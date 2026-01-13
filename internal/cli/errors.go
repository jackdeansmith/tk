package cli

import (
	"fmt"
	"strings"
)

// NotFoundError indicates a task, wait, or project was not found.
type NotFoundError struct {
	Type string // "task", "wait", or "project"
	ID   string // the ID that was not found
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %s not found", e.Type, e.ID)
}

// CycleError indicates a dependency cycle would be created.
type CycleError struct {
	From  string   // the task/wait being modified
	To    string   // the blocker being added
	Cycle []string // the cycle path
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("%s cannot depend on %s (would create cycle: %s)",
		e.From, e.To, strings.Join(e.Cycle, " -> "))
}

// ValidationError indicates a validation failure.
type ValidationError struct {
	Field   string // the field that failed validation
	Message string // what went wrong
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
	}
	return e.Message
}

// BlockedError indicates an operation cannot proceed due to blockers.
type BlockedError struct {
	ID       string   // the task/wait being operated on
	Blockers []string // the incomplete blockers
	Hint     string   // suggestion for how to proceed
}

func (e *BlockedError) Error() string {
	msg := fmt.Sprintf("%s has incomplete blockers: %s", e.ID, strings.Join(e.Blockers, ", "))
	if e.Hint != "" {
		msg += "\n" + e.Hint
	}
	return msg
}

// DependentsError indicates an operation cannot proceed due to dependents.
type DependentsError struct {
	ID         string   // the task/wait being operated on
	Dependents []string // items that depend on this
	Hint       string   // suggestion for how to proceed
}

func (e *DependentsError) Error() string {
	msg := fmt.Sprintf("%s has dependents: %s", e.ID, strings.Join(e.Dependents, ", "))
	if e.Hint != "" {
		msg += "\n" + e.Hint
	}
	return msg
}

// ProjectStatusError indicates a project status doesn't allow the operation.
type ProjectStatusError struct {
	Operation string // what was attempted
	Project   string // project name
	Status    string // current status
}

func (e *ProjectStatusError) Error() string {
	return fmt.Sprintf("cannot %s %s project %q", e.Operation, e.Status, e.Project)
}

// FormatError returns a user-friendly error message.
// It prefixes the error with "error: " for consistent CLI output.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return "error: " + err.Error()
}
