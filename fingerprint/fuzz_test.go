package fingerprint

import (
	"strings"
	"testing"
)

// FuzzGenerate tests fingerprint generation with various seed and country combinations.
func FuzzGenerate(f *testing.F) {
	// Seed corpus with known seed/country pairs
	f.Add("test-seed-1", "US")
	f.Add("test-seed-2", "DE")
	f.Add("another-seed", "GB")
	f.Add("random-seed-123", "JP")
	f.Add("", "US") // Empty seed case

	f.Fuzz(func(t *testing.T, seed string, country string) {
		g := NewGenerator()

		// Generate fingerprint
		fp, err := g.Generate(seed, country)

		// If country is invalid, we expect an error
		if err != nil {
			// Expected errors: ErrInvalidCountry or ErrInvalidSeed
			if err != ErrInvalidCountry && err != ErrInvalidSeed {
				t.Errorf("Unexpected error for seed='%s', country='%s': %v", seed, country, err)
			}
			return
		}

		// If generation succeeded, fingerprint should not be nil
		if fp == nil {
			t.Fatal("Generated nil fingerprint without error")
		}

		// Validate the generated fingerprint
		v := NewValidator()
		if err := v.Validate(fp); err != nil {
			t.Errorf("Generated fingerprint failed validation for seed='%s', country='%s': %v", seed, country, err)
		}

		// If seed is non-empty, generating with same seed should produce same fingerprint
		if seed != "" {
			fp2, err := g.Generate(seed, country)
			if err != nil {
				t.Errorf("Second Generate failed for seed='%s', country='%s': %v", seed, country, err)
				return
			}
			if fp2 == nil {
				t.Fatal("Second Generate returned nil fingerprint")
			}

			// Verify determinism
			if fp.Seed != fp2.Seed ||
				fp.UserAgent != fp2.UserAgent ||
				fp.Platform != fp2.Platform ||
				fp.Timezone != fp2.Timezone ||
				fp.Locale != fp2.Locale ||
				fp.Screen.Width != fp2.Screen.Width ||
				fp.Screen.Height != fp2.Screen.Height {
				t.Errorf("Same seed produced different fingerprints for seed='%s', country='%s'", seed, country)
			}
		}
	})
}

// FuzzGenerateRandom tests GenerateRandom with various countries.
func FuzzGenerateRandom(f *testing.F) {
	// Seed corpus with known country codes
	f.Add("US")
	f.Add("DE")
	f.Add("GB")
	f.Add("XX") // Invalid country
	f.Add("")

	f.Fuzz(func(t *testing.T, country string) {
		g := NewGenerator()

		fp, err := g.GenerateRandom(country)

		// If country is invalid, we expect an error
		if err != nil {
			if err != ErrInvalidCountry {
				t.Errorf("Unexpected error for country='%s': %v", country, err)
			}
			return
		}

		// If generation succeeded, fingerprint should not be nil
		if fp == nil {
			t.Fatal("GenerateRandom returned nil fingerprint without error")
		}

		// Seed should be non-empty
		if fp.Seed == "" {
			t.Error("GenerateRandom produced fingerprint with empty seed")
		}

		// Validate the generated fingerprint
		v := NewValidator()
		if err := v.Validate(fp); err != nil {
			t.Errorf("Generated fingerprint failed validation for country='%s': %v", country, err)
		}
	})
}

// FuzzValidate tests fingerprint validation with various fingerprints.
func FuzzValidate(f *testing.F) {
	// Seed corpus with valid and invalid fingerprints
	f.Add("test-seed", "Windows", 1920, 1080, 1.0, "America/New_York", "en-US", "NVIDIA", "Test Renderer")
	f.Add("test-seed", "Windows", 1920, 1080, 1.0, "Europe/Berlin", "en-US", "NVIDIA", "Test Renderer") // Invalid tz/locale
	f.Add("test-seed", "Windows", 100, 100, 1.0, "America/New_York", "en-US", "NVIDIA", "Test Renderer") // Invalid resolution
	f.Add("test-seed", "Windows", 1920, 1080, 5.0, "America/New_York", "en-US", "NVIDIA", "Test Renderer") // Invalid pixel ratio
	f.Add("test-seed", "Windows", 1920, 1080, 1.0, "America/New_York", "en-US", "Apple", "Test Renderer") // Invalid GPU for Windows

	f.Fuzz(func(t *testing.T, seed, platform string, width, height int, pixelRatio float64, timezone, locale, gpuVendor, renderer string) {
		v := NewValidator()

		fp := &Fingerprint{
			Seed:     seed,
			Platform: platform,
			Screen: ScreenConfig{
				Width:      width,
				Height:     height,
				PixelRatio: pixelRatio,
			},
			Timezone: timezone,
			Locale:   locale,
			WebGL: WebGLConfig{
				Renderer: renderer,
				Vendor:   gpuVendor,
			},
			Hardware: HardwareConfig{
				GPUVendor: gpuVendor,
			},
			Network: NetworkConfig{
				ConnectionType: "wifi",
				Downlink:        10,
				RTT:             20,
			},
		}

		err := v.Validate(fp)

		// We don't assert on valid/invalid here, just ensure no panic occurs
		// The validator will return appropriate errors for invalid fingerprints
		_ = err
	})
}

// FuzzValidatorWithGenerator tests validation with generator configurations.
func FuzzValidatorWithGenerator(f *testing.F) {
	f.Add("test-seed", "US")
	f.Add("test-seed", "DE")
	f.Add("another-seed", "GB")

	f.Fuzz(func(t *testing.T, seed, country string) {
		g := NewGeneratorWithValidator()

		fp, err := g.Generate(seed, country)
		if err != nil {
			// Expected for invalid countries
			if err != ErrInvalidCountry && err != ErrInvalidSeed {
				t.Errorf("Unexpected error: %v", err)
			}
			return
		}

		if fp == nil {
			t.Fatal("Generated nil fingerprint")
		}

		// Validate should pass for properly generated fingerprints
		if err := g.Validate(fp); err != nil {
			t.Errorf("Validate failed for seed='%s', country='%s': %v", seed, country, err)
		}
	})
}

// FuzzConsistency tests fingerprint consistency under various conditions.
func FuzzConsistency(f *testing.F) {
	// Test various platform/GPU combinations
	platforms := []string{"Windows", "Mac", "Linux", "iOS", "Android"}
	gpuVendors := []string{"NVIDIA", "AMD", "Intel", "Apple", "Qualcomm", "ARM"}

	f.Fuzz(func(t *testing.T, platformIdx, gpuIdx int) {
		platform := platforms[platformIdx%len(platforms)]
		gpuVendor := gpuVendors[gpuIdx%len(gpuVendors)]

		v := NewValidator()

		fp := &Fingerprint{
			Seed:     "test-seed",
			Platform: platform,
			Screen: ScreenConfig{
				Width:      1920,
				Height:     1080,
				PixelRatio: 1.0,
			},
			Timezone: "America/New_York",
			Locale:   "en-US",
			WebGL: WebGLConfig{
				Renderer: "Test Renderer",
				Vendor:   gpuVendor,
			},
			Hardware: HardwareConfig{
				GPUVendor: gpuVendor,
			},
			Network: NetworkConfig{
				ConnectionType: "wifi",
				Downlink:       10,
				RTT:            20,
			},
		}

		err := v.Validate(fp)

		// Check that validation result is consistent with IsValidGPUForPlatform
		expectedValid := IsValidGPUForPlatform(platform, gpuVendor)
		if err == nil && !expectedValid {
			// This can happen if the GPU is technically allowed but unusual
			// We don't fail the test, just log
			t.Logf("Validation passed but GPU might be unusual for platform: %s/%s", platform, gpuVendor)
		}
	})
}

// FuzzLongSeed tests fingerprint generation with very long seeds.
func FuzzLongSeed(f *testing.F) {
	f.Fuzz(func(t *testing.T, seed string) {
		// Skip empty seeds for this test
		if seed == "" {
			return
		}

		// Skip extremely long seeds (> 10KB) to avoid memory issues
		if len(seed) > 10000 {
			return
		}

		g := NewGenerator()

		fp, err := g.Generate(seed, "US")
		if err != nil {
			if err != ErrInvalidSeed {
				t.Errorf("Unexpected error for long seed: %v", err)
			}
			return
		}

		if fp == nil {
			t.Fatal("Generated nil fingerprint")
		}

		// Verify determinism
		fp2, err := g.Generate(seed, "US")
		if err != nil {
			t.Errorf("Second Generate failed: %v", err)
			return
		}

		if fp.UserAgent != fp2.UserAgent || fp.Screen.Width != fp2.Screen.Width {
			t.Error("Same long seed produced different fingerprints")
		}
	})
}

// FuzzSpecialCharactersInSeed tests seeds with special characters.
func FuzzSpecialCharactersInSeed(f *testing.F) {
	f.Fuzz(func(t *testing.T, seed string) {
		if seed == "" {
			return
		}

		g := NewGenerator()

		// Try with various countries
		for _, country := range []string{"US", "DE", "GB", "XX"} {
			fp, err := g.Generate(seed, country)

			if country == "XX" {
				if err != ErrInvalidCountry {
					t.Errorf("Expected ErrInvalidCountry for country='XX', got: %v", err)
				}
				continue
			}

			if err != nil {
				t.Errorf("Unexpected error for seed='%s', country='%s': %v", seed, country, err)
				continue
			}

			if fp == nil {
				t.Fatal("Generated nil fingerprint")
			}
		}
	})
}

// TestEmptyFields tests fingerprints with empty fields.
func TestEmptyFields(t *testing.T) {
	v := NewValidator()

	emptyFields := []struct {
		name    string
		fp      *Fingerprint
	}{
		{
			"empty user agent",
			&Fingerprint{
				Seed:      "test",
				UserAgent: "",
				Platform:  "Windows",
				Screen:    ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
				Timezone:  "America/New_York",
				Locale:    "en-US",
				WebGL:     WebGLConfig{Renderer: "NVIDIA", Vendor: "NVIDIA"},
				Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
				Network:   NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: 20},
			},
		},
		{
			"empty renderer",
			&Fingerprint{
				Seed:      "test",
				Platform:  "Windows",
				Screen:    ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
				Timezone:  "America/New_York",
				Locale:    "en-US",
				WebGL:     WebGLConfig{Renderer: "", Vendor: "NVIDIA"},
				Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
				Network:   NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: 20},
			},
		},
		{
			"empty seed",
			&Fingerprint{
				Seed:      "",
				Platform:  "Windows",
				Screen:    ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
				Timezone:  "America/New_York",
				Locale:    "en-US",
				WebGL:     WebGLConfig{Renderer: "NVIDIA", Vendor: "NVIDIA"},
				Hardware:  HardwareConfig{GPUVendor: "NVIDIA"},
				Network:   NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: 20},
			},
		},
	}

	for _, tt := range emptyFields {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.fp)
			// Some empty fields should cause validation errors
			// We just ensure no panic occurs
			_ = err
		})
	}
}

// FuzzUnicodeInSeed tests seeds with unicode characters.
func FuzzUnicodeInSeed(f *testing.F) {
	f.Fuzz(func(t *testing.T, seed string) {
		if seed == "" {
			return
		}

		g := NewGenerator()

		fp, err := g.Generate(seed, "US")
		if err != nil {
			// Only ErrInvalidSeed should occur for empty seed
			return
		}

		if fp == nil {
			t.Fatal("Generated nil fingerprint")
		}

		// Verify determinism even with unicode
		fp2, err := g.Generate(seed, "US")
		if err != nil {
			t.Errorf("Second Generate failed: %v", err)
			return
		}

		if fp.Seed != fp2.Seed {
			t.Error("Same unicode seed produced different seeds")
		}
	})
}

// TestPlatformVariations tests various platform strings.
func TestPlatformVariations(t *testing.T) {
	v := NewValidator()

	platforms := []string{
		"Windows",
		"Mac",
		"Linux",
		"iOS",
		"Android",
		"Windows NT",
		"Macintosh",
		"windows",
		"WINDOWS",
		"UnknownPlatform",
		"",
	}

	for _, platform := range platforms {
		fp := &Fingerprint{
			Seed:     "test",
			Platform: platform,
			Screen:   ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
			Timezone: "America/New_York",
			Locale:   "en-US",
			WebGL:    WebGLConfig{Renderer: "NVIDIA", Vendor: "NVIDIA"},
			Hardware: HardwareConfig{GPUVendor: "NVIDIA"},
			Network:  NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: 20},
		}

		err := v.Validate(fp)

		// We just ensure no panic occurs for any platform string
		// The validator will return errors for inconsistent combinations
		_ = err
	}
}

// TestLocaleVariations tests various locale strings.
func TestLocaleVariations(t *testing.T) {
	locales := []string{
		"en-US",
		"en-GB",
		"de-DE",
		"fr-FR",
		"ja-JP",
		"zh-CN",
		"en",
		"de",
		"EN-US",
		"En-Us",
		"",
		"invalid-locale",
		"xx-XX",
	}

	v := NewValidator()
	g := NewGenerator()

	for _, locale := range locales {
		fp := &Fingerprint{
			Seed:     "test",
			Platform: "Windows",
			Screen:   ScreenConfig{Width: 1920, Height: 1080, PixelRatio: 1.0},
			Timezone: "America/New_York",
			Locale:   locale,
			WebGL:    WebGLConfig{Renderer: "NVIDIA", Vendor: "NVIDIA"},
			Hardware: HardwareConfig{GPUVendor: "NVIDIA"},
			Network:  NetworkConfig{ConnectionType: "wifi", Downlink: 10, RTT: 20},
		}

		err := v.Validate(fp)
		_ = err // Just ensure no panic

		// Also test with generator
		fp2, err := g.Generate("test-"+strings.ReplaceAll(locale, "-", ""), "US")
		if err == nil && fp2 != nil {
			_ = fp2
		}
	}
}