package commands

import (
	"context"

	"github.com/tmos/fingerbrower/fingerprint"
)

type FingerprintService struct {
	generator *fingerprint.Generator
}

func NewFingerprintService() *FingerprintService {
	return &FingerprintService{
		generator: fingerprint.NewGenerator(),
	}
}

func (s *FingerprintService) GenerateFingerprint(ctx context.Context, seed, country string) (*fingerprint.Fingerprint, error) {
	if seed == "" {
		return s.generator.GenerateRandom(country)
	}
	return s.generator.Generate(seed, country)
}

func (s *FingerprintService) GenerateRandomFingerprint(ctx context.Context, country string) (*fingerprint.Fingerprint, error) {
	return s.generator.GenerateRandom(country)
}

func (s *FingerprintService) ValidateFingerprint(ctx context.Context, fp *fingerprint.Fingerprint) error {
	validator := fingerprint.NewValidator()
	return validator.Validate(fp)
}

type Fingerprint = fingerprint.Fingerprint
