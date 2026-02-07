package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/graph"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
)

// Priority range constants.
const (
	MinPriority = 1
	MaxPriority = 4
)

// ValidateTitle checks that a task title is not empty or whitespace-only.
func ValidateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("task title must not be empty")
	}
	return nil
}

// ValidatePriority checks that priority is within the valid range (1-4).
// A priority of 0 is allowed as it means "use default".
func ValidatePriority(priority int) error {
	if priority == 0 {
		return nil // 0 means "use default"
	}
	if priority < MinPriority || priority > MaxPriority {
		return fmt.Errorf("invalid priority %d: must be between %d and %d", priority, MinPriority, MaxPriority)
	}
	return nil
}

// TaskOptions contains options for creating a new task.
type TaskOptions struct {
	Priority     int
	Tags         []string
	Notes        string
	Assignee     string
	DueDate      *time.Time
	AutoComplete bool
	BlockedBy    []string
}

// TaskChanges represents fields that can be updated on a task.
type TaskChanges struct {
	Title        *string
	Priority     *int
	Tags         *[]string
	Notes        *string
	Assignee     *string
	DueDate      **time.Time // pointer to pointer to allow setting to nil
	AutoComplete *bool
	BlockedBy    *[]string
}

// IncompleteBlockersError indicates that a task cannot be completed because it
// has incomplete blockers. This is a distinct error type so callers can
// differentiate blocker errors from other completion failures (e.g. task not
// found, task not open).
type IncompleteBlockersError struct {
	TaskID   string
	Blockers []string
}

func (e *IncompleteBlockersError) Error() string {
	return fmt.Sprintf("task has incomplete blockers: %s (use --force to complete anyway)",
		strings.Join(e.Blockers, ", "))
}

// CompletionResult contains the results of completing a task.
type CompletionResult struct {
	// Unblocked lists tasks/waits that are now unblocked.
	Unblocked []string
	// Activated lists waits that went from dormant to active.
	Activated []string
	// AutoCompleted lists tasks that were auto-completed as a cascade.
	AutoCompleted []string
}

// AddTask creates a new task in the given project.
func AddTask(s *storage.Storage, prefix string, title string, opts TaskOptions) (*model.Task, error) {
	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, err
	}

	// Validate project is active
	if pf.Status != model.ProjectStatusActive {
		return nil, fmt.Errorf("cannot add task to %s project %q", pf.Status, pf.Name)
	}

	// Validate title
	if err := ValidateTitle(title); err != nil {
		return nil, err
	}

	// Validate priority if specified
	if err := ValidatePriority(opts.Priority); err != nil {
		return nil, err
	}

	// Validate blockers if provided
	if len(opts.BlockedBy) > 0 {
		if err := validateBlockers(pf, opts.BlockedBy); err != nil {
			return nil, err
		}
	}

	// Set default priority if not specified
	priority := opts.Priority
	if priority == 0 {
		priority = 3 // Default priority
	}

	// Create task with next ID
	now := time.Now()
	taskID := model.FormatTaskID(pf.Prefix, pf.NextID, pf.NextID)
	task := model.Task{
		ID:           taskID,
		Title:        title,
		Status:       model.TaskStatusOpen,
		Priority:     priority,
		BlockedBy:    normalizeBlockerIDs(opts.BlockedBy, pf.NextID),
		Tags:         opts.Tags,
		Notes:        opts.Notes,
		Assignee:     opts.Assignee,
		DueDate:      opts.DueDate,
		AutoComplete: opts.AutoComplete,
		Created:      now,
		Updated:      now,
	}

	pf.Tasks = append(pf.Tasks, task)
	pf.NextID++

	if err := s.SaveProject(pf); err != nil {
		return nil, err
	}

	return &task, nil
}

// EditTask modifies an existing task.
func EditTask(s *storage.Storage, taskID string, changes TaskChanges) error {
	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Validate title if being changed
	if changes.Title != nil {
		if err := ValidateTitle(*changes.Title); err != nil {
			return err
		}
	}

	// Validate priority if being changed (0 is not valid when explicitly setting)
	if changes.Priority != nil {
		p := *changes.Priority
		if p < MinPriority || p > MaxPriority {
			return fmt.Errorf("invalid priority %d: must be between %d and %d", p, MinPriority, MaxPriority)
		}
	}

	// Validate new blockers if being changed
	if changes.BlockedBy != nil {
		if err := validateBlockers(pf, *changes.BlockedBy); err != nil {
			return err
		}
		// Check for cycles
		g := graph.BuildGraph(pf)
		for _, blockerID := range *changes.BlockedBy {
			if cycle := g.CheckCycle(taskID, blockerID); cycle != nil {
				return fmt.Errorf("adding blocker would create cycle: %s", strings.Join(cycle, " -> "))
			}
		}
	}

	// Apply changes
	if changes.Title != nil {
		task.Title = *changes.Title
	}
	if changes.Priority != nil {
		task.Priority = *changes.Priority
	}
	if changes.Tags != nil {
		task.Tags = *changes.Tags
	}
	if changes.Notes != nil {
		task.Notes = *changes.Notes
	}
	if changes.Assignee != nil {
		task.Assignee = *changes.Assignee
	}
	if changes.DueDate != nil {
		task.DueDate = *changes.DueDate
	}
	if changes.AutoComplete != nil {
		task.AutoComplete = *changes.AutoComplete
	}
	if changes.BlockedBy != nil {
		task.BlockedBy = normalizeBlockerIDs(*changes.BlockedBy, pf.NextID-1)
	}

	task.Updated = time.Now()

	return s.SaveProject(pf)
}

// CompleteTask marks a task as done.
// If force is false and the task has incomplete blockers, returns an error.
// Returns information about cascading effects (unblocked items, auto-completed tasks).
func CompleteTask(s *storage.Storage, taskID string, force bool) (*CompletionResult, error) {
	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return nil, fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != model.TaskStatusOpen {
		return nil, fmt.Errorf("task %s is not open (status: %s)", taskID, task.Status)
	}

	// Check for incomplete blockers
	blockerStates := computeBlockerStates(pf)
	incompleteBlockers := []string{}
	for _, blockerID := range task.BlockedBy {
		if resolved, ok := blockerStates[blockerID]; !ok || !resolved {
			incompleteBlockers = append(incompleteBlockers, blockerID)
		}
	}

	if len(incompleteBlockers) > 0 {
		if !force {
			return nil, &IncompleteBlockersError{
				TaskID:   taskID,
				Blockers: incompleteBlockers,
			}
		}
		// Remove unfulfilled blockers when forcing
		task.BlockedBy = removeBlockers(task.BlockedBy, incompleteBlockers)
	}

	// Mark as done
	now := time.Now()
	task.Status = model.TaskStatusDone
	task.DoneAt = &now
	task.Updated = now

	// Calculate cascading effects
	result := &CompletionResult{}

	// Update blocker states with this task now done
	blockerStates[taskID] = true

	// Find items that are now unblocked
	g := graph.BuildGraph(pf)
	for _, dependentID := range g.Blocking(taskID) {
		// Check if all blockers are now resolved
		item := findItem(pf, dependentID)
		if item == nil {
			continue
		}

		allResolved := true
		for _, bid := range item.blockedBy {
			if resolved, ok := blockerStates[bid]; !ok || !resolved {
				allResolved = false
				break
			}
		}

		if allResolved && item.status == model.TaskStatusOpen {
			result.Unblocked = append(result.Unblocked, dependentID)
		}
	}

	// Find waits that are now activated (were dormant, now active)
	for i := range pf.Waits {
		w := &pf.Waits[i]
		if w.Status != model.WaitStatusOpen {
			continue
		}

		// Check if this wait was dormant (had this task as a blocker)
		wasDormant := false
		for _, bid := range w.BlockedBy {
			if bid == taskID {
				wasDormant = true
				break
			}
		}

		if wasDormant {
			// Check if all blockers are now resolved
			allResolved := true
			for _, bid := range w.BlockedBy {
				if resolved, ok := blockerStates[bid]; !ok || !resolved {
					allResolved = false
					break
				}
			}
			if allResolved {
				result.Activated = append(result.Activated, w.ID)
			}
		}
	}

	// Handle auto-complete cascade
	result.AutoCompleted = processAutoComplete(pf, blockerStates)

	if err := s.SaveProject(pf); err != nil {
		return nil, err
	}

	return result, nil
}

// processAutoComplete handles cascading auto-completion of tasks.
// Returns list of task IDs that were auto-completed.
func processAutoComplete(pf *model.ProjectFile, blockerStates model.BlockerStatus) []string {
	var autoCompleted []string
	changed := true

	for changed {
		changed = false
		for i := range pf.Tasks {
			t := &pf.Tasks[i]
			if t.Status != model.TaskStatusOpen || !t.AutoComplete {
				continue
			}

			// Check if all blockers are resolved
			allResolved := true
			for _, bid := range t.BlockedBy {
				if resolved, ok := blockerStates[bid]; !ok || !resolved {
					allResolved = false
					break
				}
			}

			if allResolved {
				// Auto-complete this task
				now := time.Now()
				t.Status = model.TaskStatusDone
				t.DoneAt = &now
				t.Updated = now
				blockerStates[t.ID] = true
				autoCompleted = append(autoCompleted, t.ID)
				changed = true
			}
		}
	}

	return autoCompleted
}

// DropTask marks a task as dropped.
// If dropDeps is true, dependent items are also dropped recursively.
// If removeDeps is true, this task is removed from dependents' blocked_by lists.
func DropTask(s *storage.Storage, taskID string, reason string, dropDeps, removeDeps bool) error {
	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != model.TaskStatusOpen {
		return fmt.Errorf("task %s is not open (status: %s)", taskID, task.Status)
	}

	// Check for dependents
	g := graph.BuildGraph(pf)
	dependents := g.Blocking(taskID)

	// Filter to only open dependents
	openDependents := []string{}
	for _, depID := range dependents {
		item := findItem(pf, depID)
		if item != nil && item.status == model.TaskStatusOpen {
			openDependents = append(openDependents, depID)
		}
	}

	if len(openDependents) > 0 && !dropDeps && !removeDeps {
		return fmt.Errorf("task has dependents: %s (use --drop-deps or --remove-deps)",
			strings.Join(openDependents, ", "))
	}

	if dropDeps {
		// Drop all dependents recursively
		if err := dropDependents(pf, taskID, reason); err != nil {
			return err
		}
	}

	if removeDeps {
		// Remove this task from all dependents' blocked_by lists
		removeSelfFromDependents(pf, taskID)
	}

	// Drop the task
	now := time.Now()
	task.Status = model.TaskStatusDropped
	task.DroppedAt = &now
	task.DropReason = reason
	task.Updated = now

	return s.SaveProject(pf)
}

// dropDependents recursively drops all items that depend on the given ID.
func dropDependents(pf *model.ProjectFile, id string, reason string) error {
	g := graph.BuildGraph(pf)
	dependents := g.TransitiveBlocking(id)
	now := time.Now()

	for _, depID := range dependents {
		if model.IsTaskID(depID) {
			task := findTask(pf, depID)
			if task != nil && task.Status == model.TaskStatusOpen {
				task.Status = model.TaskStatusDropped
				task.DroppedAt = &now
				task.DropReason = reason
				task.Updated = now
			}
		} else {
			wait := findWait(pf, depID)
			if wait != nil && wait.Status == model.WaitStatusOpen {
				wait.Status = model.WaitStatusDropped
				wait.DroppedAt = &now
				wait.DropReason = reason
			}
		}
	}

	return nil
}

// removeSelfFromDependents removes the given ID from all dependents' blocked_by lists.
func removeSelfFromDependents(pf *model.ProjectFile, id string) {
	for i := range pf.Tasks {
		pf.Tasks[i].BlockedBy = removeFromSlice(pf.Tasks[i].BlockedBy, id)
	}
	for i := range pf.Waits {
		pf.Waits[i].BlockedBy = removeFromSlice(pf.Waits[i].BlockedBy, id)
	}
}

// ReopenTask reopens a done or dropped task.
func ReopenTask(s *storage.Storage, taskID string) error {
	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Status == model.TaskStatusOpen {
		return fmt.Errorf("task %s is already open", taskID)
	}

	task.Status = model.TaskStatusOpen
	task.DoneAt = nil
	task.DroppedAt = nil
	task.DropReason = ""
	task.Updated = time.Now()

	return s.SaveProject(pf)
}

// DeferTask defers a task by creating a time-based wait.
// Returns error if the task already has open waits.
func DeferTask(s *storage.Storage, taskID string, until time.Time) (*model.Wait, error) {
	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return nil, fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	if task.Status != model.TaskStatusOpen {
		return nil, fmt.Errorf("task %s is not open (status: %s)", taskID, task.Status)
	}

	// Check if task already has open wait blockers
	for _, blockerID := range task.BlockedBy {
		if model.IsWaitID(blockerID) {
			wait := findWait(pf, blockerID)
			if wait != nil && wait.Status == model.WaitStatusOpen {
				return nil, fmt.Errorf("task already has open wait: %s", blockerID)
			}
		}
	}

	// Create a time-based wait
	now := time.Now()
	waitID := model.FormatWaitID(pf.Prefix, pf.NextID, pf.NextID)
	wait := model.Wait{
		ID:     waitID,
		Status: model.WaitStatusOpen,
		ResolutionCriteria: model.ResolutionCriteria{
			Type:  model.ResolutionTypeTime,
			After: &until,
		},
		Created: now,
	}

	pf.Waits = append(pf.Waits, wait)
	pf.NextID++

	// Add wait to task's blocked_by
	task.BlockedBy = append(task.BlockedBy, waitID)
	task.Updated = now

	if err := s.SaveProject(pf); err != nil {
		return nil, err
	}

	return &wait, nil
}

// MoveTask moves a task to a different project.
func MoveTask(s *storage.Storage, taskID string, toPrefix string) error {
	fromPrefix := model.ExtractPrefix(taskID)
	if fromPrefix == "" {
		return fmt.Errorf("invalid task ID: %s", taskID)
	}

	toPrefix = strings.ToUpper(toPrefix)
	if fromPrefix == toPrefix {
		return fmt.Errorf("task is already in project %s", toPrefix)
	}

	// Load source project
	srcPf, err := s.LoadProject(fromPrefix)
	if err != nil {
		return err
	}

	// Load destination project
	dstPf, err := s.LoadProject(toPrefix)
	if err != nil {
		return err
	}

	// Find and remove task from source
	var task *model.Task
	for i, t := range srcPf.Tasks {
		if strings.EqualFold(t.ID, taskID) {
			task = &srcPf.Tasks[i]
			srcPf.Tasks = append(srcPf.Tasks[:i], srcPf.Tasks[i+1:]...)
			break
		}
	}

	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Check if task has blockers in the source project
	for _, blockerID := range task.BlockedBy {
		blockerPrefix := model.ExtractPrefix(blockerID)
		if blockerPrefix == fromPrefix {
			return fmt.Errorf("task has blockers in source project: %s", blockerID)
		}
	}

	// Check if task is blocking anything in the source project
	g := graph.BuildGraph(srcPf)
	dependents := g.Blocking(taskID)
	if len(dependents) > 0 {
		return fmt.Errorf("task is blocking items in source project: %s", strings.Join(dependents, ", "))
	}

	// Assign new ID in destination project
	newID := model.FormatTaskID(dstPf.Prefix, dstPf.NextID, dstPf.NextID)
	task.ID = newID
	task.Updated = time.Now()

	// Clear blockers that reference the old project (they won't work in new project)
	newBlockers := []string{}
	for _, blockerID := range task.BlockedBy {
		blockerPrefix := model.ExtractPrefix(blockerID)
		if blockerPrefix != fromPrefix {
			newBlockers = append(newBlockers, blockerID)
		}
	}
	task.BlockedBy = newBlockers

	dstPf.Tasks = append(dstPf.Tasks, *task)
	dstPf.NextID++

	// Save both projects
	if err := s.SaveProject(srcPf); err != nil {
		return err
	}
	return s.SaveProject(dstPf)
}

// AddBlocker adds a blocker to a task.
func AddBlocker(s *storage.Storage, taskID, blockerID string) error {
	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Validate blocker exists
	if err := validateBlockers(pf, []string{blockerID}); err != nil {
		return err
	}

	// Check for existing blocker
	for _, bid := range task.BlockedBy {
		if strings.EqualFold(bid, blockerID) {
			return fmt.Errorf("blocker %s already exists on task %s", blockerID, taskID)
		}
	}

	// Check for cycles
	g := graph.BuildGraph(pf)
	if cycle := g.CheckCycle(taskID, blockerID); cycle != nil {
		return fmt.Errorf("adding blocker would create cycle: %s", strings.Join(cycle, " -> "))
	}

	task.BlockedBy = append(task.BlockedBy, model.NormalizeID(blockerID, pf.NextID-1))
	task.Updated = time.Now()

	return s.SaveProject(pf)
}

// RemoveBlocker removes a blocker from a task.
func RemoveBlocker(s *storage.Storage, taskID, blockerID string) error {
	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Find and remove blocker
	found := false
	for i, bid := range task.BlockedBy {
		if strings.EqualFold(bid, blockerID) {
			task.BlockedBy = append(task.BlockedBy[:i], task.BlockedBy[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("blocker %s not found on task %s", blockerID, taskID)
	}

	task.Updated = time.Now()

	return s.SaveProject(pf)
}

// Helper functions

// findTask finds a task by ID in a project file.
func findTask(pf *model.ProjectFile, taskID string) *model.Task {
	for i := range pf.Tasks {
		if strings.EqualFold(pf.Tasks[i].ID, taskID) {
			return &pf.Tasks[i]
		}
	}
	return nil
}

// findWait finds a wait by ID in a project file.
func findWait(pf *model.ProjectFile, waitID string) *model.Wait {
	for i := range pf.Waits {
		if strings.EqualFold(pf.Waits[i].ID, waitID) {
			return &pf.Waits[i]
		}
	}
	return nil
}

// itemInfo holds common properties for both tasks and waits.
type itemInfo struct {
	id        string
	status    model.TaskStatus
	blockedBy []string
}

// findItem finds a task or wait by ID and returns common info.
func findItem(pf *model.ProjectFile, id string) *itemInfo {
	if model.IsWaitID(id) {
		w := findWait(pf, id)
		if w == nil {
			return nil
		}
		return &itemInfo{
			id:        w.ID,
			status:    model.TaskStatus(w.Status),
			blockedBy: w.BlockedBy,
		}
	}
	t := findTask(pf, id)
	if t == nil {
		return nil
	}
	return &itemInfo{
		id:        t.ID,
		status:    t.Status,
		blockedBy: t.BlockedBy,
	}
}

// validateBlockers checks that all blocker IDs exist in the project.
func validateBlockers(pf *model.ProjectFile, blockerIDs []string) error {
	for _, id := range blockerIDs {
		if model.IsWaitID(id) {
			if findWait(pf, id) == nil {
				return fmt.Errorf("blocker %s not found", id)
			}
		} else if model.IsTaskID(id) {
			if findTask(pf, id) == nil {
				return fmt.Errorf("blocker %s not found", id)
			}
		} else {
			return fmt.Errorf("invalid blocker ID: %s", id)
		}
	}
	return nil
}

// normalizeBlockerIDs normalizes blocker IDs to canonical form.
func normalizeBlockerIDs(ids []string, maxID int) []string {
	if len(ids) == 0 {
		return nil
	}
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = model.NormalizeID(id, maxID)
	}
	return result
}

// computeBlockerStates builds a map of ID -> resolved status for all items.
func computeBlockerStates(pf *model.ProjectFile) model.BlockerStatus {
	states := make(model.BlockerStatus)
	for _, t := range pf.Tasks {
		states[t.ID] = t.Status == model.TaskStatusDone || t.Status == model.TaskStatusDropped
	}
	for _, w := range pf.Waits {
		states[w.ID] = w.Status == model.WaitStatusDone || w.Status == model.WaitStatusDropped
	}
	return states
}

// removeBlockers removes specified IDs from a slice.
func removeBlockers(slice []string, toRemove []string) []string {
	removeSet := make(map[string]bool)
	for _, id := range toRemove {
		removeSet[strings.ToUpper(id)] = true
	}

	result := []string{}
	for _, id := range slice {
		if !removeSet[strings.ToUpper(id)] {
			result = append(result, id)
		}
	}
	return result
}

// removeFromSlice removes a single ID from a slice (case-insensitive).
func removeFromSlice(slice []string, id string) []string {
	result := []string{}
	for _, s := range slice {
		if !strings.EqualFold(s, id) {
			result = append(result, s)
		}
	}
	return result
}
