package commands

import (
	"context"
	"fmt"
	"hash/fnv"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

// ChromePaths defines known Chrome binary locations for different platforms.
var ChromePaths = []string{
	// macOS
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
	// Windows
	`C:\Program Files\Google\Chrome\Application\chrome.exe`,
	`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	// Linux
	"/usr/bin/google-chrome",
	"/usr/bin/google-chrome-stable",
	"/usr/bin/chromium",
	"/usr/bin/chromium-browser",
}

// LocalChromeManager manages local Chrome browser instances.
type LocalChromeManager struct {
	store     *sqlite.InstanceStore
	portAlloc *instance.PortAllocator
	dataDir   string
	mu        sync.Mutex
	processes map[string]*exec.Cmd // instanceID -> cmd
}

func NewLocalChromeManager(db *sqlite.DB) *LocalChromeManager {
	return &LocalChromeManager{
		store:     sqlite.NewInstanceStore(db),
		portAlloc: instance.NewPortAllocator(9222, 65535),
		dataDir:   os.TempDir(),
		processes: make(map[string]*exec.Cmd),
	}
}

// DetectChromePath finds the Chrome binary on the system.
func DetectChromePath() (string, error) {
	if override := strings.TrimSpace(os.Getenv("BROWSER_BINARY")); override != "" {
		info, err := os.Stat(override)
		if err != nil {
			return "", fmt.Errorf("BROWSER_BINARY set to %q but not found: %w", override, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("BROWSER_BINARY points to a directory: %s", override)
		}
		return override, nil
	}

	for _, path := range ChromePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("Chrome not found in any known location")
}

func chromeReadyTimeout() time.Duration {
	if strings.TrimSpace(os.Getenv("BROWSER_BINARY")) != "" {
		return 90 * time.Second
	}
	return 30 * time.Second
}

// DetectChromeVersion returns the Chrome version string.
func DetectChromeVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// Start launches a new Chrome instance with the given configuration.
func (m *LocalChromeManager) Start(ctx context.Context, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error) {
	m.mu.Lock()

	// Detect Chrome path if not already cached
	chromePath, err := DetectChromePath()
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to detect Chrome: %w", err)
	}

	// Allocate port
	port, err := m.portAlloc.Allocate()
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("no available port: %w", err)
	}

	// Generate instance ID and user data dir
	instanceID := uuid.New().String()
	userDataDir := filepath.Join(m.dataDir, fmt.Sprintf("fingerbrower-%s", instanceID))
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to create user data dir: %w", err)
	}

	instanceName := strings.TrimSpace(cfg.Name)
	if instanceName == "" {
		if cfg.AccountLabel != "" {
			instanceName = cfg.AccountLabel
		} else {
			instanceName = fmt.Sprintf("Instance-%s", instanceID[:8])
		}
	}

	infoURL, err := m.writeInstanceInfoPage(userDataDir, instanceName, cfg, port, instanceID)
	if err != nil {
		fmt.Printf("Warning: failed to write instance info page: %v\n", err)
		infoURL = ""
	}

	// Build Chrome arguments
	args := m.buildArgs(chromePath, port, userDataDir, cfg, infoURL)

	// Start Chrome
	cmd := exec.CommandContext(ctx, chromePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}

	m.mu.Unlock()

	// Wait for Chrome to be ready
	readyCtx, cancel := context.WithTimeout(ctx, chromeReadyTimeout())
	defer cancel()

	if err := m.waitForReady(readyCtx, port); err != nil {
		cmd.Process.Kill()
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("Chrome not ready: %w", err)
	}

	if err := m.applyFingerprintOverrides(ctx, port, cfg); err != nil {
		cmd.Process.Kill()
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		return nil, err
	}

	m.mu.Lock()
	now := time.Now()
	browserInstance := &instance.BrowserInstance{
		ID:           instanceID,
		Name:         instanceName,
		Status:       instance.StatusRunning,
		Fingerprint:  cfg.Fingerprint,
		ProxyID:      "",
		ProxyURL:     "",
		AccountID:    cfg.AccountID,
		CDPEndpoint:  fmt.Sprintf("ws://localhost:%d", port),
		PID:          cmd.Process.Pid,
		Port:         port,
		UserDataDir:  userDataDir,
		Group:        cfg.Group,
		Headless:     cfg.Headless,
		StartedAt:    now,
		LastActiveAt: now,
		CreatedAt:    now,
	}
	if cfg.Proxy != nil {
		browserInstance.ProxyID = cfg.Proxy.ID
		browserInstance.ProxyURL = cfg.Proxy.URL
	}

	m.processes[instanceID] = cmd
	m.mu.Unlock()

	// Save to store
	if _, err := m.store.Save(browserInstance); err != nil {
		// Log but don't fail - instance is running
		fmt.Printf("Warning: failed to save instance to store: %v\n", err)
	}

	go m.monitorProcess(instanceID, cmd, port)

	return browserInstance, nil
}

// Stop terminates a Chrome instance.
func (m *LocalChromeManager) Stop(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get instance from store
	inst, err := m.store.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Find and kill process
	cmd, ok := m.processes[id]
	if !ok {
		// Try to find by PID
		proc, _ := os.FindProcess(inst.PID)
		if proc != nil {
			proc.Kill()
		}
	} else if cmd.Process != nil {
		cmd.Process.Kill()
	}

	// Release port
	m.portAlloc.Release(inst.Port)

	inst.Status = instance.StatusStopped
	inst.LastActiveAt = time.Now()
	if err := m.store.Update(inst); err != nil {
		return err
	}

	delete(m.processes, id)
	return nil
}

// Restart starts a stopped instance with existing configuration.
func (m *LocalChromeManager) Restart(ctx context.Context, inst *instance.BrowserInstance, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error) {
	m.mu.Lock()

	port, err := m.portAlloc.Allocate()
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("no available port: %w", err)
	}

	userDataDir := inst.UserDataDir
	if userDataDir == "" {
		userDataDir = filepath.Join(m.dataDir, fmt.Sprintf("fingerbrower-%s", inst.ID))
	}
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to create user data dir: %w", err)
	}

	instanceName := strings.TrimSpace(cfg.Name)
	if instanceName == "" {
		if cfg.AccountLabel != "" {
			instanceName = cfg.AccountLabel
		} else {
			instanceName = inst.Name
		}
	}
	if instanceName == "" {
		instanceName = fmt.Sprintf("Instance-%s", inst.ID[:8])
	}

	infoURL, err := m.writeInstanceInfoPage(userDataDir, instanceName, cfg, port, inst.ID)
	if err != nil {
		fmt.Printf("Warning: failed to write instance info page: %v\n", err)
		infoURL = ""
	}

	chromePath, err := DetectChromePath()
	if err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to detect Chrome: %w", err)
	}

	args := m.buildArgs(chromePath, port, userDataDir, cfg, infoURL)
	cmd := exec.CommandContext(ctx, chromePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to start Chrome: %w", err)
	}

	m.mu.Unlock()

	readyCtx, cancel := context.WithTimeout(ctx, chromeReadyTimeout())
	defer cancel()
	if err := m.waitForReady(readyCtx, port); err != nil {
		cmd.Process.Kill()
		m.portAlloc.Release(port)
		return nil, fmt.Errorf("Chrome not ready: %w", err)
	}

	if err := m.applyFingerprintOverrides(ctx, port, cfg); err != nil {
		cmd.Process.Kill()
		m.portAlloc.Release(port)
		return nil, err
	}

	m.mu.Lock()
	now := time.Now()
	inst.Name = instanceName
	inst.Status = instance.StatusRunning
	inst.Fingerprint = cfg.Fingerprint
	inst.ProxyID = ""
	inst.ProxyURL = ""
	if cfg.Proxy != nil {
		inst.ProxyID = cfg.Proxy.ID
		inst.ProxyURL = cfg.Proxy.URL
	}
	inst.AccountID = cfg.AccountID
	inst.CDPEndpoint = fmt.Sprintf("ws://localhost:%d", port)
	inst.PID = cmd.Process.Pid
	inst.Port = port
	inst.UserDataDir = userDataDir
	inst.Group = cfg.Group
	inst.Headless = cfg.Headless
	inst.StartedAt = now
	inst.LastActiveAt = now
	if err := m.store.Update(inst); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	m.processes[inst.ID] = cmd
	m.mu.Unlock()

	go m.monitorProcess(inst.ID, cmd, port)

	return inst, nil
}

// Delete removes a stopped instance and its data directory.
func (m *LocalChromeManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, err := m.store.Get(id)
	if err != nil {
		return err
	}
	if inst.UserDataDir != "" {
		if err := os.RemoveAll(inst.UserDataDir); err != nil {
			return fmt.Errorf("failed to remove user data dir: %w", err)
		}
	}
	if err := m.store.Delete(id); err != nil {
		return err
	}
	delete(m.processes, id)
	return nil
}

// buildArgs constructs Chrome command-line arguments.
func (m *LocalChromeManager) buildArgs(chromePath string, port int, userDataDir string, cfg *instance.InstanceConfig, infoURL string) []string {
	args := []string{
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--user-data-dir=" + userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-sync",
		"--disable-translate",
	}

	// Add fingerprint parameters if provided
	if cfg.Fingerprint != nil {
		fp := cfg.Fingerprint
		useSelfBuilt := strings.TrimSpace(os.Getenv("BROWSER_BINARY")) != ""
		if fp.UserAgent != "" {
			args = append(args, "--user-agent="+fp.UserAgent)
		}
		if useSelfBuilt {
			if fp.Platform != "" {
				args = append(args, "--platform="+fp.Platform)
			}
			if fp.Locale != "" {
				args = append(args, "--locale="+fp.Locale)
			}
			if fp.Timezone != "" {
				args = append(args, "--timezone="+fp.Timezone)
			}
			if fp.Seed != "" {
				args = append(args, "--fingerprint="+fp.Seed)
				args = append(args, "--fingerprint-noise=1")
			}
			if fp.Screen.Width > 0 {
				args = append(args, "--screen-width="+strconv.Itoa(fp.Screen.Width))
				args = append(args, "--screen-avail-width="+strconv.Itoa(fp.Screen.Width))
			}
			if fp.Screen.Height > 0 {
				args = append(args, "--screen-height="+strconv.Itoa(fp.Screen.Height))
				args = append(args, "--screen-avail-height="+strconv.Itoa(fp.Screen.Height))
			}
			if fp.Screen.PixelRatio > 0 {
				args = append(args, "--device-pixel-ratio="+strconv.FormatFloat(fp.Screen.PixelRatio, 'f', -1, 64))
			}
			if fp.Screen.Width > 0 || fp.Screen.Height > 0 {
				args = append(args, "--screen-avail-left=0", "--screen-avail-top=0")
			}
			if fp.Hardware.CPUCores > 0 {
				args = append(args, "--hardware-concurrency="+strconv.Itoa(fp.Hardware.CPUCores))
			}
			if fp.Hardware.MemoryGB > 0 {
				args = append(args, "--device-memory="+strconv.Itoa(fp.Hardware.MemoryGB))
			}
			if fp.WebGL.Vendor != "" {
				args = append(args, "--webgl-vendor="+fp.WebGL.Vendor)
			}
			if fp.WebGL.Renderer != "" {
				args = append(args, "--webgl-renderer="+fp.WebGL.Renderer)
			}
			if len(fp.WebGL.Extensions) > 0 {
				args = append(args, "--webgl-extensions="+strings.Join(fp.WebGL.Extensions, ","))
			}
			if fp.Seed != "" {
				if seed := fingerprintSeedInt(fp.Seed); seed > 0 {
					args = append(args, "--audio-fingerprint-seed="+strconv.Itoa(seed))
				}
			}
		}
	}

	// Proxy configuration
	if cfg.Proxy != nil && cfg.Proxy.URL != "" {
		args = append(args, "--proxy-server="+cfg.Proxy.URL)
	}

	// Headless mode
	if cfg.Headless {
		args = append(args, "--headless=new")
	}

	// Platform-specific args
	if runtime.GOOS == "darwin" {
		args = append(args, "--disable-background-networking")
		args = append(args, "--disable-default-apps")
		args = append(args, "--disable-os-applications")
	}

	if infoURL != "" {
		args = append(args, "--new-window", infoURL)
	}

	return args
}

func (m *LocalChromeManager) monitorProcess(instanceID string, cmd *exec.Cmd, port int) {
	err := cmd.Wait()
	if err != nil {
		fmt.Printf("Instance %s exited: %v\n", instanceID, err)
	}

	m.mu.Lock()
	current, ok := m.processes[instanceID]
	if !ok || current != cmd {
		m.mu.Unlock()
		return
	}
	delete(m.processes, instanceID)
	m.portAlloc.Release(port)

	inst, err := m.store.Get(instanceID)
	if err != nil {
		m.mu.Unlock()
		return
	}
	inst.Status = instance.StatusStopped
	inst.LastActiveAt = time.Now()
	if err := m.store.Update(inst); err != nil {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
}

func fingerprintSeedInt(seed string) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(seed))
	return int(hasher.Sum32() & 0x7fffffff)
}

func (m *LocalChromeManager) applyFingerprintOverrides(ctx context.Context, port int, cfg *instance.InstanceConfig) error {
	if cfg == nil || cfg.Fingerprint == nil {
		return nil
	}
	// Note: Timezone is already set via Chrome command-line args (--timezone=) in buildChromeArgs.
	// We skip CDP timezone override to avoid "Timezone override is already in effect" error.
	// If timezone is empty in config, it means no timezone override was requested.
	timezone := strings.TrimSpace(cfg.Fingerprint.Timezone)
	if timezone == "" {
		return nil
	}
	// Timezone was set via command line, skip CDP override
	return nil
}

// waitForReady waits for Chrome to be ready to accept CDP connections.
func (m *LocalChromeManager) waitForReady(ctx context.Context, port int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	url := fmt.Sprintf("http://localhost:%d/json", port)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				return nil
			}
		}
	}
}

// isPortOpen checks if a port is accepting connections.
func isPortOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

type instanceInfoData struct {
	Title               string
	InstanceID          string
	AccountID           string
	AccountLabel        string
	ProxyURL            string
	Port                int
	Group               string
	FingerprintSeed     string
	FingerprintPlatform string
	Timezone            string
}

func (m *LocalChromeManager) writeInstanceInfoPage(userDataDir string, instanceName string, cfg *instance.InstanceConfig, port int, instanceID string) (string, error) {
	infoPath := filepath.Join(userDataDir, "instance-info.html")
	fpSeed := ""
	fpPlatform := ""
	fpTimezone := ""
	if cfg.Fingerprint != nil {
		fpSeed = cfg.Fingerprint.Seed
		fpPlatform = cfg.Fingerprint.Platform
		fpTimezone = cfg.Fingerprint.Timezone
	}

	data := instanceInfoData{
		Title:               instanceName,
		InstanceID:          instanceID,
		AccountID:           cfg.AccountID,
		AccountLabel:        cfg.AccountLabel,
		ProxyURL:            "",
		Port:                port,
		Group:               cfg.Group,
		FingerprintSeed:     fpSeed,
		FingerprintPlatform: fpPlatform,
		Timezone:            fpTimezone,
	}
	if cfg.Proxy != nil {
		data.ProxyURL = cfg.Proxy.URL
	}

	tpl := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <title>{{.Title}}</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 24px; color: #111827; }
    h1 { font-size: 20px; margin-bottom: 12px; }
    .meta { display: grid; grid-template-columns: 140px 1fr; row-gap: 8px; column-gap: 12px; }
    .label { color: #6b7280; }
    .value { color: #111827; word-break: break-all; }
  </style>
</head>
<body>
  <h1>{{.Title}}</h1>
  <div class="meta">
    <div class="label">Instance ID</div><div class="value">{{.InstanceID}}</div>
    <div class="label">Account</div><div class="value">{{if .AccountLabel}}{{.AccountLabel}}{{else}}-{{end}}</div>
    <div class="label">Account ID</div><div class="value">{{if .AccountID}}{{.AccountID}}{{else}}-{{end}}</div>
    <div class="label">Proxy</div><div class="value">{{if .ProxyURL}}{{.ProxyURL}}{{else}}-{{end}}</div>
    <div class="label">Port</div><div class="value">{{.Port}}</div>
    <div class="label">Group</div><div class="value">{{if .Group}}{{.Group}}{{else}}-{{end}}</div>
    <div class="label">FP Seed</div><div class="value">{{if .FingerprintSeed}}{{.FingerprintSeed}}{{else}}-{{end}}</div>
    <div class="label">FP Platform</div><div class="value">{{if .FingerprintPlatform}}{{.FingerprintPlatform}}{{else}}-{{end}}</div>
    <div class="label">Timezone</div><div class="value">{{if .Timezone}}{{.Timezone}}{{else}}-{{end}}</div>
  </div>
</body>
</html>`

	t, err := template.New("instance-info").Parse(tpl)
	if err != nil {
		return "", err
	}
	file, err := os.Create(infoPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := t.Execute(file, data); err != nil {
		return "", err
	}

	u := url.URL{Scheme: "file", Path: infoPath}
	return u.String(), nil
}
