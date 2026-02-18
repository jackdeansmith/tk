package main

import (
	"fmt"
	"os"

	"github.com/jacksmith/tk/internal/cli"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
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

	infos, err := ops.ListProjectInfos(s, projectsAll)
	if err != nil {
		return err
	}

	if len(infos) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	table := cli.NewTable()
	for _, info := range infos {
		table.AddRow(info.Prefix, info.Name, formatProjectStatus(info.Status))
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
