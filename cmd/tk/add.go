package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new task",
	Long: `Add a new task to a project.

If --project is not specified, uses the default_project from .tkconfig.yaml.

Examples:
  tk add "Dig test hole"
  tk add "Dig test hole" --project=backyard
  tk add "Dig test hole" -p BY --priority=1 --tag=weekend
  tk add "Dig test hole" -p BY --blocked-by=BY-05,BY-03W`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

var (
	addProject      string
	addPriority     int
	addP1           bool
	addP2           bool
	addP3           bool
	addP4           bool
	addTags         []string
	addNotes        string
	addAssignee     string
	addDueDate      string
	addAutoComplete bool
	addBlockedBy    string
)

func init() {
	addCmd.Flags().StringVarP(&addProject, "project", "p", "", "project prefix or ID")
	addCmd.Flags().IntVar(&addPriority, "priority", 0, "task priority (1-4)")
	addCmd.Flags().BoolVar(&addP1, "p1", false, "shorthand for --priority=1")
	addCmd.Flags().BoolVar(&addP2, "p2", false, "shorthand for --priority=2")
	addCmd.Flags().BoolVar(&addP3, "p3", false, "shorthand for --priority=3")
	addCmd.Flags().BoolVar(&addP4, "p4", false, "shorthand for --priority=4")
	addCmd.Flags().StringArrayVar(&addTags, "tag", nil, "add a tag (can be repeated)")
	addCmd.Flags().StringVar(&addNotes, "notes", "", "task notes")
	addCmd.Flags().StringVar(&addAssignee, "assignee", "", "task assignee")
	addCmd.Flags().StringVar(&addDueDate, "due-date", "", "due date (YYYY-MM-DD)")
	addCmd.Flags().BoolVar(&addAutoComplete, "auto-complete", false, "auto-complete when blockers done")
	addCmd.Flags().StringVar(&addBlockedBy, "blocked-by", "", "comma-separated blocker IDs")

	addCmd.RegisterFlagCompletionFunc("project", completeProjectIDs)
	addCmd.RegisterFlagCompletionFunc("tag", completeTags)

	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	title := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	// Resolve project (uses default from config if empty)
	pf, err := ops.ResolveProject(s, addProject)
	if err != nil {
		return err
	}

	priority := resolvePriorityShorthand(addPriority, addP1, addP2, addP3, addP4)

	if priority != 0 {
		if err := ops.ValidatePriority(priority); err != nil {
			return err
		}
	}

	opts := ops.TaskOptions{
		Priority:     priority,
		Tags:         addTags,
		Notes:        addNotes,
		Assignee:     addAssignee,
		AutoComplete: addAutoComplete,
	}

	if addDueDate != "" {
		t, err := time.Parse("2006-01-02", addDueDate)
		if err != nil {
			return fmt.Errorf("invalid due date format (expected YYYY-MM-DD): %v", err)
		}
		opts.DueDate = &t
	}

	if addBlockedBy != "" {
		opts.BlockedBy = strings.Split(addBlockedBy, ",")
		for i, id := range opts.BlockedBy {
			opts.BlockedBy[i] = strings.TrimSpace(id)
		}
	}

	// Use default priority from config if not specified
	if opts.Priority == 0 {
		cfg, err := s.LoadConfig()
		if err == nil && cfg.DefaultPriority > 0 {
			opts.Priority = cfg.DefaultPriority
		}
	}

	task, err := ops.AddTask(s, pf.Prefix, title, opts)
	if err != nil {
		return err
	}

	fmt.Printf("%s %s\n", task.ID, task.Title)
	return nil
}
