package main

import (
	"fmt"
	"os"

	"github.com/jacksmith/tk/internal/cli"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List all projects",
	Long: `List all projects in the tk repository.

By default, only active projects are shown.
Use --all to include paused and done projects.`,
	RunE: runProjects,
}

var projectsAll bool

func init() {
	projectsCmd.Flags().BoolVar(&projectsAll, "all", false, "include paused and done projects")
	rootCmd.AddCommand(projectsCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	prefixes, err := s.ListProjects()
	if err != nil {
		return err
	}

	if len(prefixes) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	table := cli.NewTable()

	for _, prefix := range prefixes {
		pf, err := s.LoadProject(prefix)
		if err != nil {
			// Skip projects that can't be loaded
			continue
		}

		// Filter by status unless --all is specified
		if !projectsAll && pf.Status != model.ProjectStatusActive {
			continue
		}

		// Format status with color
		statusStr := formatProjectStatus(pf.Status)

		table.AddRow(pf.Prefix, pf.Name, statusStr)
	}

	table.Render(os.Stdout)
	return nil
}

func formatProjectStatus(status model.ProjectStatus) string {
	switch status {
	case model.ProjectStatusActive:
		return cli.Green(string(status))
	case model.ProjectStatusPaused:
		return cli.Yellow(string(status))
	case model.ProjectStatusDone:
		return cli.Gray(string(status))
	default:
		return string(status)
	}
}
