// Package main is the entry point for the tk CLI.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "tk",
	Short: "tk - a task tracker with first-class dependency support",
	Long: `tk is a personal task management system with first-class support for
external blockers. It helps you track tasks, their dependencies, and
things you're waiting on.

Tasks live in projects and can be blocked by other tasks or waits.
Waits represent external conditions outside your control.`,
	Version: Version,
	// Show help when no subcommand is provided
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Disable the default completion command (we'll add our own in Phase 9)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Set version template
	rootCmd.SetVersionTemplate("tk version {{.Version}}\n")
}
