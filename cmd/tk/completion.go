package main

import (
	"os"
	"strings"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for tk.

To load completions:

Bash:
  $ source <(tk completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ tk completion bash > /etc/bash_completion.d/tk
  # macOS:
  $ tk completion bash > $(brew --prefix)/etc/bash_completion.d/tk

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ tk completion zsh > "${fpath[1]}/_tk"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ tk completion fish | source
  # To load completions for each session, execute once:
  $ tk completion fish > ~/.config/fish/completions/tk.fish
`,
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	Long:  "Generate the autocompletion script for bash.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletion(os.Stdout)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	Long:  "Generate the autocompletion script for zsh.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	Long:  "Generate the autocompletion script for fish.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(os.Stdout, true)
	},
}

func init() {
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)
	rootCmd.AddCommand(completionCmd)
}

// completeProjectIDs returns a completion function for project prefixes and IDs.
func completeProjectIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	s, err := storage.Open(".")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prefixes, err := s.ListProjects()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	toCompleteLower := strings.ToLower(toComplete)

	for _, prefix := range prefixes {
		pf, err := s.LoadProject(prefix)
		if err != nil {
			continue
		}

		// Add both prefix and project ID as completions
		if strings.HasPrefix(strings.ToLower(prefix), toCompleteLower) {
			completions = append(completions, prefix+"\t"+pf.Name)
		}
		if strings.HasPrefix(strings.ToLower(pf.ID), toCompleteLower) {
			completions = append(completions, pf.ID+"\t"+pf.Name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeTaskIDs returns a completion function for task IDs in active projects.
func completeTaskIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeIDs(false, true, toComplete)
}

// completeWaitIDs returns a completion function for wait IDs in active projects.
func completeWaitIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeIDs(true, false, toComplete)
}

// completeAnyIDs returns a completion function for both task and wait IDs.
func completeAnyIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return completeIDs(true, true, toComplete)
}

// completeIDs is a helper that returns task and/or wait IDs.
func completeIDs(includeWaits, includeTasks bool, toComplete string) ([]string, cobra.ShellCompDirective) {
	s, err := storage.Open(".")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prefixes, err := s.ListProjects()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	toCompleteLower := strings.ToLower(toComplete)

	for _, prefix := range prefixes {
		pf, err := s.LoadProject(prefix)
		if err != nil {
			continue
		}

		// Only include active projects
		if pf.Status != model.ProjectStatusActive {
			continue
		}

		if includeTasks {
			for _, t := range pf.Tasks {
				if strings.HasPrefix(strings.ToLower(t.ID), toCompleteLower) {
					// Include status in completion description
					status := string(t.Status)
					completions = append(completions, t.ID+"\t"+status+": "+truncate(t.Title, 40))
				}
			}
		}

		if includeWaits {
			for _, w := range pf.Waits {
				if strings.HasPrefix(strings.ToLower(w.ID), toCompleteLower) {
					status := string(w.Status)
					completions = append(completions, w.ID+"\t"+status+": "+truncate(w.DisplayText(), 40))
				}
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeTags returns a completion function for tags across all projects.
func completeTags(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	s, err := storage.Open(".")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prefixes, err := s.ListProjects()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Collect unique tags
	tagSet := make(map[string]bool)
	toCompleteLower := strings.ToLower(toComplete)

	for _, prefix := range prefixes {
		pf, err := s.LoadProject(prefix)
		if err != nil {
			continue
		}

		for _, t := range pf.Tasks {
			for _, tag := range t.Tags {
				if strings.HasPrefix(strings.ToLower(tag), toCompleteLower) {
					tagSet[tag] = true
				}
			}
		}
	}

	var completions []string
	for tag := range tagSet {
		completions = append(completions, tag)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeTaskIDsThenTags completes task IDs for the first argument and tags for the second.
func completeTaskIDsThenTags(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completeTaskIDs(cmd, args, toComplete)
	}
	return completeTags(cmd, args, toComplete)
}

// truncate shortens a string to the given length, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
