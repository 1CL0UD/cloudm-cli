package liftshift

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/jackc/pgx/v5"
)

// TerminateConnections terminates active connections to the target database
func TerminateConnections(config *Config) error {
	fmt.Println("Terminating active connections to target database...")

	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		config.DstHost, config.DstPort, config.DstAdminUser, config.DstAdminPassword)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer conn.Close(ctx)

	query := fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid();", config.DstDB)
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to terminate connections: %w", err)
	}
	defer rows.Close()

	fmt.Println("Connections terminated.")
	return nil
}

// RestoreToTarget restores the dumps to the target database
func RestoreToTarget(config *Config, structFile, dataFile string) error {
	fmt.Println("Cleaning and restoring to target database...")

	// Set PGPASSWORD environment variable
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", config.DstAdminPassword))

	// Terminate connections
	err := TerminateConnections(config)
	if err != nil {
		return fmt.Errorf("failed to terminate connections: %w", err)
	}

	// Drop and recreate schema
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.DstHost, config.DstPort, config.DstAdminUser, config.DstAdminPassword, config.DstDB)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}

	// Drop schema
	_, err = conn.Exec(ctx, "DROP SCHEMA IF EXISTS public CASCADE;")
	if err != nil {
		conn.Close(ctx)
		return fmt.Errorf("failed to drop schema: %w", err)
	}

	// Create schema
	_, err = conn.Exec(ctx, "CREATE SCHEMA public;")
	if err != nil {
		conn.Close(ctx)
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Create pg_trgm extension
	_, err = conn.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pg_trgm;")
	if err != nil {
		conn.Close(ctx)
		return fmt.Errorf("failed to create pg_trgm extension: %w", err)
	}

	conn.Close(ctx)

	// Now restore using the actual dump files
	fmt.Println("Restoring database structure...")
	structRestoreCmd := exec.Command(
		"pg_restore",
		"-h", config.DstHost,
		"-p", fmt.Sprintf("%d", config.DstPort),
		"-U", config.DstAdminUser,
		"-d", config.DstDB,
		"-n", "public",
		"-j", fmt.Sprintf("%d", config.ParallelJobs),
		"--no-owner", "--no-privileges",
		structFile,
	)
	structRestoreCmd.Env = env

	structOutput, err := structRestoreCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("structure restore failed: %w\nOutput: %s", err, string(structOutput))
	}

	fmt.Println("Restoring database data...")
	dataRestoreCmd := exec.Command(
		"pg_restore",
		"-h", config.DstHost,
		"-p", fmt.Sprintf("%d", config.DstPort),
		"-U", config.DstAdminUser,
		"-d", config.DstDB,
		"-n", "public",
		"-j", fmt.Sprintf("%d", config.DataParallelJobs),
		"--no-owner", "--no-privileges",
		dataFile,
	)
	dataRestoreCmd.Env = env

	dataOutput, err := dataRestoreCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("data restore failed: %w\nOutput: %s", err, string(dataOutput))
	}

	fmt.Printf("Database structure and data restored successfully from %s and %s\n", structFile, dataFile)
	return nil
}
