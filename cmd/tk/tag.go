package main

import (
	"fmt"
	"strings"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag <id> <tag>",
	Short: "Add a tag to a task",
	Long: `Add a tag to a task. Shortcut for --add-tag.

Examples:
  tk tag BY-07 weekend
  tk tag BY-07 urgent`,
	Args: cobra.ExactArgs(2),
	RunE: runTag,
}

var untagCmd = &cobra.Command{
	Use:   "untag <id> <tag>",
	Short: "Remove a tag from a task",
	Long: `Remove a tag from a task. Shortcut for --remove-tag.

Examples:
  tk untag BY-07 weekend
  tk untag BY-07 urgent`,
	Args: cobra.ExactArgs(2),
	RunE: runUntag,
}

func init() {
	rootCmd.AddCommand(tagCmd)
	rootCmd.AddCommand(untagCmd)
}

func runTag(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	tag := strings.ToLower(strings.TrimSpace(args[1]))

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

	// Check if tag already exists
	for _, t := range task.Tags {
		if strings.EqualFold(t, tag) {
			fmt.Printf("%s already has tag %q.\n", taskID, tag)
			return nil
		}
	}

	// Add tag
	newTags := append(task.Tags, tag)
	changes := ops.TaskChanges{Tags: &newTags}

	if err := ops.EditTask(s, taskID, changes); err != nil {
		return err
	}

	fmt.Printf("Added tag %q to %s.\n", tag, taskID)
	return nil
}

func runUntag(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	tag := strings.ToLower(strings.TrimSpace(args[1]))

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

	// Check if tag exists
	found := false
	var newTags []string
	for _, t := range task.Tags {
		if strings.EqualFold(t, tag) {
			found = true
		} else {
			newTags = append(newTags, t)
		}
	}

	if !found {
		fmt.Printf("%s does not have tag %q.\n", taskID, tag)
		return nil
	}

	changes := ops.TaskChanges{Tags: &newTags}

	if err := ops.EditTask(s, taskID, changes); err != nil {
		return err
	}

	fmt.Printf("Removed tag %q from %s.\n", tag, taskID)
	return nil
}
