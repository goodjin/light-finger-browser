package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPromoteChannelUpdatesManifestChannel(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeChannelFixture(t, root, "alpha", "alpha", "124.0.0", "signed")

	svc := NewReleaseService()
	result, err := svc.PromoteChannel(context.Background(), &ReleasePromotionRequest{
		ArtifactsRoot:      root,
		FromChannel:        "alpha",
		ToChannel:          "beta",
		UpdateRootManifest: true,
	})
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if result.ToChannel != "beta" {
		t.Fatalf("expected beta channel, got %q", result.ToChannel)
	}

	manifest := readManifestFixture(t, filepath.Join(root, "channels", "beta", "artifacts.json"))
	if manifest.Channel != "beta" {
		t.Fatalf("expected beta manifest channel, got %q", manifest.Channel)
	}

	rootManifest := readManifestFixture(t, filepath.Join(root, "artifacts.json"))
	if rootManifest.Channel != "beta" {
		t.Fatalf("expected root manifest beta, got %q", rootManifest.Channel)
	}
}

func TestPromoteChannelToStableArchivesPrevious(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeChannelFixture(t, root, "stable", "stable", "123.0.0", "signed")
	writeChannelFixture(t, root, "beta", "beta", "124.0.0", "signed")

	svc := NewReleaseService()
	result, err := svc.PromoteChannel(context.Background(), &ReleasePromotionRequest{
		ArtifactsRoot: root,
		FromChannel:   "beta",
		ToChannel:     "stable",
	})
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if result.PreviousStable != "123.0.0" {
		t.Fatalf("expected previous stable 123.0.0, got %q", result.PreviousStable)
	}
	if !dirExists(filepath.Join(root, "channels", "stable-previous")) {
		t.Fatalf("expected stable-previous to exist")
	}
}

func TestRollbackStableSwapsChannels(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeChannelFixture(t, root, "stable", "stable", "124.0.0", "signed")
	writeChannelFixture(t, root, "stable-previous", "stable", "123.0.0", "signed")

	svc := NewReleaseService()
	result, err := svc.RollbackStable(context.Background(), &ReleaseRollbackRequest{
		ArtifactsRoot:      root,
		UpdateRootManifest: true,
	})
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if result.CurrentStable != "123.0.0" {
		t.Fatalf("expected current stable 123.0.0, got %q", result.CurrentStable)
	}
	if result.PreviousStable != "124.0.0" {
		t.Fatalf("expected previous stable 124.0.0, got %q", result.PreviousStable)
	}

	rootManifest := readManifestFixture(t, filepath.Join(root, "artifacts.json"))
	if rootManifest.Channel != "stable" {
		t.Fatalf("expected root manifest stable, got %q", rootManifest.Channel)
	}
}

func writeChannelFixture(t *testing.T, root, dirChannel, manifestChannel, version, signingStatus string) {
	t.Helper()

	channelDir := filepath.Join(root, "channels", dirChannel)
	artifactPath := filepath.Join(channelDir, "darwin-arm64", "Chromium.app", "Contents", "MacOS", "Chromium")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(artifactPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	manifest := browserArtifactManifest{
		Version: 2,
		Channel: manifestChannel,
		Artifacts: []browserArtifactRecord{
			{
				OS:      "darwin",
				Arch:    "arm64",
				Path:    "darwin-arm64/Chromium.app/Contents/MacOS/Chromium",
				Version: version,
			},
		},
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	manifestPath := filepath.Join(channelDir, "artifacts.json")
	if err := os.WriteFile(manifestPath, payload, 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	metaDir := filepath.Join(channelDir, "metadata")
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		t.Fatalf("mkdir metadata: %v", err)
	}
	buildPayload := map[string]string{
		"browser_version": version,
		"signing_status":  signingStatus,
	}
	buildBytes, err := json.MarshalIndent(buildPayload, "", "  ")
	if err != nil {
		t.Fatalf("marshal build: %v", err)
	}
	if err := os.WriteFile(filepath.Join(metaDir, "build.json"), buildBytes, 0644); err != nil {
		t.Fatalf("write build.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(metaDir, "checksums.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write checksums.json: %v", err)
	}
}

func readManifestFixture(t *testing.T, path string) *browserArtifactManifest {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest browserArtifactManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return &manifest
}
