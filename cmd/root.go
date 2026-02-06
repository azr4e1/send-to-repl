package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

const Version = "v0.1.0"

var (
	rootCmd = &cobra.Command{
		Use:     "gorepl",
		Short:   "A repl multiplexer",
		Long:    "gorepl allows use to spin up a repl and send lines to it from anywhere",
		Version: Version,
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
