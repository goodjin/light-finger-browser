package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	// DefaultInstancePort is the default CDP port for browser instances.
	DefaultInstancePort = 9222

	// ConfigFileName is the name of the config file.
	ConfigFileName = "config.json"
)

// AppConfig represents the application configuration.
type AppConfig struct {
	InstancePort int  `json:"instance_port"`
	Headless    bool `json:"headless"`
}

// ConfigService manages application configuration.
type ConfigService struct {
	configPath string
	config     *AppConfig
	mu         sync.RWMutex
}

// NewConfigService creates a new ConfigService.
func NewConfigService() *ConfigService {
	appDataDir := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "fingerbrower")
	configPath := filepath.Join(appDataDir, ConfigFileName)
	return &ConfigService{
		configPath: configPath,
		config:     nil,
	}
}

// Load loads the configuration from disk.
func (s *ConfigService) Load() (*AppConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If already loaded, return cached config
	if s.config != nil {
		return s.config, nil
	}

	// Try to load from file
	if _, err := os.Stat(s.configPath); err == nil {
		data, err := os.ReadFile(s.configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		var cfg AppConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		s.config = &cfg
		return &cfg, nil
	}

	// Config file doesn't exist, create default
	s.config = &AppConfig{
		InstancePort: DefaultInstancePort,
		Headless:     false,
	}

	// Save default config
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	return s.config, nil
}

// saveLocked saves the configuration to disk. Caller must hold the lock.
func (s *ConfigService) saveLocked() error {
	// Ensure directory exists
	dir := filepath.Dir(s.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Save saves the configuration to disk.
func (s *ConfigService) Save(cfg *AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = cfg
	return s.saveLocked()
}

// GetInstancePort returns the configured instance port or default.
func (s *ConfigService) GetInstancePort() (int, error) {
	cfg, err := s.Load()
	if err != nil {
		return DefaultInstancePort, err
	}

	if cfg.InstancePort <= 0 || cfg.InstancePort > 65535 {
		return DefaultInstancePort, nil
	}

	return cfg.InstancePort, nil
}

// SetInstancePort updates the instance port configuration.
func (s *ConfigService) SetInstancePort(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config == nil {
		// Load without lock (we already hold it)
		if err := s.loadLocked(); err != nil {
			return err
		}
	}

	s.config.InstancePort = port
	return s.saveLocked()
}

// GetHeadless returns whether instances should be started in headless mode.
func (s *ConfigService) GetHeadless() (bool, error) {
	cfg, err := s.Load()
	if err != nil {
		return false, err
	}
	return cfg.Headless, nil
}

// SetHeadless updates the headless mode configuration.
func (s *ConfigService) SetHeadless(headless bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config == nil {
		// Load without lock (we already hold it)
		if err := s.loadLocked(); err != nil {
			return err
		}
	}

	s.config.Headless = headless
	return s.saveLocked()
}

// loadLocked loads the configuration from disk. Caller must hold the lock.
func (s *ConfigService) loadLocked() error {
	// Try to load from file
	if _, err := os.Stat(s.configPath); err == nil {
		data, err := os.ReadFile(s.configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		var cfg AppConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}

		s.config = &cfg
		return nil
	}

	// Config file doesn't exist, create default
	s.config = &AppConfig{
		InstancePort: DefaultInstancePort,
		Headless:     false,
	}

	// Save default config
	return s.saveLocked()
}

// GetConfig returns the current configuration.
func (s *ConfigService) GetConfig() (*AppConfig, error) {
	return s.Load()
}
