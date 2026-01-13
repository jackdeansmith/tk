package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{Type: "task", ID: "BY-999"}
	assert.Equal(t, "task BY-999 not found", err.Error())

	err = &NotFoundError{Type: "wait", ID: "BY-03W"}
	assert.Equal(t, "wait BY-03W not found", err.Error())

	err = &NotFoundError{Type: "project", ID: "backyard"}
	assert.Equal(t, "project backyard not found", err.Error())
}

func TestCycleError(t *testing.T) {
	err := &CycleError{
		From:  "BY-07",
		To:    "BY-05",
		Cycle: []string{"BY-05", "BY-03", "BY-07", "BY-05"},
	}
	expected := "BY-07 cannot depend on BY-05 (would create cycle: BY-05 -> BY-03 -> BY-07 -> BY-05)"
	assert.Equal(t, expected, err.Error())
}

func TestValidationError(t *testing.T) {
	// With field
	err := &ValidationError{Field: "prefix", Message: "must be 2-3 uppercase letters"}
	assert.Equal(t, "invalid prefix: must be 2-3 uppercase letters", err.Error())

	// Without field
	err = &ValidationError{Message: "project name is required"}
	assert.Equal(t, "project name is required", err.Error())
}

func TestBlockedError(t *testing.T) {
	// Without hint
	err := &BlockedError{
		ID:       "BY-07",
		Blockers: []string{"BY-05", "BY-03W"},
	}
	assert.Equal(t, "BY-07 has incomplete blockers: BY-05, BY-03W", err.Error())

	// With hint
	err = &BlockedError{
		ID:       "BY-07",
		Blockers: []string{"BY-05"},
		Hint:     "Use --force to complete anyway.",
	}
	expected := "BY-07 has incomplete blockers: BY-05\nUse --force to complete anyway."
	assert.Equal(t, expected, err.Error())
}

func TestDependentsError(t *testing.T) {
	// Without hint
	err := &DependentsError{
		ID:         "BY-07",
		Dependents: []string{"BY-08", "BY-09"},
	}
	assert.Equal(t, "BY-07 has dependents: BY-08, BY-09", err.Error())

	// With hint
	err = &DependentsError{
		ID:         "BY-07",
		Dependents: []string{"BY-08"},
		Hint:       "Use --drop-deps to drop them too, or --remove-deps to unlink them.",
	}
	expected := "BY-07 has dependents: BY-08\nUse --drop-deps to drop them too, or --remove-deps to unlink them."
	assert.Equal(t, expected, err.Error())
}

func TestProjectStatusError(t *testing.T) {
	err := &ProjectStatusError{
		Operation: "add task to",
		Project:   "backyard",
		Status:    "paused",
	}
	assert.Equal(t, "cannot add task to paused project \"backyard\"", err.Error())

	err = &ProjectStatusError{
		Operation: "add task to",
		Project:   "archive",
		Status:    "done",
	}
	assert.Equal(t, "cannot add task to done project \"archive\"", err.Error())
}

func TestFormatError(t *testing.T) {
	// nil error
	assert.Equal(t, "", FormatError(nil))

	// Simple error
	assert.Equal(t, "error: something went wrong", FormatError(errors.New("something went wrong")))

	// NotFoundError
	err := &NotFoundError{Type: "task", ID: "BY-999"}
	assert.Equal(t, "error: task BY-999 not found", FormatError(err))

	// CycleError
	cycleErr := &CycleError{
		From:  "BY-07",
		To:    "BY-05",
		Cycle: []string{"BY-05", "BY-07", "BY-05"},
	}
	assert.Equal(t, "error: BY-07 cannot depend on BY-05 (would create cycle: BY-05 -> BY-07 -> BY-05)",
		FormatError(cycleErr))
}
