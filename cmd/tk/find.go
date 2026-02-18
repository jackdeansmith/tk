package main

import (
	"fmt"
	"os"

	"github.com/jacksmith/tk/internal/cli"
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
	findCmd.RegisterFlagCompletionFunc("project", completeProjectIDs)
	rootCmd.AddCommand(findCmd)
}

func runFind(cmd *cobra.Command, args []string) error {
	query := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	ops.AutoCheck(s)

	result, err := ops.FindItems(s, query, findProject)
	if err != nil {
		return err
	}

	if len(result.Tasks) == 0 && len(result.Waits) == 0 {
		fmt.Printf("No results found for %q\n", query)
		return nil
	}

	if len(result.Tasks) > 0 {
		fmt.Println("Tasks:")
		table := cli.NewTable()
		table.SetMaxWidth(2, cli.DefaultMaxTitleWidth)
		for _, m := range result.Tasks {
			table.AddRow(m.Task.ID, formatTaskState(m.State), m.Task.Title)
		}
		table.Render(os.Stdout)
	}

	if len(result.Waits) > 0 {
		if len(result.Tasks) > 0 {
			fmt.Println()
		}
		fmt.Println("Waits:")
		table := cli.NewTable()
		for _, m := range result.Waits {
			table.AddRow(m.Wait.ID, m.Wait.DisplayText())
		}
		table.Render(os.Stdout)
	}

	return nil
}
