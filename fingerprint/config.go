package fingerprint

// countryConfigs maps country codes to their fingerprint configurations.
// Each configuration defines platform, user agent, timezone, locale prefix,
// screen resolutions, WebGL renderers, and GPU options for that country.
var countryConfigs = map[string]CountryConfig{
	"US": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "America/New_York",
		LocalePrefix: "en",
		Resolutions: [][]int{
			{1920, 1080}, {1366, 768}, {1536, 864}, {2560, 1440}, {1280, 720},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1070",
			"NVIDIA GeForce GTX 1080",
			"NVIDIA GeForce RTX 2060",
			"NVIDIA GeForce RTX 3070",
			"AMD Radeon RX 580",
			"AMD Radeon RX 590",
			"AMD Radeon RX 6700 XT",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1070", Renderer: "NVIDIA GeForce GTX 1070"},
			{Vendor: "NVIDIA", Model: "GTX 1080", Renderer: "NVIDIA GeForce GTX 1080"},
			{Vendor: "NVIDIA", Model: "RTX 2060", Renderer: "NVIDIA GeForce RTX 2060"},
			{Vendor: "NVIDIA", Model: "RTX 3070", Renderer: "NVIDIA GeForce RTX 3070"},
			{Vendor: "AMD", Model: "RX 580", Renderer: "AMD Radeon RX 580"},
			{Vendor: "AMD", Model: "RX 590", Renderer: "AMD Radeon RX 590"},
			{Vendor: "AMD", Model: "RX 6700 XT", Renderer: "AMD Radeon RX 6700 XT"},
		},
	},
	"DE": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "Europe/Berlin",
		LocalePrefix: "de",
		Resolutions: [][]int{
			{1920, 1080}, {1366, 768}, {1536, 864}, {3840, 2160}, {2560, 1440},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1060",
			"NVIDIA GeForce GTX 1070",
			"AMD Radeon RX 580",
			"AMD Radeon RX Vega 56",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1060", Renderer: "NVIDIA GeForce GTX 1060"},
			{Vendor: "NVIDIA", Model: "GTX 1070", Renderer: "NVIDIA GeForce GTX 1070"},
			{Vendor: "AMD", Model: "RX 580", Renderer: "AMD Radeon RX 580"},
			{Vendor: "AMD", Model: "RX Vega 56", Renderer: "AMD Radeon RX Vega 56"},
		},
	},
	"GB": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "Europe/London",
		LocalePrefix: "en",
		Resolutions: [][]int{
			{1920, 1080}, {1366, 768}, {1440, 900}, {1536, 864}, {1280, 720},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1050 Ti",
			"NVIDIA GeForce GTX 1060",
			"AMD Radeon RX 550",
			"Intel UHD Graphics 630",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1050 Ti", Renderer: "NVIDIA GeForce GTX 1050 Ti"},
			{Vendor: "NVIDIA", Model: "GTX 1060", Renderer: "NVIDIA GeForce GTX 1060"},
			{Vendor: "AMD", Model: "RX 550", Renderer: "AMD Radeon RX 550"},
			{Vendor: "Intel", Model: "UHD 630", Renderer: "Intel UHD Graphics 630"},
		},
	},
	"FR": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "Europe/Paris",
		LocalePrefix: "fr",
		Resolutions: [][]int{
			{1920, 1080}, {1600, 900}, {1366, 768}, {1536, 864}, {1280, 720},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1070",
			"AMD Radeon RX 580",
			"AMD Radeon RX 590",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1070", Renderer: "NVIDIA GeForce GTX 1070"},
			{Vendor: "AMD", Model: "RX 580", Renderer: "AMD Radeon RX 580"},
			{Vendor: "AMD", Model: "RX 590", Renderer: "AMD Radeon RX 590"},
		},
	},
	"JP": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "Asia/Tokyo",
		LocalePrefix: "ja",
		Resolutions: [][]int{
			{1920, 1080}, {1366, 768}, {1280, 720}, {1536, 864}, {2560, 1440},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1060",
			"NVIDIA GeForce GTX 1070",
			"AMD Radeon RX 580",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1060", Renderer: "NVIDIA GeForce GTX 1060"},
			{Vendor: "NVIDIA", Model: "GTX 1070", Renderer: "NVIDIA GeForce GTX 1070"},
			{Vendor: "AMD", Model: "RX 580", Renderer: "AMD Radeon RX 580"},
		},
	},
	"CN": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "Asia/Shanghai",
		LocalePrefix: "zh",
		Resolutions: [][]int{
			{1920, 1080}, {1366, 768}, {1440, 900}, {1536, 864}, {1280, 720},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1050",
			"NVIDIA GeForce GTX 1050 Ti",
			"AMD Radeon RX 550",
			"Intel UHD Graphics 620",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1050", Renderer: "NVIDIA GeForce GTX 1050"},
			{Vendor: "NVIDIA", Model: "GTX 1050 Ti", Renderer: "NVIDIA GeForce GTX 1050 Ti"},
			{Vendor: "AMD", Model: "RX 550", Renderer: "AMD Radeon RX 550"},
			{Vendor: "Intel", Model: "UHD 620", Renderer: "Intel UHD Graphics 620"},
		},
	},
	"CA": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "America/Toronto",
		LocalePrefix: "en",
		Resolutions: [][]int{
			{1920, 1080}, {1366, 768}, {1536, 864}, {1280, 720}, {2560, 1440},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1070",
			"AMD Radeon RX 580",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1070", Renderer: "NVIDIA GeForce GTX 1070"},
			{Vendor: "AMD", Model: "RX 580", Renderer: "AMD Radeon RX 580"},
		},
	},
	"AU": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "Australia/Sydney",
		LocalePrefix: "en",
		Resolutions: [][]int{
			{1920, 1080}, {1366, 768}, {1440, 900}, {1280, 720}, {1536, 864},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1050 Ti",
			"AMD Radeon RX 560",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1050 Ti", Renderer: "NVIDIA GeForce GTX 1050 Ti"},
			{Vendor: "AMD", Model: "RX 560", Renderer: "AMD Radeon RX 560"},
		},
	},
	"BR": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "America/Sao_Paulo",
		LocalePrefix: "pt",
		Resolutions: [][]int{
			{1366, 768}, {1920, 1080}, {1280, 720}, {1536, 864}, {1600, 900},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 750 Ti",
			"AMD Radeon R7 370",
			"Intel HD Graphics 4000",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 750 Ti", Renderer: "NVIDIA GeForce GTX 750 Ti"},
			{Vendor: "AMD", Model: "R7 370", Renderer: "AMD Radeon R7 370"},
			{Vendor: "Intel", Model: "HD 4000", Renderer: "Intel HD Graphics 4000"},
		},
	},
	"IN": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "Asia/Kolkata",
		LocalePrefix: "en",
		Resolutions: [][]int{
			{1366, 768}, {1920, 1080}, {1280, 720}, {1536, 864}, {1600, 900},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1050",
			"AMD Radeon RX 560",
			"Intel UHD Graphics 620",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1050", Renderer: "NVIDIA GeForce GTX 1050"},
			{Vendor: "AMD", Model: "RX 560", Renderer: "AMD Radeon RX 560"},
			{Vendor: "Intel", Model: "UHD 620", Renderer: "Intel UHD Graphics 620"},
		},
	},
	"IT": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "Europe/Rome",
		LocalePrefix: "it",
		Resolutions: [][]int{
			{1920, 1080}, {1366, 768}, {1536, 864}, {1280, 720}, {1600, 900},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1060",
			"AMD Radeon RX 580",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1060", Renderer: "NVIDIA GeForce GTX 1060"},
			{Vendor: "AMD", Model: "RX 580", Renderer: "AMD Radeon RX 580"},
		},
	},
	"ES": {
		Platform:     "Windows",
		UserAgent:    "Windows NT 10.0; Win64; x64",
		Timezone:     "Europe/Madrid",
		LocalePrefix: "es",
		Resolutions: [][]int{
			{1920, 1080}, {1366, 768}, {1536, 864}, {1280, 720}, {1440, 900},
		},
		WebGLRenderers: []string{
			"NVIDIA GeForce GTX 1070",
			"AMD Radeon RX 580",
		},
		GPUs: []GPUConfig{
			{Vendor: "NVIDIA", Model: "GTX 1070", Renderer: "NVIDIA GeForce GTX 1070"},
			{Vendor: "AMD", Model: "RX 580", Renderer: "AMD Radeon RX 580"},
		},
	},
}

// GetCountryConfig returns the country configuration for a given country code.
func GetCountryConfig(country string) (CountryConfig, bool) {
	config, ok := countryConfigs[country]
	return config, ok
}

// SupportedCountries returns a list of all supported country codes.
func SupportedCountries() []string {
	countries := make([]string, 0, len(countryConfigs))
	for country := range countryConfigs {
		countries = append(countries, country)
	}
	return countries
}

// IsSupportedCountry checks if a country code is supported.
func IsSupportedCountry(country string) bool {
	_, ok := countryConfigs[country]
	return ok
}