package ops

import (
	"fmt"
	"strings"

	"github.com/jacksmith/tk/internal/graph"
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
)

// ValidationErrorType represents the type of validation error.
type ValidationErrorType string

const (
	ValidationErrorOrphanBlocker   ValidationErrorType = "orphan_blocker"
	ValidationErrorCycle           ValidationErrorType = "cycle"
	ValidationErrorDuplicateID     ValidationErrorType = "duplicate_id"
	ValidationErrorInvalidID       ValidationErrorType = "invalid_id"
	ValidationErrorMissingRequired ValidationErrorType = "missing_required"
)

// ValidationError represents a data integrity issue.
type ValidationError struct {
	Type    ValidationErrorType
	ItemID  string
	Message string
	Details []string // Additional context (e.g., cycle path)
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s - %s", e.ItemID, e.Type, e.Message)
}

// ValidationFix represents an auto-repair action taken.
type ValidationFix struct {
	Type        ValidationErrorType
	ItemID      string
	Description string
}

// Validate checks all projects for data integrity issues.
// Returns a list of validation errors found.
func Validate(s *storage.Storage) ([]ValidationError, error) {
	prefixes, err := s.ListProjects()
	if err != nil {
		return nil, err
	}

	var allErrors []ValidationError
	for _, prefix := range prefixes {
		errors, err := validateProject(s, prefix)
		if err != nil {
			return nil, err
		}
		allErrors = append(allErrors, errors...)
	}

	return allErrors, nil
}

// validateProject validates a single project.
func validateProject(s *storage.Storage, prefix string) ([]ValidationError, error) {
	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, err
	}

	var errors []ValidationError

	// Build set of valid IDs
	validIDs := make(map[string]bool)
	for _, t := range pf.Tasks {
		validIDs[t.ID] = true
	}
	for _, w := range pf.Waits {
		validIDs[w.ID] = true
	}

	// Check for duplicate IDs
	seenIDs := make(map[string]bool)
	for _, t := range pf.Tasks {
		normalizedID := strings.ToUpper(t.ID)
		if seenIDs[normalizedID] {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorDuplicateID,
				ItemID:  t.ID,
				Message: "duplicate task ID",
			})
		}
		seenIDs[normalizedID] = true
	}
	for _, w := range pf.Waits {
		normalizedID := strings.ToUpper(w.ID)
		if seenIDs[normalizedID] {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorDuplicateID,
				ItemID:  w.ID,
				Message: "duplicate wait ID",
			})
		}
		seenIDs[normalizedID] = true
	}

	// Check for invalid IDs
	for _, t := range pf.Tasks {
		if !model.IsTaskID(t.ID) {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorInvalidID,
				ItemID:  t.ID,
				Message: "invalid task ID format",
			})
		}
	}
	for _, w := range pf.Waits {
		if !model.IsWaitID(w.ID) {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorInvalidID,
				ItemID:  w.ID,
				Message: "invalid wait ID format",
			})
		}
	}

	// Check for orphan blockers (references to non-existent items)
	for _, t := range pf.Tasks {
		for _, blockerID := range t.BlockedBy {
			normalizedID := model.NormalizeID(blockerID, 0)
			if !validIDs[normalizedID] && !validIDs[blockerID] {
				errors = append(errors, ValidationError{
					Type:    ValidationErrorOrphanBlocker,
					ItemID:  t.ID,
					Message: fmt.Sprintf("references non-existent blocker: %s", blockerID),
					Details: []string{blockerID},
				})
			}
		}
	}
	for _, w := range pf.Waits {
		for _, blockerID := range w.BlockedBy {
			normalizedID := model.NormalizeID(blockerID, 0)
			if !validIDs[normalizedID] && !validIDs[blockerID] {
				errors = append(errors, ValidationError{
					Type:    ValidationErrorOrphanBlocker,
					ItemID:  w.ID,
					Message: fmt.Sprintf("references non-existent blocker: %s", blockerID),
					Details: []string{blockerID},
				})
			}
		}
	}

	// Check for cycles
	g := graph.BuildGraph(pf)
	cycleErrors := detectCycles(pf, g)
	errors = append(errors, cycleErrors...)

	// Check for missing required fields
	for _, t := range pf.Tasks {
		if t.Title == "" {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorMissingRequired,
				ItemID:  t.ID,
				Message: "task missing required field: title",
			})
		}
	}
	for _, w := range pf.Waits {
		if w.ResolutionCriteria.Type == "" {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorMissingRequired,
				ItemID:  w.ID,
				Message: "wait missing required field: resolution_criteria.type",
			})
		}
		if w.ResolutionCriteria.Type == model.ResolutionTypeTime && w.ResolutionCriteria.After == nil {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorMissingRequired,
				ItemID:  w.ID,
				Message: "time wait missing required field: resolution_criteria.after",
			})
		}
		if w.ResolutionCriteria.Type == model.ResolutionTypeManual &&
			w.ResolutionCriteria.Question == "" && w.Title == "" {
			errors = append(errors, ValidationError{
				Type:    ValidationErrorMissingRequired,
				ItemID:  w.ID,
				Message: "manual wait missing required field: question or title",
			})
		}
	}

	return errors, nil
}

// detectCycles finds all cycles in the dependency graph.
func detectCycles(pf *model.ProjectFile, g *graph.Graph) []ValidationError {
	var errors []ValidationError
	visited := make(map[string]bool)

	// Check each node for cycles
	for _, t := range pf.Tasks {
		if visited[t.ID] {
			continue
		}
		for _, blockerID := range t.BlockedBy {
			if cycle := g.CheckCycle(t.ID, blockerID); cycle != nil {
				// Only report if this is a real existing cycle (not potential)
				// Check if the edge actually exists
				hasEdge := false
				for _, bid := range t.BlockedBy {
					if strings.EqualFold(bid, blockerID) {
						hasEdge = true
						break
					}
				}
				if hasEdge {
					errors = append(errors, ValidationError{
						Type:    ValidationErrorCycle,
						ItemID:  t.ID,
						Message: fmt.Sprintf("dependency cycle detected"),
						Details: cycle,
					})
					// Mark all items in cycle as visited to avoid duplicate reports
					for _, id := range cycle {
						visited[id] = true
					}
				}
			}
		}
	}

	for _, w := range pf.Waits {
		if visited[w.ID] {
			continue
		}
		for _, blockerID := range w.BlockedBy {
			if cycle := g.CheckCycle(w.ID, blockerID); cycle != nil {
				hasEdge := false
				for _, bid := range w.BlockedBy {
					if strings.EqualFold(bid, blockerID) {
						hasEdge = true
						break
					}
				}
				if hasEdge {
					errors = append(errors, ValidationError{
						Type:    ValidationErrorCycle,
						ItemID:  w.ID,
						Message: fmt.Sprintf("dependency cycle detected"),
						Details: cycle,
					})
					for _, id := range cycle {
						visited[id] = true
					}
				}
			}
		}
	}

	return errors
}

// ValidateAndFix validates all projects and auto-repairs fixable issues.
// Returns a list of fixes that were applied.
func ValidateAndFix(s *storage.Storage) ([]ValidationFix, error) {
	prefixes, err := s.ListProjects()
	if err != nil {
		return nil, err
	}

	var allFixes []ValidationFix
	for _, prefix := range prefixes {
		fixes, err := validateAndFixProject(s, prefix)
		if err != nil {
			return nil, err
		}
		allFixes = append(allFixes, fixes...)
	}

	return allFixes, nil
}

// validateAndFixProject validates and fixes a single project.
func validateAndFixProject(s *storage.Storage, prefix string) ([]ValidationFix, error) {
	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, err
	}

	var fixes []ValidationFix
	modified := false

	// Build set of valid IDs
	validIDs := make(map[string]bool)
	for _, t := range pf.Tasks {
		validIDs[strings.ToUpper(t.ID)] = true
	}
	for _, w := range pf.Waits {
		validIDs[strings.ToUpper(w.ID)] = true
	}

	// Fix orphan blockers by removing them
	for i := range pf.Tasks {
		t := &pf.Tasks[i]
		var cleanBlockers []string
		for _, blockerID := range t.BlockedBy {
			normalizedID := strings.ToUpper(model.NormalizeID(blockerID, 0))
			if validIDs[normalizedID] {
				cleanBlockers = append(cleanBlockers, blockerID)
			} else {
				fixes = append(fixes, ValidationFix{
					Type:        ValidationErrorOrphanBlocker,
					ItemID:      t.ID,
					Description: fmt.Sprintf("removed orphan blocker: %s", blockerID),
				})
				modified = true
			}
		}
		t.BlockedBy = cleanBlockers
	}

	for i := range pf.Waits {
		w := &pf.Waits[i]
		var cleanBlockers []string
		for _, blockerID := range w.BlockedBy {
			normalizedID := strings.ToUpper(model.NormalizeID(blockerID, 0))
			if validIDs[normalizedID] {
				cleanBlockers = append(cleanBlockers, blockerID)
			} else {
				fixes = append(fixes, ValidationFix{
					Type:        ValidationErrorOrphanBlocker,
					ItemID:      w.ID,
					Description: fmt.Sprintf("removed orphan blocker: %s", blockerID),
				})
				modified = true
			}
		}
		w.BlockedBy = cleanBlockers
	}

	if modified {
		if err := s.SaveProject(pf); err != nil {
			return nil, err
		}
	}

	return fixes, nil
}

// ValidateProject validates a single project by prefix.
func ValidateProject(s *storage.Storage, prefix string) ([]ValidationError, error) {
	return validateProject(s, prefix)
}
