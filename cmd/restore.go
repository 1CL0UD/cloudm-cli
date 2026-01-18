package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/1CL0UD/cloudm-cli/internal/config"
	"github.com/1CL0UD/cloudm-cli/internal/filesystem"
	"github.com/1CL0UD/cloudm-cli/internal/logger"
	"github.com/1CL0UD/cloudm-cli/internal/postgres"
	"github.com/1CL0UD/cloudm-cli/pkg/executor"
	"github.com/spf13/cobra"
)

var (
	inputDir string
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore from dump files",
	Long:  `Restore database from existing dump files`,
	RunE:  runRestore,
}

func init() {
	restoreCmd.Flags().StringVarP(&inputDir, "input", "i", "", "input directory containing dump files (required)")
	restoreCmd.Flags().BoolVar(&skipBackup, "skip-backup", false, "skip pre-migration backup")
	restoreCmd.Flags().BoolVar(&structureOnly, "structure-only", false, "restore only schema structure")
	restoreCmd.Flags().BoolVar(&dataOnly, "data-only", false, "restore only data")
	restoreCmd.MarkFlagRequired("input")
}

func runRestore(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
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

	log.Info("Starting database restore from: %s", inputDir)

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

	// Validate dump files exist
	if !structureOnly {
		if err := filesystem.ValidateDataDump(inputDir); err != nil {
			log.Error("Data dump file not found: %v", err)
			return err
		}
	}
	if !dataOnly {
		if err := filesystem.ValidateStructureDump(inputDir); err != nil {
			log.Error("Structure dump file not found: %v", err)
			return err
		}
	}
	log.Success("Dump files validated")

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

	structureDump, dataDump := filesystem.GetDumpPaths(inputDir)

	// Backup target (unless skipped)
	if !skipBackup && !cfg.Options.SkipBackup {
		log.Phase("Backup target database")

		backupFile := filesystem.GetBackupPath(inputDir)
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
		log.Success("Backup completed: %s", backupFile)
	}

	// Prepare target (terminate connections, drop/recreate schema, create extensions)
	log.Phase("Prepare target database")
	log.Info("Preparing target database...")
	if err := postgres.PrepareTarget(ctx, cfg.Target, cfg.Options.Extensions); err != nil {
		log.Error("Failed to prepare target: %v", err)
		return err
	}
	log.Success("Target prepared (schema recreated, extensions created)")

	// Restore structure (unless data-only)
	if !dataOnly {
		log.Phase("Restore database structure")
		log.Info("Restoring database structure (parallel jobs: %d)...", cfg.Options.ParallelJobs)
		if err := postgres.RestoreStructure(postgres.RestoreOptions{
			Host:         cfg.Target.Host,
			Port:         cfg.Target.Port,
			User:         cfg.Target.AdminUser,
			Password:     cfg.Target.AdminPassword,
			Database:     cfg.Target.Database,
			Schema:       "public",
			InputFile:    structureDump,
			ParallelJobs: cfg.Options.ParallelJobs,
		}); err != nil {
			log.Error("Structure restore failed: %v", err)
			return err
		}
		log.Success("Structure restored successfully")
	}

	// Restore data (unless structure-only)
	if !structureOnly {
		log.Phase("Restore database data")
		log.Info("Restoring database data (parallel jobs: %d)...", cfg.Options.DataParallelJobs)
		if err := postgres.RestoreData(postgres.RestoreOptions{
			Host:         cfg.Target.Host,
			Port:         cfg.Target.Port,
			User:         cfg.Target.AdminUser,
			Password:     cfg.Target.AdminPassword,
			Database:     cfg.Target.Database,
			Schema:       "public",
			InputFile:    dataDump,
			ParallelJobs: cfg.Options.DataParallelJobs,
		}); err != nil {
			log.Error("Data restore failed: %v", err)
			return err
		}
		log.Success("Data restored successfully")
	}

	// Configure ownership
	log.Phase("Configure ownership")
	if err := postgres.CreateAppUserIfNotExists(ctx, cfg.Target, cfg.Target.AppUser); err != nil {
		log.Error("Failed to create app user: %v", err)
		return err
	}

	log.Info("Transferring ownership to %s...", cfg.Target.AppUser)
	if err := postgres.TransferOwnership(ctx, cfg.Target, cfg.Target.AppUser); err != nil {
		log.Error("Ownership transfer failed: %v", err)
		return err
	}
	log.Success("Ownership configured successfully")

	log.Success("Restore completed in %s", time.Since(startTime).Round(time.Second))

	return nil
}