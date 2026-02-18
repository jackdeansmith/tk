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
	waitsCmd.RegisterFlagCompletionFunc("project", completeProjectIDs)
	rootCmd.AddCommand(waitsCmd)
}

func runWaits(cmd *cobra.Command, args []string) error {
	if err := validateWaitsStatusFilters(); err != nil {
		return err
	}

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	// waits command always runs check (per spec), regardless of autocheck config
	checkResult, _ := ops.RunCheck(s)
	if checkResult != nil && len(checkResult.ResolvedWaits) > 0 {
		for _, wid := range checkResult.ResolvedWaits {
			fmt.Printf("Auto-resolved: %s\n", wid)
		}
		fmt.Println()
	}

	filter := ops.WaitFilter{
		Project: waitsProject,
		All:     waitsAll,
	}
	if state := resolveWaitStateFilter(); state != nil {
		filter.State = state
	}

	results, err := ops.ListWaits(s, filter)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("No waits found.")
		return nil
	}

	table := cli.NewTable()
	for _, r := range results {
		table.AddRow(r.Wait.ID, formatWaitState(r.State), r.Wait.DisplayText())
	}
	table.Render(os.Stdout)
	return nil
}

func resolveWaitStateFilter() *model.WaitState {
	var state model.WaitState
	switch {
	case waitsActionable:
		state = model.WaitStateActionable
	case waitsDormant:
		state = model.WaitStateDormant
	case waitsDone:
		state = model.WaitStateDone
	case waitsDropped:
		state = model.WaitStateDropped
	default:
		return nil
	}
	return &state
}

func validateWaitsStatusFilters() error {
	var active []string
	if waitsActionable {
		active = append(active, "--actionable")
	}
	if waitsDormant {
		active = append(active, "--dormant")
	}
	if waitsDone {
		active = append(active, "--done")
	}
	if waitsDropped {
		active = append(active, "--dropped")
	}
	if waitsAll {
		active = append(active, "--all")
	}

	if len(active) > 1 {
		return fmt.Errorf("conflicting status filters: %s (use only one at a time)", strings.Join(active, ", "))
	}
	return nil
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
