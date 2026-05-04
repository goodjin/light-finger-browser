package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectChromePathUsesEnvOverride(t *testing.T) {
	tempDir := t.TempDir()
	bin := filepath.Join(tempDir, "chromium")
	if err := os.WriteFile(bin, []byte("binary"), 0755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	t.Setenv("BROWSER_BINARY", bin)
	path, err := DetectChromePath()
	if err != nil {
		t.Fatalf("expected override path, got error: %v", err)
	}
	if path != bin {
		t.Fatalf("expected override path %q, got %q", bin, path)
	}
}

func TestDetectChromePathReturnsErrorForMissingOverride(t *testing.T) {
	tempDir := t.TempDir()
	missing := filepath.Join(tempDir, "missing-chrome")
	t.Setenv("BROWSER_BINARY", missing)

	_, err := DetectChromePath()
	if err == nil {
		t.Fatal("expected error for missing BROWSER_BINARY override")
	}
	if !strings.Contains(err.Error(), "BROWSER_BINARY") {
		t.Fatalf("expected BROWSER_BINARY error, got %v", err)
	}
}
