package main

import (
	"github.com/spf13/cobra"
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List ready tasks",
	Long: `List all tasks that are ready to work on.

This is an alias for 'tk list --ready'.

Ready tasks are open tasks with no incomplete blockers.`,
	RunE: runReady,
}

func init() {
	rootCmd.AddCommand(readyCmd)
}

func runReady(cmd *cobra.Command, args []string) error {
	// Set the ready flag and delegate to list
	listReady = true
	return runList(cmd, args)
}
