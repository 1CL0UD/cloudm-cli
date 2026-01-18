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

var (
	outputDir     string
	structureOnly bool
	dataOnly      bool
)

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump source database",
	Long:  `Dump structure and/or data from source database to local files`,
	RunE:  runDump,
}

func init() {
	dumpCmd.Flags().StringVar(&outputDir, "output", "", "output directory for dumps")
	dumpCmd.Flags().BoolVar(&structureOnly, "structure-only", false, "dump only schema structure")
	dumpCmd.Flags().BoolVar(&dataOnly, "data-only", false, "dump only data")
}

func runDump(cmd *cobra.Command, args []string) error {
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

	log.Info("Starting database dump")

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

	// Validate source configuration
	if cfg.Source.Host == "" || cfg.Source.Database == "" {
		log.Error("Source database configuration is incomplete")
		return fmt.Errorf("source database configuration is incomplete")
	}

	// Initialize executor
	exec := executor.New(log, dryRun)

	// Check required tools
	if err := exec.CheckRequiredTools(); err != nil {
		log.Error("Pre-flight check failed: %v", err)
		return err
	}

	// Test connection
	log.Info("Testing connection to source database...")
	if err := postgres.TestConnection(cfg.Source); err != nil {
		log.Error("Failed to connect to source database: %v", err)
		return err
	}
	log.Success("Connected to source database: %s/%s", cfg.Source.Host, cfg.Source.Database)

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
		outDir = "./migrations"
	}

	// Create migration directory
	migrationDir, err := filesystem.CreateMigrationDir(outDir)
	if err != nil {
		log.Error("Failed to create output directory: %v", err)
		return err
	}
	log.Info("Output directory: %s", migrationDir)

	structureDump, dataDump := filesystem.GetDumpPaths(migrationDir)

	// Dump structure (unless data-only)
	if !dataOnly {
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

		size, _ := filesystem.GetFileSize(structureDump)
		log.Success("Structure dump completed: %s (%s)", structureDump, size)
	}

	// Dump data (unless structure-only)
	if !structureOnly {
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

		size, _ := filesystem.GetFileSize(dataDump)
		log.Success("Data dump completed: %s (%s)", dataDump, size)
	}

	log.Success("Dump completed in %s", time.Since(startTime).Round(time.Second))
	log.Info("Files saved to: %s", migrationDir)

	return nil
}