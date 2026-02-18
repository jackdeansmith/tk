package main

import (
	"fmt"

	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <id>",
	Short: "Move a task to a different project",
	Long: `Move a task to a different project.

The task cannot have blockers or dependents in the source project.
The task will get a new ID in the destination project.

Examples:
  tk move BY-07 --to=HH
  tk move BY-07 --to=household`,
	Args:              cobra.ExactArgs(1),
	RunE:              runMove,
	ValidArgsFunction: completeTaskIDs,
}

var moveTo string

func init() {
	moveCmd.Flags().StringVar(&moveTo, "to", "", "destination project prefix or ID")
	moveCmd.MarkFlagRequired("to")
	moveCmd.RegisterFlagCompletionFunc("to", completeProjectIDs)
	rootCmd.AddCommand(moveCmd)
}

func runMove(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	destPf, err := ops.ResolveProject(s, moveTo)
	if err != nil {
		return fmt.Errorf("destination project %q not found", moveTo)
	}

	if err := ops.MoveTask(s, taskID, destPf.Prefix); err != nil {
		return err
	}

	fmt.Printf("%s moved to project %s.\n", taskID, destPf.Prefix)
	return nil
}
