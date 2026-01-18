package postgres

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RestoreOptions struct {
	Host          string
	Port          int
	User          string
	Password      string
	Database      string
	Schema        string
	InputFile     string
	ParallelJobs  int
	StructureOnly bool
	DataOnly      bool
}

// RestoreStructure restores database structure (schema only)
func RestoreStructure(opts RestoreOptions) error {
	args := buildRestoreArgs(opts, true, false)
	return runPgRestore(args, opts.Password)
}

// RestoreData restores database data only
func RestoreData(opts RestoreOptions) error {
	args := buildRestoreArgs(opts, false, true)
	return runPgRestore(args, opts.Password)
}

// RestoreFull restores both structure and data
func RestoreFull(opts RestoreOptions) error {
	args := buildRestoreArgs(opts, false, false)
	return runPgRestore(args, opts.Password)
}

// buildRestoreArgs builds pg_restore command arguments
func buildRestoreArgs(opts RestoreOptions, structureOnly, dataOnly bool) []string {
	args := []string{
		"-h", opts.Host,
		"-p", fmt.Sprintf("%d", opts.Port),
		"-U", opts.User,
		"-d", opts.Database,
	}

	// Add schema restriction if specified
	if opts.Schema != "" {
		args = append(args, "-n", opts.Schema)
	}

	// Structure only flag
	if structureOnly {
		args = append(args, "-s")
	}

	// Data only flag
	if dataOnly {
		args = append(args, "-a")
	}

	// Parallel jobs
	if opts.ParallelJobs > 0 {
		args = append(args, "-j", fmt.Sprintf("%d", opts.ParallelJobs))
	}

	// Don't restore ownership or privileges (we handle this separately)
	args = append(args, "--no-owner", "--no-privileges")

	// Input file
	args = append(args, opts.InputFile)

	return args
}

// runPgRestore executes pg_restore with the given arguments
func runPgRestore(args []string, password string) error {
	cmd := exec.Command("pg_restore", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// pg_restore often returns warnings that aren't fatal
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "error") || strings.Contains(stderrStr, "FATAL") {
			return fmt.Errorf("pg_restore failed: %w\nstderr: %s", err, stderrStr)
		}
	}

	return nil
}

// BackupDatabase creates a full backup of a database
func BackupDatabase(host string, port int, user, password, database, outputFile string) error {
	args := []string{
		"-h", host,
		"-p", fmt.Sprintf("%d", port),
		"-U", user,
		"-d", database,
		"-Fc",
		"-f", outputFile,
	}

	cmd := exec.Command("pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("backup failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}