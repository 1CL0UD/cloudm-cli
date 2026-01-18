package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/1CL0UD/cloudm-cli/internal/config"
	"github.com/jackc/pgx/v5"
)

type TableStats struct {
	Schema   string
	Table    string
	RowCount int64
	Size     string
}

// GetTableStats retrieves table statistics from a database
func GetTableStats(ctx context.Context, cfg config.DatabaseConfig, schema string) ([]TableStats, error) {
	connStr := GetConnectionString(cfg)
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	return getTableStatsFromConn(ctx, conn, schema)
}

// GetTargetTableStats retrieves table statistics from a target database
func GetTargetTableStats(ctx context.Context, cfg config.TargetConfig, schema string) ([]TableStats, error) {
	connStr := GetTargetConnectionString(cfg)
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	return getTableStatsFromConn(ctx, conn, schema)
}

// getTableStatsFromConn retrieves table stats using an existing connection
func getTableStatsFromConn(ctx context.Context, conn *pgx.Conn, schema string) ([]TableStats, error) {
	rows, err := conn.Query(ctx, `
		SELECT 
			schemaname, 
			tablename, 
			n_live_tup,
			pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)) as size
		FROM pg_stat_user_tables 
		WHERE schemaname = $1 
		ORDER BY tablename`, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to query table stats: %w", err)
	}
	defer rows.Close()

	var stats []TableStats
	for rows.Next() {
		var s TableStats
		if err := rows.Scan(&s.Schema, &s.Table, &s.RowCount, &s.Size); err != nil {
			return nil, fmt.Errorf("failed to scan table stats: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// CompareTableStats compares source and target table statistics
func CompareTableStats(source, target []TableStats) (report string, hasDiscrepancy bool) {
	var sb strings.Builder

	// Create lookup map for target tables
	targetMap := make(map[string]TableStats)
	for _, t := range target {
		key := fmt.Sprintf("%s.%s", t.Schema, t.Table)
		targetMap[key] = t
	}

	sb.WriteString("Table Comparison:\n")
	sb.WriteString(fmt.Sprintf("%-40s %15s %15s %10s\n", "Table", "Source Rows", "Target Rows", "Status"))
	sb.WriteString(strings.Repeat("-", 85) + "\n")

	for _, srcTable := range source {
		key := fmt.Sprintf("%s.%s", srcTable.Schema, srcTable.Table)
		tgtTable, exists := targetMap[key]

		var status string
		var targetRows int64
		if !exists {
			status = "MISSING"
			hasDiscrepancy = true
		} else if srcTable.RowCount != tgtTable.RowCount {
			status = "MISMATCH"
			hasDiscrepancy = true
			targetRows = tgtTable.RowCount
		} else {
			status = "OK"
			targetRows = tgtTable.RowCount
		}

		sb.WriteString(fmt.Sprintf("%-40s %15d %15d %10s\n",
			key, srcTable.RowCount, targetRows, status))
	}

	return sb.String(), hasDiscrepancy
}

// TerminateConnections terminates all active connections to a database
func TerminateConnections(ctx context.Context, cfg config.TargetConfig, database string) error {
	// Connect to the postgres database to terminate connections
	connStr := GetTargetPostgresConnectionString(cfg)
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		SELECT pg_terminate_backend(pid) 
		FROM pg_stat_activity 
		WHERE datname = $1 AND pid <> pg_backend_pid()`, database)
	if err != nil {
		return fmt.Errorf("failed to terminate connections: %w", err)
	}

	return nil
}

// CreateExtensions creates PostgreSQL extensions
func CreateExtensions(ctx context.Context, cfg config.TargetConfig, extensions []string) error {
	connStr := GetTargetConnectionString(cfg)
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	for _, ext := range extensions {
		_, err = conn.Exec(ctx, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", ext))
		if err != nil {
			return fmt.Errorf("failed to create extension %s: %w", ext, err)
		}
	}

	return nil
}

// DropSchema drops a schema and all its objects
func DropSchema(ctx context.Context, cfg config.TargetConfig, schema string) error {
	connStr := GetTargetConnectionString(cfg)
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
	if err != nil {
		return fmt.Errorf("failed to drop schema: %w", err)
	}

	return nil
}

// CreateSchema creates a schema
func CreateSchema(ctx context.Context, cfg config.TargetConfig, schema string) error {
	connStr := GetTargetConnectionString(cfg)
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema))
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// PrepareTarget prepares the target database for migration
func PrepareTarget(ctx context.Context, cfg config.TargetConfig, extensions []string) error {
	// Terminate connections
	if err := TerminateConnections(ctx, cfg, cfg.Database); err != nil {
		return fmt.Errorf("failed to terminate connections: %w", err)
	}

	// Drop and recreate schema
	if err := DropSchema(ctx, cfg, "public"); err != nil {
		return fmt.Errorf("failed to drop schema: %w", err)
	}

	if err := CreateSchema(ctx, cfg, "public"); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Create extensions
	if len(extensions) > 0 {
		if err := CreateExtensions(ctx, cfg, extensions); err != nil {
			return fmt.Errorf("failed to create extensions: %w", err)
		}
	}

	return nil
}