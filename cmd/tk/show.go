package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/cli"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show task or wait details",
	Long: `Show full details for a task or wait.

The ID can be a task ID (e.g., BY-07) or a wait ID (e.g., BY-03W).
IDs are case-insensitive.

Shows all fields including blockers with their status.`,
	Args:              cobra.ExactArgs(1),
	RunE:              runShow,
	ValidArgsFunction: completeAnyIDs,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	id := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	ops.AutoCheck(s)

	if model.IsWaitID(id) {
		return showWait(s, id)
	}
	return showTask(s, id)
}

func showTask(s ops.Store, id string) error {
	result, pf, err := ops.ShowTask(s, id)
	if err != nil {
		return err
	}

	task := &result.Task

	fmt.Printf("%s: %s\n", task.ID, task.Title)
	fmt.Printf("Status:        %s (%s)\n", task.Status, result.State)
	fmt.Printf("Priority:      %d (%s)\n", task.Priority, priorityLabel(task.Priority))

	if len(task.Tags) > 0 {
		fmt.Printf("Tags:          %s\n", strings.Join(task.Tags, ", "))
	} else {
		fmt.Printf("Tags:          -\n")
	}

	if task.Assignee != "" {
		fmt.Printf("Assignee:      %s\n", task.Assignee)
	} else {
		fmt.Printf("Assignee:      -\n")
	}

	if task.DueDate != nil {
		fmt.Printf("Due:           %s\n", task.DueDate.Format("2006-01-02"))
	} else {
		fmt.Printf("Due:           -\n")
	}

	fmt.Printf("Auto-complete: %s\n", boolToYesNo(task.AutoComplete))

	fmt.Printf("Created:       %s\n", task.Created.Format(time.RFC3339))
	fmt.Printf("Updated:       %s\n", task.Updated.Format(time.RFC3339))
	if task.DoneAt != nil {
		fmt.Printf("Done at:       %s\n", task.DoneAt.Format(time.RFC3339))
	}
	if task.DroppedAt != nil {
		fmt.Printf("Dropped at:    %s\n", task.DroppedAt.Format(time.RFC3339))
	}
	if task.DropReason != "" {
		fmt.Printf("Drop reason:   %s\n", task.DropReason)
	}

	if len(task.BlockedBy) > 0 {
		fmt.Println()
		fmt.Println("Blocked by:")
		for _, blockerID := range task.BlockedBy {
			info := ops.GetBlockerInfo(pf, blockerID)
			fmt.Printf("  %s %s %s\n", info.ID, formatStatusBracket(info.Status), info.DisplayText)
		}
	}

	if task.Notes != "" {
		fmt.Println()
		fmt.Println("Notes:")
		for _, line := range strings.Split(task.Notes, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}

	return nil
}

func showWait(s ops.Store, id string) error {
	result, pf, err := ops.ShowWait(s, id)
	if err != nil {
		return err
	}

	wait := &result.Wait

	displayText := wait.DisplayText()
	fmt.Printf("%s: %s\n", wait.ID, displayText)
	fmt.Printf("Status:      %s (%s)\n", wait.Status, result.State)
	fmt.Printf("Type:        %s\n", wait.ResolutionCriteria.Type)

	if wait.ResolutionCriteria.Type == model.ResolutionTypeManual {
		fmt.Printf("Question:    %s\n", wait.ResolutionCriteria.Question)
		if wait.ResolutionCriteria.CheckAfter != nil {
			fmt.Printf("Check after: %s\n", wait.ResolutionCriteria.CheckAfter.Format(time.RFC3339))
		}
	} else if wait.ResolutionCriteria.Type == model.ResolutionTypeTime {
		if wait.ResolutionCriteria.After != nil {
			fmt.Printf("After:       %s\n", wait.ResolutionCriteria.After.Format(time.RFC3339))
		}
	}

	if wait.Title != "" && wait.Title != displayText {
		fmt.Printf("Title:       %s\n", wait.Title)
	}

	fmt.Printf("Created:     %s\n", wait.Created.Format(time.RFC3339))
	if wait.DoneAt != nil {
		fmt.Printf("Done at:     %s\n", wait.DoneAt.Format(time.RFC3339))
	}
	if wait.DroppedAt != nil {
		fmt.Printf("Dropped at:  %s\n", wait.DroppedAt.Format(time.RFC3339))
	}

	if wait.Resolution != "" {
		fmt.Printf("Resolution:  %s\n", wait.Resolution)
	}
	if wait.DropReason != "" {
		fmt.Printf("Drop reason: %s\n", wait.DropReason)
	}

	if len(wait.BlockedBy) > 0 {
		fmt.Println()
		fmt.Println("Blocked by:")
		for _, blockerID := range wait.BlockedBy {
			info := ops.GetBlockerInfo(pf, blockerID)
			fmt.Printf("  %s %s %s\n", info.ID, formatStatusBracket(info.Status), info.DisplayText)
		}
	}

	if wait.Notes != "" {
		fmt.Println()
		fmt.Println("Notes:")
		for _, line := range strings.Split(wait.Notes, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}

	return nil
}

func formatStatusBracket(status string) string {
	switch status {
	case "done":
		return cli.Gray("[done]")
	case "dropped":
		return cli.Gray("[dropped]")
	case "open":
		return cli.Yellow("[open]")
	default:
		return fmt.Sprintf("[%s]", status)
	}
}

func priorityLabel(p int) string {
	switch p {
	case 1:
		return "urgent"
	case 2:
		return "high"
	case 3:
		return "medium"
	case 4:
		return "backlog"
	default:
		return "unknown"
	}
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
