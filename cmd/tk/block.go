package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jacksmith/tk/internal/cli"
	"github.com/jacksmith/tk/internal/graph"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var blockCmd = &cobra.Command{
	Use:   "block <id>",
	Short: "Add a blocker to a task",
	Long: `Add a blocker (task or wait) to a task.

The blocker must be in the same project. Adding a blocker that
would create a dependency cycle is not allowed.

Examples:
  tk block BY-07 --by=BY-05
  tk block BY-07 --by=BY-03W`,
	Args: cobra.ExactArgs(1),
	RunE: runBlock,
}

var unblockCmd = &cobra.Command{
	Use:   "unblock <id>",
	Short: "Remove a blocker from a task",
	Long: `Remove a blocker from a task.

Examples:
  tk unblock BY-07 --from=BY-05
  tk unblock BY-07 --from=BY-03W`,
	Args: cobra.ExactArgs(1),
	RunE: runUnblock,
}

var blockedByCmd = &cobra.Command{
	Use:   "blocked-by <id>",
	Short: "Show what is blocking an item",
	Long: `Show all direct blockers of a task or wait.

Examples:
  tk blocked-by BY-07`,
	Args: cobra.ExactArgs(1),
	RunE: runBlockedBy,
}

var blockingCmd = &cobra.Command{
	Use:   "blocking <id>",
	Short: "Show what an item is blocking",
	Long: `Show all items directly blocked by a task or wait.

Examples:
  tk blocking BY-07`,
	Args: cobra.ExactArgs(1),
	RunE: runBlocking,
}

var (
	blockBy     string
	unblockFrom string
)

func init() {
	blockCmd.Flags().StringVar(&blockBy, "by", "", "blocker ID (task or wait)")
	blockCmd.MarkFlagRequired("by")
	rootCmd.AddCommand(blockCmd)

	unblockCmd.Flags().StringVar(&unblockFrom, "from", "", "blocker ID to remove")
	unblockCmd.MarkFlagRequired("from")
	rootCmd.AddCommand(unblockCmd)

	rootCmd.AddCommand(blockedByCmd)
	rootCmd.AddCommand(blockingCmd)
}

func runBlock(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if err := ops.AddBlocker(s, taskID, blockBy); err != nil {
		return err
	}

	fmt.Printf("%s is now blocked by %s.\n", taskID, blockBy)
	return nil
}

func runUnblock(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if err := ops.RemoveBlocker(s, taskID, unblockFrom); err != nil {
		return err
	}

	fmt.Printf("%s is no longer blocked by %s.\n", taskID, unblockFrom)
	return nil
}

func runBlockedBy(cmd *cobra.Command, args []string) error {
	id := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	prefix := model.ExtractPrefix(id)
	if prefix == "" {
		return fmt.Errorf("invalid ID format: %s", id)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	// Find the item's blockers
	var blockers []string
	if model.IsWaitID(id) {
		for _, w := range pf.Waits {
			if strings.EqualFold(w.ID, id) {
				blockers = w.BlockedBy
				break
			}
		}
	} else {
		for _, t := range pf.Tasks {
			if strings.EqualFold(t.ID, id) {
				blockers = t.BlockedBy
				break
			}
		}
	}

	if len(blockers) == 0 {
		fmt.Printf("%s has no blockers.\n", id)
		return nil
	}

	table := cli.NewTable()
	for _, blockerID := range blockers {
		info := getBlockerInfo(pf, blockerID)
		table.AddRow(info)
	}
	table.Render(os.Stdout)
	return nil
}

func runBlocking(cmd *cobra.Command, args []string) error {
	id := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	prefix := model.ExtractPrefix(id)
	if prefix == "" {
		return fmt.Errorf("invalid ID format: %s", id)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	g := graph.BuildGraph(pf)
	blocking := g.Blocking(id)

	if len(blocking) == 0 {
		fmt.Printf("%s is not blocking anything.\n", id)
		return nil
	}

	table := cli.NewTable()
	for _, blockedID := range blocking {
		info := getBlockerInfo(pf, blockedID)
		table.AddRow(info)
	}
	table.Render(os.Stdout)
	return nil
}
