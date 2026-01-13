package main

import (
	"fmt"
	"strings"

	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Auto-resolve time-based waits",
	Long: `Run auto-resolution for time-based waits that have passed their 'after' date.

This command:
1. Finds all time waits where 'after' has passed
2. Marks them as done
3. Reports cascading effects (unblocked items, auto-completed tasks)

This is automatically run by 'tk waits' and optionally by other read commands
when autocheck is enabled in .tkconfig.yaml.

Examples:
  tk check`,
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	result, err := ops.RunCheck(s)
	if err != nil {
		return err
	}

	if len(result.ResolvedWaits) == 0 && len(result.Unblocked) == 0 && len(result.AutoCompleted) == 0 {
		fmt.Println("No time waits ready to resolve.")
		return nil
	}

	if len(result.ResolvedWaits) > 0 {
		fmt.Printf("Resolved waits: %s\n", strings.Join(result.ResolvedWaits, ", "))
	}

	if len(result.Unblocked) > 0 {
		fmt.Printf("Unblocked: %s\n", strings.Join(result.Unblocked, ", "))
	}

	if len(result.AutoCompleted) > 0 {
		fmt.Printf("Auto-completed: %s\n", strings.Join(result.AutoCompleted, ", "))
	}

	return nil
}
