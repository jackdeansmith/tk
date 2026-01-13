package main

import (
	"fmt"

	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var dropCmd = &cobra.Command{
	Use:   "drop <id>",
	Short: "Drop a task",
	Long: `Drop a task (mark as not needed).

If the task has dependent items (tasks or waits blocked by it):
- Use --drop-deps to also drop all dependent items recursively
- Use --remove-deps to unlink this task from dependents

Examples:
  tk drop BY-07
  tk drop BY-07 --reason="No longer needed"
  tk drop BY-07 --drop-deps
  tk drop BY-07 --remove-deps`,
	Args:              cobra.ExactArgs(1),
	RunE:              runDrop,
	ValidArgsFunction: completeTaskIDs,
}

var (
	dropReason     string
	dropDropDeps   bool
	dropRemoveDeps bool
)

func init() {
	dropCmd.Flags().StringVar(&dropReason, "reason", "", "reason for dropping")
	dropCmd.Flags().BoolVar(&dropDropDeps, "drop-deps", false, "also drop dependent items")
	dropCmd.Flags().BoolVar(&dropRemoveDeps, "remove-deps", false, "unlink from dependent items")
	rootCmd.AddCommand(dropCmd)
}

func runDrop(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if err := ops.DropTask(s, taskID, dropReason, dropDropDeps, dropRemoveDeps); err != nil {
		return err
	}

	fmt.Printf("%s dropped.\n", taskID)
	return nil
}
