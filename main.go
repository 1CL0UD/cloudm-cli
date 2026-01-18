package main

import (
	"os"
	"github.com/1CL0UD/cloudm-cli/cmd"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
	GitCommit = "unknown"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}