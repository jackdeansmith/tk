package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done <id>...",
	Short: "Mark task(s) as done",
	Long: `Mark one or more tasks as done.

If a task has incomplete blockers, an error is shown.
Use --force to remove incomplete blockers and complete anyway.

Multiple tasks can be specified (batch mode):
  tk done BY-07 BY-08 BY-09

In batch mode, tasks that can be completed will be completed,
and errors will be reported for tasks that couldn't be completed.

Examples:
  tk done BY-07
  tk done BY-07 --force
  tk done BY-07 BY-08 BY-09`,
	Args:              cobra.MinimumNArgs(1),
	RunE:              runDone,
	ValidArgsFunction: completeTaskIDs,
}

var doneForce bool

func init() {
	doneCmd.Flags().BoolVar(&doneForce, "force", false, "remove incomplete blockers and complete")
	rootCmd.AddCommand(doneCmd)
}

func runDone(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	var errs []string
	var successes []string
	hasBlockerError := false

	for _, taskID := range args {
		result, err := ops.CompleteTask(s, taskID, doneForce)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", taskID, err))
			var blockerErr *ops.IncompleteBlockersError
			if errors.As(err, &blockerErr) {
				hasBlockerError = true
			}
			continue
		}

		successes = append(successes, taskID)

		// Print result for this task
		fmt.Printf("%s done.\n", taskID)

		if len(result.Unblocked) > 0 {
			fmt.Printf("Unblocked: %s\n", strings.Join(result.Unblocked, ", "))
		}
		if len(result.Activated) > 0 {
			fmt.Printf("Waits now active: %s\n", strings.Join(result.Activated, ", "))
		}
		if len(result.AutoCompleted) > 0 {
			fmt.Printf("Auto-completed: %s\n", strings.Join(result.AutoCompleted, ", "))
		}
	}

	// Report errors
	if len(errs) > 0 {
		fmt.Println()
		for _, e := range errs {
			fmt.Printf("error: %s\n", e)
		}
		if !doneForce && hasBlockerError {
			fmt.Println("\nUse --force to remove blockers and complete anyway.")
		}
		// Return error if all failed
		if len(successes) == 0 {
			return fmt.Errorf("failed to complete any tasks")
		}
	}

	return nil
}
