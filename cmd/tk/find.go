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

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search tasks and waits",
	Long: `Search for tasks and waits by keyword.

Performs a case-insensitive substring search in:
- Task titles
- Task notes
- Wait titles
- Wait questions

Use -p/--project to limit search to a specific project.

Results are grouped by type (Tasks, Waits) and show ID and matching text.`,
	Args: cobra.ExactArgs(1),
	RunE: runFind,
}

var findProject string

func init() {
	findCmd.Flags().StringVarP(&findProject, "project", "p", "", "limit search to project (prefix or ID)")
	rootCmd.AddCommand(findCmd)
}

func runFind(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(args[0])

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

	// Determine which projects to search
	var projects []*model.ProjectFile

	if findProject != "" {
		pf, err := s.LoadProject(findProject)
		if err != nil {
			pf, err = s.LoadProjectByID(findProject)
			if err != nil {
				return fmt.Errorf("project %q not found", findProject)
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
			if pf.Status == model.ProjectStatusActive {
				projects = append(projects, pf)
			}
		}
	}

	// Search for matches
	var taskMatches []taskMatch
	var waitMatches []waitMatch

	for _, pf := range projects {
		blockerStates := computeBlockerStates(pf)

		// Search tasks
		for _, t := range pf.Tasks {
			if matchesTask(&t, query) {
				state := model.ComputeTaskState(&t, blockerStates)
				taskMatches = append(taskMatches, taskMatch{task: t, state: state})
			}
		}

		// Search waits
		for _, w := range pf.Waits {
			if matchesWait(&w, query) {
				waitMatches = append(waitMatches, waitMatch{wait: w})
			}
		}
	}

	// Print results
	if len(taskMatches) == 0 && len(waitMatches) == 0 {
		fmt.Printf("No results found for %q\n", args[0])
		return nil
	}

	if len(taskMatches) > 0 {
		fmt.Println("Tasks:")
		table := cli.NewTable()
		for _, m := range taskMatches {
			table.AddRow(
				m.task.ID,
				formatTaskState(m.state),
				m.task.Title,
			)
		}
		table.Render(os.Stdout)
	}

	if len(waitMatches) > 0 {
		if len(taskMatches) > 0 {
			fmt.Println()
		}
		fmt.Println("Waits:")
		table := cli.NewTable()
		for _, m := range waitMatches {
			table.AddRow(
				m.wait.ID,
				m.wait.DisplayText(),
			)
		}
		table.Render(os.Stdout)
	}

	return nil
}

type taskMatch struct {
	task  model.Task
	state model.TaskState
}

type waitMatch struct {
	wait model.Wait
}

func matchesTask(t *model.Task, query string) bool {
	// Search in title
	if strings.Contains(strings.ToLower(t.Title), query) {
		return true
	}
	// Search in notes
	if strings.Contains(strings.ToLower(t.Notes), query) {
		return true
	}
	return false
}

func matchesWait(w *model.Wait, query string) bool {
	// Search in title
	if strings.Contains(strings.ToLower(w.Title), query) {
		return true
	}
	// Search in question
	if strings.Contains(strings.ToLower(w.ResolutionCriteria.Question), query) {
		return true
	}
	// Search in notes
	if strings.Contains(strings.ToLower(w.Notes), query) {
		return true
	}
	return false
}
