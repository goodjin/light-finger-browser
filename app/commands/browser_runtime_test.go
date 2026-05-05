package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBrowserBinaryPathPrefersExplicitEnv(t *testing.T) {
	t.Parallel()

	env := func(key string) string {
		if key == "BROWSER_BINARY" {
			return "/custom/selfbuilt"
		}
		return ""
	}

	path, source := resolveBrowserBinaryPath("linux", "amd64", "/tmp/app", "", env)
	if path != "/custom/selfbuilt" {
		t.Fatalf("expected explicit env path, got %q", path)
	}
	if source != "env:BROWSER_BINARY" {
		t.Fatalf("expected env source, got %q", source)
	}
}

func TestResolveBrowserBinaryPathUsesManifest(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "resources", "selfbuilt", "darwin-arm64", "Chromium.app", "Contents", "MacOS", "Chromium")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "resources", "selfbuilt", "artifacts.json")
	manifest := `{"version":1,"artifacts":[{"os":"darwin","arch":"arm64","path":"darwin-arm64/Chromium.app/Contents/MacOS/Chromium"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	path, source := resolveBrowserBinaryPath("darwin", "arm64", filepath.Join(tempDir, "Fingerbrower.app", "Contents", "MacOS", "fingerbrower"), tempDir, func(string) string {
		return ""
	})
	if path != artifactPath {
		t.Fatalf("expected manifest artifact path %q, got %q", artifactPath, path)
	}
	if source != "artifact-manifest" {
		t.Fatalf("expected artifact manifest source, got %q", source)
	}
}

func TestResolveBrowserBinaryPathFallsBackToBundledCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	bundledPath := filepath.Join(tempDir, "resources", "selfbuilt", "selfbuilt")
	if err := os.MkdirAll(filepath.Dir(bundledPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(bundledPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write bundled path: %v", err)
	}

	path, source := resolveBrowserBinaryPath("linux", "amd64", filepath.Join(tempDir, "fingerbrower"), tempDir, func(string) string {
		return ""
	})
	if path != bundledPath {
		t.Fatalf("expected bundled path %q, got %q", bundledPath, path)
	}
	if source != "bundled" {
		t.Fatalf("expected bundled source, got %q", source)
	}
}

func TestResolveBrowserBinaryPathUsesChannelManifest(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "resources", "selfbuilt", "channels", "alpha", "darwin-arm64", "Chromium.app", "Contents", "MacOS", "Chromium")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "resources", "selfbuilt", "channels", "alpha", "artifacts.json")
	manifest := `{"version":2,"channel":"alpha","artifacts":[{"os":"darwin","arch":"arm64","path":"darwin-arm64/Chromium.app/Contents/MacOS/Chromium","version":"124.0.0"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	path, source := resolveBrowserBinaryPath("darwin", "arm64", filepath.Join(tempDir, "Fingerbrower.app", "Contents", "MacOS", "fingerbrower"), tempDir, func(key string) string {
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

func TestResolveBrowserBinaryPathFallsBackToStableChannel(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "resources", "selfbuilt", "channels", "stable", "darwin-arm64", "Chromium.app", "Contents", "MacOS", "Chromium")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "resources", "selfbuilt", "channels", "stable", "artifacts.json")
	manifest := `{"version":2,"channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","path":"darwin-arm64/Chromium.app/Contents/MacOS/Chromium","version":"124.0.0"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	path, source := resolveBrowserBinaryPath("darwin", "arm64", filepath.Join(tempDir, "Fingerbrower.app", "Contents", "MacOS", "fingerbrower"), tempDir, func(key string) string {
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

func TestResolveBrowserBinaryPathRespectsVersionSelection(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	artifactPath := filepath.Join(tempDir, "resources", "selfbuilt", "channels", "stable", "darwin-arm64", "Chromium.app", "Contents", "MacOS", "Chromium")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "resources", "selfbuilt", "channels", "stable", "artifacts.json")
	manifest := `{"version":2,"channel":"stable","artifacts":[{"os":"darwin","arch":"arm64","path":"darwin-arm64/Chromium.app/Contents/MacOS/Chromium","version":"124.0.0"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	path, source := resolveBrowserBinaryPath("darwin", "arm64", filepath.Join(tempDir, "Fingerbrower.app", "Contents", "MacOS", "fingerbrower"), tempDir, func(key string) string {
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
