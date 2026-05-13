package instance

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// ProcessManager handles browser process lifecycle.
type ProcessManager struct {
	binaryPath string
	dataDir    string
	portAlloc  *PortAllocator
	store      Store
	mu         sync.Mutex
	processes  map[string]*exec.Cmd
	// readyFunc is an optional function to check if process is ready.
	// If nil, defaults to checking TCP port availability.
	readyFunc func(port int) bool
}

// NewProcessManager creates a new ProcessManager.
func NewProcessManager(binaryPath, dataDir string, portAlloc *PortAllocator, store Store) *ProcessManager {
	return &ProcessManager{
		binaryPath: binaryPath,
		dataDir:    dataDir,
		portAlloc:  portAlloc,
		store:      store,
		processes:  make(map[string]*exec.Cmd),
		readyFunc:  nil, // defaults to TCP port check
	}
}

// SetReadyFunc sets a custom readiness check function.
// This is useful for testing without requiring real TCP connections.
func (pm *ProcessManager) SetReadyFunc(f func(port int) bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.readyFunc = f
}

// Start launches a new browser instance.
func (pm *ProcessManager) Start(ctx context.Context, cfg *InstanceConfig) (*BrowserInstance, error) {
	pm.mu.Lock()

	// 1. Allocate port
	port, err := pm.portAlloc.Allocate()
	if err != nil {
		pm.mu.Unlock()
		return nil, fmt.Errorf("failed to allocate port: %w", err)
	}

	// 2. Create user data directory
	uuidVal, err := NewUUID()
	if err != nil {
		pm.portAlloc.Release(port)
		pm.mu.Unlock()
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}
	userDataDir := filepath.Join(pm.dataDir, uuidVal.String())
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		pm.portAlloc.Release(port)
		pm.mu.Unlock()
		return nil, fmt.Errorf("failed to create user data dir: %w", err)
	}

	// 3. Construct launch arguments
	args := pm.buildArgs(port, userDataDir, cfg)

	// 4. Start process
	cmd := exec.CommandContext(ctx, pm.binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		pm.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		pm.mu.Unlock()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// 5. Wait for process to be ready (release lock to avoid deadlock)
	pm.mu.Unlock()

	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := pm.waitForReady(readyCtx, port); err != nil {
		cmd.Process.Kill()
		pm.portAlloc.Release(port)
		os.RemoveAll(userDataDir)
		return nil, fmt.Errorf("process not ready: %w", err)
	}

	pm.mu.Lock()
	now := time.Now()
	instance := &BrowserInstance{
		ID:           uuidVal.String(),
		Name:         cfg.Name,
		Status:       StatusRunning,
		Fingerprint:  cfg.Fingerprint,
		ProxyID:      "", // Default empty
		AccountID:    cfg.AccountID,
		CDPEndpoint:  fmt.Sprintf("ws://localhost:%d", port),
		PID:          cmd.Process.Pid,
		Port:         port,
		UserDataDir:  userDataDir,
		Group:        cfg.Group,
		StartedAt:    now,
		LastActiveAt: now,
		CreatedAt:    now,
	}
	if cfg.Proxy != nil {
		instance.ProxyID = cfg.Proxy.ID
	}

	pm.processes[instance.ID] = cmd
	pm.mu.Unlock()

	return instance, nil
}

// Stop terminates a browser instance gracefully.
func (pm *ProcessManager) Stop(ctx context.Context, id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	instance, err := pm.store.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	proc, _ := os.FindProcess(instance.PID)
	if proc != nil {
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}

		// Wait for process to exit with timeout
		done := make(chan error, 1)
		go func() {
			_, err := proc.Wait()
			done <- err
		}()

		select {
		case <-ctx.Done():
			// Force kill if timeout
			proc.Kill()
			return ctx.Err()
		case err := <-done:
			if err != nil {
				return fmt.Errorf("process wait error: %w", err)
			}
		}
	}

	// Release port
	pm.portAlloc.Release(instance.Port)

	// Cleanup user data directory
	if err := os.RemoveAll(instance.UserDataDir); err != nil {
		return fmt.Errorf("failed to remove user data dir: %w", err)
	}

	delete(pm.processes, id)

	return nil
}

// buildArgs constructs the command-line arguments for the browser.
func (pm *ProcessManager) buildArgs(port int, userDataDir string, cfg *InstanceConfig) []string {
	args := []string{
		"--remote-debugging-port=" + strconv.Itoa(port),
		"--user-data-dir=" + userDataDir,
	}

	if cfg.Proxy != nil && cfg.Proxy.URL != "" {
		args = append(args, "--proxy-server="+cfg.Proxy.URL)
	}

	if cfg.Headless {
		args = append(args, "--headless")
	}

	return args
}

// waitForReady waits for the browser process to be ready to accept connections.
func (pm *ProcessManager) waitForReady(ctx context.Context, port int) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if pm.checkReady(port) {
				return nil
			}
		}
	}
}

// checkReady checks if the process is ready.
// Uses custom readyFunc if set, otherwise defaults to TCP port check.
func (pm *ProcessManager) checkReady(port int) bool {
	pm.mu.Lock()
	ready := pm.readyFunc
	pm.mu.Unlock()
	if ready != nil {
		return ready(port)
	}
	return pm.isPortOpen(port)
}

// isPortOpen checks if a port is open and accepting connections.
func (pm *ProcessManager) isPortOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetProcess returns the process command for a given instance ID.
func (pm *ProcessManager) GetProcess(id string) *exec.Cmd {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.processes[id]
}
