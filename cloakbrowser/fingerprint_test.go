package cloakbrowser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmos/fingerbrower/fingerprint"
)

func TestApplyFingerprint(t *testing.T) {
	// Create a dummy binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	script := `#!/bin/bash
sleep 10
exit 0
`
	if err := os.WriteFile(binaryPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19223)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Start the browser first
	fp := &fingerprint.Fingerprint{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
		Platform:  "Windows",
		Screen:    fingerprint.ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
		Timezone:  "America/New_York",
		Locale:    "en-US",
	}

	ctx := context.Background()
	if err := client.Start(ctx, fp); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer client.Stop()

	// Apply a new fingerprint
	newFP := &fingerprint.Fingerprint{
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15",
		Platform:  "Mac",
		Screen:    fingerprint.ScreenConfig{Width: 2560, Height: 1440, PixelRatio: 1.0},
		Timezone:  "America/Los_Angeles",
		Locale:    "en-US",
		Canvas:    fingerprint.CanvasConfig{Hash: "new-canvas-hash"},
		WebGL:     fingerprint.WebGLConfig{Renderer: "Apple M1", Vendor: "Apple"},
	}

	if err := client.ApplyFingerprint(ctx, newFP); err != nil {
		t.Errorf("ApplyFingerprint() error = %v", err)
	}
}

func TestApplyFingerprintNil(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19224)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.ApplyFingerprint(context.Background(), nil)
	if err == nil {
		t.Error("ApplyFingerprint(nil) expected error, got nil")
	}
}

func TestConvertToCloakFormat(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19225)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	fp := &fingerprint.Fingerprint{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
		Platform:  "Windows",
		Screen:    fingerprint.ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.25},
		Timezone:  "America/New_York",
		Locale:    "en-US",
		Canvas:    fingerprint.CanvasConfig{Hash: "canvas-hash"},
		WebGL:     fingerprint.WebGLConfig{Renderer: "NVIDIA GeForce GTX 1070", Vendor: "NVIDIA"},
		Audio:     fingerprint.AudioConfig{Hash: "audio-hash"},
	}

	cloakFP := client.convertToCloakFormat(fp)

	if cloakFP.UserAgent != fp.UserAgent {
		t.Errorf("UserAgent = %s, want %s", cloakFP.UserAgent, fp.UserAgent)
	}
	if cloakFP.Platform != fp.Platform {
		t.Errorf("Platform = %s, want %s", cloakFP.Platform, fp.Platform)
	}
	if cloakFP.ScreenWidth != fp.Screen.Width {
		t.Errorf("ScreenWidth = %d, want %d", cloakFP.ScreenWidth, fp.Screen.Width)
	}
	if cloakFP.CanvasMode != "random" {
		t.Errorf("CanvasMode = %s, want random", cloakFP.CanvasMode)
	}
}

func TestSupportsCDPFingerprint(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19226)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.supportsCDPFingerprint() {
		t.Error("supportsCDPFingerprint() = true, want false")
	}
}
