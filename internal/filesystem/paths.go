package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CreateMigrationDir creates a timestamped migration directory
func CreateMigrationDir(baseDir string) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	migrationDir := filepath.Join(baseDir, timestamp)

	if err := os.MkdirAll(migrationDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create migration directory: %w", err)
	}

	return migrationDir, nil
}

// GetDumpPaths returns the paths for structure and data dumps
func GetDumpPaths(migrationDir string) (structure, data string) {
	structure = filepath.Join(migrationDir, "structure.dump")
	data = filepath.Join(migrationDir, "data.dump")
	return
}

// GetBackupPath returns the path for a backup file
func GetBackupPath(migrationDir string) string {
	return filepath.Join(migrationDir, "backup_pre_migration.dump")
}

// GetLogPaths returns paths for log files
func GetLogPaths(migrationDir string) (mainLog, timeLog, validationLog string) {
	mainLog = filepath.Join(migrationDir, "migration.log")
	timeLog = filepath.Join(migrationDir, "migration_time.txt")
	validationLog = filepath.Join(migrationDir, "validation.log")
	return
}

// ValidateDumpFiles checks if required dump files exist
func ValidateDumpFiles(migrationDir string) error {
	structure, data := GetDumpPaths(migrationDir)

	if _, err := os.Stat(structure); os.IsNotExist(err) {
		return fmt.Errorf("structure dump file not found: %s", structure)
	}

	if _, err := os.Stat(data); os.IsNotExist(err) {
		return fmt.Errorf("data dump file not found: %s", data)
	}

	return nil
}

// ValidateStructureDump checks if structure dump file exists
func ValidateStructureDump(migrationDir string) error {
	structure, _ := GetDumpPaths(migrationDir)

	if _, err := os.Stat(structure); os.IsNotExist(err) {
		return fmt.Errorf("structure dump file not found: %s", structure)
	}

	return nil
}

// ValidateDataDump checks if data dump file exists
func ValidateDataDump(migrationDir string) error {
	_, data := GetDumpPaths(migrationDir)

	if _, err := os.Stat(data); os.IsNotExist(err) {
		return fmt.Errorf("data dump file not found: %s", data)
	}

	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetFileSize returns the size of a file in human-readable format
func GetFileSize(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return formatBytes(info.Size()), nil
}

// formatBytes formats bytes to human-readable size
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}