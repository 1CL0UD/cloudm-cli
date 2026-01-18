package cmd

import (
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
	// TODO: Implementation
	return nil
}