package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/1CL0UD/cloudm-cli/internal/config"
	"github.com/jackc/pgx/v5"
)

// TestConnection tests if a database connection can be established
func TestConnection(cfg config.DatabaseConfig) error {
	connStr := GetConnectionString(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Test with a simple query
	var result int
	err = conn.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("failed to execute test query: %w", err)
	}

	return nil
}

// TestTargetConnection tests connection to target database with admin credentials
func TestTargetConnection(cfg config.TargetConfig) error {
	connStr := GetTargetConnectionString(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Test with a simple query
	var result int
	err = conn.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("failed to execute test query: %w", err)
	}

	return nil
}

// GetConnectionString builds a PostgreSQL connection string for source databases
func GetConnectionString(cfg config.DatabaseConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=prefer",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
}

// GetTargetConnectionString builds a PostgreSQL connection string for target databases
func GetTargetConnectionString(cfg config.TargetConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=prefer",
		cfg.AdminUser,
		cfg.AdminPassword,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
}

// GetTargetPostgresConnectionString builds a connection string to postgres database (for admin operations)
func GetTargetPostgresConnectionString(cfg config.TargetConfig) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/postgres?sslmode=prefer",
		cfg.AdminUser,
		cfg.AdminPassword,
		cfg.Host,
		cfg.Port,
	)
}

// Connect establishes a connection to the database
func Connect(ctx context.Context, connStr string) (*pgx.Conn, error) {
	return pgx.Connect(ctx, connStr)
}