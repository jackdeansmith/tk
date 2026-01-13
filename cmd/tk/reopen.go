package main

import (
	"fmt"

	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var reopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Reopen a done or dropped task",
	Long: `Reopen a task that was previously completed or dropped.

Sets the status back to open and clears done_at, dropped_at, and drop_reason.
Does not affect dependent items.

Examples:
  tk reopen BY-07`,
	Args:              cobra.ExactArgs(1),
	RunE:              runReopen,
	ValidArgsFunction: completeTaskIDs,
}

func init() {
	rootCmd.AddCommand(reopenCmd)
}

func runReopen(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if err := ops.ReopenTask(s, taskID); err != nil {
		return err
	}

	fmt.Printf("%s reopened.\n", taskID)
	return nil
}
