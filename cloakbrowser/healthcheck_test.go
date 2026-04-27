package cloakbrowser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsRunning(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	// Create a binary that runs for a while
	script := `#!/bin/bash
sleep 5
exit 0
`
	if err := os.WriteFile(binaryPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19230)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Before start, should not be running
	if client.IsRunning() {
		t.Error("IsRunning() = true before Start()")
	}

	// Start the browser
	ctx := context.Background()
	if err := client.Start(ctx, nil); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// After start, should be running
	if !client.IsRunning() {
		t.Error("IsRunning() = false after Start()")
	}

	// Stop the browser
	if err := client.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Wait for process to terminate
	time.Sleep(100 * time.Millisecond)

	// After stop, should not be running
	if client.IsRunning() {
		t.Error("IsRunning() = true after Stop()")
	}
}

func TestHealthCheckNotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19231)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// HealthCheck on non-running process should fail
	err = client.HealthCheck(1 * time.Second)
	if err == nil {
		t.Error("HealthCheck() on non-running process expected error, got nil")
	}
}

func TestClientGetPort(t *testing.T) {
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "cloakbrowser")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	client, err := NewClient(binaryPath, 19232)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.GetPort() != 19232 {
		t.Errorf("GetPort() = %d, want 19232", client.GetPort())
	}
}
