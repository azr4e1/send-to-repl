package cmd

import (
	"github.com/spf13/cobra"
)

var (
	sendCmd = &cobra.Command{
		Use:   "send",
		Short: "Send data to repl",
		Run:   Send,
		Args:  cobra.MaximumNArgs(1),
	}
)

func Send(command *cobra.Command, args []string) {}

func init() {
	rootCmd.AddCommand(sendCmd)
}
