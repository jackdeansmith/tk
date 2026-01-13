package main

import (
	"fmt"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project <id>",
	Short: "Show project summary",
	Long: `Show a summary of the specified project.

The project can be specified by its ID (e.g., "backyard") or prefix (e.g., "BY").

Output shows counts of open tasks (broken down by ready, blocked, waiting),
done tasks, dropped tasks, and waits.`,
	Args: cobra.ExactArgs(1),
	RunE: runProject,
}

func init() {
	rootCmd.AddCommand(projectCmd)
}

func runProject(cmd *cobra.Command, args []string) error {
	projectRef := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	// Try loading by prefix first, then by ID
	pf, err := s.LoadProject(projectRef)
	if err != nil {
		// Try by ID
		pf, err = s.LoadProjectByID(projectRef)
		if err != nil {
			return fmt.Errorf("project %q not found", projectRef)
		}
	}

	// Compute blocker states for state calculations
	blockerStates := computeBlockerStates(pf)

	// Count tasks by state
	var openCount, readyCount, blockedCount, waitingCount, doneCount, droppedCount int
	for _, t := range pf.Tasks {
		switch t.Status {
		case model.TaskStatusDone:
			doneCount++
		case model.TaskStatusDropped:
			droppedCount++
		case model.TaskStatusOpen:
			openCount++
			state := model.ComputeTaskState(&t, blockerStates)
			switch state {
			case model.TaskStateReady:
				readyCount++
			case model.TaskStateBlocked:
				blockedCount++
			case model.TaskStateWaiting:
				waitingCount++
			}
		}
	}

	// Count waits
	var openWaits, doneWaits, droppedWaits int
	for _, w := range pf.Waits {
		switch w.Status {
		case model.WaitStatusOpen:
			openWaits++
		case model.WaitStatusDone:
			doneWaits++
		case model.WaitStatusDropped:
			droppedWaits++
		}
	}

	// Print summary
	fmt.Printf("%s: %s\n", pf.Prefix, pf.Name)
	if pf.Description != "" {
		fmt.Printf("%s\n", pf.Description)
	}
	fmt.Printf("Status: %s\n", pf.Status)
	fmt.Println()

	// Task summary
	if openCount > 0 {
		fmt.Printf("%d open", openCount)
		details := []string{}
		if readyCount > 0 {
			details = append(details, fmt.Sprintf("%d ready", readyCount))
		}
		if blockedCount > 0 {
			details = append(details, fmt.Sprintf("%d blocked", blockedCount))
		}
		if waitingCount > 0 {
			details = append(details, fmt.Sprintf("%d waiting", waitingCount))
		}
		if len(details) > 0 {
			fmt.Printf(" (")
			for i, d := range details {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", d)
			}
			fmt.Printf(")")
		}
	} else {
		fmt.Printf("0 open")
	}
	fmt.Printf(", %d done", doneCount)
	if droppedCount > 0 {
		fmt.Printf(", %d dropped", droppedCount)
	}
	fmt.Println()

	// Wait summary
	if openWaits > 0 || doneWaits > 0 || droppedWaits > 0 {
		fmt.Printf("%d waits", openWaits)
		if doneWaits > 0 {
			fmt.Printf(" (%d resolved)", doneWaits)
		}
		fmt.Println()
	}

	return nil
}

// computeBlockerStates builds a map of blocker ID to resolved status.
// true = resolved (done/dropped), false = still open (blocks)
func computeBlockerStates(pf *model.ProjectFile) model.BlockerStatus {
	states := make(model.BlockerStatus)

	for _, t := range pf.Tasks {
		states[t.ID] = t.Status == model.TaskStatusDone || t.Status == model.TaskStatusDropped
	}

	for _, w := range pf.Waits {
		states[w.ID] = w.Status == model.WaitStatusDone || w.Status == model.WaitStatusDropped
	}

	return states
}
