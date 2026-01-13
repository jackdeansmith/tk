package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/cli"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Long: `List tasks with optional filtering.

By default, lists all open tasks in active projects.

Filter flags:
  --ready       Show only tasks that are ready to work on
  --blocked     Show only tasks blocked by other tasks
  --waiting     Show only tasks waiting on external conditions
  --done        Show only completed tasks
  --dropped     Show only dropped tasks
  --all         Show all tasks regardless of status

  -p, --project Limit to a specific project (by prefix or ID)
  --priority    Filter by priority (1-4)
  --p1/--p2/--p3/--p4  Shorthand for --priority=N
  --tag         Filter by tag (can be repeated, requires all tags)
  --overdue     Show only tasks with due date in the past

Tasks are sorted by ID.`,
	RunE: runList,
}

var (
	listProject  string
	listReady    bool
	listBlocked  bool
	listWaiting  bool
	listDone     bool
	listDropped  bool
	listAll      bool
	listPriority int
	listP1       bool
	listP2       bool
	listP3       bool
	listP4       bool
	listTags     []string
	listOverdue  bool
)

func init() {
	listCmd.Flags().StringVarP(&listProject, "project", "p", "", "filter by project (prefix or ID)")
	listCmd.Flags().BoolVar(&listReady, "ready", false, "show only ready tasks")
	listCmd.Flags().BoolVar(&listBlocked, "blocked", false, "show only blocked tasks")
	listCmd.Flags().BoolVar(&listWaiting, "waiting", false, "show only waiting tasks")
	listCmd.Flags().BoolVar(&listDone, "done", false, "show only done tasks")
	listCmd.Flags().BoolVar(&listDropped, "dropped", false, "show only dropped tasks")
	listCmd.Flags().BoolVar(&listAll, "all", false, "show all tasks")
	listCmd.Flags().IntVar(&listPriority, "priority", 0, "filter by priority (1-4)")
	listCmd.Flags().BoolVar(&listP1, "p1", false, "shorthand for --priority=1")
	listCmd.Flags().BoolVar(&listP2, "p2", false, "shorthand for --priority=2")
	listCmd.Flags().BoolVar(&listP3, "p3", false, "shorthand for --priority=3")
	listCmd.Flags().BoolVar(&listP4, "p4", false, "shorthand for --priority=4")
	listCmd.Flags().StringArrayVar(&listTags, "tag", nil, "filter by tag (can be repeated)")
	listCmd.Flags().BoolVar(&listOverdue, "overdue", false, "show only overdue tasks")

	// Register completion functions
	listCmd.RegisterFlagCompletionFunc("project", completeProjectIDs)
	listCmd.RegisterFlagCompletionFunc("tag", completeTags)

	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Resolve priority shorthand
	priority := listPriority
	if listP1 {
		priority = 1
	} else if listP2 {
		priority = 2
	} else if listP3 {
		priority = 3
	} else if listP4 {
		priority = 4
	}

	// Determine which projects to include
	var projects []*model.ProjectFile

	if listProject != "" {
		// Single project specified
		pf, err := s.LoadProject(listProject)
		if err != nil {
			pf, err = s.LoadProjectByID(listProject)
			if err != nil {
				return fmt.Errorf("project %q not found", listProject)
			}
		}
		projects = append(projects, pf)
	} else {
		// All active projects
		prefixes, err := s.ListProjects()
		if err != nil {
			return err
		}
		for _, prefix := range prefixes {
			pf, err := s.LoadProject(prefix)
			if err != nil {
				continue
			}
			// Only include active projects unless --all
			if !listAll && pf.Status != model.ProjectStatusActive {
				continue
			}
			projects = append(projects, pf)
		}
	}

	// Collect all tasks with filtering
	table := cli.NewTable()
	now := time.Now()
	hasResults := false

	for _, pf := range projects {
		blockerStates := computeBlockerStates(pf)

		for _, t := range pf.Tasks {
			if shouldIncludeTask(&t, blockerStates, priority, now) {
				hasResults = true
				state := model.ComputeTaskState(&t, blockerStates)
				table.AddRow(
					t.ID,
					formatTaskState(state),
					formatPriority(t.Priority),
					t.Title,
					formatTags(t.Tags),
				)
			}
		}
	}

	if !hasResults {
		fmt.Println("No tasks found.")
		return nil
	}

	table.Render(os.Stdout)
	return nil
}

func shouldIncludeTask(t *model.Task, blockerStates model.BlockerStatus, priority int, now time.Time) bool {
	state := model.ComputeTaskState(t, blockerStates)

	// Status filters
	if listDone {
		if state != model.TaskStateDone {
			return false
		}
	} else if listDropped {
		if state != model.TaskStateDropped {
			return false
		}
	} else if listReady {
		if state != model.TaskStateReady {
			return false
		}
	} else if listBlocked {
		if state != model.TaskStateBlocked {
			return false
		}
	} else if listWaiting {
		if state != model.TaskStateWaiting {
			return false
		}
	} else if !listAll {
		// Default: show only open tasks
		if t.Status != model.TaskStatusOpen {
			return false
		}
	}

	// Priority filter
	if priority > 0 && t.Priority != priority {
		return false
	}

	// Tag filter (AND logic - must have all specified tags)
	if len(listTags) > 0 {
		taskTags := make(map[string]bool)
		for _, tag := range t.Tags {
			taskTags[strings.ToLower(tag)] = true
		}
		for _, requiredTag := range listTags {
			if !taskTags[strings.ToLower(requiredTag)] {
				return false
			}
		}
	}

	// Overdue filter
	if listOverdue {
		if t.DueDate == nil || !t.DueDate.Before(now) {
			return false
		}
	}

	return true
}

func formatTaskState(state model.TaskState) string {
	switch state {
	case model.TaskStateReady:
		return cli.Green("[ready]")
	case model.TaskStateBlocked:
		return cli.Red("[blocked]")
	case model.TaskStateWaiting:
		return cli.Yellow("[waiting]")
	case model.TaskStateDone:
		return cli.Gray("[done]")
	case model.TaskStateDropped:
		return cli.Gray("[dropped]")
	default:
		return fmt.Sprintf("[%s]", state)
	}
}

func formatPriority(p int) string {
	return fmt.Sprintf("P%d", p)
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var formatted []string
	for _, tag := range tags {
		formatted = append(formatted, "["+tag+"]")
	}
	return strings.Join(formatted, " ")
}
