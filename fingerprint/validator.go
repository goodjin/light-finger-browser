package fingerprint

import (
	"fmt"
	"strings"
)

// Validator validates fingerprints for consistency.
type Validator struct{}

// NewValidator creates a new Validator instance.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks the consistency of a fingerprint.
// It validates:
// - Hardware consistency (UA platform vs GPU vendor)
// - Timezone and locale consistency
// - Screen resolution validity
func (v *Validator) Validate(f *Fingerprint) error {
	if f == nil {
		return ErrInvalidFingerprint
	}

	// Rule-1 & Rule-2: UA Platform vs GPU Vendor consistency
	if err := v.validateHardwareConsistency(f); err != nil {
		return err
	}

	// Rule-3: Timezone and locale consistency
	if err := v.validateTimezoneLocale(f); err != nil {
		return err
	}

	// Rule-4: Screen resolution validity
	if err := v.validateScreenResolution(f); err != nil {
		return err
	}

	// Rule-5: WebGL configuration validity
	if err := v.validateWebGL(f); err != nil {
		return err
	}

	// Rule-6: Network configuration validity
	if err := v.validateNetwork(f); err != nil {
		return err
	}

	return nil
}

// validateHardwareConsistency checks if the GPU vendor is compatible with the platform.
func (v *Validator) validateHardwareConsistency(f *Fingerprint) error {
	platform := f.Platform
	gpuVendor := f.Hardware.GPUVendor

	if !IsValidGPUForPlatform(platform, gpuVendor) {
		return &ValidationError{
			Message: fmt.Sprintf("inconsistent hardware: GPU vendor '%s' is not valid for platform '%s'", gpuVendor, platform),
		}
	}

	// Additional check: Apple GPU should not appear with Windows UA
	if platform == "Windows" && gpuVendor == "Apple" {
		return &ValidationError{
			Message: "inconsistent hardware: Apple GPU cannot be used with Windows platform",
		}
	}

	// macOS should typically have Apple GPU
	if platform == "Mac" && gpuVendor != "Apple" && gpuVendor != "AMD" && gpuVendor != "Intel" {
		return &ValidationError{
			Message: fmt.Sprintf("inconsistent hardware: unusual GPU vendor '%s' for Mac platform", gpuVendor),
		}
	}

	return nil
}

// validateTimezoneLocale checks if timezone and locale are consistent.
func (v *Validator) validateTimezoneLocale(f *Fingerprint) error {
	timezone := f.Timezone
	locale := f.Locale

	if !IsValidTimezoneLocale(timezone, locale) {
		return &ValidationError{
			Message: fmt.Sprintf("inconsistent timezone/locale: timezone '%s' does not match locale '%s'", timezone, locale),
		}
	}

	// Additional validation: check if timezone offset is reasonable for the locale
	if strings.HasPrefix(locale, "en") {
		usTimezones := []string{"America/New_York", "America/Los_Angeles", "America/Chicago", "America/Denver"}
		for _, tz := range usTimezones {
			if timezone == tz && !strings.HasPrefix(locale, "en-US") && !strings.HasPrefix(locale, "en-CA") {
				// This is a warning, not an error
			}
		}
	}

	return nil
}

// validateScreenResolution checks if screen resolution is valid.
func (v *Validator) validateScreenResolution(f *Fingerprint) error {
	width := f.Screen.Width
	height := f.Screen.Height

	// Check if resolution is within valid bounds
	if width < 320 || height < 480 {
		return &ValidationError{
			Message: fmt.Sprintf("invalid screen resolution: %dx%d is too small", width, height),
		}
	}

	if width > 7680 || height > 4320 {
		return &ValidationError{
			Message: fmt.Sprintf("invalid screen resolution: %dx%d exceeds maximum", width, height),
		}
	}

	// Check if it's a standard resolution
	if !IsValidScreenResolution(width, height) {
		// Not all resolutions are in the valid list, but we only warn for unusual ones
		aspectRatio := float64(width) / float64(height)
		if aspectRatio < 1.2 || aspectRatio > 3.0 {
			return &ValidationError{
				Message: fmt.Sprintf("unusual aspect ratio: %d:%d (%.2f)", width, height, aspectRatio),
			}
		}
	}

	// Pixel ratio should be reasonable
	pixelRatio := f.Screen.PixelRatio
	if pixelRatio < 1.0 || pixelRatio > 4.0 {
		return &ValidationError{
			Message: fmt.Sprintf("unusual pixel ratio: %.2f", pixelRatio),
		}
	}

	return nil
}

// validateWebGL checks if WebGL configuration is valid.
func (v *Validator) validateWebGL(f *Fingerprint) error {
	webgl := f.WebGL

	// Renderer should not be empty
	if webgl.Renderer == "" {
		return &ValidationError{
			Message: "WebGL renderer cannot be empty",
		}
	}

	// Vendor should be one of the known vendors
	validVendors := []string{"NVIDIA", "AMD", "Intel", "Apple", "Qualcomm", "ARM", "Unknown"}
	isValidVendor := false
	for _, vendor := range validVendors {
		if webgl.Vendor == vendor {
			isValidVendor = true
			break
		}
	}
	if !isValidVendor {
		return &ValidationError{
			Message: fmt.Sprintf("unknown WebGL vendor: %s", webgl.Vendor),
		}
	}

	return nil
}

// validateNetwork checks if network configuration is valid.
func (v *Validator) validateNetwork(f *Fingerprint) error {
	network := f.Network

	// Connection type should be valid
	validConnectionTypes := []string{"wifi", "wired", "4g", "3g", "2g", "unknown"}
	isValidType := false
	for _, ct := range validConnectionTypes {
		if network.ConnectionType == ct {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return &ValidationError{
			Message: fmt.Sprintf("unknown connection type: %s", network.ConnectionType),
		}
	}

	// Downlink should be non-negative
	if network.Downlink < 0 {
		return &ValidationError{
			Message: fmt.Sprintf("invalid downlink: %.2f cannot be negative", network.Downlink),
		}
	}

	// RTT should be positive
	if network.RTT <= 0 {
		return &ValidationError{
			Message: fmt.Sprintf("invalid RTT: %d must be positive", network.RTT),
		}
	}

	// RTT should be within reasonable bounds
	if network.RTT > 5000 {
		return &ValidationError{
			Message: fmt.Sprintf("unusual RTT: %dms is extremely high", network.RTT),
		}
	}

	return nil
}

// ValidateWithGenerator validates a fingerprint using the generator's configurations.
func (v *Validator) ValidateWithGenerator(f *Fingerprint, g *Generator) error {
	if err := v.Validate(f); err != nil {
		return err
	}

	// Additional check: verify fingerprint matches expected country config
	// by regenerating with same seed and comparing
	config, ok := g.configs[f.Locale[:2]]
	if !ok {
		return nil // Can't verify, skip
	}

	// Check platform matches
	if f.Platform != config.Platform {
		return &ValidationError{
			Message: fmt.Sprintf("platform mismatch: fingerprint has '%s' but config expects '%s'", f.Platform, config.Platform),
		}
	}

	return nil
}