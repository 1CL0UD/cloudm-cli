package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values
	if cfg.Source.Port == 0 {
		cfg.Source.Port = 5432
	}
	if cfg.Target.Port == 0 {
		cfg.Target.Port = 5432
	}
	if cfg.Options.ParallelJobs == 0 {
		cfg.Options.ParallelJobs = 4
	}
	if cfg.Options.DataParallelJobs == 0 {
		cfg.Options.DataParallelJobs = 2
	}
	if cfg.Options.OutputDir == "" {
		cfg.Options.OutputDir = "./migrations"
	}

	// Expand environment variables
	if err := ExpandEnvVars(&cfg); err != nil {
		return nil, fmt.Errorf("failed to expand environment variables: %w", err)
	}

	return &cfg, nil
}

// ExpandEnvVars replaces ${VAR} with environment variable values
func ExpandEnvVars(cfg *Config) error {
	// Expand source password
	cfg.Source.Password = expandString(cfg.Source.Password)
	cfg.Source.PasswordFile = expandString(cfg.Source.PasswordFile)
	cfg.Source.Host = expandString(cfg.Source.Host)
	cfg.Source.User = expandString(cfg.Source.User)
	cfg.Source.Database = expandString(cfg.Source.Database)

	// Expand target password
	cfg.Target.Password = expandString(cfg.Target.Password)
	cfg.Target.PasswordFile = expandString(cfg.Target.PasswordFile)
	cfg.Target.Host = expandString(cfg.Target.Host)
	cfg.Target.User = expandString(cfg.Target.User)
	cfg.Target.Database = expandString(cfg.Target.Database)
	cfg.Target.AdminUser = expandString(cfg.Target.AdminUser)
	cfg.Target.AdminPassword = expandString(cfg.Target.AdminPassword)
	cfg.Target.AppUser = expandString(cfg.Target.AppUser)
	cfg.Target.AppUserPassword = expandString(cfg.Target.AppUserPassword)

	// If password file is specified, read password from file
	if cfg.Source.PasswordFile != "" && cfg.Source.Password == "" {
		password, err := os.ReadFile(cfg.Source.PasswordFile)
		if err != nil {
			return fmt.Errorf("failed to read source password file: %w", err)
		}
		cfg.Source.Password = strings.TrimSpace(string(password))
	}

	if cfg.Target.PasswordFile != "" && cfg.Target.Password == "" {
		password, err := os.ReadFile(cfg.Target.PasswordFile)
		if err != nil {
			return fmt.Errorf("failed to read target password file: %w", err)
		}
		cfg.Target.Password = strings.TrimSpace(string(password))
	}

	return nil
}

// Validate validates the configuration
func Validate(cfg *Config) error {
	var errors []string

	// Validate source configuration
	if cfg.Source.Host == "" {
		errors = append(errors, "source.host is required")
	}
	if cfg.Source.Database == "" {
		errors = append(errors, "source.database is required")
	}
	if cfg.Source.User == "" {
		errors = append(errors, "source.user is required")
	}
	if cfg.Source.Password == "" {
		errors = append(errors, "source.password is required (set via config or SRC_PASSWORD env var)")
	}

	// Validate target configuration
	if cfg.Target.Host == "" {
		errors = append(errors, "target.host is required")
	}
	if cfg.Target.Database == "" {
		errors = append(errors, "target.database is required")
	}
	if cfg.Target.AdminUser == "" {
		errors = append(errors, "target.admin_user is required")
	}
	if cfg.Target.AdminPassword == "" {
		errors = append(errors, "target.admin_password is required (set via config or DST_ADMIN_PASSWORD env var)")
	}
	if cfg.Target.AppUser == "" {
		errors = append(errors, "target.app_user is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// expandString replaces ${VAR} or $VAR with environment variable values
func expandString(s string) string {
	return os.Expand(s, func(key string) string {
		return os.Getenv(key)
	})
}