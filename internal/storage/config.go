package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// userConfigFile is the name of the user configuration file (sibling to .tk/).
	userConfigFile = ".tkconfig.yaml"

	// Default configuration values
	DefaultAutoCheck       = false
	DefaultDefaultProject  = "default"
	DefaultDefaultPriority = 3
)

// Config represents user configuration from .tkconfig.yaml.
// This file is user-managed and never written by tk.
type Config struct {
	// AutoCheck enables auto-running `tk check` on read commands.
	AutoCheck bool `yaml:"autocheck"`

	// DefaultProject is the default project for `tk add` when -p not specified.
	DefaultProject string `yaml:"default_project"`

	// DefaultPriority is the default priority for new tasks (1-4).
	DefaultPriority int `yaml:"default_priority"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		AutoCheck:       DefaultAutoCheck,
		DefaultProject:  DefaultDefaultProject,
		DefaultPriority: DefaultDefaultPriority,
	}
}

// LoadConfig loads .tkconfig.yaml if it exists, otherwise returns defaults.
// The config file is a sibling to .tk/ (in the same directory).
// Partial config files are merged with defaults.
func (s *Storage) LoadConfig() (*Config, error) {
	configPath := filepath.Join(s.root, userConfigFile)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file - return defaults
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", userConfigFile, err)
	}

	// Start with defaults
	cfg := DefaultConfig()

	// Parse YAML and merge with defaults
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", userConfigFile, err)
	}

	return cfg, nil
}

// ConfigPath returns the path to the user config file.
func (s *Storage) ConfigPath() string {
	return filepath.Join(s.root, userConfigFile)
}
