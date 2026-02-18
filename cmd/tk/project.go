package main

import (
	"fmt"
	"strings"

	"github.com/jacksmith/tk/internal/cli"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var projectCmd = &cobra.Command{
	Use:   "project <id>",
	Short: "Show project summary or manage projects",
	Long: `Show a summary of the specified project, or manage projects with subcommands.

The project can be specified by its ID (e.g., "backyard") or prefix (e.g., "BY").

Output shows counts of open tasks (broken down by ready, blocked, waiting),
done tasks, dropped tasks, and waits.

Subcommands:
  new      Create a new project
  edit     Edit an existing project
  delete   Delete a project`,
	Args:              cobra.ExactArgs(1),
	RunE:              runProject,
	ValidArgsFunction: completeProjectIDs,
}

var projectNewCmd = &cobra.Command{
	Use:   "new [id]",
	Short: "Create a new project",
	Long: `Create a new project.

The --prefix and --name flags are required. The positional id argument is
optional; if omitted, the project id is derived from the prefix (lowercased).

Examples:
  tk project new --prefix=BY --name="Backyard Redo"
  tk project new backyard --prefix=BY --name="Backyard Redo"
  tk project new --prefix=EL --name="Electronics" --description="PCB projects"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProjectNew,
}

var projectEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a project",
	Long: `Edit a project's metadata.

Use flags to change specific fields, or -i to edit in $EDITOR.

Examples:
  tk project edit backyard --name="New Name"
  tk project edit backyard --status=paused
  tk project edit backyard --prefix=NW    # triggers ID migration
  tk project edit backyard -i`,
	Args:              cobra.ExactArgs(1),
	RunE:              runProjectEdit,
	ValidArgsFunction: completeProjectIDs,
}

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a project",
	Long: `Delete a project and all its tasks/waits.

The --force flag is required to confirm deletion.

Examples:
  tk project delete backyard --force`,
	Args:              cobra.ExactArgs(1),
	RunE:              runProjectDelete,
	ValidArgsFunction: completeProjectIDs,
}

var (
	projectNewPrefix      string
	projectNewName        string
	projectNewDescription string

	projectEditName        string
	projectEditDescription string
	projectEditStatus      string
	projectEditPrefix      string
	projectEditInteractive bool

	projectDeleteForce bool
)

func init() {
	projectNewCmd.Flags().StringVar(&projectNewPrefix, "prefix", "", "project prefix (2-3 uppercase letters)")
	projectNewCmd.Flags().StringVar(&projectNewName, "name", "", "project display name")
	projectNewCmd.Flags().StringVar(&projectNewDescription, "description", "", "project description")
	projectNewCmd.MarkFlagRequired("prefix")
	projectNewCmd.MarkFlagRequired("name")
	projectCmd.AddCommand(projectNewCmd)

	projectEditCmd.Flags().StringVar(&projectEditName, "name", "", "set project name")
	projectEditCmd.Flags().StringVar(&projectEditDescription, "description", "", "set project description")
	projectEditCmd.Flags().StringVar(&projectEditStatus, "status", "", "set project status (active/paused/done)")
	projectEditCmd.Flags().StringVar(&projectEditPrefix, "prefix", "", "change project prefix (triggers ID migration)")
	projectEditCmd.Flags().BoolVarP(&projectEditInteractive, "interactive", "i", false, "edit in $EDITOR")
	projectCmd.AddCommand(projectEditCmd)

	projectDeleteCmd.Flags().BoolVar(&projectDeleteForce, "force", false, "confirm deletion")
	projectCmd.AddCommand(projectDeleteCmd)

	rootCmd.AddCommand(projectCmd)
}

func runProject(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	summary, err := ops.GetProjectSummary(s, args[0])
	if err != nil {
		return err
	}

	fmt.Printf("%s: %s\n", summary.Project.Prefix, summary.Project.Name)
	if summary.Project.Description != "" {
		fmt.Printf("%s\n", summary.Project.Description)
	}
	fmt.Printf("Status: %s\n", summary.Project.Status)
	fmt.Println()

	if summary.OpenCount > 0 {
		fmt.Printf("%d open", summary.OpenCount)
		var details []string
		if summary.ReadyCount > 0 {
			details = append(details, fmt.Sprintf("%d ready", summary.ReadyCount))
		}
		if summary.BlockedCount > 0 {
			details = append(details, fmt.Sprintf("%d blocked", summary.BlockedCount))
		}
		if summary.WaitingCount > 0 {
			details = append(details, fmt.Sprintf("%d waiting", summary.WaitingCount))
		}
		if len(details) > 0 {
			fmt.Printf(" (%s)", strings.Join(details, ", "))
		}
	} else {
		fmt.Printf("0 open")
	}
	fmt.Printf(", %d done", summary.DoneCount)
	if summary.DroppedCount > 0 {
		fmt.Printf(", %d dropped", summary.DroppedCount)
	}
	fmt.Println()

	if summary.OpenWaits > 0 || summary.DoneWaits > 0 || summary.DroppedWaits > 0 {
		fmt.Printf("%d waits", summary.OpenWaits)
		if summary.DoneWaits > 0 {
			fmt.Printf(" (%d resolved)", summary.DoneWaits)
		}
		fmt.Println()
	}

	return nil
}

func runProjectNew(cmd *cobra.Command, args []string) error {
	projectID := ""
	if len(args) > 0 {
		projectID = args[0]
	}

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if err := ops.CreateProject(s, projectID, projectNewPrefix, projectNewName, projectNewDescription); err != nil {
		return err
	}

	fmt.Printf("Created project %s (%s).\n", projectNewPrefix, projectNewName)
	return nil
}

func runProjectEdit(cmd *cobra.Command, args []string) error {
	projectRef := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	pf, err := ops.ResolveProject(s, projectRef)
	if err != nil {
		return err
	}
	prefix := pf.Prefix

	if projectEditInteractive {
		return runProjectEditInteractive(s, pf)
	}

	if cmd.Flags().Changed("prefix") && projectEditPrefix != prefix {
		if err := ops.ChangeProjectPrefix(s, prefix, projectEditPrefix); err != nil {
			return err
		}
		fmt.Printf("Project prefix changed from %s to %s.\n", prefix, strings.ToUpper(projectEditPrefix))
		prefix = strings.ToUpper(projectEditPrefix)
	}

	changes := ops.ProjectChanges{}
	hasChanges := false

	if cmd.Flags().Changed("name") {
		changes.Name = &projectEditName
		hasChanges = true
	}
	if cmd.Flags().Changed("description") {
		changes.Description = &projectEditDescription
		hasChanges = true
	}
	if cmd.Flags().Changed("status") {
		status := model.ProjectStatus(projectEditStatus)
		switch status {
		case model.ProjectStatusActive, model.ProjectStatusPaused, model.ProjectStatusDone:
			changes.Status = &status
			hasChanges = true
		default:
			return fmt.Errorf("invalid status: %s (expected active/paused/done)", projectEditStatus)
		}
	}

	if hasChanges {
		if err := ops.EditProject(s, prefix, changes); err != nil {
			return err
		}
	}

	if !hasChanges && !cmd.Flags().Changed("prefix") {
		return fmt.Errorf("no changes specified")
	}

	fmt.Printf("Project %s updated.\n", prefix)
	return nil
}

type editableProject struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Status      string `yaml:"status"`
}

func runProjectEditInteractive(s ops.Store, pf *model.ProjectFile) error {
	editable := editableProject{
		Name:        pf.Name,
		Description: pf.Description,
		Status:      string(pf.Status),
	}

	content, err := yaml.Marshal(&editable)
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	header := fmt.Sprintf("# Editing project %s (%s)\n# Note: To change prefix, use --prefix flag instead.\n# Save and close editor to apply changes. Exit without saving to cancel.\n\n", pf.Prefix, pf.ID)
	content = append([]byte(header), content...)

	edited, err := cli.EditInEditor(content, ".yaml")
	if err != nil {
		return err
	}

	var newEditable editableProject
	if err := yaml.Unmarshal(edited, &newEditable); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	changes := ops.ProjectChanges{}
	if newEditable.Name != pf.Name {
		changes.Name = &newEditable.Name
	}
	if newEditable.Description != pf.Description {
		changes.Description = &newEditable.Description
	}
	if newEditable.Status != string(pf.Status) {
		status := model.ProjectStatus(newEditable.Status)
		switch status {
		case model.ProjectStatusActive, model.ProjectStatusPaused, model.ProjectStatusDone:
			changes.Status = &status
		default:
			return fmt.Errorf("invalid status: %s (expected active/paused/done)", newEditable.Status)
		}
	}

	if err := ops.EditProject(s, pf.Prefix, changes); err != nil {
		return err
	}

	fmt.Printf("Project %s updated.\n", pf.Prefix)
	return nil
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	projectRef := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	pf, err := ops.ResolveProject(s, projectRef)
	if err != nil {
		return err
	}

	if !projectDeleteForce {
		var openTasks, openWaits int
		for _, t := range pf.Tasks {
			if t.Status == model.TaskStatusOpen {
				openTasks++
			}
		}
		for _, w := range pf.Waits {
			if w.Status == model.WaitStatusOpen {
				openWaits++
			}
		}

		if openTasks > 0 || openWaits > 0 {
			parts := []string{}
			if openTasks > 0 {
				noun := "task"
				if openTasks != 1 {
					noun = "tasks"
				}
				parts = append(parts, fmt.Sprintf("%d open %s", openTasks, noun))
			}
			if openWaits > 0 {
				noun := "wait"
				if openWaits != 1 {
					noun = "waits"
				}
				parts = append(parts, fmt.Sprintf("%d open %s", openWaits, noun))
			}
			return fmt.Errorf("project %s has %s (use --force to delete anyway)", pf.Prefix, strings.Join(parts, " and "))
		}
	}

	if err := ops.DeleteProject(s, pf.Prefix, projectDeleteForce); err != nil {
		return err
	}

	fmt.Printf("Deleted project %s.\n", pf.Prefix)
	return nil
}
