package commands

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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
	for _, path := range ChromePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("Chrome not found in any known location")
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

	// Build Chrome arguments
	args := m.buildArgs(chromePath, port, userDataDir, cfg)

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
	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := m.waitForReady(readyCtx, port); err != nil {
		cmd.Process.Kill()
		m.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("Chrome not ready: %w", err)
	}

	m.mu.Lock()
	now := time.Now()
	browserInstance := &instance.BrowserInstance{
		ID:          instanceID,
		Status:      instance.StatusRunning,
		Fingerprint: cfg.Fingerprint,
		ProxyID:     "",
		AccountID:   cfg.AccountID,
		CDPEndpoint: fmt.Sprintf("ws://localhost:%d", port),
		PID:         cmd.Process.Pid,
		Port:        port,
		UserDataDir: userDataDir,
		Group:       cfg.Group,
		StartedAt:    now,
		LastActiveAt: now,
		CreatedAt:    now,
	}
	if cfg.Proxy != nil {
		browserInstance.ProxyID = cfg.Proxy.ID
	}

	m.processes[instanceID] = cmd
	m.mu.Unlock()

	// Save to store
	if _, err := m.store.Save(browserInstance); err != nil {
		// Log but don't fail - instance is running
		fmt.Printf("Warning: failed to save instance to store: %v\n", err)
	}

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

	// Cleanup user data dir
	os.RemoveAll(inst.UserDataDir)

	// Delete from store
	m.store.Delete(id)

	delete(m.processes, id)
	return nil
}

// buildArgs constructs Chrome command-line arguments.
func (m *LocalChromeManager) buildArgs(chromePath string, port int, userDataDir string, cfg *instance.InstanceConfig) []string {
	args := []string{
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--user-data-dir=" + userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-blink-features=AutomationControlled",
		"--disable-dev-shm-usage",
		"--disable-extensions",
		"--disable-sync",
		"--disable-translate",
	}

	// Add fingerprint parameters if provided
	if cfg.Fingerprint != nil {
		fp := cfg.Fingerprint
		if fp.UserAgent != "" {
			args = append(args, "--user-agent="+fp.UserAgent)
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

	return args
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
