package main

import (
	"fmt"
	"strings"

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
	Args:              cobra.ExactArgs(2),
	RunE:              runTag,
	ValidArgsFunction: completeTaskIDsThenTags,
}

var untagCmd = &cobra.Command{
	Use:   "untag <id> <tag>",
	Short: "Remove a tag from a task",
	Long: `Remove a tag from a task. Shortcut for --remove-tag.

Examples:
  tk untag BY-07 weekend
  tk untag BY-07 urgent`,
	Args:              cobra.ExactArgs(2),
	RunE:              runUntag,
	ValidArgsFunction: completeTaskIDsThenTags,
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

	added, err := ops.AddTag(s, taskID, tag)
	if err != nil {
		return err
	}

	if !added {
		fmt.Printf("%s already has tag %q.\n", taskID, tag)
		return nil
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

	removed, err := ops.RemoveTag(s, taskID, tag)
	if err != nil {
		return err
	}

	if !removed {
		fmt.Printf("%s does not have tag %q.\n", taskID, tag)
		return nil
	}

	fmt.Printf("Removed tag %q from %s.\n", tag, taskID)
	return nil
}
