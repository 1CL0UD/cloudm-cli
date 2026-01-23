package configloader

import (
	"os"
	"strconv"

	"github.com/1CL0UD/cloudm-cli/configs"
)

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() *configs.Config {
	config := configs.NewDefaultConfig()

	// Source database configuration
	if host := os.Getenv("SRC_HOST"); host != "" {
		config.SrcHost = host
	}
	if portStr := os.Getenv("SRC_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.SrcPort = port
		}
	}
	if db := os.Getenv("SRC_DB"); db != "" {
		config.SrcDB = db
	}
	if user := os.Getenv("SRC_USER"); user != "" {
		config.SrcUser = user
	}
	if password := os.Getenv("SRC_PASSWORD"); password != "" {
		config.SrcPassword = password
	}

	// Target database configuration
	if host := os.Getenv("DST_HOST"); host != "" {
		config.DstHost = host
	}
	if portStr := os.Getenv("DST_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.DstPort = port
		}
	}
	if db := os.Getenv("DST_DB"); db != "" {
		config.DstDB = db
	}
	if user := os.Getenv("DST_ADMIN_USER"); user != "" {
		config.DstAdminUser = user
	}
	if password := os.Getenv("DST_ADMIN_PASSWORD"); password != "" {
		config.DstAdminPassword = password
	}
	if user := os.Getenv("APP_USER"); user != "" {
		config.AppUser = user
	}

	// Options
	if skipBackupStr := os.Getenv("SKIP_BACKUP"); skipBackupStr != "" {
		if skipBackup, err := strconv.ParseBool(skipBackupStr); err == nil {
			config.SkipBackup = skipBackup
		}
	}
	if parallelJobsStr := os.Getenv("PARALLEL_JOBS"); parallelJobsStr != "" {
		if jobs, err := strconv.Atoi(parallelJobsStr); err == nil {
			config.ParallelJobs = jobs
		}
	}
	if dataParallelJobsStr := os.Getenv("DATA_PARALLEL_JOBS"); dataParallelJobsStr != "" {
		if jobs, err := strconv.Atoi(dataParallelJobsStr); err == nil {
			config.DataParallelJobs = jobs
		}
	}
	if dryRunStr := os.Getenv("DRY_RUN"); dryRunStr != "" {
		if dryRun, err := strconv.ParseBool(dryRunStr); err == nil {
			config.DryRun = dryRun
		}
	}

	return config
}