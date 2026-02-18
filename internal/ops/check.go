package ops

import (
	"time"

	"github.com/jacksmith/tk/internal/model"
)

// CheckResult contains the results of running auto-resolution check.
type CheckResult struct {
	// ResolvedWaits lists waits that were auto-resolved (time waits that passed).
	ResolvedWaits []string
	// Unblocked lists tasks/waits that are now unblocked.
	Unblocked []string
	// AutoCompleted lists tasks that were auto-completed as a cascade.
	AutoCompleted []string
}

// RunCheck auto-resolves time-based waits that have passed their 'after' date.
// Returns information about all cascading effects.
func RunCheck(s Store) (*CheckResult, error) {
	return RunCheckAt(s, time.Now())
}

// RunCheckAt runs the check using the specified time (useful for testing).
func RunCheckAt(s Store, now time.Time) (*CheckResult, error) {
	result := &CheckResult{}

	// Get all projects
	prefixes, err := s.ListProjects()
	if err != nil {
		return nil, err
	}

	for _, prefix := range prefixes {
		projectResult, err := runCheckOnProject(s, prefix, now)
		if err != nil {
			return nil, err
		}

		result.ResolvedWaits = append(result.ResolvedWaits, projectResult.ResolvedWaits...)
		result.Unblocked = append(result.Unblocked, projectResult.Unblocked...)
		result.AutoCompleted = append(result.AutoCompleted, projectResult.AutoCompleted...)
	}

	return result, nil
}

// runCheckOnProject runs the check on a single project.
func runCheckOnProject(s Store, prefix string, now time.Time) (*CheckResult, error) {
	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, err
	}

	result := &CheckResult{}
	modified := false

	// Build initial blocker states
	blockerStates := ComputeBlockerStates(pf)

	// Find time waits that are ready to resolve
	for i := range pf.Waits {
		w := &pf.Waits[i]

		// Skip if not open
		if w.Status != model.WaitStatusOpen {
			continue
		}

		// Skip if not a time wait
		if w.ResolutionCriteria.Type != model.ResolutionTypeTime {
			continue
		}

		// Skip if no 'after' date
		if w.ResolutionCriteria.After == nil {
			continue
		}

		// Check if wait is dormant (has unresolved blockers)
		isDormant := false
		for _, blockerID := range w.BlockedBy {
			if resolved, ok := blockerStates[blockerID]; !ok || !resolved {
				isDormant = true
				break
			}
		}
		if isDormant {
			continue
		}

		// Check if time has passed
		if now.Before(*w.ResolutionCriteria.After) {
			continue
		}

		// Auto-resolve this wait
		doneAt := now
		w.Status = model.WaitStatusDone
		w.DoneAt = &doneAt
		blockerStates[w.ID] = true
		result.ResolvedWaits = append(result.ResolvedWaits, w.ID)
		modified = true
	}

	// If any waits were resolved, check for cascading effects
	if len(result.ResolvedWaits) > 0 {
		// Find items that are now unblocked
		result.Unblocked = findNewlyUnblocked(pf, blockerStates)

		// Handle auto-complete cascade
		autoCompleted := processAutoComplete(pf, blockerStates)
		result.AutoCompleted = autoCompleted
		if len(autoCompleted) > 0 {
			modified = true
		}
	}

	if modified {
		if err := s.SaveProject(pf); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// findNewlyUnblocked finds items that have all blockers resolved.
func findNewlyUnblocked(pf *model.ProjectFile, blockerStates model.BlockerStatus) []string {
	var unblocked []string

	// Check tasks
	for _, t := range pf.Tasks {
		if t.Status != model.TaskStatusOpen {
			continue
		}
		if len(t.BlockedBy) == 0 {
			continue
		}

		allResolved := true
		for _, bid := range t.BlockedBy {
			if resolved, ok := blockerStates[bid]; !ok || !resolved {
				allResolved = false
				break
			}
		}

		if allResolved {
			unblocked = append(unblocked, t.ID)
		}
	}

	// Check waits
	for _, w := range pf.Waits {
		if w.Status != model.WaitStatusOpen {
			continue
		}
		if len(w.BlockedBy) == 0 {
			continue
		}

		allResolved := true
		for _, bid := range w.BlockedBy {
			if resolved, ok := blockerStates[bid]; !ok || !resolved {
				allResolved = false
				break
			}
		}

		if allResolved {
			unblocked = append(unblocked, w.ID)
		}
	}

	return unblocked
}
