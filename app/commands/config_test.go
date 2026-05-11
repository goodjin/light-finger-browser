package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigServiceLoad(t *testing.T) {
	// Create a temp config directory
	tmpDir := filepath.Join(os.TempDir(), "fingerbrower-config-test")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// Create config service with temp path
	svc := &ConfigService{
		configPath: filepath.Join(tmpDir, "config.json"),
	}

	// Load should create default config
	cfg, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.InstancePort != DefaultInstancePort {
		t.Errorf("Expected default port %d, got %d", DefaultInstancePort, cfg.InstancePort)
	}

	if cfg.Headless != false {
		t.Errorf("Expected Headless=false, got %v", cfg.Headless)
	}
}

func TestConfigServiceSetInstancePort(t *testing.T) {
	// Create a temp config directory
	tmpDir := filepath.Join(os.TempDir(), "fingerbrower-config-test2")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	svc := &ConfigService{
		configPath: filepath.Join(tmpDir, "config.json"),
	}

	// Set a custom port
	err := svc.SetInstancePort(9500)
	if err != nil {
		t.Fatalf("SetInstancePort() failed: %v", err)
	}

	// Reload and verify
	port, err := svc.GetInstancePort()
	if err != nil {
		t.Fatalf("GetInstancePort() failed: %v", err)
	}

	if port != 9500 {
		t.Errorf("Expected port 9500, got %d", port)
	}
}

func TestConfigServiceSetHeadless(t *testing.T) {
	// Create a temp config directory
	tmpDir := filepath.Join(os.TempDir(), "fingerbrower-config-test3")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	svc := &ConfigService{
		configPath: filepath.Join(tmpDir, "config.json"),
	}

	// Set headless mode
	err := svc.SetHeadless(true)
	if err != nil {
		t.Fatalf("SetHeadless() failed: %v", err)
	}

	// Reload and verify
	headless, err := svc.GetHeadless()
	if err != nil {
		t.Fatalf("GetHeadless() failed: %v", err)
	}

	if !headless {
		t.Error("Expected Headless=true")
	}
}
