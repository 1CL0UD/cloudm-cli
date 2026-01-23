package configs

import (
	"fmt"
)

// Config holds the configuration for the cloudm-cli application
type Config struct {
	// Source (Staging) database configuration
	SrcHost     string
	SrcPort     int
	SrcDB       string
	SrcUser     string
	SrcPassword string

	// Target (RDS Production) database configuration
	DstHost           string
	DstPort           int
	DstDB             string
	DstAdminUser      string
	DstAdminPassword  string
	AppUser           string

	// Options
	SkipBackup      bool
	ParallelJobs    int
	DataParallelJobs int
	DryRun          bool
	Timestamp       string
}

// NewDefaultConfig creates a new configuration with default values
func NewDefaultConfig() *Config {
	return &Config{
		SrcPort:        5432,
		DstPort:        5432,
		SrcUser:        "postgres",
		DstAdminUser:   "postgres",
		AppUser:        "app_user",
		ParallelJobs:   4,
		DataParallelJobs: 2,
		SkipBackup:     false,
		DryRun:         false,
	}
}

// Validate validates the configuration values
func (c *Config) Validate() error {
	if c.SrcHost == "" {
		return fmt.Errorf("source host is required")
	}
	if c.SrcDB == "" {
		return fmt.Errorf("source database is required")
	}
	if c.DstHost == "" {
		return fmt.Errorf("destination host is required")
	}
	if c.DstDB == "" {
		return fmt.Errorf("destination database is required")
	}
	if c.SrcPassword == "" {
		return fmt.Errorf("source password is required")
	}
	if c.DstAdminPassword == "" {
		return fmt.Errorf("destination admin password is required")
	}

	if c.SrcPort <= 0 || c.SrcPort > 65535 {
		return fmt.Errorf("source port must be between 1 and 65535")
	}
	if c.DstPort <= 0 || c.DstPort > 65535 {
		return fmt.Errorf("destination port must be between 1 and 65535")
	}

	if c.ParallelJobs <= 0 {
		return fmt.Errorf("parallel jobs must be greater than 0")
	}
	if c.DataParallelJobs <= 0 {
		return fmt.Errorf("data parallel jobs must be greater than 0")
	}

	return nil
}