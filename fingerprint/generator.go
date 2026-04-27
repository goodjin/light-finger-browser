package fingerprint

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strings"
)

// Generator generates fingerprints based on seeds and country configurations.
type Generator struct {
	configs map[string]CountryConfig
}

// NewGenerator creates a new Generator with the default country configurations.
func NewGenerator() *Generator {
	configs := make(map[string]CountryConfig)
	for country, config := range countryConfigs {
		configs[country] = config
	}
	return &Generator{configs: configs}
}

// Generate creates a deterministic fingerprint based on the given seed and country.
// If the seed is empty, an error is returned. If the country is not supported,
// ErrInvalidCountry is returned.
func (g *Generator) Generate(seed string, country string) (*Fingerprint, error) {
	if seed == "" {
		return nil, ErrInvalidSeed
	}

	config, ok := g.configs[country]
	if !ok {
		return nil, ErrInvalidCountry
	}

	// Create deterministic random number generator from seed
	hash := sha256.Sum256([]byte(seed))
	seedNum := int64(binary.BigEndian.Uint64(hash[:8]))
	rng := NewDeterministicRand(seedNum)

	// Generate fingerprint components
	ua := g.generateUserAgent(rng, config)
	screen := g.generateScreen(rng, config)
	timezone := config.Timezone
	locale := g.generateLocale(rng, config)
	webgl := g.generateWebGL(rng, config)
	canvas := g.generateCanvas(rng)
	audio := g.generateAudio(rng)
	hardware := g.generateHardware(rng, config)
	network := g.generateNetwork(rng)

	fp := &Fingerprint{
		Seed:      seed,
		UserAgent: ua,
		Platform:  config.Platform,
		Screen:    screen,
		Timezone:  timezone,
		Locale:    locale,
		WebGL:     webgl,
		Canvas:    canvas,
		Audio:     audio,
		Hardware:  hardware,
		Network:   network,
	}

	return fp, nil
}

// GenerateRandom generates a fingerprint with a random seed for the given country.
func (g *Generator) GenerateRandom(country string) (*Fingerprint, error) {
	seed, err := generateRandomSeed()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random seed: %w", err)
	}
	return g.Generate(seed, country)
}

// generateUserAgent generates a user agent string based on the configuration.
func (g *Generator) generateUserAgent(rng *DeterministicRand, config CountryConfig) string {
	browserVersions := []string{
		"Chrome/120.0.0.0", "Chrome/119.0.0.0", "Chrome/118.0.0.0",
		"Firefox/121.0", "Firefox/120.0",
		"Safari/17.2", "Safari/17.1",
		"Edge/120.0.0.0", "Edge/119.0.0.0",
	}

	version := browserVersions[rng.Intn(len(browserVersions))]

	switch config.Platform {
	case "Windows":
		return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) %s Safari/537.36", version)
	case "Mac":
		return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) %s Safari/537.36", version)
	case "Linux":
		return fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) %s Safari/537.36", version)
	case "iOS":
		devices := []string{"iPhone", "iPad"}
		device := devices[rng.Intn(len(devices))]
		return fmt.Sprintf("Mozilla/5.0 (%s; CPU OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1", device)
	case "Android":
		return fmt.Sprintf("Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) %s Mobile Safari/537.36", version)
	default:
		return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) %s Safari/537.36", version)
	}
}

// generateScreen generates a screen configuration based on available resolutions.
func (g *Generator) generateScreen(rng *DeterministicRand, config CountryConfig) ScreenConfig {
	if len(config.Resolutions) == 0 {
		return ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0}
	}

	res := config.Resolutions[rng.Intn(len(config.Resolutions))]
	width := res[0]
	height := res[1]

	pixelRatios := []float64{1.0, 1.25, 1.5, 1.0, 1.0, 1.25}
	pixelRatio := pixelRatios[rng.Intn(len(pixelRatios))]

	return ScreenConfig{
		Width:      width,
		Height:     height,
		PixelRatio: pixelRatio,
	}
}

// generateLocale generates a locale string based on the country configuration.
func (g *Generator) generateLocale(rng *DeterministicRand, config CountryConfig) string {
	prefix := config.LocalePrefix

	switch prefix {
	case "en":
		localeVariants := []string{"en-US", "en-GB", "en-CA", "en-AU", "en"}
		return localeVariants[rng.Intn(len(localeVariants))]
	case "de":
		localeVariants := []string{"de-DE", "de-AT", "de-CH", "de"}
		return localeVariants[rng.Intn(len(localeVariants))]
	case "fr":
		localeVariants := []string{"fr-FR", "fr-CA", "fr-BE", "fr"}
		return localeVariants[rng.Intn(len(localeVariants))]
	case "ja":
		return "ja-JP"
	case "zh":
		localeVariants := []string{"zh-CN", "zh-TW", "zh-HK"}
		return localeVariants[rng.Intn(len(localeVariants))]
	case "es":
		return "es-ES"
	case "it":
		return "it-IT"
	case "pt":
		localeVariants := []string{"pt-BR", "pt-PT"}
		return localeVariants[rng.Intn(len(localeVariants))]
	default:
		return prefix
	}
}

// generateWebGL generates a WebGL configuration based on available renderers.
func (g *Generator) generateWebGL(rng *DeterministicRand, config CountryConfig) WebGLConfig {
	if len(config.WebGLRenderers) == 0 {
		return WebGLConfig{
			Renderer:   "Unknown",
			Vendor:     "Unknown",
			Extensions: []string{},
		}
	}

	renderer := config.WebGLRenderers[rng.Intn(len(config.WebGLRenderers))]
	vendor := g.extractVendor(renderer)

	extensions := []string{
		"WEBGL_debug_renderer_info",
		"EXT_color_buffer_float",
		"WEBGL_depth_texture",
		"OES_texture_float",
		"OES_texture_float_linear",
	}

	numExtensions := rng.Intn(3) + 2
	selectedExtensions := make([]string, 0, numExtensions)
	for i := 0; i < numExtensions && i < len(extensions); i++ {
		selectedExtensions = append(selectedExtensions, extensions[i])
	}

	return WebGLConfig{
		Renderer:   renderer,
		Vendor:     vendor,
		Extensions: selectedExtensions,
	}
}

// extractVendor extracts the GPU vendor from a renderer string.
func (g *Generator) extractVendor(renderer string) string {
	lower := strings.ToLower(renderer)
	if strings.Contains(lower, "nvidia") {
		return "NVIDIA"
	}
	if strings.Contains(lower, "amd") || strings.Contains(lower, "radeon") {
		return "AMD"
	}
	if strings.Contains(lower, "intel") {
		return "Intel"
	}
	if strings.Contains(lower, "apple") {
		return "Apple"
	}
	if strings.Contains(lower, "qualcomm") || strings.Contains(lower, "adreno") {
		return "Qualcomm"
	}
	if strings.Contains(lower, "mali") {
		return "ARM"
	}
	return "Unknown"
}

// generateCanvas generates a canvas hash.
func (g *Generator) generateCanvas(rng *DeterministicRand) CanvasConfig {
	hashBytes := make([]byte, 16)
	for i := range hashBytes {
		hashBytes[i] = byte(rng.Intn(256))
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(hashBytes))[:32]
	return CanvasConfig{Hash: hash}
}

// generateAudio generates an audio hash.
func (g *Generator) generateAudio(rng *DeterministicRand) AudioConfig {
	hashBytes := make([]byte, 8)
	for i := range hashBytes {
		hashBytes[i] = byte(rng.Intn(256))
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(hashBytes))[:16]
	return AudioConfig{Hash: hash}
}

// generateHardware generates a hardware configuration.
func (g *Generator) generateHardware(rng *DeterministicRand, config CountryConfig) HardwareConfig {
	cpuCoresOptions := []int{2, 4, 6, 8, 12, 16}
	memoryGBOptions := []int{4, 8, 16, 32}

	cpuCores := cpuCoresOptions[rng.Intn(len(cpuCoresOptions))]
	memoryGB := memoryGBOptions[rng.Intn(len(memoryGBOptions))]

	var gpuVendor, gpuModel, gpuRenderer string

	if len(config.GPUs) > 0 {
		gpu := config.GPUs[rng.Intn(len(config.GPUs))]
		gpuVendor = gpu.Vendor
		gpuModel = gpu.Model
		gpuRenderer = gpu.Renderer
	} else if len(config.WebGLRenderers) > 0 {
		renderer := config.WebGLRenderers[rng.Intn(len(config.WebGLRenderers))]
		gpuVendor = g.extractVendor(renderer)
		gpuModel = "Unknown"
		gpuRenderer = renderer
	} else {
		gpuVendor = "Unknown"
		gpuModel = "Unknown"
		gpuRenderer = "Unknown"
	}

	return HardwareConfig{
		CPUCores:    cpuCores,
		MemoryGB:    memoryGB,
		GPUVendor:   gpuVendor,
		GPUModel:    gpuModel,
		GPURenderer: gpuRenderer,
	}
}

// generateNetwork generates a network configuration.
func (g *Generator) generateNetwork(rng *DeterministicRand) NetworkConfig {
	connectionTypes := []string{"wifi", "wired", "4g", "3g", "unknown"}
	connectionType := connectionTypes[rng.Intn(len(connectionTypes))]

	var downlink float64
	var rtt int

	switch connectionType {
	case "wifi":
		downlink = float64(rng.Intn(100)+50) + rng.Float64()
		rtt = rng.Intn(20) + 5
	case "wired":
		downlink = float64(rng.Intn(1000)+100) + rng.Float64()
		rtt = rng.Intn(10) + 2
	case "4g":
		downlink = float64(rng.Intn(50)+10) + rng.Float64()
		rtt = rng.Intn(100) + 30
	case "3g":
		downlink = float64(rng.Intn(5)+1) + rng.Float64()
		rtt = rng.Intn(200) + 100
	default:
		downlink = float64(rng.Intn(50)) + rng.Float64()
		rtt = rng.Intn(100) + 20
	}

	return NetworkConfig{
		ConnectionType: connectionType,
		Downlink:       math.Round(downlink*100) / 100,
		RTT:            rtt,
	}
}

// generateRandomSeed generates a cryptographically random seed string.
func generateRandomSeed() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", bytes), nil
}

// DeterministicRand is a deterministic random number generator for reproducibility.
type DeterministicRand struct {
	source int64
}

// NewDeterministicRand creates a new DeterministicRand with the given seed.
func NewDeterministicRand(seed int64) *DeterministicRand {
	return &DeterministicRand{source: seed}
}

// Intn returns a deterministic random int in [0, n).
func (r *DeterministicRand) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	r.source = (r.source*1103515245 + 12345) & 0x7fffffff
	return int(r.source % int64(n))
}

// Float64 returns a deterministic random float64 in [0, 1).
func (r *DeterministicRand) Float64() float64 {
	r.source = (r.source*1103515245 + 12345) & 0x7fffffff
	return float64(r.source) / float64(0x7fffffff)
}

// BigIntn returns a deterministic random big.Int in [0, n).
func (r *DeterministicRand) BigIntn(n *big.Int) *big.Int {
	r.source = (r.source*1103515245 + 12345) & 0x7fffffff
	result := new(big.Int).Mod(big.NewInt(r.source), n)
	return result
}