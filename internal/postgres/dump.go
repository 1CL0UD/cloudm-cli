package postgres

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type DumpOptions struct {
	Host          string
	Port          int
	User          string
	Password      string
	Database      string
	Schema        string
	OutputFile    string
	StructureOnly bool
	DataOnly      bool
	ExcludeTables []string
}

// DumpStructure dumps database structure (schema only)
func DumpStructure(opts DumpOptions) error {
	args := buildDumpArgs(opts, true, false)
	return runPgDump(args, opts.Password)
}

// DumpData dumps database data only
func DumpData(opts DumpOptions) error {
	args := buildDumpArgs(opts, false, true)
	return runPgDump(args, opts.Password)
}

// DumpFull dumps both structure and data
func DumpFull(opts DumpOptions) error {
	args := buildDumpArgs(opts, false, false)
	return runPgDump(args, opts.Password)
}

// buildDumpArgs builds pg_dump command arguments
func buildDumpArgs(opts DumpOptions, structureOnly, dataOnly bool) []string {
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

	// Exclude tables
	for _, table := range opts.ExcludeTables {
		args = append(args, "--exclude-table="+table)
	}

	// Output format: custom (binary, compressed)
	args = append(args, "-Fc")

	// Output file
	args = append(args, "-f", opts.OutputFile)

	return args
}

// runPgDump executes pg_dump with the given arguments
func runPgDump(args []string, password string) error {
	cmd := exec.Command("pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", password))

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

// GetDumpInfo returns information about a dump file
func GetDumpInfo(dumpFile string) (string, error) {
	cmd := exec.Command("pg_restore", "-l", dumpFile)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get dump info: %w", err)
	}

	return string(output), nil
}