package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/graph"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
)

// WaitOptions contains options for creating a new wait.
type WaitOptions struct {
	Title      string
	Type       model.ResolutionType
	Question   string     // For manual waits
	After      *time.Time // For time waits
	CheckAfter *time.Time // For manual waits (optional)
	Notes      string
	BlockedBy  []string
}

// WaitChanges represents fields that can be updated on a wait.
type WaitChanges struct {
	Title      *string
	Question   *string
	After      **time.Time
	CheckAfter **time.Time
	Notes      *string
	BlockedBy  *[]string
}

// AddWait creates a new wait in the given project.
func AddWait(s *storage.Storage, prefix string, opts WaitOptions) (*model.Wait, error) {
	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, err
	}

	// Validate project is active
	if pf.Status != model.ProjectStatusActive {
		return nil, fmt.Errorf("cannot add wait to %s project %q", pf.Status, pf.Name)
	}

	// Validate resolution type and required fields
	switch opts.Type {
	case model.ResolutionTypeTime:
		if opts.After == nil {
			return nil, fmt.Errorf("time waits require 'after' date")
		}
	case model.ResolutionTypeManual:
		if opts.Question == "" && opts.Title == "" {
			return nil, fmt.Errorf("manual waits require 'question' or 'title'")
		}
	default:
		return nil, fmt.Errorf("invalid resolution type: %s", opts.Type)
	}

	// Validate blockers if provided
	if len(opts.BlockedBy) > 0 {
		if err := validateBlockers(pf, opts.BlockedBy); err != nil {
			return nil, err
		}
	}

	// Create wait with next ID
	now := time.Now()
	waitID := model.FormatWaitID(pf.Prefix, pf.NextID, pf.NextID)
	wait := model.Wait{
		ID:     waitID,
		Title:  opts.Title,
		Status: model.WaitStatusOpen,
		ResolutionCriteria: model.ResolutionCriteria{
			Type:       opts.Type,
			Question:   opts.Question,
			After:      opts.After,
			CheckAfter: opts.CheckAfter,
		},
		BlockedBy: normalizeBlockerIDs(opts.BlockedBy, pf.NextID),
		Notes:     opts.Notes,
		Created:   now,
	}

	pf.Waits = append(pf.Waits, wait)
	pf.NextID++

	if err := s.SaveProject(pf); err != nil {
		return nil, err
	}

	return &wait, nil
}

// EditWait modifies an existing wait.
func EditWait(s *storage.Storage, waitID string, changes WaitChanges) error {
	prefix := model.ExtractPrefix(waitID)
	if prefix == "" {
		return fmt.Errorf("invalid wait ID: %s", waitID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	wait := findWait(pf, waitID)
	if wait == nil {
		return fmt.Errorf("wait %s not found", waitID)
	}

	// Validate new blockers if being changed
	if changes.BlockedBy != nil {
		if err := validateBlockers(pf, *changes.BlockedBy); err != nil {
			return err
		}
		// Check for cycles
		g := graph.BuildGraph(pf)
		for _, blockerID := range *changes.BlockedBy {
			if cycle := g.CheckCycle(waitID, blockerID); cycle != nil {
				return fmt.Errorf("adding blocker would create cycle: %s", strings.Join(cycle, " -> "))
			}
		}
	}

	// Apply changes
	if changes.Title != nil {
		wait.Title = *changes.Title
	}
	if changes.Question != nil {
		wait.ResolutionCriteria.Question = *changes.Question
	}
	if changes.After != nil {
		wait.ResolutionCriteria.After = *changes.After
	}
	if changes.CheckAfter != nil {
		wait.ResolutionCriteria.CheckAfter = *changes.CheckAfter
	}
	if changes.Notes != nil {
		wait.Notes = *changes.Notes
	}
	if changes.BlockedBy != nil {
		wait.BlockedBy = normalizeBlockerIDs(*changes.BlockedBy, pf.NextID-1)
	}

	return s.SaveProject(pf)
}

// ResolveWait marks a wait as done.
// For time waits, allows early resolution by updating 'after' to current time.
// Returns error if wait is dormant (blocked by incomplete items).
func ResolveWait(s *storage.Storage, waitID string, resolution string) error {
	prefix := model.ExtractPrefix(waitID)
	if prefix == "" {
		return fmt.Errorf("invalid wait ID: %s", waitID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	wait := findWait(pf, waitID)
	if wait == nil {
		return fmt.Errorf("wait %s not found", waitID)
	}

	if wait.Status != model.WaitStatusOpen {
		return fmt.Errorf("wait %s is not open (status: %s)", waitID, wait.Status)
	}

	// Check if wait is dormant (has unresolved blockers)
	blockerStates := computeBlockerStates(pf)
	for _, blockerID := range wait.BlockedBy {
		if resolved, ok := blockerStates[blockerID]; !ok || !resolved {
			return fmt.Errorf("wait is dormant (blocked by %s)", blockerID)
		}
	}

	// For time waits that haven't passed yet, update 'after' to now
	now := time.Now()
	if wait.ResolutionCriteria.Type == model.ResolutionTypeTime {
		if wait.ResolutionCriteria.After != nil && wait.ResolutionCriteria.After.After(now) {
			wait.ResolutionCriteria.After = &now
		}
	}

	// Mark as done
	wait.Status = model.WaitStatusDone
	wait.DoneAt = &now
	wait.Resolution = resolution

	return s.SaveProject(pf)
}

// DropWait marks a wait as dropped.
// If dropDeps is true, dependent items are also dropped recursively.
// If removeDeps is true, this wait is removed from dependents' blocked_by lists.
func DropWait(s *storage.Storage, waitID string, reason string, dropDeps, removeDeps bool) error {
	prefix := model.ExtractPrefix(waitID)
	if prefix == "" {
		return fmt.Errorf("invalid wait ID: %s", waitID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	wait := findWait(pf, waitID)
	if wait == nil {
		return fmt.Errorf("wait %s not found", waitID)
	}

	if wait.Status != model.WaitStatusOpen {
		return fmt.Errorf("wait %s is not open (status: %s)", waitID, wait.Status)
	}

	// Check for dependents (items blocked by this wait)
	g := graph.BuildGraph(pf)
	dependents := g.Blocking(waitID)

	// Filter to only open dependents
	openDependents := []string{}
	for _, depID := range dependents {
		item := findItem(pf, depID)
		if item != nil && item.status == model.TaskStatusOpen {
			openDependents = append(openDependents, depID)
		}
	}

	if len(openDependents) > 0 && !dropDeps && !removeDeps {
		return fmt.Errorf("wait has dependents: %s (use --drop-deps or --remove-deps)",
			strings.Join(openDependents, ", "))
	}

	if dropDeps {
		// Drop all dependents recursively
		if err := dropDependents(pf, waitID, reason); err != nil {
			return err
		}
	}

	if removeDeps {
		// Remove this wait from all dependents' blocked_by lists
		removeSelfFromDependents(pf, waitID)
	}

	// Drop the wait
	now := time.Now()
	wait.Status = model.WaitStatusDropped
	wait.DroppedAt = &now
	wait.DropReason = reason

	return s.SaveProject(pf)
}

// DeferWait pushes back the dates on a wait.
// For time waits, updates the 'after' field.
// For manual waits, updates the 'check_after' field.
func DeferWait(s *storage.Storage, waitID string, until time.Time) error {
	prefix := model.ExtractPrefix(waitID)
	if prefix == "" {
		return fmt.Errorf("invalid wait ID: %s", waitID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	wait := findWait(pf, waitID)
	if wait == nil {
		return fmt.Errorf("wait %s not found", waitID)
	}

	if wait.Status != model.WaitStatusOpen {
		return fmt.Errorf("wait %s is not open (status: %s)", waitID, wait.Status)
	}

	switch wait.ResolutionCriteria.Type {
	case model.ResolutionTypeTime:
		wait.ResolutionCriteria.After = &until
	case model.ResolutionTypeManual:
		wait.ResolutionCriteria.CheckAfter = &until
	default:
		return fmt.Errorf("cannot defer wait of type: %s", wait.ResolutionCriteria.Type)
	}

	return s.SaveProject(pf)
}

// AddWaitBlocker adds a blocker to a wait.
func AddWaitBlocker(s *storage.Storage, waitID, blockerID string) error {
	prefix := model.ExtractPrefix(waitID)
	if prefix == "" {
		return fmt.Errorf("invalid wait ID: %s", waitID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	wait := findWait(pf, waitID)
	if wait == nil {
		return fmt.Errorf("wait %s not found", waitID)
	}

	// Validate blocker exists
	if err := validateBlockers(pf, []string{blockerID}); err != nil {
		return err
	}

	// Check for existing blocker
	for _, bid := range wait.BlockedBy {
		if strings.EqualFold(bid, blockerID) {
			return fmt.Errorf("blocker %s already exists on wait %s", blockerID, waitID)
		}
	}

	// Check for cycles
	g := graph.BuildGraph(pf)
	if cycle := g.CheckCycle(waitID, blockerID); cycle != nil {
		return fmt.Errorf("adding blocker would create cycle: %s", strings.Join(cycle, " -> "))
	}

	wait.BlockedBy = append(wait.BlockedBy, model.NormalizeID(blockerID, pf.NextID-1))

	return s.SaveProject(pf)
}

// RemoveWaitBlocker removes a blocker from a wait.
func RemoveWaitBlocker(s *storage.Storage, waitID, blockerID string) error {
	prefix := model.ExtractPrefix(waitID)
	if prefix == "" {
		return fmt.Errorf("invalid wait ID: %s", waitID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	wait := findWait(pf, waitID)
	if wait == nil {
		return fmt.Errorf("wait %s not found", waitID)
	}

	// Find and remove blocker
	found := false
	for i, bid := range wait.BlockedBy {
		if strings.EqualFold(bid, blockerID) {
			wait.BlockedBy = append(wait.BlockedBy[:i], wait.BlockedBy[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("blocker %s not found on wait %s", blockerID, waitID)
	}

	return s.SaveProject(pf)
}
