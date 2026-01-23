package liftshift

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jackc/pgx/v5"
)

// CheckRequirements verifies that required tools are available
func CheckRequirements() error {
	requiredTools := []string{"pg_dump", "pg_restore", "psql"}

	for _, tool := range requiredTools {
		_, err := exec.LookPath(tool)
		if err != nil {
			return fmt.Errorf("%s not found in PATH", tool)
		}
	}

	return nil
}

// TestConnection tests the database connection
func TestConnection(host, port, user, database, password string) error {
	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return fmt.Errorf("cannot connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Test the connection with a simple query
	var result int
	err = conn.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("failed to execute test query: %w", err)
	}

	return nil
}

// CheckDatabaseConnections tests both source and target database connections
func CheckDatabaseConnections(config *Config) error {
	fmt.Println("Testing source database connection...")
	err := TestConnection(
		config.SrcHost,
		fmt.Sprintf("%d", config.SrcPort),
		config.SrcUser,
		config.SrcDB,
		config.SrcPassword,
	)
	if err != nil {
		return fmt.Errorf("source database connection failed: %w", err)
	}

	fmt.Println("Testing target database connection...")
	err = TestConnection(
		config.DstHost,
		fmt.Sprintf("%d", config.DstPort),
		config.DstAdminUser,
		config.DstDB,
		config.DstAdminPassword,
	)
	if err != nil {
		return fmt.Errorf("target database connection failed: %w", err)
	}

	return nil
}
