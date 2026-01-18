package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/1CL0UD/cloudm-cli/internal/config"
	"github.com/1CL0UD/cloudm-cli/internal/filesystem"
	"github.com/1CL0UD/cloudm-cli/internal/logger"
	"github.com/1CL0UD/cloudm-cli/internal/postgres"
	"github.com/1CL0UD/cloudm-cli/pkg/executor"
	"github.com/spf13/cobra"
)

var (
	skipBackup bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run full migration pipeline",
	Long:  `Executes the complete migration: backup → dump → restore → ownership → validate`,
	RunE:  runMigrate,
}

func init() {
	migrateCmd.Flags().BoolVar(&skipBackup, "skip-backup", false, "skip pre-migration backup")
}

func runMigrate(cmd *cobra.Command, args []string) error {
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

	log.Info("Starting database migration")

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

	// Initialize executor
	exec := executor.New(log, dryRun)

	// Check required tools
	if err := exec.CheckRequiredTools(); err != nil {
		log.Error("Pre-flight check failed: %v", err)
		return err
	}
	log.Success("Pre-flight checks passed")

	// Test connections
	log.Step(1, 5, "Testing database connections")
	if err := postgres.TestConnection(cfg.Source); err != nil {
		log.Error("Failed to connect to source database: %v", err)
		return err
	}
	log.Success("Connected to source database: %s/%s", cfg.Source.Host, cfg.Source.Database)

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

	// Create migration directory
	outputDir := cfg.Options.OutputDir
	if outputDir == "" {
		outputDir = "./migrations"
	}
	migrationDir, err := filesystem.CreateMigrationDir(outputDir)
	if err != nil {
		log.Error("Failed to create migration directory: %v", err)
		return err
	}
	log.Info("Migration directory: %s", migrationDir)

	// Update log file to migration directory
	mainLog, timeLog, validationLog := filesystem.GetLogPaths(migrationDir)
	_ = timeLog // Will be used for timing report

	var phases []logger.PhaseReport
	var files []string

	// Phase 0: Backup target (unless skipped)
	if !skipBackup && !cfg.Options.SkipBackup {
		log.Phase("STEP 0: Backup target database")
		backupStart := time.Now()

		backupFile := filesystem.GetBackupPath(migrationDir)
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

		phases = append(phases, logger.PhaseReport{Name: "Backup", Duration: time.Since(backupStart)})
		files = append(files, backupFile)
		log.Success("Backup completed: %s", backupFile)
	} else {
		log.Info("Skipping backup (--skip-backup flag or config)")
	}

	// Phase 1: Dump from source
	log.Phase("STEP 1: Dump from source database")
	dumpStart := time.Now()

	structureDump, dataDump := filesystem.GetDumpPaths(migrationDir)

	// Dump structure
	log.Info("Dumping database structure...")
	if err := postgres.DumpStructure(postgres.DumpOptions{
		Host:       cfg.Source.Host,
		Port:       cfg.Source.Port,
		User:       cfg.Source.User,
		Password:   cfg.Source.Password,
		Database:   cfg.Source.Database,
		Schema:     "public",
		OutputFile: structureDump,
	}); err != nil {
		log.Error("Structure dump failed: %v", err)
		return err
	}
	files = append(files, structureDump)
	log.Success("Structure dump completed: %s", structureDump)

	// Dump data
	log.Info("Dumping database data...")
	if err := postgres.DumpData(postgres.DumpOptions{
		Host:          cfg.Source.Host,
		Port:          cfg.Source.Port,
		User:          cfg.Source.User,
		Password:      cfg.Source.Password,
		Database:      cfg.Source.Database,
		Schema:        "public",
		OutputFile:    dataDump,
		ExcludeTables: cfg.Options.ExcludeTables,
	}); err != nil {
		log.Error("Data dump failed: %v", err)
		return err
	}
	files = append(files, dataDump)
	log.Success("Data dump completed: %s", dataDump)

	phases = append(phases, logger.PhaseReport{Name: "Dump", Duration: time.Since(dumpStart)})

	// Phase 2: Prepare and restore to target
	log.Phase("STEP 2: Clean & restore to target database")
	restoreStart := time.Now()

	// Prepare target (terminate connections, drop/recreate schema, create extensions)
	log.Info("Preparing target database...")
	if err := postgres.PrepareTarget(ctx, cfg.Target, cfg.Options.Extensions); err != nil {
		log.Error("Failed to prepare target: %v", err)
		return err
	}
	log.Success("Target prepared (schema recreated, extensions created)")

	// Restore structure
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

	// Restore data
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

	phases = append(phases, logger.PhaseReport{Name: "Restore", Duration: time.Since(restoreStart)})

	// Phase 3: Configure ownership
	log.Phase("STEP 3: Configure ownership for " + cfg.Target.AppUser)
	ownershipStart := time.Now()

	// Create app user if needed
	if err := postgres.CreateAppUserIfNotExists(ctx, cfg.Target, cfg.Target.AppUser); err != nil {
		log.Error("Failed to create app user: %v", err)
		return err
	}

	// Transfer ownership
	log.Info("Transferring ownership to %s...", cfg.Target.AppUser)
	if err := postgres.TransferOwnership(ctx, cfg.Target, cfg.Target.AppUser); err != nil {
		log.Error("Ownership transfer failed: %v", err)
		return err
	}
	log.Success("Ownership configured successfully")

	phases = append(phases, logger.PhaseReport{Name: "Ownership", Duration: time.Since(ownershipStart)})

	// Phase 4: Validation
	log.Phase("STEP 4: Validation")

	// Get table stats from both databases
	sourceStats, err := postgres.GetTableStats(ctx, cfg.Source, "public")
	if err != nil {
		log.Warning("Failed to get source stats: %v", err)
	}

	targetStats, err := postgres.GetTargetTableStats(ctx, cfg.Target, "public")
	if err != nil {
		log.Warning("Failed to get target stats: %v", err)
	}

	// Compare and generate report
	if sourceStats != nil && targetStats != nil {
		if err := logger.GenerateValidationReport(sourceStats, targetStats, validationLog); err != nil {
			log.Warning("Failed to generate validation report: %v", err)
		} else {
			files = append(files, validationLog)
			log.Info("Validation report saved to: %s", validationLog)
		}

		_, hasDiscrepancy := postgres.CompareTableStats(sourceStats, targetStats)
		if hasDiscrepancy {
			log.Warning("Discrepancies found! Review validation report.")
		} else {
			log.Success("All tables validated successfully!")
		}
	}

	// Generate timing report
	endTime := time.Now()
	report := logger.MigrationReport{
		StartTime: startTime,
		EndTime:   endTime,
		Phases:    phases,
		Files:     files,
		Success:   true,
	}

	if err := logger.GenerateReport(report, filepath.Join(migrationDir, "migration_time.txt")); err != nil {
		log.Warning("Failed to generate timing report: %v", err)
	}
	files = append(files, mainLog)

	// Cleanup if configured
	if !cfg.Options.KeepDumps {
		log.Info("Cleaning up dump files...")
		if err := filesystem.CleanupDumps(migrationDir); err != nil {
			log.Warning("Failed to cleanup dumps: %v", err)
		}
	}

	// Summary
	log.Phase("MIGRATION COMPLETED SUCCESSFULLY")
	log.Info("Total time: %s", time.Since(startTime).Round(time.Second))
	log.Info("Migration directory: %s", migrationDir)

	if !skipBackup && !cfg.Options.SkipBackup {
		log.Info("")
		log.Info("IMPORTANT: Pre-migration backup saved to %s", filesystem.GetBackupPath(migrationDir))
		log.Info("Keep this backup until you've verified the migration is successful")
	}

	log.Info("")
	log.Info("Next steps:")
	log.Info("1. Review validation report: %s", validationLog)
	log.Info("2. Test application connectivity")
	log.Info("3. If %s was created, change its password", cfg.Target.AppUser)
	log.Info("4. Verify critical business processes")
	log.Info("5. Once verified, clean up dump files and old backup")

	return nil
}