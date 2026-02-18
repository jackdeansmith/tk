package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var dumpCmd = &cobra.Command{
	Use:   "dump <project>",
	Short: "Export project as plain text",
	Long: `Export a project as human-readable plain text.

This is a one-way export for viewing/sharing - it cannot be re-imported.

The project can be specified by its ID (e.g., "backyard") or prefix (e.g., "BY").

Examples:
  tk dump backyard
  tk dump BY`,
	Args:              cobra.ExactArgs(1),
	RunE:              runDump,
	ValidArgsFunction: completeProjectIDs,
}

func init() {
	rootCmd.AddCommand(dumpCmd)
}

func runDump(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	pf, err := ops.ResolveProject(s, args[0])
	if err != nil {
		return err
	}

	blockerStates := ops.ComputeBlockerStates(pf)

	fmt.Printf("# %s: %s\n", pf.Prefix, pf.Name)
	if pf.Description != "" {
		fmt.Printf("# %s\n", pf.Description)
	}
	fmt.Printf("# Status: %s\n", pf.Status)
	fmt.Printf("# Created: %s\n", pf.Created.Format(time.RFC3339))
	fmt.Println()

	if len(pf.Tasks) > 0 {
		fmt.Println("## Tasks")
		fmt.Println()
		for _, t := range pf.Tasks {
			state := model.ComputeTaskState(&t, blockerStates)
			dumpTask(&t, state)
			fmt.Println()
		}
	}

	if len(pf.Waits) > 0 {
		fmt.Println("## Waits")
		fmt.Println()
		now := time.Now()
		for _, w := range pf.Waits {
			state := model.ComputeWaitState(&w, blockerStates, now)
			dumpWait(&w, state)
			fmt.Println()
		}
	}

	return nil
}

func dumpTask(t *model.Task, state model.TaskState) {
	fmt.Printf("### %s: %s\n", t.ID, t.Title)
	fmt.Printf("Status: %s (%s)\n", t.Status, state)
	fmt.Printf("Priority: P%d\n", t.Priority)

	if len(t.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(t.Tags, ", "))
	}
	if t.Assignee != "" {
		fmt.Printf("Assignee: %s\n", t.Assignee)
	}
	if t.DueDate != nil {
		fmt.Printf("Due: %s\n", t.DueDate.Format("2006-01-02"))
	}
	if t.AutoComplete {
		fmt.Println("Auto-complete: yes")
	}
	if len(t.BlockedBy) > 0 {
		fmt.Printf("Blocked by: %s\n", strings.Join(t.BlockedBy, ", "))
	}
	fmt.Printf("Created: %s\n", t.Created.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", t.Updated.Format(time.RFC3339))
	if t.DoneAt != nil {
		fmt.Printf("Done at: %s\n", t.DoneAt.Format(time.RFC3339))
	}
	if t.DroppedAt != nil {
		fmt.Printf("Dropped at: %s\n", t.DroppedAt.Format(time.RFC3339))
	}
	if t.DropReason != "" {
		fmt.Printf("Drop reason: %s\n", t.DropReason)
	}
	if t.Notes != "" {
		fmt.Println()
		fmt.Println("Notes:")
		for _, line := range strings.Split(t.Notes, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
}

func dumpWait(w *model.Wait, state model.WaitState) {
	fmt.Printf("### %s: %s\n", w.ID, w.DisplayText())
	fmt.Printf("Status: %s (%s)\n", w.Status, state)
	fmt.Printf("Type: %s\n", w.ResolutionCriteria.Type)
	if w.Title != "" {
		fmt.Printf("Title: %s\n", w.Title)
	}
	if w.ResolutionCriteria.Type == model.ResolutionTypeManual {
		if w.ResolutionCriteria.Question != "" {
			fmt.Printf("Question: %s\n", w.ResolutionCriteria.Question)
		}
		if w.ResolutionCriteria.CheckAfter != nil {
			fmt.Printf("Check after: %s\n", w.ResolutionCriteria.CheckAfter.Format(time.RFC3339))
		}
	} else if w.ResolutionCriteria.Type == model.ResolutionTypeTime {
		if w.ResolutionCriteria.After != nil {
			fmt.Printf("After: %s\n", w.ResolutionCriteria.After.Format(time.RFC3339))
		}
	}
	if len(w.BlockedBy) > 0 {
		fmt.Printf("Blocked by: %s\n", strings.Join(w.BlockedBy, ", "))
	}
	fmt.Printf("Created: %s\n", w.Created.Format(time.RFC3339))
	if w.DoneAt != nil {
		fmt.Printf("Done at: %s\n", w.DoneAt.Format(time.RFC3339))
	}
	if w.Resolution != "" {
		fmt.Printf("Resolution: %s\n", w.Resolution)
	}
	if w.DroppedAt != nil {
		fmt.Printf("Dropped at: %s\n", w.DroppedAt.Format(time.RFC3339))
	}
	if w.DropReason != "" {
		fmt.Printf("Drop reason: %s\n", w.DropReason)
	}
	if w.Notes != "" {
		fmt.Println()
		fmt.Println("Notes:")
		for _, line := range strings.Split(w.Notes, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}
}
