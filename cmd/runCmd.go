package cmd

import (
	"github.com/spf13/cobra"
)

var (
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the shell",
		Run:   Run,
		Args:  cobra.MaximumNArgs(1),
	}
)

func Run(command *cobra.Command, args []string) {}

func init() {
	rootCmd.AddCommand(runCmd)
	// runCmd.Flags().IntVarP()
}
