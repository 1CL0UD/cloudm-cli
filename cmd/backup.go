package cmd

import (
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
	// TODO: Implementation
	return nil
}