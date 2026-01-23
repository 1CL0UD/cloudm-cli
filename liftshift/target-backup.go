package liftshift

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// BackupTargetDatabase creates a backup of the target database
func BackupTargetDatabase(config *Config) (string, error) {
	if config.SkipBackup {
		fmt.Println("Skipping backup (SKIP_BACKUP=true)")
		return "", nil
	}

	fmt.Println("Backing up target database...")

	// Create timestamp for the backup file
	timestamp := time.Now().Format("20060102_150405")
	backupFile := fmt.Sprintf("backup_pre_migration_%s.dump", timestamp)

	// Set PGPASSWORD environment variable for the duration of the command
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", config.DstAdminPassword))

	// Build the pg_dump command
	cmd := exec.Command(
		"pg_dump",
		"-h", config.DstHost,
		"-p", fmt.Sprintf("%d", config.DstPort),
		"-U", config.DstAdminUser,
		"-d", config.DstDB,
		"-Fc", // Custom format
		"-f", backupFile,
	)

	// Set the environment with the password
	cmd.Env = env

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("backup failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Backup completed successfully: %s\n", backupFile)
	return backupFile, nil
}
