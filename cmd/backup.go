package cmd

import (
	"fmt"
	"time"

	"github.com/1CL0UD/cloudm-cli/internal/config"
	"github.com/1CL0UD/cloudm-cli/internal/filesystem"
	"github.com/1CL0UD/cloudm-cli/internal/logger"
	"github.com/1CL0UD/cloudm-cli/internal/postgres"
	"github.com/1CL0UD/cloudm-cli/pkg/executor"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup target database",
	Long:  `Create a backup of the target database`,
	RunE:  runBackup,
}

func init() {
	backupCmd.Flags().StringVar(&outputDir, "output", "", "output directory for backup")
}

func runBackup(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

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

	log.Info("Starting database backup")

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

	// Validate target configuration
	if cfg.Target.Host == "" || cfg.Target.Database == "" {
		log.Error("Target database configuration is incomplete")
		return fmt.Errorf("target database configuration is incomplete")
	}

	// Initialize executor
	exec := executor.New(log, dryRun)

	// Check required tools
	if err := exec.CheckRequiredTools(); err != nil {
		log.Error("Pre-flight check failed: %v", err)
		return err
	}

	// Test connection
	log.Info("Testing connection to target database...")
	if err := postgres.TestTargetConnection(cfg.Target); err != nil {
		log.Error("Failed to connect to target database: %v", err)
		return err
	}
	log.Success("Connected to target database: %s/%s", cfg.Target.Host, cfg.Target.Database)

	if dryRun {
		log.DryRun("Dry run mode - no changes will be made")
		log.Success("Dry run completed successfully")
		return nil
	}

	// Determine output directory
	outDir := outputDir
	if outDir == "" {
		outDir = cfg.Options.OutputDir
	}
	if outDir == "" {
		outDir = "./backups"
	}

	// Create backup directory
	backupDir, err := filesystem.CreateMigrationDir(outDir)
	if err != nil {
		log.Error("Failed to create output directory: %v", err)
		return err
	}

	backupFile := filesystem.GetBackupPath(backupDir)

	log.Info("Creating backup of %s/%s...", cfg.Target.Host, cfg.Target.Database)
	if err := postgres.BackupDatabase(
		cfg.Target.Host,
		cfg.Target.Port,
		cfg.Target.AdminUser,
		cfg.Target.AdminPassword,
		cfg.Target.Database,
		backupFile,
	); err != nil {
		log.Error("Backup failed: %v", err)
		return err
	}

	// Get file size
	size, _ := filesystem.GetFileSize(backupFile)

	log.Success("Backup completed in %s", time.Since(startTime).Round(time.Second))
	log.Info("Backup file: %s (%s)", backupFile, size)

	return nil
}