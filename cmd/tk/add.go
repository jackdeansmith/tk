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
	addProject     string
	addPriority    int
	addTags        []string
	addNotes       string
	addAssignee    string
	addDueDate     string
	addAutoComplete bool
	addBlockedBy   string
)

func init() {
	addCmd.Flags().StringVarP(&addProject, "project", "p", "", "project prefix or ID")
	addCmd.Flags().IntVar(&addPriority, "priority", 0, "task priority (1-4)")
	addCmd.Flags().StringArrayVar(&addTags, "tag", nil, "add a tag (can be repeated)")
	addCmd.Flags().StringVar(&addNotes, "notes", "", "task notes")
	addCmd.Flags().StringVar(&addAssignee, "assignee", "", "task assignee")
	addCmd.Flags().StringVar(&addDueDate, "due-date", "", "due date (YYYY-MM-DD)")
	addCmd.Flags().BoolVar(&addAutoComplete, "auto-complete", false, "auto-complete when blockers done")
	addCmd.Flags().StringVar(&addBlockedBy, "blocked-by", "", "comma-separated blocker IDs")

	// Register completion functions
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

	// Determine project
	project := addProject
	if project == "" {
		cfg, err := s.LoadConfig()
		if err != nil {
			return err
		}
		if cfg.DefaultProject == "" {
			return fmt.Errorf("no project specified and no default_project in config")
		}
		// Load the default project to get its prefix
		pf, err := s.LoadProjectByID(cfg.DefaultProject)
		if err != nil {
			return fmt.Errorf("default project %q not found", cfg.DefaultProject)
		}
		project = pf.Prefix
	} else {
		// Try to resolve project reference (could be prefix or ID)
		pf, err := s.LoadProject(project)
		if err != nil {
			pf, err = s.LoadProjectByID(project)
			if err != nil {
				return fmt.Errorf("project %q not found", project)
			}
		}
		project = pf.Prefix
	}

	// Validate priority if provided
	if addPriority != 0 {
		if err := ops.ValidatePriority(addPriority); err != nil {
			return err
		}
	}

	// Build options
	opts := ops.TaskOptions{
		Priority:     addPriority,
		Tags:         addTags,
		Notes:        addNotes,
		Assignee:     addAssignee,
		AutoComplete: addAutoComplete,
	}

	// Parse due date if provided
	if addDueDate != "" {
		t, err := time.Parse("2006-01-02", addDueDate)
		if err != nil {
			return fmt.Errorf("invalid due date format (expected YYYY-MM-DD): %v", err)
		}
		opts.DueDate = &t
	}

	// Parse blocked-by if provided
	if addBlockedBy != "" {
		opts.BlockedBy = strings.Split(addBlockedBy, ",")
		// Trim whitespace from each ID
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

	task, err := ops.AddTask(s, project, title, opts)
	if err != nil {
		return err
	}

	fmt.Printf("%s %s\n", task.ID, task.Title)
	return nil
}
