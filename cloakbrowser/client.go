package cloakbrowser

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
)

// Client manages CloakBrowser instances.
type Client struct {
	binaryPath string
	port       int
	userDataDir string
	cmd        *exec.Cmd
	mu         sync.Mutex
}

// NewClient creates a new CloakBrowser client.
func NewClient(binaryPath string, port int) (*Client, error) {
	// Verify CloakBrowser binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, fmt.Errorf("CloakBrowser not found at %s: %w", binaryPath, err)
	}

	// Create user data directory
	userDataDir := fmt.Sprintf("/tmp/cloakbrowser-%d", port)
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create user data dir: %w", err)
	}

	return &Client{
		binaryPath: binaryPath,
		port:       port,
		userDataDir: userDataDir,
	}, nil
}

// Start launches the CloakBrowser process with the given fingerprint.
func (c *Client) Start(ctx context.Context, fp *fingerprint.Fingerprint) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd != nil && c.cmd.Process != nil {
		return fmt.Errorf("CloakBrowser already running on port %d", c.port)
	}

	args := c.buildArgs(fp)

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	cmd.Dir = c.userDataDir

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start CloakBrowser: %w", err)
	}

	c.cmd = cmd
	return nil
}

// Stop terminates the CloakBrowser process.
func (c *Client) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd == nil || c.cmd.Process == nil {
		return nil
	}

	if err := c.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill CloakBrowser process: %w", err)
	}

	c.cmd.Wait()
	c.cmd = nil
	return nil
}

// IsRunning checks if the CloakBrowser process is running.
func (c *Client) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd == nil || c.cmd.Process == nil {
		return false
	}

	err := c.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// GetPort returns the CDP port for this client.
func (c *Client) GetPort() int {
	return c.port
}

// buildArgs builds the CloakBrowser command-line arguments.
func (c *Client) buildArgs(fp *fingerprint.Fingerprint) []string {
	args := []string{
		"--port=" + strconv.Itoa(c.port),
		"--user-data-dir=" + c.userDataDir,
	}

	if fp != nil {
		// User-Agent
		if fp.UserAgent != "" {
			args = append(args, "--user-agent="+fp.UserAgent)
		}

		// Platform
		if fp.Platform != "" {
			args = append(args, "--platform="+fp.Platform)
		}

		// Screen dimensions
		args = append(args, fmt.Sprintf("--screen-width=%d", fp.Screen.Width))
		args = append(args, fmt.Sprintf("--screen-height=%d", fp.Screen.Height))
		args = append(args, fmt.Sprintf("--screen-pixel-ratio=%.2f", fp.Screen.PixelRatio))

		// Timezone
		if fp.Timezone != "" {
			args = append(args, "--timezone="+fp.Timezone)
		}

		// Locale
		if fp.Locale != "" {
			args = append(args, "--locale="+fp.Locale)
		}

		// Canvas fingerprint (using hash as seed)
		if fp.Canvas.Hash != "" {
			args = append(args, "--canvas-hash="+fp.Canvas.Hash)
		}

		// WebGL fingerprint
		if fp.WebGL.Renderer != "" {
			args = append(args, "--webgl-renderer="+fp.WebGL.Renderer)
		}
		if fp.WebGL.Vendor != "" {
			args = append(args, "--webgl-vendor="+fp.WebGL.Vendor)
		}

		// Audio fingerprint
		if fp.Audio.Hash != "" {
			args = append(args, "--audio-hash="+fp.Audio.Hash)
		}
	}

	// Additional CloakBrowser-specific anti-detection flags
	args = append(args,
		"--disable-blink-features=AutomationControlled",
		"--no-sandbox",
		"--disable-dev-shm-usage",
	)

	return args
}

// GetCDPEndpoint returns the CDP WebSocket endpoint URL.
func (c *Client) GetCDPEndpoint() string {
	return fmt.Sprintf("ws://localhost:%d/devtools/browser", c.port)
}

// GetWebSocketURL returns the WebSocket URL for a specific target.
func (c *Client) GetWebSocketURL(targetID string) string {
	return fmt.Sprintf("ws://localhost:%d/devtools/page/%s", c.port, targetID)
}

// ListTargets returns a list of available CDP targets (pages/tabs).
func (c *Client) ListTargets() ([]*CDPTarget, error) {
	// CloakBrowser exposes CDP targets via a JSON endpoint
	url := fmt.Sprintf("http://localhost:%d/json", c.port)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get targets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CDP endpoint returned status %d", resp.StatusCode)
	}

	var targets []*CDPTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("failed to decode targets: %w", err)
	}

	return targets, nil
}

// CDPTarget represents a Chrome DevTools Protocol target (page/tab).
type CDPTarget struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	WebSocket string `json:"webSocketDebuggerUrl"`
}

// GetFirstTargetWebSocketURL returns the WebSocket URL for the first available target.
func (c *Client) GetFirstTargetWebSocketURL() (string, error) {
	targets, err := c.ListTargets()
	if err != nil {
		return "", err
	}

	for _, t := range targets {
		if t.Type == "page" || t.Type == "webview" {
			return t.WebSocket, nil
		}
	}

	return "", fmt.Errorf("no available targets found")
}

// HealthCheck verifies the CloakBrowser process is responsive.
func (c *Client) HealthCheck(timeout time.Duration) error {
	if !c.IsRunning() {
		return fmt.Errorf("CloakBrowser process not running")
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", c.port), timeout)
	if err != nil {
		return fmt.Errorf("CDP port not accessible: %w", err)
	}
	conn.Close()

	return nil
}

// CloakFingerprint represents CloakBrowser-specific fingerprint configuration.
type CloakFingerprint struct {
	UserAgent    string
	Platform     string
	ScreenWidth  int
	ScreenHeight int
	PixelRatio   float64
	Timezone     string
	Locale       string
	CanvasMode   string
	WebGLVendor  string
	WebGLRenderer string
}

// ApplyFingerprint applies the given fingerprint to a running CloakBrowser instance.
// If the browser supports CDP fingerprint injection, it will be applied without restart.
// Otherwise, the browser will be restarted with the new fingerprint.
func (c *Client) ApplyFingerprint(ctx context.Context, fp *fingerprint.Fingerprint) error {
	if fp == nil {
		return fmt.Errorf("fingerprint is nil")
	}

	cloakFP := c.convertToCloakFormat(fp)

	// Try CDP-based fingerprint injection first
	if c.supportsCDPFingerprint() {
		if err := c.applyFingerprintViaCDP(ctx, cloakFP); err == nil {
			return nil
		}
	}

	// Fallback: restart with new fingerprint
	return c.restartWithFingerprint(ctx, cloakFP)
}

// convertToCloakFormat converts a fingerprint.Fingerprint to CloakFingerprint.
func (c *Client) convertToCloakFormat(fp *fingerprint.Fingerprint) *CloakFingerprint {
	return &CloakFingerprint{
		UserAgent:     fp.UserAgent,
		Platform:     fp.Platform,
		ScreenWidth:  fp.Screen.Width,
		ScreenHeight: fp.Screen.Height,
		PixelRatio:   fp.Screen.PixelRatio,
		Timezone:     fp.Timezone,
		Locale:       fp.Locale,
		CanvasMode:   "random",
		WebGLVendor:  fp.WebGL.Vendor,
		WebGLRenderer: fp.WebGL.Renderer,
	}
}

// supportsCDPFingerprint returns true if the browser supports CDP-based fingerprint injection.
func (c *Client) supportsCDPFingerprint() bool {
	// CloakBrowser may support CDP-based fingerprint injection
	// Return false for now, CDP injection can be implemented later
	return false
}

// applyFingerprintViaCDP attempts to apply fingerprint via CDP commands.
func (c *Client) applyFingerprintViaCDP(ctx context.Context, fp *CloakFingerprint) error {
	// CDP fingerprint injection not yet implemented
	return fmt.Errorf("CDP fingerprint injection not supported")
}

// restartWithFingerprint restarts the browser with the new fingerprint.
func (c *Client) restartWithFingerprint(ctx context.Context, fp *CloakFingerprint) error {
	// Stop the current process
	if err := c.Stop(); err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}

	// Create a minimal fingerprint for restart
	restartFP := &fingerprint.Fingerprint{
		UserAgent: fp.UserAgent,
		Platform: fp.Platform,
		Screen: fingerprint.ScreenConfig{
			Width:      fp.ScreenWidth,
			Height:     fp.ScreenHeight,
			PixelRatio: fp.PixelRatio,
		},
		Timezone: fp.Timezone,
		Locale:   fp.Locale,
		WebGL: fingerprint.WebGLConfig{
			Vendor:   fp.WebGLVendor,
			Renderer: fp.WebGLRenderer,
		},
	}

	// Restart with new fingerprint
	return c.Start(ctx, restartFP)
}
