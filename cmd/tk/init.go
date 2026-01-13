package main

import (
	"fmt"

	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new tk repository",
	Long: `Create a .tk/ directory with a default project.

This creates the necessary directory structure for tk to manage tasks.
If --name and --prefix are not specified, creates a project called "Default"
with prefix "DF".

Fails if .tk/ already exists in the current directory.`,
	RunE: runInit,
}

var (
	initName   string
	initPrefix string
)

func init() {
	initCmd.Flags().StringVar(&initName, "name", "", "name for the default project")
	initCmd.Flags().StringVar(&initPrefix, "prefix", "", "prefix for the default project (2-3 uppercase letters)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	s, err := storage.Init(".", initName, initPrefix)
	if err != nil {
		return err
	}

	// Load the project to get the actual values used
	prefixes, err := s.ListProjects()
	if err != nil {
		return err
	}

	if len(prefixes) > 0 {
		pf, err := s.LoadProject(prefixes[0])
		if err != nil {
			return err
		}
		fmt.Printf("Initialized tk in .tk/\n")
		fmt.Printf("Created project %q with prefix %s\n", pf.Name, pf.Prefix)
	} else {
		fmt.Printf("Initialized tk in .tk/\n")
	}

	return nil
}
