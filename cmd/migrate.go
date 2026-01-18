package cmd

import (
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
	// TODO: Implementation
	return nil
}