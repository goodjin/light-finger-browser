package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

type browserRuntimeManager interface {
	Start(ctx context.Context, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error)
	Stop(ctx context.Context, id string) error
	Restart(ctx context.Context, inst *instance.BrowserInstance, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error)
	Delete(id string) error
}

type browserEngine string

const (
	browserEngineSelfBuilt browserEngine = "self-built"
	browserEngineLocal     browserEngine = "local"
)

type browserEngineConfig struct {
	Engine                browserEngine
	BrowserBinaryPath     string
	BrowserBinarySource   string
}

func loadBrowserEngineConfig() browserEngineConfig {
	rawEngine := strings.TrimSpace(strings.ToLower(os.Getenv("BROWSER_ENGINE")))
	switch rawEngine {
	case "", "self-built", "selfbuilt":
		rawEngine = string(browserEngineSelfBuilt)
	case "local", "local-chrome", "chrome":
		rawEngine = string(browserEngineLocal)
	default:
		rawEngine = string(browserEngineSelfBuilt)
	}

	executablePath, _ := os.Executable()
	workingDir, _ := os.Getwd()
	path, source := resolveBrowserBinaryPath(runtime.GOOS, runtime.GOARCH, executablePath, workingDir, os.Getenv)

	return browserEngineConfig{
		Engine:               browserEngine(rawEngine),
		BrowserBinaryPath:    path,
		BrowserBinarySource:  source,
	}
}

func newBrowserRuntimeManager(db *sqlite.DB) browserRuntimeManager {
	cfg := loadBrowserEngineConfig()
	if cfg.Engine == browserEngineLocal {
		return NewLocalChromeManager(db)
	}
	return NewSelfBuiltBrowserManager(db, cfg.BrowserBinaryPath)
}

type browserArtifactManifest struct {
	Version   int                          `json:"version"`
	Channel   string                       `json:"channel,omitempty"`
	Artifacts []browserArtifactRecord      `json:"artifacts"`
	Metadata  *browserArtifactManifestMeta `json:"metadata,omitempty"`
}

type browserArtifactRecord struct {
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	Path        string            `json:"path"`
	Version     string            `json:"version,omitempty"`
	Checksums   map[string]string `json:"checksums,omitempty"`
	Description string            `json:"description,omitempty"`
}

type browserArtifactManifestMeta struct {
	Build     string `json:"build,omitempty"`
	Checksums string `json:"checksums,omitempty"`
}

func resolveBrowserBinaryPath(goos, goarch, executablePath, workingDir string, getenv func(string) string) (string, string) {
	if getenv != nil {
		for _, envKey := range []string{"BROWSER_BINARY", "SELF_BUILT_BROWSER_PATH"} {
			if candidate := strings.TrimSpace(getenv(envKey)); candidate != "" {
				return candidate, "env:" + envKey
			}
		}
	}

	selectedChannel, selectedVersion := resolveBrowserChannelSelection(getenv)
	if selectedChannel == "" {
		selectedChannel = "stable"
	}

	for _, manifestPath := range browserArtifactManifestCandidates(executablePath, workingDir, goos, selectedChannel) {
		if candidate := resolveBrowserPathFromManifest(manifestPath, goos, goarch, selectedChannel, selectedVersion); candidate != "" {
			return candidate, "artifact-manifest"
		}
	}
	if selectedVersion != "" || selectedChannel != "stable" {
		for _, manifestPath := range browserArtifactManifestCandidates(executablePath, workingDir, goos, "stable") {
			if candidate := resolveBrowserPathFromManifest(manifestPath, goos, goarch, "stable", ""); candidate != "" {
				return candidate, "artifact-manifest"
			}
		}
	}

	for _, candidate := range bundledBrowserCandidates(executablePath, workingDir, goos) {
		if fileExists(candidate) {
			return candidate, "bundled"
		}
	}

	return "", ""
}

func browserArtifactManifestCandidates(executablePath, workingDir, goos, channel string) []string {
	var candidates []string
	if executablePath != "" {
		execDir := filepath.Dir(executablePath)
		switch goos {
		case "darwin":
			candidates = appendUniquePath(candidates,
				filepath.Clean(filepath.Join(execDir, "..", "Resources", "selfbuilt", "artifacts.json")),
				filepath.Clean(filepath.Join(execDir, "..", "Resources", "selfbuilt", "channels", channel, "artifacts.json")),
				filepath.Clean(filepath.Join(execDir, "..", "Resources", "artifacts.json")),
				filepath.Join(execDir, "artifacts.json"),
			)
		default:
			candidates = appendUniquePath(candidates,
				filepath.Join(execDir, "selfbuilt", "artifacts.json"),
				filepath.Join(execDir, "selfbuilt", "channels", channel, "artifacts.json"),
				filepath.Join(execDir, "resources", "selfbuilt", "artifacts.json"),
				filepath.Join(execDir, "resources", "selfbuilt", "channels", channel, "artifacts.json"),
				filepath.Join(execDir, "resources", "artifacts.json"),
				filepath.Join(execDir, "artifacts.json"),
			)
		}
	}
	if workingDir != "" {
		candidates = appendUniquePath(candidates,
			filepath.Join(workingDir, "resources", "selfbuilt", "artifacts.json"),
			filepath.Join(workingDir, "resources", "selfbuilt", "channels", channel, "artifacts.json"),
			filepath.Join(workingDir, "resources", "artifacts.json"),
		)
	}
	return candidates
}

func bundledBrowserCandidates(executablePath, workingDir, goos string) []string {
	var candidates []string
	if executablePath != "" {
		execDir := filepath.Dir(executablePath)
		switch goos {
		case "darwin":
			candidates = appendUniquePath(candidates,
				filepath.Clean(filepath.Join(execDir, "..", "Resources", "selfbuilt", "Chromium.app", "Contents", "MacOS", "Chromium")),
				filepath.Clean(filepath.Join(execDir, "..", "Resources", "SelfBuiltBrowser", "Chromium.app", "Contents", "MacOS", "Chromium")),
				filepath.Clean(filepath.Join(execDir, "..", "Resources", "Chromium.app", "Contents", "MacOS", "Chromium")),
				filepath.Join(execDir, "selfbuilt"),
			)
		case "windows":
			candidates = appendUniquePath(candidates,
				filepath.Join(execDir, "selfbuilt", "selfbuilt.exe"),
				filepath.Join(execDir, "selfbuilt", "chrome.exe"),
				filepath.Join(execDir, "SelfBuiltBrowser", "selfbuilt.exe"),
				filepath.Join(execDir, "SelfBuiltBrowser", "chrome.exe"),
				filepath.Join(execDir, "selfbuilt.exe"),
			)
		default:
			candidates = appendUniquePath(candidates,
				filepath.Join(execDir, "selfbuilt", "selfbuilt"),
				filepath.Join(execDir, "selfbuilt", "chrome"),
				filepath.Join(execDir, "SelfBuiltBrowser", "selfbuilt"),
				filepath.Join(execDir, "SelfBuiltBrowser", "chrome"),
				filepath.Join(execDir, "selfbuilt"),
				filepath.Join(execDir, "chrome"),
			)
		}
	}

	if workingDir != "" {
		switch goos {
		case "darwin":
			candidates = appendUniquePath(candidates,
				filepath.Join(workingDir, "resources", "selfbuilt", "Chromium.app", "Contents", "MacOS", "Chromium"),
				filepath.Join(workingDir, "resources", "Chromium.app", "Contents", "MacOS", "Chromium"),
			)
		case "windows":
			candidates = appendUniquePath(candidates,
				filepath.Join(workingDir, "resources", "selfbuilt", "selfbuilt.exe"),
				filepath.Join(workingDir, "resources", "selfbuilt", "chrome.exe"),
			)
		default:
			candidates = appendUniquePath(candidates,
				filepath.Join(workingDir, "resources", "selfbuilt", "selfbuilt"),
				filepath.Join(workingDir, "resources", "selfbuilt", "chrome"),
			)
		}
	}

	return candidates
}

func resolveBrowserPathFromManifest(manifestPath, goos, goarch, channel, version string) string {
	if !fileExists(manifestPath) {
		return ""
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return ""
	}

	var manifest browserArtifactManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return ""
	}
	if manifest.Version == 0 {
		manifest.Version = 1
	}

	if manifest.Version >= 2 {
		manifestChannel := normalizeBrowserChannel(manifest.Channel)
		if manifestChannel == "" {
			return ""
		}
		if channel != "" && manifestChannel != channel {
			return ""
		}
		baseDir := resolveChannelManifestBaseDir(manifestPath, manifestChannel)
		return resolveManifestArtifactPath(&manifest, baseDir, goos, goarch, version)
	}

	if channel != "" && channel != "stable" {
		return ""
	}
	if version != "" {
		return ""
	}

	return resolveManifestArtifactPath(&manifest, filepath.Dir(manifestPath), goos, goarch, "")
}

func resolveManifestArtifactPath(manifest *browserArtifactManifest, baseDir, goos, goarch, version string) string {
	for _, artifact := range manifest.Artifacts {
		if normalizeBrowserArtifactOS(artifact.OS) != normalizeBrowserArtifactOS(goos) {
			continue
		}
		if normalizeBrowserArtifactArch(artifact.Arch) != normalizeBrowserArtifactArch(goarch) {
			continue
		}
		if version != "" && strings.TrimSpace(artifact.Version) != version {
			continue
		}
		candidate := strings.TrimSpace(artifact.Path)
		if candidate == "" {
			continue
		}
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(baseDir, candidate)
		}
		if fileExists(candidate) {
			return candidate
		}
	}

	return ""
}

func resolveChannelManifestBaseDir(manifestPath, channel string) string {
	manifestDir := filepath.Dir(manifestPath)
	channelSuffix := filepath.Join("channels", channel)
	if strings.HasSuffix(filepath.Clean(manifestDir), channelSuffix) {
		return manifestDir
	}
	return filepath.Join(filepath.Dir(manifestPath), "channels", channel)
}

func resolveBrowserChannelSelection(getenv func(string) string) (string, string) {
	if getenv == nil {
		return "", ""
	}
	channel := normalizeBrowserChannel(getenv("BROWSER_CHANNEL"))
	version := strings.TrimSpace(getenv("BROWSER_VERSION"))
	return channel, version
}

func normalizeBrowserArtifactOS(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "macos", "osx":
		return "darwin"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeBrowserChannel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "alpha":
		return "alpha"
	case "beta":
		return "beta"
	case "stable":
		return "stable"
	case "":
		return ""
	default:
		return ""
	}
}

func normalizeBrowserArtifactArch(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "x64", "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func appendUniquePath(paths []string, candidates ...string) []string {
	existing := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		existing[path] = struct{}{}
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := existing[candidate]; ok {
			continue
		}
		paths = append(paths, candidate)
		existing[candidate] = struct{}{}
	}
	return paths
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
