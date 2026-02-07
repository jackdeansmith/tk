package main

import (
	"fmt"
	"strings"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var noteCmd = &cobra.Command{
	Use:   "note <id> <text>...",
	Short: "Append a note to a task",
	Long: `Append text to a task's notes field.

If the task already has notes, the new text is appended with a newline separator.
All arguments after the task ID are joined with spaces to form the note text.

Examples:
  tk note BY-07 Need to check with supplier first
  tk note BY-07 "Called supplier, they said 2 weeks"`,
	Args:              cobra.MinimumNArgs(2),
	RunE:              runNote,
	ValidArgsFunction: completeTaskIDs,
}

func init() {
	rootCmd.AddCommand(noteCmd)
}

func runNote(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	text := strings.Join(args[1:], " ")

	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("note text must not be empty")
	}

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	// Find the task
	var task *model.Task
	for i := range pf.Tasks {
		if strings.EqualFold(pf.Tasks[i].ID, taskID) {
			task = &pf.Tasks[i]
			break
		}
	}
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Build new notes: append with newline separator if existing notes
	var newNotes string
	if task.Notes == "" {
		newNotes = text
	} else {
		newNotes = task.Notes + "\n" + text
	}

	changes := ops.TaskChanges{Notes: &newNotes}

	if err := ops.EditTask(s, taskID, changes); err != nil {
		return err
	}

	fmt.Printf("Note added to %s.\n", taskID)
	return nil
}
