package fingerprint

import (
	"testing"
)

// TestTCM101 tests Generate US fingerprint generation success.
func TestTCM101(t *testing.T) {
	g := NewGenerator()
	fp, err := g.Generate("test-seed-us", "US")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if fp == nil {
		t.Fatal("Expected fingerprint, got nil")
	}
	if fp.Seed != "test-seed-us" {
		t.Errorf("Expected seed 'test-seed-us', got '%s'", fp.Seed)
	}
	if fp.Platform != "Windows" {
		t.Errorf("Expected platform 'Windows', got '%s'", fp.Platform)
	}
	if fp.UserAgent == "" {
		t.Error("Expected non-empty UserAgent")
	}
	if fp.Timezone != "America/New_York" {
		t.Errorf("Expected timezone 'America/New_York', got '%s'", fp.Timezone)
	}
}

// TestTCM102 tests Generate DE fingerprint generation success.
func TestTCM102(t *testing.T) {
	g := NewGenerator()
	fp, err := g.Generate("test-seed-de", "DE")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if fp == nil {
		t.Fatal("Expected fingerprint, got nil")
	}
	if fp.Seed != "test-seed-de" {
		t.Errorf("Expected seed 'test-seed-de', got '%s'", fp.Seed)
	}
	if fp.Platform != "Windows" {
		t.Errorf("Expected platform 'Windows', got '%s'", fp.Platform)
	}
	if fp.Timezone != "Europe/Berlin" {
		t.Errorf("Expected timezone 'Europe/Berlin', got '%s'", fp.Timezone)
	}
}

// TestTCM103 tests GenerateRandom generates random fingerprints.
func TestTCM103(t *testing.T) {
	g := NewGenerator()
	fp1, err := g.GenerateRandom("US")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if fp1 == nil {
		t.Fatal("Expected fingerprint, got nil")
	}
	if fp1.Seed == "" {
		t.Error("Expected non-empty seed")
	}

	// Generate another random fingerprint
	fp2, err := g.GenerateRandom("US")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if fp2 == nil {
		t.Fatal("Expected fingerprint, got nil")
	}

	// They should be different (different seeds)
	if fp1.Seed == fp2.Seed {
		t.Error("Expected different seeds for GenerateRandom calls, but got same seed")
	}

	// Fingerprints should have different values
	if fp1.UserAgent == fp2.UserAgent && fp1.Screen.Width == fp2.Screen.Width && fp1.Screen.Height == fp2.Screen.Height {
		t.Error("Expected different fingerprints for different GenerateRandom calls")
	}
}

// TestTCM104 tests that same seed generates same fingerprint.
func TestTCM104(t *testing.T) {
	g := NewGenerator()
	seed := "deterministic-test-seed"

	fp1, err := g.Generate(seed, "US")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	fp2, err := g.Generate(seed, "US")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify all fields are identical
	if fp1.Seed != fp2.Seed {
		t.Errorf("Expected seed '%s', got '%s'", fp1.Seed, fp2.Seed)
	}
	if fp1.UserAgent != fp2.UserAgent {
		t.Errorf("Expected UserAgent '%s', got '%s'", fp1.UserAgent, fp2.UserAgent)
	}
	if fp1.Platform != fp2.Platform {
		t.Errorf("Expected Platform '%s', got '%s'", fp1.Platform, fp2.Platform)
	}
	if fp1.Screen.Width != fp2.Screen.Width {
		t.Errorf("Expected Screen.Width '%d', got '%d'", fp1.Screen.Width, fp2.Screen.Width)
	}
	if fp1.Screen.Height != fp2.Screen.Height {
		t.Errorf("Expected Screen.Height '%d', got '%d'", fp1.Screen.Height, fp2.Screen.Height)
	}
	if fp1.Screen.PixelRatio != fp2.Screen.PixelRatio {
		t.Errorf("Expected Screen.PixelRatio '%.2f', got '%.2f'", fp1.Screen.PixelRatio, fp2.Screen.PixelRatio)
	}
	if fp1.Timezone != fp2.Timezone {
		t.Errorf("Expected Timezone '%s', got '%s'", fp1.Timezone, fp2.Timezone)
	}
	if fp1.Locale != fp2.Locale {
		t.Errorf("Expected Locale '%s', got '%s'", fp1.Locale, fp2.Locale)
	}
	if fp1.WebGL.Renderer != fp2.WebGL.Renderer {
		t.Errorf("Expected WebGL.Renderer '%s', got '%s'", fp1.WebGL.Renderer, fp2.WebGL.Renderer)
	}
	if fp1.WebGL.Vendor != fp2.WebGL.Vendor {
		t.Errorf("Expected WebGL.Vendor '%s', got '%s'", fp1.WebGL.Vendor, fp2.WebGL.Vendor)
	}
	if fp1.Hardware.GPUVendor != fp2.Hardware.GPUVendor {
		t.Errorf("Expected Hardware.GPUVendor '%s', got '%s'", fp1.Hardware.GPUVendor, fp2.Hardware.GPUVendor)
	}
	if fp1.Hardware.CPUCores != fp2.Hardware.CPUCores {
		t.Errorf("Expected Hardware.CPUCores '%d', got '%d'", fp1.Hardware.CPUCores, fp2.Hardware.CPUCores)
	}
	if fp1.Network.ConnectionType != fp2.Network.ConnectionType {
		t.Errorf("Expected Network.ConnectionType '%s', got '%s'", fp1.Network.ConnectionType, fp2.Network.ConnectionType)
	}
}

// TestGenerateInvalidSeed tests that empty seed returns error.
func TestGenerateInvalidSeed(t *testing.T) {
	g := NewGenerator()
	fp, err := g.Generate("", "US")
	if err != ErrInvalidSeed {
		t.Errorf("Expected ErrInvalidSeed, got: %v", err)
	}
	if fp != nil {
		t.Error("Expected nil fingerprint for invalid seed")
	}
}

// TestGenerateInvalidCountry tests that invalid country returns error.
func TestGenerateInvalidCountry(t *testing.T) {
	g := NewGenerator()
	fp, err := g.Generate("test-seed", "XX")
	if err != ErrInvalidCountry {
		t.Errorf("Expected ErrInvalidCountry, got: %v", err)
	}
	if fp != nil {
		t.Error("Expected nil fingerprint for invalid country")
	}
}

// TestGenerateRandomInvalidCountry tests that GenerateRandom with invalid country returns error.
func TestGenerateRandomInvalidCountry(t *testing.T) {
	g := NewGenerator()
	fp, err := g.GenerateRandom("XX")
	if err != ErrInvalidCountry {
		t.Errorf("Expected ErrInvalidCountry, got: %v", err)
	}
	if fp != nil {
		t.Error("Expected nil fingerprint for invalid country")
	}
}

// TestSupportedCountries tests that supported countries are available.
func TestSupportedCountries(t *testing.T) {
	countries := SupportedCountries()
	if len(countries) < 10 {
		t.Errorf("Expected at least 10 supported countries, got %d", len(countries))
	}

	// Check specific countries
	expectedCountries := []string{"US", "DE", "GB", "FR", "JP", "CN", "CA", "AU", "BR", "IN", "IT", "ES"}
	for _, expected := range expectedCountries {
		found := false
		for _, country := range countries {
			if country == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected country '%s' to be supported", expected)
		}
	}
}

// TestIsSupportedCountry tests IsSupportedCountry function.
func TestIsSupportedCountry(t *testing.T) {
	if !IsSupportedCountry("US") {
		t.Error("Expected US to be supported")
	}
	if !IsSupportedCountry("DE") {
		t.Error("Expected DE to be supported")
	}
	if IsSupportedCountry("XX") {
		t.Error("Expected XX to not be supported")
	}
}

// TestDifferentSeedsDifferentFingerprints tests that different seeds produce different fingerprints.
func TestDifferentSeedsDifferentFingerprints(t *testing.T) {
	g := NewGenerator()

	fp1, err := g.Generate("seed-1", "US")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	fp2, err := g.Generate("seed-2", "US")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// They should have different UserAgents or Screen configs
	sameUA := fp1.UserAgent == fp2.UserAgent
	sameScreen := fp1.Screen.Width == fp2.Screen.Width && fp1.Screen.Height == fp2.Screen.Height
	sameTimezone := fp1.Timezone == fp2.Timezone
	sameLocale := fp1.Locale == fp2.Locale

	if sameUA && sameScreen && sameTimezone && sameLocale {
		t.Error("Expected different fingerprints for different seeds, but got same")
	}
}

// TestDeterministicRandIntn tests the deterministic random number generator.
func TestDeterministicRandIntn(t *testing.T) {
	rng := NewDeterministicRand(12345)

	// Generate sequence of random numbers
	vals := make([]int, 10)
	for i := 0; i < 10; i++ {
		vals[i] = rng.Intn(100)
	}

	// Re-create and verify same sequence
	rng2 := NewDeterministicRand(12345)
	for i := 0; i < 10; i++ {
		val2 := rng2.Intn(100)
		if vals[i] != val2 {
			t.Errorf("DeterministicRand not deterministic: index %d got %d, expected %d", i, val2, vals[i])
		}
	}
}

// TestDeterministicRandFloat64 tests the deterministic random float64 generator.
func TestDeterministicRandFloat64(t *testing.T) {
	rng := NewDeterministicRand(54321)

	// Generate sequence of random floats
	vals := make([]float64, 10)
	for i := 0; i < 10; i++ {
		vals[i] = rng.Float64()
	}

	// Re-create and verify same sequence
	rng2 := NewDeterministicRand(54321)
	for i := 0; i < 10; i++ {
		val2 := rng2.Float64()
		if vals[i] != val2 {
			t.Errorf("DeterministicRand.Float64 not deterministic: index %d got %f, expected %f", i, val2, vals[i])
		}
	}
}