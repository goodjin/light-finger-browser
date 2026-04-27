package cloakbrowser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
)

func TestNewClient(t *testing.T) {
	// Create a dummy binary for testing
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\nsleep 60\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	tests := []struct {
		name      string
		binaryPath string
		port      int
		wantErr   bool
	}{
		{
			name:      "valid binary",
			binaryPath: binaryPath,
			port:      9222,
			wantErr:   false,
		},
		{
			name:      "binary not found",
			binaryPath: "/nonexistent/cloakbrowser",
			port:      9223,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.binaryPath, tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if client == nil {
					t.Error("NewClient() returned nil client without error")
				}
				if client.GetPort() != tt.port {
					t.Errorf("NewClient() port = %d, want %d", client.GetPort(), tt.port)
				}
			}
		})
	}
}

func TestClientStartStop(t *testing.T) {
	// Create a dummy binary that exits quickly
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	// Script that just sleeps then exits
	script := `#!/bin/bash
sleep 2
exit 0
`
	if err := os.WriteFile(binaryPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19222)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Test Start
	fp := &fingerprint.Fingerprint{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
		Platform:  "Windows",
		Screen:    fingerprint.ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
		Timezone:  "America/New_York",
		Locale:    "en-US",
	}

	ctx := context.Background()
	if err := client.Start(ctx, fp); err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if !client.IsRunning() {
		t.Error("IsRunning() = false after Start()")
	}

	// Test Stop
	if err := client.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Wait a bit for process to be killed
	time.Sleep(100 * time.Millisecond)

	if client.IsRunning() {
		t.Error("IsRunning() = true after Stop()")
	}
}

func TestClientBuildArgs(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 9222)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	fp := &fingerprint.Fingerprint{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
		Platform:  "Windows",
		Screen:    fingerprint.ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
		Timezone:  "America/New_York",
		Locale:    "en-US",
		Canvas:    fingerprint.CanvasConfig{Hash: "abc123"},
		WebGL:     fingerprint.WebGLConfig{Renderer: "NVIDIA GeForce GTX 1070", Vendor: "NVIDIA"},
		Audio:     fingerprint.AudioConfig{Hash: "def456"},
	}

	args := client.buildArgs(fp)

	// Verify key arguments are present
	argMap := make(map[string]bool)
	for _, arg := range args {
		argMap[arg] = true
	}

	if !argMap["--port=9222"] {
		t.Error("buildArgs() missing --port argument")
	}
	if !argMap["--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"] {
		t.Error("buildArgs() missing user-agent argument")
	}
	if !argMap["--screen-width=1920"] {
		t.Error("buildArgs() missing screen-width argument")
	}
	if !argMap["--canvas-hash=abc123"] {
		t.Error("buildArgs() missing canvas-hash argument")
	}
}

func TestClientGetCDPEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 9222)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	endpoint := client.GetCDPEndpoint()
	expected := "ws://localhost:9222/devtools/browser"
	if endpoint != expected {
		t.Errorf("GetCDPEndpoint() = %s, want %s", endpoint, expected)
	}
}

func TestClientGetWebSocketURL(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 9222)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	url := client.GetWebSocketURL("tab-123")
	expected := "ws://localhost:9222/devtools/page/tab-123"
	if url != expected {
		t.Errorf("GetWebSocketURL() = %s, want %s", url, expected)
	}
}
