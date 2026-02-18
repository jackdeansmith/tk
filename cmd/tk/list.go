package main

import (
	"fmt"
	"os"
	"strings"

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
	if err := validateListStatusFilters(); err != nil {
		return err
	}

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	ops.AutoCheck(s)

	// Build filter from flags
	filter := ops.TaskFilter{
		Project:  listProject,
		All:      listAll,
		Priority: resolvePriorityShorthand(listPriority, listP1, listP2, listP3, listP4),
		Tags:     listTags,
		Overdue:  listOverdue,
	}
	if state := resolveTaskStateFilter(); state != nil {
		filter.State = state
	}

	results, err := ops.ListTasks(s, filter)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	table := cli.NewTable()
	table.SetMaxWidth(3, cli.DefaultMaxTitleWidth)
	for _, r := range results {
		table.AddRow(
			r.Task.ID,
			formatTaskState(r.State),
			formatPriority(r.Task.Priority),
			r.Task.Title,
			formatTags(r.Task.Tags),
		)
	}
	table.Render(os.Stdout)
	return nil
}

// resolveTaskStateFilter maps the boolean status flags to a *model.TaskState.
func resolveTaskStateFilter() *model.TaskState {
	var state model.TaskState
	switch {
	case listReady:
		state = model.TaskStateReady
	case listBlocked:
		state = model.TaskStateBlocked
	case listWaiting:
		state = model.TaskStateWaiting
	case listDone:
		state = model.TaskStateDone
	case listDropped:
		state = model.TaskStateDropped
	default:
		return nil
	}
	return &state
}

func validateListStatusFilters() error {
	var active []string
	if listReady {
		active = append(active, "--ready")
	}
	if listBlocked {
		active = append(active, "--blocked")
	}
	if listWaiting {
		active = append(active, "--waiting")
	}
	if listDone {
		active = append(active, "--done")
	}
	if listDropped {
		active = append(active, "--dropped")
	}
	if listAll {
		active = append(active, "--all")
	}

	if len(active) > 1 {
		return fmt.Errorf("conflicting status filters: %s (use only one at a time)", strings.Join(active, ", "))
	}
	return nil
}

// resolvePriorityShorthand resolves --p1/--p2/--p3/--p4 flags into a priority int.
func resolvePriorityShorthand(priority int, p1, p2, p3, p4 bool) int {
	switch {
	case p1:
		return 1
	case p2:
		return 2
	case p3:
		return 3
	case p4:
		return 4
	default:
		return priority
	}
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
