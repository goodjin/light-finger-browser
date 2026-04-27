package fingerprint

import (
	"strings"
	"testing"
)

// TestTCM105 tests that Windows UA + Apple GPU returns error.
func TestTCM105(t *testing.T) {
	v := NewValidator()
	g := NewGenerator()

	// Generate a Windows fingerprint
	fp, err := g.Generate("test-seed-windows", "US")
	if err != nil {
		t.Fatalf("Failed to generate fingerprint: %v", err)
	}

	// Manually set Apple GPU (invalid for Windows)
	fp.Hardware.GPUVendor = "Apple"
	fp.Hardware.GPUModel = "M1"
	fp.Hardware.GPURenderer = "Apple M1"

	err = v.Validate(fp)
	if err == nil {
		t.Error("Expected error for Windows + Apple GPU, got nil")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	} else {
		if !strings.Contains(validationErr.Message, "Apple GPU") && !strings.Contains(validationErr.Message, "platform") {
			t.Errorf("Expected error about Apple GPU or platform, got: %s", validationErr.Message)
		}
	}
}

// TestTCM106 tests that macOS UA + Apple GPU validates successfully.
func TestTCM106(t *testing.T) {
	v := NewValidator()
	g := NewGenerator()

	// Generate a fingerprint and manually set Mac platform with Apple GPU
	fp, err := g.Generate("test-seed-mac", "US")
	if err != nil {
		t.Fatalf("Failed to generate fingerprint: %v", err)
	}

	// Set Mac platform with Apple GPU (valid)
	fp.Platform = "Mac"
	fp.Hardware.GPUVendor = "Apple"
	fp.Hardware.GPUModel = "M1"
	fp.Hardware.GPURenderer = "Apple M1"

	err = v.Validate(fp)
	if err != nil {
		t.Errorf("Expected no error for Mac + Apple GPU, got: %v", err)
	}
}

// TestTCM107 tests that invalid country returns error.
func TestTCM107(t *testing.T) {
	v := NewValidator()
	g := NewGenerator()

	// Generate with invalid country
	fp, err := g.Generate("test-seed", "XX")
	if err != nil {
		// This should fail at generation time
		if err != ErrInvalidCountry {
			t.Errorf("Expected ErrInvalidCountry, got: %v", err)
		}
		return
	}

	// If generation somehow succeeded (shouldn't), validation should catch it
	err = v.Validate(fp)
	if err != nil {
		t.Logf("Validation error (expected): %v", err)
	}
}

// TestTCM108 tests that same seed repeated calls return same fingerprint.
func TestTCM108(t *testing.T) {
	g := NewGenerator()
	v := NewValidator()
	seed := "repeatable-seed"

	// Generate multiple times with same seed
	fp1, err := g.Generate(seed, "US")
	if err != nil {
		t.Fatalf("First Generate failed: %v", err)
	}

	fp2, err := g.Generate(seed, "US")
	if err != nil {
		t.Fatalf("Second Generate failed: %v", err)
	}

	// Both should be identical
	if fp1.Seed != fp2.Seed ||
		fp1.UserAgent != fp2.UserAgent ||
		fp1.Platform != fp2.Platform ||
		fp1.Timezone != fp2.Timezone ||
		fp1.Locale != fp2.Locale {
		t.Error("Same seed should produce identical fingerprint")
	}

	// Both should validate successfully
	err = v.Validate(fp1)
	if err != nil {
		t.Errorf("First fingerprint validation failed: %v", err)
	}

	err = v.Validate(fp2)
	if err != nil {
		t.Errorf("Second fingerprint validation failed: %v", err)
	}
}

// TestValidateNilFingerprint tests that nil fingerprint returns error.
func TestValidateNilFingerprint(t *testing.T) {
	v := NewValidator()
	err := v.Validate(nil)
	if err != ErrInvalidFingerprint {
		t.Errorf("Expected ErrInvalidFingerprint, got: %v", err)
	}
}

// TestValidateInconsistentTimezoneLocale tests timezone/locale consistency.
func TestValidateInconsistentTimezoneLocale(t *testing.T) {
	v := NewValidator()
	g := NewGenerator()

	fp, err := g.Generate("test-seed-tz", "US")
	if err != nil {
		t.Fatalf("Failed to generate fingerprint: %v", err)
	}

	// Manually set inconsistent timezone/locale
	fp.Timezone = "Europe/Berlin"
	fp.Locale = "en-US"

	err = v.Validate(fp)
	if err == nil {
		t.Error("Expected error for inconsistent timezone/locale, got nil")
	}
}

// TestValidateScreenResolution tests screen resolution validation.
func TestValidateScreenResolution(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name      string
		width     int
		height    int
		expectErr bool
	}{
		{"valid resolution", 1920, 1080, false},
		{"valid resolution 1366x768", 1366, 768, false},
		{"too small", 100, 100, true},
		{"too large", 10000, 10000, true},
		{"zero width", 0, 1080, true},
		{"negative height", 1920, -1, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fp := &Fingerprint{
				Seed:      "test",
				Platform:  "Windows",
				Screen:    ScreenConfig{Width: tc.width, Height: tc.height, PixelRatio: 1.0},
				Timezone:  "America/New_York",
				Locale:    "en-US",
				WebGL:     WebGLConfig{Renderer: "NVIDIA", Vendor: "NVIDIA"},
				Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
				Network:   NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: 20},
			}

			err := v.Validate(fp)
			if tc.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestValidateUnusualPixelRatio tests unusual pixel ratio detection.
func TestValidateUnusualPixelRatio(t *testing.T) {
	v := NewValidator()

	fp := &Fingerprint{
		Seed:      "test",
		Platform:  "Windows",
		Screen:    ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 5.0}, // Unusual
		Timezone:  "America/New_York",
		Locale:    "en-US",
		WebGL:     WebGLConfig{Renderer: "NVIDIA", Vendor: "NVIDIA"},
		Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
		Network:   NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: 20},
	}

	err := v.Validate(fp)
	if err == nil {
		t.Error("Expected error for unusual pixel ratio, got nil")
	}
}

// TestValidateWebGLVendor tests WebGL vendor validation.
func TestValidateWebGLVendor(t *testing.T) {
	v := NewValidator()

	fp := &Fingerprint{
		Seed:      "test",
		Platform:  "Windows",
		Screen:    ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
		Timezone:  "America/New_York",
		Locale:    "en-US",
		WebGL:     WebGLConfig{Renderer: "Unknown GPU", Vendor: "UnknownVendor"},
		Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
		Network:   NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: 20},
	}

	err := v.Validate(fp)
	if err == nil {
		t.Error("Expected error for unknown WebGL vendor, got nil")
	}
}

// TestValidateNetworkConnectionType tests network connection type validation.
func TestValidateNetworkConnectionType(t *testing.T) {
	v := NewValidator()

	fp := &Fingerprint{
		Seed:      "test",
		Platform:  "Windows",
		Screen:    ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
		Timezone:  "America/New_York",
		Locale:    "en-US",
		WebGL:     WebGLConfig{Renderer: "NVIDIA", Vendor: "NVIDIA"},
		Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
		Network:   NetworkConfig{ConnectionType: "invalid", Downlink: 10, RTT: 20},
	}

	err := v.Validate(fp)
	if err == nil {
		t.Error("Expected error for invalid connection type, got nil")
	}
}

// TestValidateNetworkRTT tests network RTT validation.
func TestValidateNetworkRTT(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name      string
		rtt       int
		expectErr bool
	}{
		{"valid RTT", 50, false},
		{"zero RTT", 0, true},
		{"negative RTT", -10, true},
		{"extremely high RTT", 6000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fp := &Fingerprint{
				Seed:      "test",
				Platform:  "Windows",
				Screen:    ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
				Timezone:  "America/New_York",
				Locale:    "en-US",
				WebGL:     WebGLConfig{Renderer: "NVIDIA", Vendor: "NVIDIA"},
				Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
				Network:   NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: tc.rtt},
			}

			err := v.Validate(fp)
			if tc.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestValidateNetworkDownlink tests network downlink validation.
func TestValidateNetworkDownlink(t *testing.T) {
	v := NewValidator()

	fp := &Fingerprint{
		Seed:      "test",
		Platform:  "Windows",
		Screen:    ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
		Timezone:  "America/New_York",
		Locale:    "en-US",
		WebGL:     WebGLConfig{Renderer: "NVIDIA", Vendor: "NVIDIA"},
		Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
		Network:   NetworkConfig{ConnectionType: "wifi", Downlink: -5.0, RTT: 20},
	}

	err := v.Validate(fp)
	if err == nil {
		t.Error("Expected error for negative downlink, got nil")
	}
}

// TestValidateEmptyWebGLRenderer tests that empty WebGL renderer returns error.
func TestValidateEmptyWebGLRenderer(t *testing.T) {
	v := NewValidator()

	fp := &Fingerprint{
		Seed:      "test",
		Platform:  "Windows",
		Screen:    ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
		Timezone:  "America/New_York",
		Locale:    "en-US",
		WebGL:     WebGLConfig{Renderer: "", Vendor: "NVIDIA"},
		Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
		Network:   NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: 20},
	}

	err := v.Validate(fp)
	if err == nil {
		t.Error("Expected error for empty WebGL renderer, got nil")
	}
}

// TestIsValidGPUForPlatform tests GPU platform validation.
func TestIsValidGPUForPlatform(t *testing.T) {
	testCases := []struct {
		platform  string
		gpuVendor string
		valid     bool
	}{
		{"Windows", "NVIDIA", true},
		{"Windows", "AMD", true},
		{"Windows", "Intel", true},
		{"Windows", "Apple", false}, // Apple GPU not valid for Windows
		{"Mac", "Apple", true},
		{"Mac", "AMD", true},
		{"Mac", "NVIDIA", false}, // NVIDIA not valid for Mac
		{"Linux", "NVIDIA", true},
		{"Linux", "AMD", true},
		{"iOS", "Apple", true},
		{"Android", "Qualcomm", true},
		{"Android", "Apple", true},
		{"Unknown", "NVIDIA", false},
	}

	for _, tc := range testCases {
		t.Run(tc.platform+"_"+tc.gpuVendor, func(t *testing.T) {
			result := IsValidGPUForPlatform(tc.platform, tc.gpuVendor)
			if result != tc.valid {
				t.Errorf("IsValidGPUForPlatform(%s, %s) = %v, expected %v", tc.platform, tc.gpuVendor, result, tc.valid)
			}
		})
	}
}

// TestIsValidTimezoneLocale tests timezone/locale validation.
func TestIsValidTimezoneLocale(t *testing.T) {
	testCases := []struct {
		timezone string
		locale   string
		valid    bool
	}{
		{"America/New_York", "en-US", true},
		{"America/Los_Angeles", "en-US", true},
		{"Europe/Berlin", "de-DE", true},
		{"Europe/Paris", "fr-FR", true},
		{"Asia/Tokyo", "ja-JP", true},
		{"Asia/Shanghai", "zh-CN", true},
		{"America/New_York", "ja-JP", false}, // Mismatch
		{"Invalid/Timezone", "en-US", false},
	}

	for _, tc := range testCases {
		t.Run(tc.timezone+"_"+tc.locale, func(t *testing.T) {
			result := IsValidTimezoneLocale(tc.timezone, tc.locale)
			if result != tc.valid {
				t.Errorf("IsValidTimezoneLocale(%s, %s) = %v, expected %v", tc.timezone, tc.locale, result, tc.valid)
			}
		})
	}
}

// TestIsValidScreenResolution tests screen resolution validation.
func TestIsValidScreenResolution(t *testing.T) {
	testCases := []struct {
		width    int
		height   int
		expected bool
	}{
		{1920, 1080, true},
		{1366, 768, true},
		{1536, 864, true},
		{2560, 1440, true},
		{3840, 2160, true},
		{1280, 720, true},
		{9999, 9999, false}, // Not in valid list
		{100, 100, false},  // Not in valid list
	}

	for _, tc := range testCases {
		result := IsValidScreenResolution(tc.width, tc.height)
		if result != tc.expected {
			t.Errorf("IsValidScreenResolution(%d, %d) = %v, expected %v", tc.width, tc.height, result, tc.expected)
		}
	}
}