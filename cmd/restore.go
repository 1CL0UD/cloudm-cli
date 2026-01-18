package cmd

import (
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
	// TODO: Implementation
	return nil
}