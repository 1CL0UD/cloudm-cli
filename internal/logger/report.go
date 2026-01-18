package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/1CL0UD/cloudm-cli/internal/postgres"
)

type MigrationReport struct {
	StartTime    time.Time
	EndTime      time.Time
	Phases       []PhaseReport
	Files        []string
	Success      bool
	ErrorMessage string
}

type PhaseReport struct {
	Name     string
	Duration time.Duration
}

// GenerateReport generates a migration report file
func GenerateReport(report MigrationReport, outputPath string) error {
	var sb strings.Builder

	sb.WriteString("==========================================\n")
	sb.WriteString("Migration Timing Summary\n")
	sb.WriteString("==========================================\n")
	sb.WriteString(fmt.Sprintf("Start Time: %s\n", report.StartTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("End Time: %s\n", report.EndTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Total Duration: %s\n", formatDuration(report.EndTime.Sub(report.StartTime))))
	sb.WriteString(fmt.Sprintf("Status: %s\n", statusString(report.Success)))
	if !report.Success && report.ErrorMessage != "" {
		sb.WriteString(fmt.Sprintf("Error: %s\n", report.ErrorMessage))
	}
	sb.WriteString("\n")

	sb.WriteString("Phase Breakdown:\n")
	for _, phase := range report.Phases {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", phase.Name, formatDuration(phase.Duration)))
	}
	sb.WriteString("\n")

	if len(report.Files) > 0 {
		sb.WriteString("Files Generated:\n")
		for _, file := range report.Files {
			sb.WriteString(fmt.Sprintf("  %s\n", file))
		}
	}

	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}

// GenerateValidationReport generates a validation report comparing source and target
func GenerateValidationReport(source, target []postgres.TableStats, outputPath string) error {
	var sb strings.Builder

	sb.WriteString("==========================================\n")
	sb.WriteString("Migration Validation Report\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("==========================================\n\n")

	// Create lookup map for target tables
	targetMap := make(map[string]postgres.TableStats)
	for _, t := range target {
		key := fmt.Sprintf("%s.%s", t.Schema, t.Table)
		targetMap[key] = t
	}

	// Compare tables
	sb.WriteString("Table Comparison:\n")
	sb.WriteString(fmt.Sprintf("%-40s %15s %15s %10s\n", "Table", "Source Rows", "Target Rows", "Status"))
	sb.WriteString(strings.Repeat("-", 85) + "\n")

	hasDiscrepancy := false
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

	sb.WriteString("\n")
	if hasDiscrepancy {
		sb.WriteString("⚠ WARNING: Discrepancies found! Please review the above table.\n")
	} else {
		sb.WriteString("✓ All tables match! Migration validated successfully.\n")
	}

	// Add table sizes if available
	sb.WriteString("\nTarget Database Sizes:\n")
	sb.WriteString(fmt.Sprintf("%-40s %15s\n", "Table", "Size"))
	sb.WriteString(strings.Repeat("-", 55) + "\n")
	for _, t := range target {
		if t.Size != "" {
			sb.WriteString(fmt.Sprintf("%-40s %15s\n",
				fmt.Sprintf("%s.%s", t.Schema, t.Table), t.Size))
		}
	}

	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}

// formatDuration formats a duration as HH:MM:SS
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// statusString returns a human-readable status
func statusString(success bool) string {
	if success {
		return "SUCCESS"
	}
	return "FAILED"
}