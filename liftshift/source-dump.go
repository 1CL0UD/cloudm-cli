package liftshift

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DumpFromSource creates dumps of the source database (structure and data)
func DumpFromSource(config *Config) (string, string, error) {
	fmt.Println("Dumping from source database...")

	// Create timestamp for the dump files
	timestamp := time.Now().Format("20060102_150405")

	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Create the .cloudm-cli directories
	structDumpsDir := filepath.Join(homeDir, ".cloudm-cli", "structure-dumps")
	dataDumpsDir := filepath.Join(homeDir, ".cloudm-cli", "data-dumps")
	err = os.MkdirAll(structDumpsDir, 0755)
	if err != nil {
		return "", "", fmt.Errorf("failed to create structure dumps directory: %w", err)
	}
	err = os.MkdirAll(dataDumpsDir, 0755)
	if err != nil {
		return "", "", fmt.Errorf("failed to create data dumps directory: %w", err)
	}

	structDumpFile := fmt.Sprintf("structure_%s.dump", timestamp)
	dataDumpFile := fmt.Sprintf("data_%s.dump", timestamp)

	structFullpath := filepath.Join(structDumpsDir, structDumpFile)
	dataFullpath := filepath.Join(dataDumpsDir, dataDumpFile)

	// Set PGPASSWORD environment variable for the duration of the command
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", config.SrcPassword))

	// Dump structure
	fmt.Println("Dumping database structure...")
	structCmd := exec.Command(
		"pg_dump",
		"-h", config.SrcHost,
		"-p", fmt.Sprintf("%d", config.SrcPort),
		"-U", config.SrcUser,
		"-d", config.SrcDB,
		"-n", "public", // Only public schema
		"-s",  // Schema only
		"-Fc", // Custom format
		"-f", structFullpath,
	)
	structCmd.Env = env

	output, err := structCmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("structure dump failed: %w\nOutput: %s", err, string(output))
	}
	fmt.Printf("Structure dump completed: %s\n", structFullpath)

	// Dump data
	fmt.Println("Dumping database data (excluding activity_log)...")
	dataCmd := exec.Command(
		"pg_dump",
		"-h", config.SrcHost,
		"-p", fmt.Sprintf("%d", config.SrcPort),
		"-U", config.SrcUser,
		"-d", config.SrcDB,
		"-n", "public", // Only public schema
		"-a",                                  // Data only
		"--exclude-table=public.activity_log", // Exclude activity_log table
		"-Fc",                                 // Custom format
		"-f", dataFullpath,
	)
	dataCmd.Env = env

	output, err = dataCmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("data dump failed: %w\nOutput: %s", err, string(output))
	}
	fmt.Printf("Data dump completed: %s\n", dataFullpath)

	return structFullpath, dataFullpath, nil
}
