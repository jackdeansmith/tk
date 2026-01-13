package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jacksmith/tk/internal/cli"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var waitsCmd = &cobra.Command{
	Use:   "waits",
	Short: "List waits",
	Long: `List waits with optional filtering.

By default, lists all open waits in active projects.
Auto-runs 'tk check' to resolve any time-based waits that have passed.

Filter flags:
  --actionable  Show only waits ready for user action (manual waits, past check_after)
  --dormant     Show only waits blocked by incomplete items
  --done        Show only resolved waits
  --dropped     Show only dropped waits
  --all         Show all waits regardless of status

  -p, --project Limit to a specific project (by prefix or ID)

Waits are sorted by ID.`,
	RunE: runWaits,
}

var (
	waitsProject    string
	waitsActionable bool
	waitsDormant    bool
	waitsDone       bool
	waitsDropped    bool
	waitsAll        bool
)

func init() {
	waitsCmd.Flags().StringVarP(&waitsProject, "project", "p", "", "filter by project (prefix or ID)")
	waitsCmd.Flags().BoolVar(&waitsActionable, "actionable", false, "show only actionable waits")
	waitsCmd.Flags().BoolVar(&waitsDormant, "dormant", false, "show only dormant waits")
	waitsCmd.Flags().BoolVar(&waitsDone, "done", false, "show only done waits")
	waitsCmd.Flags().BoolVar(&waitsDropped, "dropped", false, "show only dropped waits")
	waitsCmd.Flags().BoolVar(&waitsAll, "all", false, "show all waits")

	// Register completion function
	waitsCmd.RegisterFlagCompletionFunc("project", completeProjectIDs)

	rootCmd.AddCommand(waitsCmd)
}

func runWaits(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	// waits command always runs check (per spec), regardless of autocheck config
	checkResult, _ := ops.RunCheck(s)

	// Show check results if any waits were resolved
	if checkResult != nil && len(checkResult.ResolvedWaits) > 0 {
		for _, wid := range checkResult.ResolvedWaits {
			fmt.Printf("Auto-resolved: %s\n", wid)
		}
		fmt.Println()
	}

	now := time.Now()

	// Determine which projects to include
	var projects []*model.ProjectFile

	if waitsProject != "" {
		// Single project specified
		pf, err := s.LoadProject(waitsProject)
		if err != nil {
			pf, err = s.LoadProjectByID(waitsProject)
			if err != nil {
				return fmt.Errorf("project %q not found", waitsProject)
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
			if !waitsAll && pf.Status != model.ProjectStatusActive {
				continue
			}
			projects = append(projects, pf)
		}
	}

	// Collect all waits with filtering
	table := cli.NewTable()
	hasResults := false

	for _, pf := range projects {
		blockerStates := computeBlockerStates(pf)

		for _, w := range pf.Waits {
			if shouldIncludeWait(&w, blockerStates, now) {
				hasResults = true
				state := model.ComputeWaitState(&w, blockerStates, now)
				table.AddRow(
					w.ID,
					formatWaitState(state),
					w.DisplayText(),
				)
			}
		}
	}

	if !hasResults {
		fmt.Println("No waits found.")
		return nil
	}

	table.Render(os.Stdout)
	return nil
}

func shouldIncludeWait(w *model.Wait, blockerStates model.BlockerStatus, now time.Time) bool {
	state := model.ComputeWaitState(w, blockerStates, now)

	// Status filters
	if waitsDone {
		return state == model.WaitStateDone
	}
	if waitsDropped {
		return state == model.WaitStateDropped
	}
	if waitsActionable {
		return state == model.WaitStateActionable
	}
	if waitsDormant {
		return state == model.WaitStateDormant
	}
	if waitsAll {
		return true
	}

	// Default: show only open waits (not done/dropped)
	return w.Status == model.WaitStatusOpen
}

func formatWaitState(state model.WaitState) string {
	switch state {
	case model.WaitStateActionable:
		return cli.Green("[actionable]")
	case model.WaitStatePending:
		return cli.Yellow("[pending]")
	case model.WaitStateDormant:
		return cli.Red("[dormant]")
	case model.WaitStateDone:
		return cli.Gray("[done]")
	case model.WaitStateDropped:
		return cli.Gray("[dropped]")
	default:
		return fmt.Sprintf("[%s]", state)
	}
}
