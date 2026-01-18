package main

import (
	"os"

	"github.com/1CL0UD/cloudm-cli/cmd"
)

// Version info - set via ldflags during build
var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func init() {
	// Set version info in cmd package
	cmd.Version = Version
	cmd.BuildDate = BuildDate
	cmd.GitCommit = GitCommit
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}