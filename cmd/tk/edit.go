package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/cli"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a task",
	Long: `Edit a task's fields.

Use flags to change specific fields, or -i to edit in $EDITOR.

Examples:
  tk edit BY-07 --title="New title"
  tk edit BY-07 --priority=2
  tk edit BY-07 --notes="Additional context"
  tk edit BY-07 --tags=weekend,hardscape    # replaces all tags
  tk edit BY-07 --add-tag=urgent            # adds tag
  tk edit BY-07 --remove-tag=weekend        # removes tag
  tk edit BY-07 --blocked-by=BY-05,BY-06    # replaces blockers
  tk edit BY-07 --add-blocked-by=BY-08      # adds blocker
  tk edit BY-07 --remove-blocked-by=BY-05   # removes blocker
  tk edit BY-07 -i                          # open in $EDITOR`,
	Args:              cobra.ExactArgs(1),
	RunE:              runEdit,
	ValidArgsFunction: completeTaskIDs,
}

var (
	editTitle          string
	editPriority       int
	editP1             bool
	editP2             bool
	editP3             bool
	editP4             bool
	editNotes          string
	editAssignee       string
	editDueDate        string
	editClearDueDate   bool
	editAutoComplete   string // "true", "false", or ""
	editTags           string
	editAddTag         []string
	editRemoveTag      []string
	editBlockedBy      string
	editAddBlockedBy   []string
	editRemoveBlockedBy []string
	editInteractive    bool
)

func init() {
	editCmd.Flags().StringVar(&editTitle, "title", "", "set task title")
	editCmd.Flags().IntVar(&editPriority, "priority", 0, "set task priority (1-4)")
	editCmd.Flags().BoolVar(&editP1, "p1", false, "shorthand for --priority=1")
	editCmd.Flags().BoolVar(&editP2, "p2", false, "shorthand for --priority=2")
	editCmd.Flags().BoolVar(&editP3, "p3", false, "shorthand for --priority=3")
	editCmd.Flags().BoolVar(&editP4, "p4", false, "shorthand for --priority=4")
	editCmd.Flags().StringVar(&editNotes, "notes", "", "set task notes")
	editCmd.Flags().StringVar(&editAssignee, "assignee", "", "set task assignee")
	editCmd.Flags().StringVar(&editDueDate, "due-date", "", "set due date (YYYY-MM-DD)")
	editCmd.Flags().BoolVar(&editClearDueDate, "clear-due-date", false, "clear due date")
	editCmd.Flags().StringVar(&editAutoComplete, "auto-complete", "", "set auto-complete (true/false)")
	editCmd.Flags().StringVar(&editTags, "tags", "", "replace all tags (comma-separated)")
	editCmd.Flags().StringArrayVar(&editAddTag, "add-tag", nil, "add a tag")
	editCmd.Flags().StringArrayVar(&editRemoveTag, "remove-tag", nil, "remove a tag")
	editCmd.Flags().StringVar(&editBlockedBy, "blocked-by", "", "replace blockers (comma-separated)")
	editCmd.Flags().StringArrayVar(&editAddBlockedBy, "add-blocked-by", nil, "add a blocker")
	editCmd.Flags().StringArrayVar(&editRemoveBlockedBy, "remove-blocked-by", nil, "remove a blocker")
	editCmd.Flags().BoolVarP(&editInteractive, "interactive", "i", false, "edit in $EDITOR")

	// Register completion functions
	editCmd.RegisterFlagCompletionFunc("add-tag", completeTags)
	editCmd.RegisterFlagCompletionFunc("remove-tag", completeTags)
	editCmd.RegisterFlagCompletionFunc("add-blocked-by", completeAnyIDs)
	editCmd.RegisterFlagCompletionFunc("remove-blocked-by", completeAnyIDs)

	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if editInteractive {
		return runEditInteractive(s, taskID)
	}

	// Build changes from flags
	changes := ops.TaskChanges{}
	hasChanges := false

	if cmd.Flags().Changed("title") {
		changes.Title = &editTitle
		hasChanges = true
	}

	// Resolve priority shorthand
	if editP1 {
		editPriority = 1
	} else if editP2 {
		editPriority = 2
	} else if editP3 {
		editPriority = 3
	} else if editP4 {
		editPriority = 4
	}

	if cmd.Flags().Changed("priority") || editP1 || editP2 || editP3 || editP4 {
		if err := ops.ValidatePriority(editPriority); err != nil {
			return err
		}
		changes.Priority = &editPriority
		hasChanges = true
	}

	if cmd.Flags().Changed("notes") {
		changes.Notes = &editNotes
		hasChanges = true
	}

	if cmd.Flags().Changed("assignee") {
		changes.Assignee = &editAssignee
		hasChanges = true
	}

	if editClearDueDate {
		var nilTime *time.Time
		changes.DueDate = &nilTime
		hasChanges = true
	} else if editDueDate != "" {
		t, err := time.Parse("2006-01-02", editDueDate)
		if err != nil {
			return fmt.Errorf("invalid due date format (expected YYYY-MM-DD): %v", err)
		}
		tPtr := &t
		changes.DueDate = &tPtr
		hasChanges = true
	}

	if editAutoComplete != "" {
		switch strings.ToLower(editAutoComplete) {
		case "true", "yes", "1":
			val := true
			changes.AutoComplete = &val
		case "false", "no", "0":
			val := false
			changes.AutoComplete = &val
		default:
			return fmt.Errorf("invalid --auto-complete value: %s (expected true/false)", editAutoComplete)
		}
		hasChanges = true
	}

	// Handle tags
	if err := handleTagChanges(s, taskID, &changes, cmd, &hasChanges); err != nil {
		return err
	}

	// Handle blockers
	if err := handleBlockerChanges(s, taskID, &changes, cmd, &hasChanges); err != nil {
		return err
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	if err := ops.EditTask(s, taskID, changes); err != nil {
		return err
	}

	fmt.Printf("%s updated.\n", taskID)
	return nil
}

func handleTagChanges(s *storage.Storage, taskID string, changes *ops.TaskChanges, cmd *cobra.Command, hasChanges *bool) error {
	// If --tags is set, it replaces all tags
	if cmd.Flags().Changed("tags") {
		var tags []string
		if editTags != "" {
			for _, tag := range strings.Split(editTags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}
		changes.Tags = &tags
		*hasChanges = true
		return nil
	}

	// Handle --add-tag and --remove-tag
	if len(editAddTag) > 0 || len(editRemoveTag) > 0 {
		// Load current task to get existing tags
		prefix := model.ExtractPrefix(taskID)
		pf, err := s.LoadProject(prefix)
		if err != nil {
			return err
		}

		var task *model.Task
		for i := range pf.Tasks {
			if strings.EqualFold(pf.Tasks[i].ID, taskID) {
				task = &pf.Tasks[i]
				break
			}
		}
		if task == nil {
			return fmt.Errorf("task %s not found", taskID)
		}

		// Build new tag list
		tagSet := make(map[string]bool)
		for _, tag := range task.Tags {
			tagSet[strings.ToLower(tag)] = true
		}

		// Add new tags
		for _, tag := range editAddTag {
			tagSet[strings.ToLower(strings.TrimSpace(tag))] = true
		}

		// Remove tags
		for _, tag := range editRemoveTag {
			delete(tagSet, strings.ToLower(strings.TrimSpace(tag)))
		}

		// Convert back to slice
		var newTags []string
		for tag := range tagSet {
			newTags = append(newTags, tag)
		}

		changes.Tags = &newTags
		*hasChanges = true
	}

	return nil
}

func handleBlockerChanges(s *storage.Storage, taskID string, changes *ops.TaskChanges, cmd *cobra.Command, hasChanges *bool) error {
	// If --blocked-by is set, it replaces all blockers
	if cmd.Flags().Changed("blocked-by") {
		var blockers []string
		if editBlockedBy != "" {
			blockers = strings.Split(editBlockedBy, ",")
			for i, b := range blockers {
				blockers[i] = strings.TrimSpace(b)
			}
		}
		changes.BlockedBy = &blockers
		*hasChanges = true
		return nil
	}

	// Handle --add-blocked-by and --remove-blocked-by
	if len(editAddBlockedBy) > 0 || len(editRemoveBlockedBy) > 0 {
		// Load current task to get existing blockers
		prefix := model.ExtractPrefix(taskID)
		pf, err := s.LoadProject(prefix)
		if err != nil {
			return err
		}

		var task *model.Task
		for i := range pf.Tasks {
			if strings.EqualFold(pf.Tasks[i].ID, taskID) {
				task = &pf.Tasks[i]
				break
			}
		}
		if task == nil {
			return fmt.Errorf("task %s not found", taskID)
		}

		// Build new blocker list
		blockerSet := make(map[string]bool)
		for _, b := range task.BlockedBy {
			blockerSet[strings.ToUpper(b)] = true
		}

		// Add new blockers
		for _, b := range editAddBlockedBy {
			blockerSet[strings.ToUpper(strings.TrimSpace(b))] = true
		}

		// Remove blockers
		for _, b := range editRemoveBlockedBy {
			delete(blockerSet, strings.ToUpper(strings.TrimSpace(b)))
		}

		// Convert back to slice
		var newBlockers []string
		for b := range blockerSet {
			newBlockers = append(newBlockers, b)
		}

		changes.BlockedBy = &newBlockers
		*hasChanges = true
	}

	return nil
}

// editableTask is a simplified task representation for interactive editing.
type editableTask struct {
	Title        string   `yaml:"title"`
	Priority     int      `yaml:"priority"`
	Tags         []string `yaml:"tags,omitempty"`
	Notes        string   `yaml:"notes,omitempty"`
	Assignee     string   `yaml:"assignee,omitempty"`
	DueDate      string   `yaml:"due_date,omitempty"`
	AutoComplete bool     `yaml:"auto_complete"`
	BlockedBy    []string `yaml:"blocked_by,omitempty"`
}

func runEditInteractive(s *storage.Storage, taskID string) error {
	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	// Find the task
	var task *model.Task
	for i := range pf.Tasks {
		if strings.EqualFold(pf.Tasks[i].ID, taskID) {
			task = &pf.Tasks[i]
			break
		}
	}
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Create editable representation
	editable := editableTask{
		Title:        task.Title,
		Priority:     task.Priority,
		Tags:         task.Tags,
		Notes:        task.Notes,
		Assignee:     task.Assignee,
		AutoComplete: task.AutoComplete,
		BlockedBy:    task.BlockedBy,
	}
	if task.DueDate != nil {
		editable.DueDate = task.DueDate.Format("2006-01-02")
	}

	// Marshal to YAML
	content, err := yaml.Marshal(&editable)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Add header comment
	header := fmt.Sprintf("# Editing task %s\n# Save and close editor to apply changes. Exit without saving to cancel.\n\n", taskID)
	content = append([]byte(header), content...)

	// Open in editor
	edited, err := cli.EditInEditor(content, ".yaml")
	if err != nil {
		return err
	}

	// Parse edited content
	var newEditable editableTask
	if err := yaml.Unmarshal(edited, &newEditable); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Build changes
	changes := ops.TaskChanges{}
	if newEditable.Title != task.Title {
		changes.Title = &newEditable.Title
	}
	if newEditable.Priority != task.Priority {
		if err := ops.ValidatePriority(newEditable.Priority); err != nil {
			return err
		}
		changes.Priority = &newEditable.Priority
	}
	if !stringSliceEqual(newEditable.Tags, task.Tags) {
		changes.Tags = &newEditable.Tags
	}
	if newEditable.Notes != task.Notes {
		changes.Notes = &newEditable.Notes
	}
	if newEditable.Assignee != task.Assignee {
		changes.Assignee = &newEditable.Assignee
	}

	// Handle due date changes
	oldDueDate := ""
	if task.DueDate != nil {
		oldDueDate = task.DueDate.Format("2006-01-02")
	}
	if newEditable.DueDate != oldDueDate {
		if newEditable.DueDate == "" {
			var nilTime *time.Time
			changes.DueDate = &nilTime
		} else {
			t, err := time.Parse("2006-01-02", newEditable.DueDate)
			if err != nil {
				return fmt.Errorf("invalid due_date format (expected YYYY-MM-DD): %v", err)
			}
			tPtr := &t
			changes.DueDate = &tPtr
		}
	}

	if newEditable.AutoComplete != task.AutoComplete {
		changes.AutoComplete = &newEditable.AutoComplete
	}
	if !stringSliceEqual(newEditable.BlockedBy, task.BlockedBy) {
		changes.BlockedBy = &newEditable.BlockedBy
	}

	if err := ops.EditTask(s, taskID, changes); err != nil {
		return err
	}

	fmt.Printf("%s updated.\n", taskID)
	return nil
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
