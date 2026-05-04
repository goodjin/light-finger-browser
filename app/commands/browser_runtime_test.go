package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCloakBrowserBinaryPathPrefersExplicitEnv(t *testing.T) {
	t.Parallel()

	env := func(key string) string {
		if key == "CLOAKBROWSER_PATH" {
			return "/custom/cloakbrowser"
		}
		return ""
	}

	path, source := resolveCloakBrowserBinaryPath("linux", "amd64", "/tmp/app", "", env)
	if path != "/custom/cloakbrowser" {
		t.Fatalf("expected explicit env path, got %q", path)
	}
	if source != "env:CLOAKBROWSER_PATH" {
		t.Fatalf("expected env source, got %q", source)
	}
}

func TestResolveCloakBrowserBinaryPathUsesManifest(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "resources", "cloakbrowser", "darwin-arm64", "Chromium.app", "Contents", "MacOS", "Chromium")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "resources", "cloakbrowser", "artifacts.json")
	manifest := `{"version":1,"artifacts":[{"os":"darwin","arch":"arm64","path":"darwin-arm64/Chromium.app/Contents/MacOS/Chromium"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	path, source := resolveCloakBrowserBinaryPath("darwin", "arm64", filepath.Join(tempDir, "Fingerbrower.app", "Contents", "MacOS", "fingerbrower"), tempDir, func(string) string {
		return ""
	})
	if path != artifactPath {
		t.Fatalf("expected manifest artifact path %q, got %q", artifactPath, path)
	}
	if source != "artifact-manifest" {
		t.Fatalf("expected artifact manifest source, got %q", source)
	}
}

func TestResolveCloakBrowserBinaryPathFallsBackToBundledCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	bundledPath := filepath.Join(tempDir, "resources", "cloakbrowser", "cloakbrowser")
	if err := os.MkdirAll(filepath.Dir(bundledPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(bundledPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write bundled path: %v", err)
	}

	path, source := resolveCloakBrowserBinaryPath("linux", "amd64", filepath.Join(tempDir, "fingerbrower"), tempDir, func(string) string {
		return ""
	})
	if path != bundledPath {
		t.Fatalf("expected bundled path %q, got %q", bundledPath, path)
	}
	if source != "bundled" {
		t.Fatalf("expected bundled source, got %q", source)
	}
}

func TestResolveCloakBrowserBinaryPathUsesChannelManifest(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "resources", "cloakbrowser", "channels", "alpha", "darwin-arm64", "Chromium.app", "Contents", "MacOS", "Chromium")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "resources", "cloakbrowser", "channels", "alpha", "artifacts.json")
	manifest := `{"version":2,"channel":"alpha","artifacts":[{"os":"darwin","arch":"arm64","path":"darwin-arm64/Chromium.app/Contents/MacOS/Chromium","version":"124.0.0"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	path, source := resolveCloakBrowserBinaryPath("darwin", "arm64", filepath.Join(tempDir, "Fingerbrower.app", "Contents", "MacOS", "fingerbrower"), tempDir, func(key string) string {
		if key == "BROWSER_CHANNEL" {
			return "alpha"
		}
		return ""
	})
	if path != artifactPath {
		t.Fatalf("expected channel manifest path %q, got %q", artifactPath, path)
	}
	if source != "artifact-manifest" {
		t.Fatalf("expected artifact manifest source, got %q", source)
	}
}

func TestResolveCloakBrowserBinaryPathFallsBackToStableChannel(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "resources", "cloakbrowser", "channels", "stable", "darwin-arm64", "Chromium.app", "Contents", "MacOS", "Chromium")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "resources", "cloakbrowser", "channels", "stable", "artifacts.json")
	manifest := `{"version":2,"channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","path":"darwin-arm64/Chromium.app/Contents/MacOS/Chromium","version":"124.0.0"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	path, source := resolveCloakBrowserBinaryPath("darwin", "arm64", filepath.Join(tempDir, "Fingerbrower.app", "Contents", "MacOS", "fingerbrower"), tempDir, func(key string) string {
		if key == "BROWSER_CHANNEL" {
			return "alpha"
		}
		return ""
	})
	if path != artifactPath {
		t.Fatalf("expected stable fallback path %q, got %q", artifactPath, path)
	}
	if source != "artifact-manifest" {
		t.Fatalf("expected artifact manifest source, got %q", source)
	}
}

func TestResolveCloakBrowserBinaryPathRespectsVersionSelection(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "resources", "cloakbrowser", "channels", "stable", "darwin-arm64", "Chromium.app", "Contents", "MacOS", "Chromium")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "resources", "cloakbrowser", "channels", "stable", "artifacts.json")
	manifest := `{"version":2,"channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","path":"darwin-arm64/Chromium.app/Contents/MacOS/Chromium","version":"124.0.0"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	path, source := resolveCloakBrowserBinaryPath("darwin", "arm64", filepath.Join(tempDir, "Fingerbrower.app", "Contents", "MacOS", "fingerbrower"), tempDir, func(key string) string {
		if key == "BROWSER_VERSION" {
			return "124.0.0"
		}
		return ""
	})
	if path != artifactPath {
		t.Fatalf("expected version-selected path %q, got %q", artifactPath, path)
	}
	if source != "artifact-manifest" {
		t.Fatalf("expected artifact manifest source, got %q", source)
	}
}
