package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	dryRun   bool
	verbose  bool
	noColor  bool
	logFile  string
)

var rootCmd = &cobra.Command{
	Use:   "dbmigrate",
	Short: "PostgreSQL database migration tool",
	Long:  `A CLI tool for migrating PostgreSQL databases with dump, restore, and validation capabilities.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./dbmigrate.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be done without executing")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose logging")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "custom log file path")

	// Add subcommands
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(dumpCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("dbmigrate")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil && verbose {
		// Config file found and successfully parsed
	}
}