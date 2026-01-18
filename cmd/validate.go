package cmd

import (
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
	// TODO: Implementation
	return nil
}