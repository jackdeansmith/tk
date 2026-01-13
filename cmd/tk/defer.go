package main

import (
	"fmt"
	"time"

	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var deferCmd = &cobra.Command{
	Use:   "defer <id>",
	Short: "Defer a task",
	Long: `Defer a task by creating a time wait and linking it.

The task must be open and cannot already have open waits.
Either --days or --until must be specified.

Examples:
  tk defer BY-07 --days=4
  tk defer BY-07 --until=2026-01-20`,
	Args: cobra.ExactArgs(1),
	RunE: runDefer,
}

var (
	deferDays  int
	deferUntil string
)

func init() {
	deferCmd.Flags().IntVar(&deferDays, "days", 0, "defer for N days")
	deferCmd.Flags().StringVar(&deferUntil, "until", "", "defer until date (YYYY-MM-DD)")
	rootCmd.AddCommand(deferCmd)
}

func runDefer(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	if deferDays == 0 && deferUntil == "" {
		return fmt.Errorf("either --days or --until must be specified")
	}

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	// Calculate target time
	var until time.Time
	if deferDays > 0 {
		// End of day N days from now
		target := time.Now().AddDate(0, 0, deferDays)
		until = time.Date(target.Year(), target.Month(), target.Day(), 23, 59, 59, 0, target.Location())
	} else {
		// Parse the date
		t, err := time.Parse("2006-01-02", deferUntil)
		if err != nil {
			return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %v", err)
		}
		// End of that day
		until = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
	}

	wait, err := ops.DeferTask(s, taskID, until)
	if err != nil {
		return err
	}

	fmt.Printf("%s deferred until %s.\n", taskID, until.Format("2006-01-02"))
	fmt.Printf("Created wait %s.\n", wait.ID)
	return nil
}
