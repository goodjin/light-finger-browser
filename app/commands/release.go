package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type ReleaseService struct{}

func NewReleaseService() *ReleaseService {
	return &ReleaseService{}
}

type ReleasePromotionRequest struct {
	ArtifactsRoot      string `json:"artifacts_root"`
	FromChannel        string `json:"from_channel"`
	ToChannel          string `json:"to_channel"`
	UpdateRootManifest bool   `json:"update_root_manifest"`
	AllowUnsigned      bool   `json:"allow_unsigned"`
}

type ReleasePromotionResult struct {
	FromChannel         string `json:"from_channel"`
	ToChannel           string `json:"to_channel"`
	PromotedVersion     string `json:"promoted_version"`
	PreviousStable      string `json:"previous_stable,omitempty"`
	RootManifestUpdated bool   `json:"root_manifest_updated"`
}

type ReleaseRollbackRequest struct {
	ArtifactsRoot      string `json:"artifacts_root"`
	UpdateRootManifest bool   `json:"update_root_manifest"`
}

type ReleaseRollbackResult struct {
	CurrentStable       string `json:"current_stable"`
	PreviousStable      string `json:"previous_stable"`
	RootManifestUpdated bool   `json:"root_manifest_updated"`
}

func (s *ReleaseService) PromoteChannel(ctx context.Context, req *ReleasePromotionRequest) (*ReleasePromotionResult, error) {
	_ = ctx
	if req == nil {
		return nil, fmt.Errorf("promotion request is required")
	}

	root := strings.TrimSpace(req.ArtifactsRoot)
	if root == "" {
		return nil, fmt.Errorf("artifacts_root is required")
	}

	fromChannel := normalizeBrowserChannel(req.FromChannel)
	toChannel := normalizeBrowserChannel(req.ToChannel)
	if fromChannel == "" || toChannel == "" {
		return nil, fmt.Errorf("invalid channel: from=%q to=%q", req.FromChannel, req.ToChannel)
	}
	if fromChannel == toChannel {
		return nil, fmt.Errorf("from_channel and to_channel must differ")
	}

	sourceDir := filepath.Join(root, "channels", fromChannel)
	if !dirExists(sourceDir) {
		return nil, fmt.Errorf("source channel not found: %s", sourceDir)
	}

	manifestPath := filepath.Join(sourceDir, "artifacts.json")
	manifest, err := loadArtifactManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	if manifest.Version >= 2 {
		manifestChannel := normalizeBrowserChannel(manifest.Channel)
		if manifestChannel != "" && manifestChannel != fromChannel {
			return nil, fmt.Errorf("manifest channel mismatch: %s", manifestChannel)
		}
	} else if toChannel != "stable" {
		return nil, fmt.Errorf("manifest v1 cannot be promoted to %s", toChannel)
	}

	buildMeta, err := loadBuildMetadata(filepath.Join(sourceDir, "metadata", "build.json"))
	if err != nil {
		return nil, err
	}
	if !fileExists(filepath.Join(sourceDir, "metadata", "checksums.json")) {
		return nil, fmt.Errorf("checksums.json missing in %s", sourceDir)
	}
	if toChannel == "stable" && !req.AllowUnsigned {
		if !isSigningStatusAccepted(buildMeta.SigningStatus) {
			return nil, fmt.Errorf("signing_status %q cannot be promoted to stable", buildMeta.SigningStatus)
		}
	}

	var previousStable string
	targetDir := filepath.Join(root, "channels", toChannel)
	if toChannel == "stable" {
		stableDir := targetDir
		previousDir := filepath.Join(root, "channels", "stable-previous")
		if dirExists(stableDir) {
			if err := os.RemoveAll(previousDir); err != nil {
				return nil, err
			}
			previousStable = buildVersionFromDir(stableDir)
			if err := os.Rename(stableDir, previousDir); err != nil {
				return nil, err
			}
		}
	} else if dirExists(targetDir) {
		if err := os.RemoveAll(targetDir); err != nil {
			return nil, err
		}
	}

	if err := copyDir(sourceDir, targetDir); err != nil {
		return nil, err
	}

	if manifest.Version >= 2 {
		manifest.Channel = toChannel
		if err := writeArtifactManifest(filepath.Join(targetDir, "artifacts.json"), manifest); err != nil {
			return nil, err
		}
	}

	rootManifestUpdated := false
	if req.UpdateRootManifest || toChannel == "stable" {
		rootManifestPath := filepath.Join(root, "artifacts.json")
		if manifest.Version >= 2 {
			if err := writeArtifactManifest(rootManifestPath, manifest); err != nil {
				return nil, err
			}
		} else if err := copyFile(filepath.Join(targetDir, "artifacts.json"), rootManifestPath); err != nil {
			return nil, err
		}
		rootManifestUpdated = true
	}

	return &ReleasePromotionResult{
		FromChannel:         fromChannel,
		ToChannel:           toChannel,
		PromotedVersion:     buildMeta.BrowserVersion,
		PreviousStable:      previousStable,
		RootManifestUpdated: rootManifestUpdated,
	}, nil
}

func (s *ReleaseService) RollbackStable(ctx context.Context, req *ReleaseRollbackRequest) (*ReleaseRollbackResult, error) {
	_ = ctx
	if req == nil {
		return nil, fmt.Errorf("rollback request is required")
	}

	root := strings.TrimSpace(req.ArtifactsRoot)
	if root == "" {
		return nil, fmt.Errorf("artifacts_root is required")
	}

	stableDir := filepath.Join(root, "channels", "stable")
	previousDir := filepath.Join(root, "channels", "stable-previous")
	if !dirExists(stableDir) {
		return nil, fmt.Errorf("stable channel not found: %s", stableDir)
	}
	if !dirExists(previousDir) {
		return nil, fmt.Errorf("stable-previous channel not found: %s", previousDir)
	}

	currentVersion := buildVersionFromDir(stableDir)
	previousVersion := buildVersionFromDir(previousDir)

	tempDir := filepath.Join(root, "channels", "stable-rollback-temp")
	if dirExists(tempDir) {
		if err := os.RemoveAll(tempDir); err != nil {
			return nil, err
		}
	}
	if err := os.Rename(stableDir, tempDir); err != nil {
		return nil, err
	}
	if err := os.Rename(previousDir, stableDir); err != nil {
		return nil, err
	}
	if err := os.Rename(tempDir, previousDir); err != nil {
		return nil, err
	}

	rootManifestUpdated := false
	if req.UpdateRootManifest {
		if err := copyFile(filepath.Join(stableDir, "artifacts.json"), filepath.Join(root, "artifacts.json")); err != nil {
			return nil, err
		}
		rootManifestUpdated = true
	}

	return &ReleaseRollbackResult{
		CurrentStable:       previousVersion,
		PreviousStable:      currentVersion,
		RootManifestUpdated: rootManifestUpdated,
	}, nil
}

type buildMetadata struct {
	BrowserVersion string `json:"browser_version"`
	SigningStatus  string `json:"signing_status"`
}

func loadBuildMetadata(path string) (*buildMetadata, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta buildMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(meta.BrowserVersion) == "" {
		return nil, fmt.Errorf("browser_version missing in %s", path)
	}
	return &meta, nil
}

func buildVersionFromDir(channelDir string) string {
	meta, err := loadBuildMetadata(filepath.Join(channelDir, "metadata", "build.json"))
	if err != nil {
		return ""
	}
	return meta.BrowserVersion
}

func isSigningStatusAccepted(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "signed", "notarized":
		return true
	default:
		return false
	}
}

func loadArtifactManifest(path string) (*browserArtifactManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest browserArtifactManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	if manifest.Version == 0 {
		manifest.Version = 1
	}
	if len(manifest.Artifacts) == 0 {
		return nil, fmt.Errorf("artifacts.json missing artifacts")
	}
	return &manifest, nil
}

func writeArtifactManifest(path string, manifest *browserArtifactManifest) error {
	if manifest == nil {
		return fmt.Errorf("manifest is required")
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0644)
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source is not a directory: %s", src)
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if rel == "." {
				return nil
			}
			return os.MkdirAll(target, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		return copyFileWithMode(path, target, info.Mode())
	})
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return copyFileWithMode(src, dst, info.Mode())
}

func copyFileWithMode(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
