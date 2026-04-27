package fingerprint

// FingerprintGenerator defines the interface for fingerprint generation.
type FingerprintGenerator interface {
	// Generate creates a deterministic fingerprint based on the given seed and country.
	// Returns ErrInvalidSeed if seed is empty, ErrInvalidCountry if country is not supported.
	Generate(seed string, country string) (*Fingerprint, error)

	// Validate checks the consistency of a fingerprint.
	// Returns nil if valid, otherwise returns a ValidationError describing the issue.
	Validate(f *Fingerprint) error

	// GenerateRandom generates a fingerprint with a random seed for the given country.
	// Returns ErrInvalidCountry if country is not supported.
	GenerateRandom(country string) (*Fingerprint, error)
}

// GeneratorWithValidator combines fingerprint generation and validation.
type GeneratorWithValidator struct {
	*Generator
	*Validator
}

// NewGeneratorWithValidator creates a new GeneratorWithValidator.
func NewGeneratorWithValidator() *GeneratorWithValidator {
	return &GeneratorWithValidator{
		Generator: NewGenerator(),
		Validator: NewValidator(),
	}
}

// Generate implements FingerprintGenerator.Generate.
func (g *GeneratorWithValidator) Generate(seed string, country string) (*Fingerprint, error) {
	return g.Generator.Generate(seed, country)
}

// Validate implements FingerprintGenerator.Validate.
func (g *GeneratorWithValidator) Validate(f *Fingerprint) error {
	return g.Validator.Validate(f)
}

// GenerateRandom implements FingerprintGenerator.GenerateRandom.
func (g *GeneratorWithValidator) GenerateRandom(country string) (*Fingerprint, error) {
	return g.Generator.GenerateRandom(country)
}

// Ensure *GeneratorWithValidator implements FingerprintGenerator
var _ FingerprintGenerator = (*GeneratorWithValidator)(nil)