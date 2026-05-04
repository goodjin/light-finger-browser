package cloakbrowser

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
)

// Client manages CloakBrowser instances.
type Client struct {
	binaryPath  string
	port        int
	userDataDir string
	cmd         *exec.Cmd
	mu          sync.Mutex
}

type LaunchOptions struct {
	ProxyURL string
	Headless bool
	StartURL string
}

// NewClient creates a new CloakBrowser client.
func NewClient(binaryPath string, port int) (*Client, error) {
	userDataDir := fmt.Sprintf("/tmp/cloakbrowser-%d", port)
	return NewClientWithUserDataDir(binaryPath, port, userDataDir)
}

// NewClientWithUserDataDir creates a new CloakBrowser client with a fixed user data dir.
func NewClientWithUserDataDir(binaryPath string, port int, userDataDir string) (*Client, error) {
	// Verify CloakBrowser binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, fmt.Errorf("CloakBrowser not found at %s: %w", binaryPath, err)
	}

	// Create user data directory
	if err := os.MkdirAll(userDataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create user data dir: %w", err)
	}

	return &Client{
		binaryPath:  binaryPath,
		port:        port,
		userDataDir: userDataDir,
	}, nil
}

// Start launches the CloakBrowser process with the given fingerprint.
func (c *Client) Start(ctx context.Context, fp *fingerprint.Fingerprint) error {
	return c.StartWithOptions(ctx, fp, nil)
}

// StartWithOptions launches the CloakBrowser process with fingerprint and launch options.
func (c *Client) StartWithOptions(ctx context.Context, fp *fingerprint.Fingerprint, opts *LaunchOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd != nil && c.cmd.Process != nil {
		return fmt.Errorf("CloakBrowser already running on port %d", c.port)
	}

	args := c.buildArgs(fp, opts)

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	cmd.Dir = c.userDataDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start CloakBrowser: %w", err)
	}

	c.cmd = cmd
	return nil
}

// Stop terminates the CloakBrowser process.
func (c *Client) Stop() error {
	c.mu.Lock()
	cmd := c.cmd
	c.cmd = nil
	c.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill CloakBrowser process: %w", err)
	}
	_ = cmd.Wait()
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
func (c *Client) buildArgs(fp *fingerprint.Fingerprint, opts *LaunchOptions) []string {
	args := []string{
		"--port=" + strconv.Itoa(c.port),
		"--remote-debugging-port=" + strconv.Itoa(c.port),
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

		// Locale
		if fp.Locale != "" {
			args = append(args, "--locale="+fp.Locale)
		}

		// Timezone
		if fp.Timezone != "" {
			args = append(args, "--timezone="+fp.Timezone)
		}

		// Canvas fingerprint seed
		if fp.Seed != "" {
			args = append(args, "--fingerprint="+fp.Seed)
			args = append(args, "--fingerprint-noise=1")
		}

		// Screen dimensions
		if fp.Screen.Width > 0 {
			args = append(args, fmt.Sprintf("--screen-width=%d", fp.Screen.Width))
			args = append(args, fmt.Sprintf("--screen-avail-width=%d", fp.Screen.Width))
		}
		if fp.Screen.Height > 0 {
			args = append(args, fmt.Sprintf("--screen-height=%d", fp.Screen.Height))
			args = append(args, fmt.Sprintf("--screen-avail-height=%d", fp.Screen.Height))
		}
		if fp.Screen.PixelRatio > 0 {
			pixelRatio := strconv.FormatFloat(fp.Screen.PixelRatio, 'f', -1, 64)
			args = append(args, "--device-pixel-ratio="+pixelRatio)
			args = append(args, "--screen-pixel-ratio="+pixelRatio)
		}
		if fp.Screen.Width > 0 || fp.Screen.Height > 0 {
			args = append(args, "--screen-avail-left=0", "--screen-avail-top=0")
		}

		// Hardware overrides
		if fp.Hardware.CPUCores > 0 {
			args = append(args, "--hardware-concurrency="+strconv.Itoa(fp.Hardware.CPUCores))
		}
		if fp.Hardware.MemoryGB > 0 {
			args = append(args, "--device-memory="+strconv.Itoa(fp.Hardware.MemoryGB))
		}

		// WebGL fingerprint
		if fp.WebGL.Renderer != "" {
			args = append(args, "--webgl-renderer="+fp.WebGL.Renderer)
		}
		if fp.WebGL.Vendor != "" {
			args = append(args, "--webgl-vendor="+fp.WebGL.Vendor)
		}
		if len(fp.WebGL.Extensions) > 0 {
			args = append(args, "--webgl-extensions="+strings.Join(fp.WebGL.Extensions, ","))
		}

		if fp.Seed != "" {
			if seed := fingerprintSeedInt(fp.Seed); seed > 0 {
				args = append(args, "--audio-fingerprint-seed="+strconv.Itoa(seed))
			}
		}

		// Legacy hash flags (older CloakBrowser builds)
		if fp.Canvas.Hash != "" {
			args = append(args, "--canvas-hash="+fp.Canvas.Hash)
		}
		if fp.Audio.Hash != "" {
			args = append(args, "--audio-hash="+fp.Audio.Hash)
		}
	}

	if opts != nil {
		if opts.ProxyURL != "" {
			args = append(args, "--proxy-server="+opts.ProxyURL)
		}
		if opts.Headless {
			args = append(args, "--headless=new")
		}
	}

	// Additional CloakBrowser-specific anti-detection flags
	args = append(args,
		"--no-sandbox",
		"--disable-dev-shm-usage",
	)

	if opts != nil && opts.StartURL != "" {
		args = append(args, "--new-window", opts.StartURL)
	}

	return args
}

func fingerprintSeedInt(seed string) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(seed))
	return int(hasher.Sum32() & 0x7fffffff)
}

// Command returns the current process command.
func (c *Client) Command() *exec.Cmd {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cmd
}

// PID returns the current process id if running.
func (c *Client) PID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd == nil || c.cmd.Process == nil {
		return 0
	}
	return c.cmd.Process.Pid
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
	UserAgent     string
	Platform      string
	ScreenWidth   int
	ScreenHeight  int
	PixelRatio    float64
	Timezone      string
	Locale        string
	CanvasMode    string
	WebGLVendor   string
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
		Platform:      fp.Platform,
		ScreenWidth:   fp.Screen.Width,
		ScreenHeight:  fp.Screen.Height,
		PixelRatio:    fp.Screen.PixelRatio,
		Timezone:      fp.Timezone,
		Locale:        fp.Locale,
		CanvasMode:    "random",
		WebGLVendor:   fp.WebGL.Vendor,
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
		Platform:  fp.Platform,
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
