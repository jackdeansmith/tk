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

var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Manage waits",
	Long: `Manage waits (external conditions that block tasks).

Subcommands:
  add      Create a new wait
  edit     Edit an existing wait
  resolve  Mark a wait as resolved
  drop     Drop a wait
  defer    Push back a wait's dates`,
}

var waitAddCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Create a new wait",
	Long: `Create a new wait in a project.

For manual waits, use --question.
For time waits, use --after.

Examples:
  tk wait add -p BY --question="Did the fabric arrive?"
  tk wait add "Fabric delivery" -p BY --question="Did the fabric arrive?"
  tk wait add -p BY --question="Did the PCBs arrive?" --check-after=2026-01-10
  tk wait add -p BY --question="Did the PCBs arrive?" --blocked-by=BY-05
  tk wait add -p BY --after=2026-01-15
  tk wait add "After Jan 15" -p BY --after=2026-01-15T14:00:00`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWaitAdd,
}

var waitEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a wait",
	Long: `Edit a wait's fields.

Use flags to change specific fields, or -i to edit in $EDITOR.

Examples:
  tk wait edit BY-03W --title="New title"
  tk wait edit BY-03W --question="Updated question?"
  tk wait edit BY-03W --check-after=2026-01-20
  tk wait edit BY-03W --notes="Tracking: 123456"
  tk wait edit BY-03W -i`,
	Args:              cobra.ExactArgs(1),
	RunE:              runWaitEdit,
	ValidArgsFunction: completeWaitIDs,
}

var waitResolveCmd = &cobra.Command{
	Use:   "resolve <id>",
	Short: "Mark a wait as resolved",
	Long: `Mark a wait as resolved (done).

For manual waits, this marks the question as answered.
For time waits, this allows early resolution.

Examples:
  tk wait resolve BY-03W
  tk wait resolve BY-03W --resolution="Arrived damaged, returning"`,
	Args:              cobra.ExactArgs(1),
	RunE:              runWaitResolve,
	ValidArgsFunction: completeWaitIDs,
}

var waitDropCmd = &cobra.Command{
	Use:   "drop <id>",
	Short: "Drop a wait",
	Long: `Drop a wait (mark as not needed).

If the wait has dependent items (tasks or waits blocked by it):
- Use --drop-deps to also drop all dependent items recursively
- Use --remove-deps to unlink this wait from dependents

Examples:
  tk wait drop BY-03W
  tk wait drop BY-03W --reason="No longer needed"
  tk wait drop BY-03W --drop-deps
  tk wait drop BY-03W --remove-deps`,
	Args:              cobra.ExactArgs(1),
	RunE:              runWaitDrop,
	ValidArgsFunction: completeWaitIDs,
}

var waitDeferCmd = &cobra.Command{
	Use:   "defer <id>",
	Short: "Defer a wait",
	Long: `Push back a wait's dates.

For time waits, updates the 'after' field.
For manual waits, updates the 'check_after' field.

Examples:
  tk wait defer BY-03W --days=4
  tk wait defer BY-03W --until=2026-01-20`,
	Args:              cobra.ExactArgs(1),
	RunE:              runWaitDefer,
	ValidArgsFunction: completeWaitIDs,
}

var (
	// wait add flags
	waitAddProject    string
	waitAddQuestion   string
	waitAddAfter      string
	waitAddCheckAfter string
	waitAddNotes      string
	waitAddBlockedBy  string

	// wait edit flags
	waitEditTitle         string
	waitEditQuestion      string
	waitEditAfter         string
	waitEditCheckAfter    string
	waitEditClearCheckAfter bool
	waitEditNotes         string
	waitEditBlockedBy     string
	waitEditAddBlockedBy  []string
	waitEditRemoveBlockedBy []string
	waitEditInteractive   bool

	// wait resolve flags
	waitResolveResolution string

	// wait drop flags
	waitDropReason     string
	waitDropDropDeps   bool
	waitDropRemoveDeps bool

	// wait defer flags
	waitDeferDays  int
	waitDeferUntil string
)

func init() {
	// wait add command
	waitAddCmd.Flags().StringVarP(&waitAddProject, "project", "p", "", "project prefix or ID")
	waitAddCmd.Flags().StringVar(&waitAddQuestion, "question", "", "question for manual wait")
	waitAddCmd.Flags().StringVar(&waitAddAfter, "after", "", "date/time for time wait (YYYY-MM-DD or RFC3339)")
	waitAddCmd.Flags().StringVar(&waitAddCheckAfter, "check-after", "", "check after date (YYYY-MM-DD or RFC3339)")
	waitAddCmd.Flags().StringVar(&waitAddNotes, "notes", "", "wait notes")
	waitAddCmd.Flags().StringVar(&waitAddBlockedBy, "blocked-by", "", "comma-separated blocker IDs")
	waitAddCmd.RegisterFlagCompletionFunc("project", completeProjectIDs)
	waitCmd.AddCommand(waitAddCmd)

	// wait edit command
	waitEditCmd.Flags().StringVar(&waitEditTitle, "title", "", "set wait title")
	waitEditCmd.Flags().StringVar(&waitEditQuestion, "question", "", "set question (manual waits)")
	waitEditCmd.Flags().StringVar(&waitEditAfter, "after", "", "set after date (time waits)")
	waitEditCmd.Flags().StringVar(&waitEditCheckAfter, "check-after", "", "set check after date")
	waitEditCmd.Flags().BoolVar(&waitEditClearCheckAfter, "clear-check-after", false, "clear check after date")
	waitEditCmd.Flags().StringVar(&waitEditNotes, "notes", "", "set notes")
	waitEditCmd.Flags().StringVar(&waitEditBlockedBy, "blocked-by", "", "replace blockers (comma-separated)")
	waitEditCmd.Flags().StringArrayVar(&waitEditAddBlockedBy, "add-blocked-by", nil, "add a blocker")
	waitEditCmd.Flags().StringArrayVar(&waitEditRemoveBlockedBy, "remove-blocked-by", nil, "remove a blocker")
	waitEditCmd.Flags().BoolVarP(&waitEditInteractive, "interactive", "i", false, "edit in $EDITOR")
	waitEditCmd.RegisterFlagCompletionFunc("add-blocked-by", completeAnyIDs)
	waitEditCmd.RegisterFlagCompletionFunc("remove-blocked-by", completeAnyIDs)
	waitCmd.AddCommand(waitEditCmd)

	// wait resolve command
	waitResolveCmd.Flags().StringVar(&waitResolveResolution, "resolution", "", "resolution description")
	waitCmd.AddCommand(waitResolveCmd)

	// wait drop command
	waitDropCmd.Flags().StringVar(&waitDropReason, "reason", "", "reason for dropping")
	waitDropCmd.Flags().BoolVar(&waitDropDropDeps, "drop-deps", false, "also drop dependent items")
	waitDropCmd.Flags().BoolVar(&waitDropRemoveDeps, "remove-deps", false, "unlink from dependent items")
	waitCmd.AddCommand(waitDropCmd)

	// wait defer command
	waitDeferCmd.Flags().IntVar(&waitDeferDays, "days", 0, "defer for N days")
	waitDeferCmd.Flags().StringVar(&waitDeferUntil, "until", "", "defer until date (YYYY-MM-DD)")
	waitCmd.AddCommand(waitDeferCmd)

	rootCmd.AddCommand(waitCmd)
}

func runWaitAdd(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	// Get title if provided
	var title string
	if len(args) > 0 {
		title = args[0]
	}

	// Determine project
	project := waitAddProject
	if project == "" {
		cfg, err := s.LoadConfig()
		if err != nil {
			return err
		}
		if cfg.DefaultProject == "" {
			return fmt.Errorf("no project specified and no default_project in config")
		}
		pf, err := s.LoadProjectByID(cfg.DefaultProject)
		if err != nil {
			return fmt.Errorf("default project %q not found", cfg.DefaultProject)
		}
		project = pf.Prefix
	} else {
		pf, err := s.LoadProject(project)
		if err != nil {
			pf, err = s.LoadProjectByID(project)
			if err != nil {
				return fmt.Errorf("project %q not found", project)
			}
		}
		project = pf.Prefix
	}

	// Determine wait type
	if waitAddQuestion == "" && waitAddAfter == "" {
		return fmt.Errorf("either --question (manual wait) or --after (time wait) is required")
	}
	if waitAddQuestion != "" && waitAddAfter != "" {
		return fmt.Errorf("cannot specify both --question and --after")
	}

	opts := ops.WaitOptions{
		Title: title,
		Notes: waitAddNotes,
	}

	// Parse blockers
	if waitAddBlockedBy != "" {
		opts.BlockedBy = strings.Split(waitAddBlockedBy, ",")
		for i, id := range opts.BlockedBy {
			opts.BlockedBy[i] = strings.TrimSpace(id)
		}
	}

	if waitAddQuestion != "" {
		opts.Type = model.ResolutionTypeManual
		opts.Question = waitAddQuestion

		// Parse check_after if provided
		if waitAddCheckAfter != "" {
			t, err := parseDateTime(waitAddCheckAfter)
			if err != nil {
				return fmt.Errorf("invalid check-after date: %v", err)
			}
			opts.CheckAfter = &t
		}
	} else {
		opts.Type = model.ResolutionTypeTime

		// Parse after date
		t, err := parseDateTime(waitAddAfter)
		if err != nil {
			return fmt.Errorf("invalid after date: %v", err)
		}
		opts.After = &t
	}

	wait, err := ops.AddWait(s, project, opts)
	if err != nil {
		return err
	}

	fmt.Printf("%s %s\n", wait.ID, wait.DisplayText())
	return nil
}

func runWaitEdit(cmd *cobra.Command, args []string) error {
	waitID := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if waitEditInteractive {
		return runWaitEditInteractive(s, waitID)
	}

	// Build changes from flags
	changes := ops.WaitChanges{}
	hasChanges := false

	if cmd.Flags().Changed("title") {
		changes.Title = &waitEditTitle
		hasChanges = true
	}

	if cmd.Flags().Changed("question") {
		changes.Question = &waitEditQuestion
		hasChanges = true
	}

	if cmd.Flags().Changed("after") {
		t, err := parseDateTime(waitEditAfter)
		if err != nil {
			return fmt.Errorf("invalid after date: %v", err)
		}
		tPtr := &t
		changes.After = &tPtr
		hasChanges = true
	}

	if waitEditClearCheckAfter {
		var nilTime *time.Time
		changes.CheckAfter = &nilTime
		hasChanges = true
	} else if cmd.Flags().Changed("check-after") {
		t, err := parseDateTime(waitEditCheckAfter)
		if err != nil {
			return fmt.Errorf("invalid check-after date: %v", err)
		}
		tPtr := &t
		changes.CheckAfter = &tPtr
		hasChanges = true
	}

	if cmd.Flags().Changed("notes") {
		changes.Notes = &waitEditNotes
		hasChanges = true
	}

	// Handle blockers
	if err := handleWaitBlockerChanges(s, waitID, &changes, cmd, &hasChanges); err != nil {
		return err
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	if err := ops.EditWait(s, waitID, changes); err != nil {
		return err
	}

	fmt.Printf("%s updated.\n", waitID)
	return nil
}

func handleWaitBlockerChanges(s *storage.Storage, waitID string, changes *ops.WaitChanges, cmd *cobra.Command, hasChanges *bool) error {
	if cmd.Flags().Changed("blocked-by") {
		var blockers []string
		if waitEditBlockedBy != "" {
			blockers = strings.Split(waitEditBlockedBy, ",")
			for i, b := range blockers {
				blockers[i] = strings.TrimSpace(b)
			}
		}
		changes.BlockedBy = &blockers
		*hasChanges = true
		return nil
	}

	if len(waitEditAddBlockedBy) > 0 || len(waitEditRemoveBlockedBy) > 0 {
		prefix := model.ExtractPrefix(waitID)
		pf, err := s.LoadProject(prefix)
		if err != nil {
			return err
		}

		var wait *model.Wait
		for i := range pf.Waits {
			if strings.EqualFold(pf.Waits[i].ID, waitID) {
				wait = &pf.Waits[i]
				break
			}
		}
		if wait == nil {
			return fmt.Errorf("wait %s not found", waitID)
		}

		blockerSet := make(map[string]bool)
		for _, b := range wait.BlockedBy {
			blockerSet[strings.ToUpper(b)] = true
		}

		for _, b := range waitEditAddBlockedBy {
			blockerSet[strings.ToUpper(strings.TrimSpace(b))] = true
		}

		for _, b := range waitEditRemoveBlockedBy {
			delete(blockerSet, strings.ToUpper(strings.TrimSpace(b)))
		}

		var newBlockers []string
		for b := range blockerSet {
			newBlockers = append(newBlockers, b)
		}

		changes.BlockedBy = &newBlockers
		*hasChanges = true
	}

	return nil
}

// editableWait is a simplified wait representation for interactive editing.
type editableWait struct {
	Title      string   `yaml:"title,omitempty"`
	Type       string   `yaml:"type"`
	Question   string   `yaml:"question,omitempty"`
	After      string   `yaml:"after,omitempty"`
	CheckAfter string   `yaml:"check_after,omitempty"`
	Notes      string   `yaml:"notes,omitempty"`
	BlockedBy  []string `yaml:"blocked_by,omitempty"`
}

func runWaitEditInteractive(s *storage.Storage, waitID string) error {
	prefix := model.ExtractPrefix(waitID)
	if prefix == "" {
		return fmt.Errorf("invalid wait ID: %s", waitID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	var wait *model.Wait
	for i := range pf.Waits {
		if strings.EqualFold(pf.Waits[i].ID, waitID) {
			wait = &pf.Waits[i]
			break
		}
	}
	if wait == nil {
		return fmt.Errorf("wait %s not found", waitID)
	}

	editable := editableWait{
		Title:     wait.Title,
		Type:      string(wait.ResolutionCriteria.Type),
		Question:  wait.ResolutionCriteria.Question,
		Notes:     wait.Notes,
		BlockedBy: wait.BlockedBy,
	}
	if wait.ResolutionCriteria.After != nil {
		editable.After = wait.ResolutionCriteria.After.Format(time.RFC3339)
	}
	if wait.ResolutionCriteria.CheckAfter != nil {
		editable.CheckAfter = wait.ResolutionCriteria.CheckAfter.Format(time.RFC3339)
	}

	content, err := yaml.Marshal(&editable)
	if err != nil {
		return fmt.Errorf("failed to marshal wait: %w", err)
	}

	header := fmt.Sprintf("# Editing wait %s\n# Note: 'type' cannot be changed.\n# Save and close editor to apply changes.\n\n", waitID)
	content = append([]byte(header), content...)

	edited, err := cli.EditInEditor(content, ".yaml")
	if err != nil {
		return err
	}

	var newEditable editableWait
	if err := yaml.Unmarshal(edited, &newEditable); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	changes := ops.WaitChanges{}
	if newEditable.Title != wait.Title {
		changes.Title = &newEditable.Title
	}
	if newEditable.Question != wait.ResolutionCriteria.Question {
		changes.Question = &newEditable.Question
	}
	if newEditable.Notes != wait.Notes {
		changes.Notes = &newEditable.Notes
	}

	oldAfter := ""
	if wait.ResolutionCriteria.After != nil {
		oldAfter = wait.ResolutionCriteria.After.Format(time.RFC3339)
	}
	if newEditable.After != oldAfter {
		if newEditable.After == "" {
			var nilTime *time.Time
			changes.After = &nilTime
		} else {
			t, err := parseDateTime(newEditable.After)
			if err != nil {
				return fmt.Errorf("invalid after date: %v", err)
			}
			tPtr := &t
			changes.After = &tPtr
		}
	}

	oldCheckAfter := ""
	if wait.ResolutionCriteria.CheckAfter != nil {
		oldCheckAfter = wait.ResolutionCriteria.CheckAfter.Format(time.RFC3339)
	}
	if newEditable.CheckAfter != oldCheckAfter {
		if newEditable.CheckAfter == "" {
			var nilTime *time.Time
			changes.CheckAfter = &nilTime
		} else {
			t, err := parseDateTime(newEditable.CheckAfter)
			if err != nil {
				return fmt.Errorf("invalid check_after date: %v", err)
			}
			tPtr := &t
			changes.CheckAfter = &tPtr
		}
	}

	if !stringSliceEqual(newEditable.BlockedBy, wait.BlockedBy) {
		changes.BlockedBy = &newEditable.BlockedBy
	}

	if err := ops.EditWait(s, waitID, changes); err != nil {
		return err
	}

	fmt.Printf("%s updated.\n", waitID)
	return nil
}

func runWaitResolve(cmd *cobra.Command, args []string) error {
	waitID := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if err := ops.ResolveWait(s, waitID, waitResolveResolution); err != nil {
		return err
	}

	fmt.Printf("%s resolved.\n", waitID)
	return nil
}

func runWaitDrop(cmd *cobra.Command, args []string) error {
	waitID := args[0]

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	if err := ops.DropWait(s, waitID, waitDropReason, waitDropDropDeps, waitDropRemoveDeps); err != nil {
		return err
	}

	fmt.Printf("%s dropped.\n", waitID)
	return nil
}

func runWaitDefer(cmd *cobra.Command, args []string) error {
	waitID := args[0]

	if waitDeferDays == 0 && waitDeferUntil == "" {
		return fmt.Errorf("either --days or --until must be specified")
	}

	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	var until time.Time
	if waitDeferDays > 0 {
		target := time.Now().AddDate(0, 0, waitDeferDays)
		until = time.Date(target.Year(), target.Month(), target.Day(), 23, 59, 59, 0, target.Location())
	} else {
		t, err := time.Parse("2006-01-02", waitDeferUntil)
		if err != nil {
			return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %v", err)
		}
		until = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
	}

	if err := ops.DeferWait(s, waitID, until); err != nil {
		return err
	}

	fmt.Printf("%s deferred until %s.\n", waitID, until.Format("2006-01-02"))
	return nil
}

// parseDateTime parses a date or datetime string.
// Accepts YYYY-MM-DD (end of day) or RFC3339.
func parseDateTime(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try date only
	if t, err := time.Parse("2006-01-02", s); err == nil {
		// End of day
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local), nil
	}

	return time.Time{}, fmt.Errorf("expected YYYY-MM-DD or RFC3339 format, got %q", s)
}
