package commands

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

type SelfBuiltBrowserManager struct {
	store      *sqlite.InstanceStore
	portAlloc  *instance.PortAllocator
	dataDir    string
	binaryPath string
	mu         sync.Mutex
	processes  map[string]*exec.Cmd // instanceID -> cmd
}

func NewSelfBuiltBrowserManager(db *sqlite.DB, binaryPath string) *SelfBuiltBrowserManager {
	return &SelfBuiltBrowserManager{
		store:      sqlite.NewInstanceStore(db),
		portAlloc:  instance.NewPortAllocator(9222, 65535),
		dataDir:    os.TempDir(),
		binaryPath: strings.TrimSpace(binaryPath),
		processes:  make(map[string]*exec.Cmd),
	}
}

func (m *SelfBuiltBrowserManager) Start(ctx context.Context, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error) {
	if strings.TrimSpace(m.binaryPath) == "" {
		return nil, fmt.Errorf("self-built browser path is not configured; set BROWSER_BINARY, or use BROWSER_ENGINE=local for development fallback")
	}

	m.mu.Lock()
	port, err := m.portAlloc.Allocate()
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("no available port: %w", err)
	}

	instanceID := uuid.New().String()
	userDataDir := filepath.Join(m.dataDir, fmt.Sprintf("fingerbrower-%s", instanceID))
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to create user data dir: %w", err)
	}

	instanceName := browserInstanceName(cfg, instanceID)
	infoURL, err := writeBrowserInstanceInfoPage(userDataDir, instanceName, cfg, port, instanceID)
	if err != nil {
		fmt.Printf("Warning: failed to write instance info page: %v\n", err)
		infoURL = ""
	}

	cmd, err := m.startBrowserProcess(ctx, port, userDataDir, cfg, infoURL)
	if err != nil {
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		m.mu.Unlock()
		return nil, err
	}
	m.mu.Unlock()

	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := waitForBrowserReady(readyCtx, port); err != nil {
		_ = cmd.Process.Kill()
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("browser not ready: %w", err)
	}

	if err := applyFingerprintOverrides(ctx, port, cfg); err != nil {
		_ = cmd.Process.Kill()
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		return nil, err
	}

	now := time.Now()
	inst := &instance.BrowserInstance{
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
		inst.ProxyID = cfg.Proxy.ID
		inst.ProxyURL = cfg.Proxy.URL
	}

	m.mu.Lock()
	m.processes[instanceID] = cmd
	m.mu.Unlock()

	if _, err := m.store.Save(inst); err != nil {
		fmt.Printf("Warning: failed to save instance to store: %v\n", err)
	}

	go m.monitorProcess(instanceID, cmd, port)

	return inst, nil
}

func (m *SelfBuiltBrowserManager) Stop(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, err := m.store.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	if cmd, ok := m.processes[id]; ok {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
	} else {
		proc, _ := os.FindProcess(inst.PID)
		if proc != nil {
			_ = proc.Kill()
		}
	}

	m.portAlloc.Release(inst.Port)
	inst.Status = instance.StatusStopped
	inst.LastActiveAt = time.Now()
	if err := m.store.Update(inst); err != nil {
		return err
	}

	delete(m.processes, id)
	return nil
}

func (m *SelfBuiltBrowserManager) Restart(ctx context.Context, inst *instance.BrowserInstance, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error) {
	if strings.TrimSpace(m.binaryPath) == "" {
		return nil, fmt.Errorf("self-built browser path is not configured; set BROWSER_BINARY environment variable")
	}

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
		instanceName = browserInstanceName(cfg, inst.ID)
	}

	infoURL, err := writeBrowserInstanceInfoPage(userDataDir, instanceName, cfg, port, inst.ID)
	if err != nil {
		fmt.Printf("Warning: failed to write instance info page: %v\n", err)
		infoURL = ""
	}

	cmd, err := m.startBrowserProcess(ctx, port, userDataDir, cfg, infoURL)
	if err != nil {
		m.portAlloc.Release(port)
		m.mu.Unlock()
		return nil, err
	}
	m.mu.Unlock()

	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := waitForBrowserReady(readyCtx, port); err != nil {
		_ = cmd.Process.Kill()
		m.portAlloc.Release(port)
		return nil, fmt.Errorf("browser not ready: %w", err)
	}

	if err := applyFingerprintOverrides(ctx, port, cfg); err != nil {
		_ = cmd.Process.Kill()
		m.portAlloc.Release(port)
		return nil, err
	}

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
		return nil, err
	}

	m.mu.Lock()
	m.processes[inst.ID] = cmd
	m.mu.Unlock()
	go m.monitorProcess(inst.ID, cmd, port)

	return inst, nil
}

func (m *SelfBuiltBrowserManager) Delete(id string) error {
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

func (m *SelfBuiltBrowserManager) startBrowserProcess(ctx context.Context, port int, userDataDir string, cfg *instance.InstanceConfig, infoURL string) (*exec.Cmd, error) {
	args := m.buildArgs(port, userDataDir, cfg, infoURL)
	cmd := exec.CommandContext(ctx, m.binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start browser: %w", err)
	}

	return cmd, nil
}

func (m *SelfBuiltBrowserManager) buildArgs(port int, userDataDir string, cfg *instance.InstanceConfig, infoURL string) []string {
	args := []string{
		"--remote-debugging-port=" + fmt.Sprintf("%d", port),
		"--user-data-dir=" + userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-sync",
		"--disable-translate",
	}

	if cfg.Fingerprint != nil {
		fp := cfg.Fingerprint
		if fp.UserAgent != "" {
			args = append(args, "--user-agent="+fp.UserAgent)
		}
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
			args = append(args, "--screen-width="+fmt.Sprintf("%d", fp.Screen.Width))
			args = append(args, "--screen-avail-width="+fmt.Sprintf("%d", fp.Screen.Width))
		}
		if fp.Screen.Height > 0 {
			args = append(args, "--screen-height="+fmt.Sprintf("%d", fp.Screen.Height))
			args = append(args, "--screen-avail-height="+fmt.Sprintf("%d", fp.Screen.Height))
		}
		if fp.Screen.PixelRatio > 0 {
			args = append(args, "--device-pixel-ratio="+fmt.Sprintf("%f", fp.Screen.PixelRatio))
		}
		if fp.Screen.Width > 0 || fp.Screen.Height > 0 {
			args = append(args, "--screen-avail-left=0", "--screen-avail-top=0")
		}
		if fp.Hardware.CPUCores > 0 {
			args = append(args, "--hardware-concurrency="+fmt.Sprintf("%d", fp.Hardware.CPUCores))
		}
		if fp.Hardware.MemoryGB > 0 {
			args = append(args, "--device-memory="+fmt.Sprintf("%d", fp.Hardware.MemoryGB))
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
				args = append(args, "--audio-fingerprint-seed="+fmt.Sprintf("%d", seed))
			}
		}
	}

	if cfg.Proxy != nil && cfg.Proxy.URL != "" {
		args = append(args, "--proxy-server="+cfg.Proxy.URL)
	}

	if cfg.Headless {
		args = append(args, "--headless=new")
	}

	if infoURL != "" {
		args = append(args, "--new-window", infoURL)
	}

	return args
}

func (m *SelfBuiltBrowserManager) monitorProcess(instanceID string, cmd *exec.Cmd, port int) {
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

func waitForBrowserReady(ctx context.Context, port int) error {
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

func applyFingerprintOverrides(ctx context.Context, port int, cfg *instance.InstanceConfig) error {
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

func browserInstanceName(cfg *instance.InstanceConfig, fallbackID string) string {
	instanceName := strings.TrimSpace(cfg.Name)
	if instanceName == "" {
		if cfg.AccountLabel != "" {
			instanceName = cfg.AccountLabel
		} else {
			instanceName = fmt.Sprintf("Instance-%s", fallbackID[:8])
		}
	}
	return instanceName
}

type browserInstanceInfoData struct {
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

func writeBrowserInstanceInfoPage(userDataDir string, instanceName string, cfg *instance.InstanceConfig, port int, instanceID string) (string, error) {
	infoPath := filepath.Join(userDataDir, "instance-info.html")
	fpSeed := ""
	fpPlatform := ""
	fpTimezone := ""
	if cfg.Fingerprint != nil {
		fpSeed = cfg.Fingerprint.Seed
		fpPlatform = cfg.Fingerprint.Platform
		fpTimezone = cfg.Fingerprint.Timezone
	}

	data := browserInstanceInfoData{
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
