package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/1CL0UD/cloudm-cli/internal/config"
	"github.com/1CL0UD/cloudm-cli/internal/filesystem"
	"github.com/1CL0UD/cloudm-cli/internal/logger"
	"github.com/1CL0UD/cloudm-cli/internal/postgres"
	"github.com/spf13/cobra"
)

var (
	detailed bool
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate migration",
	Long:  `Compare source and target databases to validate migration`,
	RunE:  runValidate,
}

func init() {
	validateCmd.Flags().BoolVar(&detailed, "detailed", false, "show detailed comparison")
}

func runValidate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize logger
	log, err := logger.New(logger.LoggerOptions{
		Verbose: verbose,
		LogFile: logFile,
		NoColor: noColor,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer log.Close()

	log.Info("Starting database validation")

	// Load configuration
	configPath := cfgFile
	if configPath == "" {
		configPath = "db.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Error("Failed to load configuration: %v", err)
		return err
	}

	// Validate configuration
	if err := config.Validate(cfg); err != nil {
		log.Error("Configuration validation failed: %v", err)
		return err
	}

	// Test connections
	log.Info("Testing database connections...")

	if err := postgres.TestConnection(cfg.Source); err != nil {
		log.Error("Failed to connect to source database: %v", err)
		return err
	}
	log.Success("Connected to source: %s/%s", cfg.Source.Host, cfg.Source.Database)

	if err := postgres.TestTargetConnection(cfg.Target); err != nil {
		log.Error("Failed to connect to target database: %v", err)
		return err
	}
	log.Success("Connected to target: %s/%s", cfg.Target.Host, cfg.Target.Database)

	// Get table stats from source
	log.Info("Fetching source database statistics...")
	sourceStats, err := postgres.GetTableStats(ctx, cfg.Source, "public")
	if err != nil {
		log.Error("Failed to get source stats: %v", err)
		return err
	}
	log.Info("Found %d tables in source database", len(sourceStats))

	// Get table stats from target
	log.Info("Fetching target database statistics...")
	targetStats, err := postgres.GetTargetTableStats(ctx, cfg.Target, "public")
	if err != nil {
		log.Error("Failed to get target stats: %v", err)
		return err
	}
	log.Info("Found %d tables in target database", len(targetStats))

	// Compare
	log.Phase("Comparison Results")
	report, hasDiscrepancy := postgres.CompareTableStats(sourceStats, targetStats)

	if detailed {
		fmt.Println(report)
	}

	// Save report to file
	outputDir := cfg.Options.OutputDir
	if outputDir == "" {
		outputDir = "./migrations"
	}

	timestamp := time.Now().Format("20060102_150405")
	reportPath := fmt.Sprintf("%s/validation_%s.log", outputDir, timestamp)

	// Ensure directory exists
	if _, err := filesystem.CreateMigrationDir(outputDir); err == nil {
		if err := logger.GenerateValidationReport(sourceStats, targetStats, reportPath); err != nil {
			log.Warning("Failed to save validation report: %v", err)
		} else {
			log.Info("Validation report saved to: %s", reportPath)
		}
	}

	// Summary
	if hasDiscrepancy {
		log.Warning("⚠ Discrepancies found between source and target databases!")
		log.Info("Review the comparison above for details")
		return fmt.Errorf("validation failed: discrepancies found")
	}

	log.Success("✓ All tables match! Migration validated successfully.")
	log.Info("Source tables: %d, Target tables: %d", len(sourceStats), len(targetStats))

	return nil
}