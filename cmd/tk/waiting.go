package main

import (
	"github.com/spf13/cobra"
)

var waitingCmd = &cobra.Command{
	Use:   "waiting",
	Short: "List actionable waits",
	Long: `List all waits that need attention.

This is an alias for 'tk waits --actionable'.

Shows manual waits that are ready for user action (not dormant,
past their check_after date if set).`,
	RunE: runWaiting,
}

func init() {
	rootCmd.AddCommand(waitingCmd)
}

func runWaiting(cmd *cobra.Command, args []string) error {
	// Set the actionable flag and delegate to waits
	waitsActionable = true
	return runWaits(cmd, args)
}
