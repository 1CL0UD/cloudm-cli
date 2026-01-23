package liftshift

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ValidateMigration compares the source and target databases to ensure migration was successful
func ValidateMigration(config *Config) error {
	fmt.Println("Validating migration...")

	validationFile := fmt.Sprintf("validation_%s.txt", time.Now().Format("20060102_150405"))

	file, err := os.Create(validationFile)
	if err != nil {
		return fmt.Errorf("failed to create validation file: %w", err)
	}
	defer file.Close()

	// Get source database stats
	sourceStats, err := getSourceTableCounts(config)
	if err != nil {
		return fmt.Errorf("failed to get source table counts: %w", err)
	}

	// Get target database stats
	targetStats, err := getTargetTableCounts(config)
	if err != nil {
		return fmt.Errorf("failed to get target table counts: %w", err)
	}

	// Write validation report
	report := generateValidationReport(config, sourceStats, targetStats)
	_, err = file.WriteString(report)
	if err != nil {
		return fmt.Errorf("failed to write validation report: %w", err)
	}

	fmt.Printf("Validation report saved to %s\n", validationFile)

	// Print summary
	fmt.Println("Validation completed. Check the report for details.")

	return nil
}

// TableInfo holds information about a table
type TableInfo struct {
	SchemaName string
	TableName  string
	RowCount   int64
}

// getSourceTableCounts gets table counts from the source database
func getSourceTableCounts(config *Config) ([]TableInfo, error) {
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.SrcHost, config.SrcPort, config.SrcUser, config.SrcPassword, config.SrcDB)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to source database: %w", err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT schemaname, tablename, n_live_tup
		FROM pg_stat_user_tables
		WHERE schemaname='public'
		ORDER BY tablename;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query source table counts: %w", err)
	}
	defer rows.Close()

	var tableInfos []TableInfo
	for rows.Next() {
		var schemaName, tableName string
		var rowCount int64
		err := rows.Scan(&schemaName, &tableName, &rowCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source table info: %w", err)
		}
		tableInfos = append(tableInfos, TableInfo{
			SchemaName: schemaName,
			TableName:  tableName,
			RowCount:   rowCount,
		})
	}

	return tableInfos, nil
}

// getTargetTableCounts gets table counts from the target database
func getTargetTableCounts(config *Config) ([]TableInfo, error) {
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.DstHost, config.DstPort, config.DstAdminUser, config.DstAdminPassword, config.DstDB)

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT schemaname, tablename, n_live_tup
		FROM pg_stat_user_tables
		WHERE schemaname='public'
		ORDER BY tablename;
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query target table counts: %w", err)
	}
	defer rows.Close()

	var tableInfos []TableInfo
	for rows.Next() {
		var schemaName, tableName string
		var rowCount int64
		err := rows.Scan(&schemaName, &tableName, &rowCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan target table info: %w", err)
		}
		tableInfos = append(tableInfos, TableInfo{
			SchemaName: schemaName,
			TableName:  tableName,
			RowCount:   rowCount,
		})
	}

	return tableInfos, nil
}

// generateValidationReport creates a validation report comparing source and target
func generateValidationReport(config *Config, sourceStats, targetStats []TableInfo) string {
	var report strings.Builder
	report.WriteString("==========================================\n")
	report.WriteString("Migration Validation Report\n")
	fmt.Fprintf(&report, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	report.WriteString("==========================================\n\n")

	fmt.Fprintf(&report, "Source Database: %s\n", config.SrcDB)
	for _, info := range sourceStats {
		fmt.Fprintf(&report, "%s.%s: %d rows\n", info.SchemaName, info.TableName, info.RowCount)
	}
	report.WriteString("\n")

	fmt.Fprintf(&report, "Target Database: %s\n", config.DstDB)
	for _, info := range targetStats {
		fmt.Fprintf(&report, "%s.%s: %d rows\n", info.SchemaName, info.TableName, info.RowCount)
	}
	report.WriteString("\n")

	return report.String()
}
