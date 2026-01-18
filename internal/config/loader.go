package config

import (
	"os"
)

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	// TODO: Implementation
	return nil, nil
}

// ExpandEnvVars replaces ${VAR} with environment variable values
func ExpandEnvVars(cfg *Config) error {
	// TODO: Implementation
	return nil
}

// Validate validates the configuration
func Validate(cfg *Config) error {
	// TODO: Implementation
	return nil
}

// expandString replaces ${VAR} or $VAR with environment variable values
func expandString(s string) string {
	return os.Expand(s, func(key string) string {
		return os.Getenv(key)
	})
}