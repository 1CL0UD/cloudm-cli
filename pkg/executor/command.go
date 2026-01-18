package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/1CL0UD/cloudm-cli/internal/logger"
)

type CommandExecutor struct {
	logger *logger.Logger
	dryRun bool
}

// New creates a new command executor
func New(logger *logger.Logger, dryRun bool) *CommandExecutor {
	return &CommandExecutor{
		logger: logger,
		dryRun: dryRun,
	}
}

// IsDryRun returns whether the executor is in dry-run mode
func (e *CommandExecutor) IsDryRun() bool {
	return e.dryRun
}

// RunPgDump executes pg_dump with given arguments
func (e *CommandExecutor) RunPgDump(args []string, env map[string]string) error {
	return e.runCommand("pg_dump", args, env)
}

// RunPgRestore executes pg_restore with given arguments
func (e *CommandExecutor) RunPgRestore(args []string, env map[string]string) error {
	return e.runCommand("pg_restore", args, env)
}

// RunPsql executes a SQL query using psql
func (e *CommandExecutor) RunPsql(query string, host string, port int, user, password, database string) (string, error) {
	args := []string{
		"-h", host,
		"-p", fmt.Sprintf("%d", port),
		"-U", user,
		"-d", database,
		"-tAc", query,
	}

	env := map[string]string{"PGPASSWORD": password}

	if e.dryRun {
		e.logger.DryRun("psql %s", strings.Join(args, " "))
		return "", nil
	}

	e.logger.Debug("Executing: psql %s", strings.Join(args, " "))

	cmd := exec.Command("psql", args...)
	cmd.Env = e.buildEnv(env)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("psql failed: %w\nstderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunPsqlScript executes a SQL script using psql
func (e *CommandExecutor) RunPsqlScript(script string, host string, port int, user, password, database string) error {
	args := []string{
		"-h", host,
		"-p", fmt.Sprintf("%d", port),
		"-U", user,
		"-d", database,
	}

	env := map[string]string{"PGPASSWORD": password}

	if e.dryRun {
		e.logger.DryRun("psql %s << SQL\n%s\nSQL", strings.Join(args, " "), script)
		return nil
	}

	e.logger.Debug("Executing SQL script on %s/%s", host, database)

	cmd := exec.Command("psql", args...)
	cmd.Env = e.buildEnv(env)
	cmd.Stdin = strings.NewReader(script)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql script failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

// CheckRequiredTools verifies that required PostgreSQL tools are available
func (e *CommandExecutor) CheckRequiredTools() error {
	tools := []string{"pg_dump", "pg_restore", "psql"}
	var missing []string

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}

	return nil
}

// runCommand executes a command with the given arguments and environment
func (e *CommandExecutor) runCommand(name string, args []string, env map[string]string) error {
	if e.dryRun {
		e.logger.DryRun("%s %s", name, strings.Join(args, " "))
		return nil
	}

	e.logger.Debug("Executing: %s %s", name, strings.Join(args, " "))

	cmd := exec.Command(name, args...)
	cmd.Env = e.buildEnv(env)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w\nstderr: %s", name, err, stderr.String())
	}

	return nil
}

// buildEnv builds the environment for command execution
func (e *CommandExecutor) buildEnv(env map[string]string) []string {
	result := os.Environ()
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}