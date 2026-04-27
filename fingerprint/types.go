package fingerprint

import "time"

// ScreenConfig represents screen configuration for a fingerprint.
type ScreenConfig struct {
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	PixelRatio float64 `json:"pixel_ratio"`
}

// WebGLConfig represents WebGL configuration for a fingerprint.
type WebGLConfig struct {
	Renderer   string   `json:"renderer"`
	Vendor     string   `json:"vendor"`
	Extensions []string `json:"extensions"`
}

// CanvasConfig represents canvas configuration for a fingerprint.
type CanvasConfig struct {
	Hash string `json:"hash"`
}

// AudioConfig represents audio configuration for a fingerprint.
type AudioConfig struct {
	Hash string `json:"hash"`
}

// HardwareConfig represents hardware configuration for a fingerprint.
type HardwareConfig struct {
	CPUCores     int    `json:"cpu_cores"`
	MemoryGB     int    `json:"memory_gb"`
	GPUVendor    string `json:"gpu_vendor"`
	GPUModel     string `json:"gpu_model"`
	GPURenderer  string `json:"gpu_renderer"`
}

// NetworkConfig represents network configuration for a fingerprint.
type NetworkConfig struct {
	ConnectionType string `json:"connection_type"`
	Downlink       float64 `json:"downlink"`
	RTT            int    `json:"rtt"`
}

// Fingerprint represents a browser fingerprint with all associated configurations.
type Fingerprint struct {
	Seed       string         `json:"seed"`
	UserAgent  string        `json:"user_agent"`
	Platform   string        `json:"platform"`
	Screen     ScreenConfig  `json:"screen"`
	Timezone   string        `json:"timezone"`
	Locale     string        `json:"locale"`
	Canvas     CanvasConfig  `json:"canvas"`
	WebGL      WebGLConfig   `json:"webgl"`
	Audio      AudioConfig   `json:"audio"`
	Hardware   HardwareConfig `json:"hardware"`
	Network    NetworkConfig `json:"network"`
}

// CountryConfig represents fingerprint configuration for a specific country.
type CountryConfig struct {
	Platform       string       `json:"platform"`
	UserAgent      string       `json:"user_agent"`
	Timezone       string       `json:"timezone"`
	LocalePrefix   string       `json:"locale_prefix"`
	Resolutions    [][]int      `json:"resolutions"`
	WebGLRenderers []string     `json:"webgl_renderers"`
	GPUs           []GPUConfig  `json:"gpus"`
}

// GPUConfig represents GPU configuration options.
type GPUConfig struct {
	Vendor   string `json:"vendor"`
	Model    string `json:"model"`
	Renderer string `json:"renderer"`
}

// Errors represents module-level errors.
var (
	ErrInvalidCountry = &ValidationError{Message: "invalid country code"}
	ErrInvalidSeed    = &ValidationError{Message: "invalid seed"}
	ErrInvalidFingerprint = &ValidationError{Message: "invalid fingerprint"}
)

// ValidationError represents a validation error with context.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new ValidationError with the given message.
func NewValidationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}

// TimezoneLocaleMap defines valid timezone to locale prefix mappings.
var TimezoneLocaleMap = map[string][]string{
	"America/New_York":       {"en"},
	"America/Los_Angeles":    {"en"},
	"America/Chicago":        {"en"},
	"America/Denver":         {"en"},
	"Europe/London":          {"en"},
	"Europe/Paris":           {"fr", "de"},
	"Europe/Berlin":          {"de"},
	"Europe/Madrid":          {"es"},
	"Europe/Rome":            {"it"},
	"Asia/Tokyo":             {"ja"},
	"Asia/Shanghai":          {"zh"},
	"Asia/Singapore":         {"en"},
	"Australia/Sydney":       {"en"},
}

// ValidScreenResolutions defines valid screen resolution ranges.
var ValidScreenResolutions = []ScreenConfig{
	{Width: 1920, Height: 1080, PixelRatio: 1.0},
	{Width: 1920, Height: 1080, PixelRatio: 1.25},
	{Width: 1920, Height: 1080, PixelRatio: 1.5},
	{Width: 1366, Height: 768, PixelRatio: 1.0},
	{Width: 1536, Height: 864, PixelRatio: 1.0},
	{Width: 2560, Height: 1440, PixelRatio: 1.0},
	{Width: 2560, Height: 1440, PixelRatio: 1.25},
	{Width: 3840, Height: 2160, PixelRatio: 1.0},
	{Width: 1280, Height: 720, PixelRatio: 1.0},
	{Width: 1440, Height: 900, PixelRatio: 1.0},
}

// IsValidScreenResolution checks if a screen resolution is valid.
func IsValidScreenResolution(width, height int) bool {
	for _, res := range ValidScreenResolutions {
		if res.Width == width && res.Height == height {
			return true
		}
	}
	return false
}

// IsValidTimezoneLocale checks if timezone and locale are consistent.
func IsValidTimezoneLocale(timezone, locale string) bool {
	allowedLocales, ok := TimezoneLocaleMap[timezone]
	if !ok {
		return false
	}
	for _, allowed := range allowedLocales {
		if len(locale) >= len(allowed) && locale[:len(allowed)] == allowed {
			return true
		}
	}
	return false
}

// PlatformGPUMap defines which GPU vendors are valid for each platform.
var PlatformGPUMap = map[string][]string{
	"Windows": {"NVIDIA", "AMD", "Intel"},
	"Mac":     {"Apple", "AMD", "Intel"},
	"Linux":   {"NVIDIA", "AMD", "Intel"},
	"iOS":     {"Apple"},
	"Android": {"Qualcomm", "Apple", "ARM", "Mali", "Adreno"},
}

// IsValidGPUForPlatform checks if a GPU vendor is valid for the given platform.
func IsValidGPUForPlatform(platform, gpuVendor string) bool {
	validGPUs, ok := PlatformGPUMap[platform]
	if !ok {
		return false
	}
	for _, valid := range validGPUs {
		if valid == gpuVendor {
			return true
		}
	}
	return false
}

// Now returns the current time for testing purposes.
var Now = func() time.Time {
	return time.Now()
}