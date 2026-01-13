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
	Args: cobra.ExactArgs(1),
	RunE: runShow,
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

	// Run autocheck if configured
	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.AutoCheck {
		_, _ = ops.RunCheck(s)
	}

	// Extract prefix from ID to find the project
	prefix := model.ExtractPrefix(id)
	if prefix == "" {
		return fmt.Errorf("invalid ID format: %s", id)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return fmt.Errorf("project with prefix %q not found", prefix)
	}

	now := time.Now()
	blockerStates := computeBlockerStates(pf)

	// Determine if this is a task or wait ID
	if model.IsWaitID(id) {
		return showWait(pf, id, blockerStates, now)
	}
	return showTask(pf, id, blockerStates)
}

func showTask(pf *model.ProjectFile, id string, blockerStates model.BlockerStatus) error {
	// Find the task
	var task *model.Task
	normalizedID := strings.ToUpper(id)
	for i := range pf.Tasks {
		if strings.ToUpper(pf.Tasks[i].ID) == normalizedID {
			task = &pf.Tasks[i]
			break
		}
		// Also try matching without leading zeros
		_, num, err := model.ParseTaskID(pf.Tasks[i].ID)
		if err == nil {
			_, searchNum, err := model.ParseTaskID(id)
			if err == nil && num == searchNum {
				task = &pf.Tasks[i]
				break
			}
		}
	}

	if task == nil {
		return fmt.Errorf("task %s not found", id)
	}

	state := model.ComputeTaskState(task, blockerStates)

	// Print header
	fmt.Printf("%s: %s\n", task.ID, task.Title)

	// Status with derived state
	fmt.Printf("Status:        %s (%s)\n", task.Status, state)

	// Priority
	fmt.Printf("Priority:      %d (%s)\n", task.Priority, priorityLabel(task.Priority))

	// Tags
	if len(task.Tags) > 0 {
		fmt.Printf("Tags:          %s\n", strings.Join(task.Tags, ", "))
	} else {
		fmt.Printf("Tags:          -\n")
	}

	// Assignee
	if task.Assignee != "" {
		fmt.Printf("Assignee:      %s\n", task.Assignee)
	} else {
		fmt.Printf("Assignee:      -\n")
	}

	// Due date
	if task.DueDate != nil {
		fmt.Printf("Due:           %s\n", task.DueDate.Format("2006-01-02"))
	} else {
		fmt.Printf("Due:           -\n")
	}

	// Auto-complete
	fmt.Printf("Auto-complete: %s\n", boolToYesNo(task.AutoComplete))

	// Timestamps
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

	// Blockers
	if len(task.BlockedBy) > 0 {
		fmt.Println()
		fmt.Println("Blocked by:")
		for _, blockerID := range task.BlockedBy {
			blockerInfo := getBlockerInfo(pf, blockerID)
			fmt.Printf("  %s\n", blockerInfo)
		}
	}

	// Notes
	if task.Notes != "" {
		fmt.Println()
		fmt.Println("Notes:")
		// Indent notes
		for _, line := range strings.Split(task.Notes, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}

	return nil
}

func showWait(pf *model.ProjectFile, id string, blockerStates model.BlockerStatus, now time.Time) error {
	// Find the wait
	var wait *model.Wait
	normalizedID := strings.ToUpper(id)
	for i := range pf.Waits {
		if strings.ToUpper(pf.Waits[i].ID) == normalizedID {
			wait = &pf.Waits[i]
			break
		}
		// Also try matching without leading zeros
		_, num, err := model.ParseWaitID(pf.Waits[i].ID)
		if err == nil {
			_, searchNum, err := model.ParseWaitID(id)
			if err == nil && num == searchNum {
				wait = &pf.Waits[i]
				break
			}
		}
	}

	if wait == nil {
		return fmt.Errorf("wait %s not found", id)
	}

	state := model.ComputeWaitState(wait, blockerStates, now)

	// Print header
	displayText := wait.DisplayText()
	fmt.Printf("%s: %s\n", wait.ID, displayText)

	// Status with derived state
	fmt.Printf("Status:      %s (%s)\n", wait.Status, state)

	// Resolution criteria
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

	// Title (if different from display text or explicit)
	if wait.Title != "" && wait.Title != displayText {
		fmt.Printf("Title:       %s\n", wait.Title)
	}

	// Timestamps
	fmt.Printf("Created:     %s\n", wait.Created.Format(time.RFC3339))
	if wait.DoneAt != nil {
		fmt.Printf("Done at:     %s\n", wait.DoneAt.Format(time.RFC3339))
	}
	if wait.DroppedAt != nil {
		fmt.Printf("Dropped at:  %s\n", wait.DroppedAt.Format(time.RFC3339))
	}

	// Resolution
	if wait.Resolution != "" {
		fmt.Printf("Resolution:  %s\n", wait.Resolution)
	}
	if wait.DropReason != "" {
		fmt.Printf("Drop reason: %s\n", wait.DropReason)
	}

	// Blockers
	if len(wait.BlockedBy) > 0 {
		fmt.Println()
		fmt.Println("Blocked by:")
		for _, blockerID := range wait.BlockedBy {
			blockerInfo := getBlockerInfo(pf, blockerID)
			fmt.Printf("  %s\n", blockerInfo)
		}
	}

	// Notes
	if wait.Notes != "" {
		fmt.Println()
		fmt.Println("Notes:")
		for _, line := range strings.Split(wait.Notes, "\n") {
			fmt.Printf("  %s\n", line)
		}
	}

	return nil
}

func getBlockerInfo(pf *model.ProjectFile, blockerID string) string {
	normalizedID := strings.ToUpper(blockerID)

	// Check if it's a task
	for _, t := range pf.Tasks {
		if strings.ToUpper(t.ID) == normalizedID {
			statusStr := formatStatusBracket(string(t.Status))
			return fmt.Sprintf("%s %s %s", t.ID, statusStr, t.Title)
		}
	}

	// Check if it's a wait
	for _, w := range pf.Waits {
		if strings.ToUpper(w.ID) == normalizedID {
			statusStr := formatStatusBracket(string(w.Status))
			return fmt.Sprintf("%s %s %s", w.ID, statusStr, w.DisplayText())
		}
	}

	// Unknown blocker
	return fmt.Sprintf("%s [unknown]", blockerID)
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
