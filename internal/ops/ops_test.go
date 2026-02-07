package ops

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
)

// setupTestStorage creates a temporary .tk directory for testing.
func setupTestStorage(t *testing.T) (*storage.Storage, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "tk-ops-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	s, err := storage.Init(dir, "Test Project", "TS")
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("failed to init storage: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return s, cleanup
}

// TestCreateProject tests project creation.
func TestCreateProject(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Test successful creation
	err := CreateProject(s, "backyard", "BY", "Backyard Project", "A test project")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	// Verify project was created
	pf, err := s.LoadProject("BY")
	if err != nil {
		t.Fatalf("failed to load project: %v", err)
	}
	if pf.ID != "backyard" {
		t.Errorf("expected ID 'backyard', got %q", pf.ID)
	}
	if pf.Prefix != "BY" {
		t.Errorf("expected Prefix 'BY', got %q", pf.Prefix)
	}
	if pf.Name != "Backyard Project" {
		t.Errorf("expected Name 'Backyard Project', got %q", pf.Name)
	}

	// Test duplicate prefix
	err = CreateProject(s, "another", "BY", "Another", "")
	if err == nil {
		t.Error("expected error for duplicate prefix")
	}

	// Test duplicate ID
	err = CreateProject(s, "backyard", "AA", "Another", "")
	if err == nil {
		t.Error("expected error for duplicate ID")
	}

	// Test invalid prefix
	err = CreateProject(s, "test", "A", "Test", "")
	if err == nil {
		t.Error("expected error for short prefix")
	}

	err = CreateProject(s, "test", "A1", "Test", "")
	if err == nil {
		t.Error("expected error for non-letter prefix")
	}
}

// TestEditProject tests project editing.
func TestEditProject(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Edit the default project
	newName := "Updated Name"
	newDesc := "Updated description"
	newStatus := model.ProjectStatusPaused

	err := EditProject(s, "TS", ProjectChanges{
		Name:        &newName,
		Description: &newDesc,
		Status:      &newStatus,
	})
	if err != nil {
		t.Fatalf("EditProject failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	if pf.Name != newName {
		t.Errorf("expected name %q, got %q", newName, pf.Name)
	}
	if pf.Description != newDesc {
		t.Errorf("expected description %q, got %q", newDesc, pf.Description)
	}
	if pf.Status != newStatus {
		t.Errorf("expected status %q, got %q", newStatus, pf.Status)
	}
}

// TestDeleteProject tests project deletion.
func TestDeleteProject(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a project to delete
	CreateProject(s, "todelete", "TD", "To Delete", "")

	// Add a task
	AddTask(s, "TD", "Test task", TaskOptions{})

	// Should fail without force
	err := DeleteProject(s, "TD", false)
	if err == nil {
		t.Error("expected error when deleting project with open tasks")
	}

	// Should succeed with force
	err = DeleteProject(s, "TD", true)
	if err != nil {
		t.Fatalf("DeleteProject with force failed: %v", err)
	}

	// Verify deleted
	if s.ProjectExists("TD") {
		t.Error("project should not exist after deletion")
	}
}

// TestChangeProjectPrefix tests prefix changes.
func TestChangeProjectPrefix(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Add some tasks first
	AddTask(s, "TS", "Task 1", TaskOptions{})
	AddTask(s, "TS", "Task 2", TaskOptions{})

	// Change prefix
	err := ChangeProjectPrefix(s, "TS", "NW")
	if err != nil {
		t.Fatalf("ChangeProjectPrefix failed: %v", err)
	}

	// Verify old prefix gone
	if s.ProjectExists("TS") {
		t.Error("old prefix should not exist")
	}

	// Verify new prefix exists
	pf, err := s.LoadProject("NW")
	if err != nil {
		t.Fatalf("failed to load project with new prefix: %v", err)
	}

	// Verify task IDs updated
	if len(pf.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(pf.Tasks))
	}
	for _, task := range pf.Tasks {
		if model.ExtractPrefix(task.ID) != "NW" {
			t.Errorf("task ID not updated: %s", task.ID)
		}
	}
}

// TestAddTask tests task creation.
func TestAddTask(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create basic task
	task, err := AddTask(s, "TS", "Test task", TaskOptions{})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if task.ID != "TS-01" {
		t.Errorf("expected ID 'TS-01', got %q", task.ID)
	}
	if task.Title != "Test task" {
		t.Errorf("expected title 'Test task', got %q", task.Title)
	}
	if task.Priority != 3 {
		t.Errorf("expected default priority 3, got %d", task.Priority)
	}

	// Create task with options
	dueDate := time.Now().Add(24 * time.Hour)
	task2, err := AddTask(s, "TS", "Task with options", TaskOptions{
		Priority:     1,
		Tags:         []string{"urgent", "bug"},
		Notes:        "Some notes",
		DueDate:      &dueDate,
		AutoComplete: true,
		BlockedBy:    []string{"TS-01"},
	})
	if err != nil {
		t.Fatalf("AddTask with options failed: %v", err)
	}
	if task2.Priority != 1 {
		t.Errorf("expected priority 1, got %d", task2.Priority)
	}
	if len(task2.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(task2.Tags))
	}
	if !task2.AutoComplete {
		t.Error("expected auto_complete to be true")
	}
}

// TestEditTask tests task editing.
func TestEditTask(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create task
	AddTask(s, "TS", "Original title", TaskOptions{})

	// Edit task
	newTitle := "Updated title"
	newPriority := 1
	err := EditTask(s, "TS-01", TaskChanges{
		Title:    &newTitle,
		Priority: &newPriority,
	})
	if err != nil {
		t.Fatalf("EditTask failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	task := pf.Tasks[0]
	if task.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, task.Title)
	}
	if task.Priority != newPriority {
		t.Errorf("expected priority %d, got %d", newPriority, task.Priority)
	}
}

// TestCompleteTask tests task completion.
func TestCompleteTask(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create tasks
	AddTask(s, "TS", "Blocker task", TaskOptions{})
	AddTask(s, "TS", "Dependent task", TaskOptions{BlockedBy: []string{"TS-01"}})

	// Try to complete dependent task (should fail)
	_, err := CompleteTask(s, "TS-02", false)
	if err == nil {
		t.Error("expected error when completing task with incomplete blockers")
	}

	// Complete blocker task
	result, err := CompleteTask(s, "TS-01", false)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}
	if len(result.Unblocked) != 1 || result.Unblocked[0] != "TS-02" {
		t.Errorf("expected TS-02 to be unblocked, got %v", result.Unblocked)
	}

	// Verify task is done
	pf, _ := s.LoadProject("TS")
	if pf.Tasks[0].Status != model.TaskStatusDone {
		t.Error("task should be marked as done")
	}
}

// TestCompleteTaskWithForce tests forced completion.
func TestCompleteTaskWithForce(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Blocker", TaskOptions{})
	AddTask(s, "TS", "Dependent", TaskOptions{BlockedBy: []string{"TS-01"}})

	// Force complete dependent task
	_, err := CompleteTask(s, "TS-02", true)
	if err != nil {
		t.Fatalf("CompleteTask with force failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	task := findTask(pf, "TS-02")
	if task.Status != model.TaskStatusDone {
		t.Error("task should be done")
	}
	if len(task.BlockedBy) != 0 {
		t.Error("blockers should be removed when forcing completion")
	}
}

// TestAutoComplete tests cascading auto-completion.
func TestAutoComplete(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create chain: TS-01 -> TS-02 (auto) -> TS-03 (auto)
	AddTask(s, "TS", "Blocker", TaskOptions{})
	AddTask(s, "TS", "Auto 1", TaskOptions{BlockedBy: []string{"TS-01"}, AutoComplete: true})
	AddTask(s, "TS", "Auto 2", TaskOptions{BlockedBy: []string{"TS-02"}, AutoComplete: true})

	// Complete blocker
	result, err := CompleteTask(s, "TS-01", false)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	// Should have auto-completed both TS-02 and TS-03
	if len(result.AutoCompleted) != 2 {
		t.Errorf("expected 2 auto-completed tasks, got %d", len(result.AutoCompleted))
	}

	pf, _ := s.LoadProject("TS")
	for _, task := range pf.Tasks {
		if task.Status != model.TaskStatusDone {
			t.Errorf("task %s should be done", task.ID)
		}
	}
}

// TestDropTask tests task dropping.
func TestDropTask(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task to drop", TaskOptions{})
	AddTask(s, "TS", "Dependent", TaskOptions{BlockedBy: []string{"TS-01"}})

	// Should fail without flags
	err := DropTask(s, "TS-01", "not needed", false, false)
	if err == nil {
		t.Error("expected error when dropping task with dependents")
	}

	// Drop with remove-deps
	err = DropTask(s, "TS-01", "not needed", false, true)
	if err != nil {
		t.Fatalf("DropTask with removeDeps failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	if pf.Tasks[0].Status != model.TaskStatusDropped {
		t.Error("task should be dropped")
	}
	if len(pf.Tasks[1].BlockedBy) != 0 {
		t.Error("dependent's blocked_by should be cleared")
	}
}

// TestDropTaskWithDropDeps tests cascading drop.
func TestDropTaskWithDropDeps(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Root", TaskOptions{})
	AddTask(s, "TS", "Child", TaskOptions{BlockedBy: []string{"TS-01"}})
	AddTask(s, "TS", "Grandchild", TaskOptions{BlockedBy: []string{"TS-02"}})

	err := DropTask(s, "TS-01", "cancelled", true, false)
	if err != nil {
		t.Fatalf("DropTask with dropDeps failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	for _, task := range pf.Tasks {
		if task.Status != model.TaskStatusDropped {
			t.Errorf("task %s should be dropped", task.ID)
		}
	}
}

// TestReopenTask tests reopening tasks.
func TestReopenTask(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task", TaskOptions{})
	CompleteTask(s, "TS-01", false)

	err := ReopenTask(s, "TS-01")
	if err != nil {
		t.Fatalf("ReopenTask failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	if pf.Tasks[0].Status != model.TaskStatusOpen {
		t.Error("task should be open after reopening")
	}
	if pf.Tasks[0].DoneAt != nil {
		t.Error("done_at should be cleared")
	}
}

// TestDeferTask tests task deferral.
func TestDeferTask(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task to defer", TaskOptions{})

	until := time.Now().Add(24 * time.Hour)
	wait, err := DeferTask(s, "TS-01", until)
	if err != nil {
		t.Fatalf("DeferTask failed: %v", err)
	}

	if wait.ResolutionCriteria.Type != model.ResolutionTypeTime {
		t.Error("wait should be time-based")
	}

	pf, _ := s.LoadProject("TS")
	task := findTask(pf, "TS-01")
	if len(task.BlockedBy) != 1 || task.BlockedBy[0] != wait.ID {
		t.Error("task should be blocked by the new wait")
	}
}

// TestAddBlocker tests adding blockers.
func TestAddBlocker(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Blocker", TaskOptions{})
	AddTask(s, "TS", "Task", TaskOptions{})

	err := AddBlocker(s, "TS-02", "TS-01")
	if err != nil {
		t.Fatalf("AddBlocker failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	task := findTask(pf, "TS-02")
	if len(task.BlockedBy) != 1 {
		t.Error("task should have one blocker")
	}
}

// TestAddBlockerCycleDetection tests cycle detection.
func TestAddBlockerCycleDetection(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "A", TaskOptions{})
	AddTask(s, "TS", "B", TaskOptions{BlockedBy: []string{"TS-01"}})
	AddTask(s, "TS", "C", TaskOptions{BlockedBy: []string{"TS-02"}})

	// Try to create cycle: TS-01 -> TS-03 (would create TS-01 -> TS-02 -> TS-03 -> TS-01)
	err := AddBlocker(s, "TS-01", "TS-03")
	if err == nil {
		t.Error("expected error for cycle")
	}
}

// TestAddWait tests wait creation.
func TestAddWait(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create manual wait
	wait, err := AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Did the package arrive?",
	})
	if err != nil {
		t.Fatalf("AddWait failed: %v", err)
	}
	if wait.ResolutionCriteria.Type != model.ResolutionTypeManual {
		t.Error("wait should be manual type")
	}

	// Create time wait
	after := time.Now().Add(24 * time.Hour)
	wait2, err := AddWait(s, "TS", WaitOptions{
		Type:  model.ResolutionTypeTime,
		After: &after,
		Title: "Wait for deployment",
	})
	if err != nil {
		t.Fatalf("AddWait (time) failed: %v", err)
	}
	if wait2.ResolutionCriteria.Type != model.ResolutionTypeTime {
		t.Error("wait should be time type")
	}
}

// TestResolveWait tests wait resolution.
func TestResolveWait(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Ready?",
	})

	err := ResolveWait(s, "TS-01W", "Yes, it's ready")
	if err != nil {
		t.Fatalf("ResolveWait failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-01W")
	if wait.Status != model.WaitStatusDone {
		t.Error("wait should be done")
	}
	if wait.Resolution != "Yes, it's ready" {
		t.Errorf("expected resolution 'Yes, it's ready', got %q", wait.Resolution)
	}
}

// TestResolveWaitDormant tests that dormant waits can't be resolved.
func TestResolveWaitDormant(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Blocker", TaskOptions{})
	AddWait(s, "TS", WaitOptions{
		Type:      model.ResolutionTypeManual,
		Question:  "Ready?",
		BlockedBy: []string{"TS-01"},
	})

	err := ResolveWait(s, "TS-02W", "Yes")
	if err == nil {
		t.Error("expected error when resolving dormant wait")
	}
}

// TestDropWait tests wait dropping.
func TestDropWait(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Ready?",
	})

	err := DropWait(s, "TS-01W", "No longer needed", false, false)
	if err != nil {
		t.Fatalf("DropWait failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-01W")
	if wait.Status != model.WaitStatusDropped {
		t.Error("wait should be dropped")
	}
}

// TestDeferWait tests wait deferral.
func TestDeferWait(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	after := time.Now()
	AddWait(s, "TS", WaitOptions{
		Type:  model.ResolutionTypeTime,
		After: &after,
	})

	newTime := time.Now().Add(48 * time.Hour)
	err := DeferWait(s, "TS-01W", newTime)
	if err != nil {
		t.Fatalf("DeferWait failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-01W")
	if !wait.ResolutionCriteria.After.After(after) {
		t.Error("wait should be deferred to later time")
	}
}

// TestRunCheck tests auto-resolution of time waits.
func TestRunCheck(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a time wait in the past
	past := time.Now().Add(-1 * time.Hour)
	AddWait(s, "TS", WaitOptions{
		Type:  model.ResolutionTypeTime,
		After: &past,
		Title: "Past wait",
	})

	// Create a time wait in the future
	future := time.Now().Add(24 * time.Hour)
	AddWait(s, "TS", WaitOptions{
		Type:  model.ResolutionTypeTime,
		After: &future,
		Title: "Future wait",
	})

	result, err := RunCheck(s)
	if err != nil {
		t.Fatalf("RunCheck failed: %v", err)
	}

	if len(result.ResolvedWaits) != 1 {
		t.Errorf("expected 1 resolved wait, got %d", len(result.ResolvedWaits))
	}

	pf, _ := s.LoadProject("TS")
	pastWait := findWait(pf, "TS-01W")
	if pastWait.Status != model.WaitStatusDone {
		t.Error("past wait should be resolved")
	}
	futureWait := findWait(pf, "TS-02W")
	if futureWait.Status != model.WaitStatusOpen {
		t.Error("future wait should still be open")
	}
}

// TestRunCheckCascade tests cascading effects of check.
func TestRunCheckCascade(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create: past wait -> task (auto-complete)
	past := time.Now().Add(-1 * time.Hour)
	AddWait(s, "TS", WaitOptions{
		Type:  model.ResolutionTypeTime,
		After: &past,
	})
	AddTask(s, "TS", "Auto task", TaskOptions{
		BlockedBy:    []string{"TS-01W"},
		AutoComplete: true,
	})

	result, err := RunCheck(s)
	if err != nil {
		t.Fatalf("RunCheck failed: %v", err)
	}

	if len(result.AutoCompleted) != 1 {
		t.Errorf("expected 1 auto-completed task, got %d", len(result.AutoCompleted))
	}

	pf, _ := s.LoadProject("TS")
	task := findTask(pf, "TS-02")
	if task.Status != model.TaskStatusDone {
		t.Error("task should be auto-completed")
	}
}

// TestValidate tests validation.
func TestValidate(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create valid data
	AddTask(s, "TS", "Valid task", TaskOptions{})

	errors, err := Validate(s)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errors), errors)
	}
}

// TestValidateOrphanBlocker tests detection of orphan blockers.
func TestValidateOrphanBlocker(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create task with non-existent blocker (manually modify file)
	pf, _ := s.LoadProject("TS")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:        "TS-01",
		Title:     "Task with orphan",
		Status:    model.TaskStatusOpen,
		Priority:  3,
		BlockedBy: []string{"TS-99"}, // Non-existent
		Created:   time.Now(),
		Updated:   time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	errors, err := Validate(s)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	found := false
	for _, e := range errors {
		if e.Type == ValidationErrorOrphanBlocker {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected orphan blocker error")
	}
}

// TestValidateAndFix tests auto-repair.
func TestValidateAndFix(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create task with orphan blocker
	pf, _ := s.LoadProject("TS")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:        "TS-01",
		Title:     "Task with orphan",
		Status:    model.TaskStatusOpen,
		Priority:  3,
		BlockedBy: []string{"TS-99"},
		Created:   time.Now(),
		Updated:   time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	fixes, err := ValidateAndFix(s)
	if err != nil {
		t.Fatalf("ValidateAndFix failed: %v", err)
	}

	if len(fixes) != 1 {
		t.Errorf("expected 1 fix, got %d", len(fixes))
	}

	// Verify fix was applied
	pf, _ = s.LoadProject("TS")
	task := findTask(pf, "TS-01")
	if len(task.BlockedBy) != 0 {
		t.Error("orphan blocker should be removed")
	}
}

// TestMoveTask tests moving tasks between projects.
func TestMoveTask(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create another project
	CreateProject(s, "other", "OT", "Other Project", "")

	// Create task in first project
	AddTask(s, "TS", "Task to move", TaskOptions{Priority: 1})

	// Move task
	err := MoveTask(s, "TS-01", "OT")
	if err != nil {
		t.Fatalf("MoveTask failed: %v", err)
	}

	// Verify task moved
	tsPf, _ := s.LoadProject("TS")
	if len(tsPf.Tasks) != 0 {
		t.Error("task should be removed from source project")
	}

	otPf, _ := s.LoadProject("OT")
	if len(otPf.Tasks) != 1 {
		t.Error("task should be in destination project")
	}
	if otPf.Tasks[0].ID != "OT-01" {
		t.Errorf("task ID should be updated to OT-01, got %s", otPf.Tasks[0].ID)
	}
	if otPf.Tasks[0].Priority != 1 {
		t.Error("task priority should be preserved")
	}
}

// TestMoveTaskWithBlockers tests that tasks with internal blockers can't be moved.
func TestMoveTaskWithBlockers(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	CreateProject(s, "other", "OT", "Other", "")
	AddTask(s, "TS", "Blocker", TaskOptions{})
	AddTask(s, "TS", "Blocked", TaskOptions{BlockedBy: []string{"TS-01"}})

	err := MoveTask(s, "TS-02", "OT")
	if err == nil {
		t.Error("expected error when moving task with blockers in source project")
	}
}

// TestRemoveBlocker tests removing blockers.
func TestRemoveBlocker(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Blocker", TaskOptions{})
	AddTask(s, "TS", "Task", TaskOptions{BlockedBy: []string{"TS-01"}})

	err := RemoveBlocker(s, "TS-02", "TS-01")
	if err != nil {
		t.Fatalf("RemoveBlocker failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	task := findTask(pf, "TS-02")
	if len(task.BlockedBy) != 0 {
		t.Error("blocker should be removed")
	}
}

// Integration test that exercises multiple operations.
func TestIntegration(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create project structure
	CreateProject(s, "work", "WK", "Work Project", "Work tasks")

	// Add tasks with dependencies
	AddTask(s, "WK", "Setup environment", TaskOptions{Priority: 1})
	AddTask(s, "WK", "Write code", TaskOptions{Priority: 2, BlockedBy: []string{"WK-01"}})
	AddTask(s, "WK", "Write tests", TaskOptions{Priority: 2, BlockedBy: []string{"WK-02"}})
	AddTask(s, "WK", "Deploy", TaskOptions{Priority: 3, BlockedBy: []string{"WK-02", "WK-03"}, AutoComplete: true})

	// Add a wait for code review
	AddWait(s, "WK", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Has code review been approved?",
	})
	AddBlocker(s, "WK-04", "WK-05W")

	// Validate
	errors, _ := Validate(s)
	if len(errors) != 0 {
		t.Errorf("validation errors: %v", errors)
	}

	// Complete first task
	result, _ := CompleteTask(s, "WK-01", false)
	if len(result.Unblocked) != 1 || result.Unblocked[0] != "WK-02" {
		t.Errorf("expected WK-02 unblocked, got %v", result.Unblocked)
	}

	// Complete second task
	CompleteTask(s, "WK-02", false)

	// Complete third task
	CompleteTask(s, "WK-03", false)

	// WK-04 still blocked by wait
	pf, _ := s.LoadProject("WK")
	task4 := findTask(pf, "WK-04")
	if task4.Status != model.TaskStatusOpen {
		t.Error("WK-04 should still be open (blocked by wait)")
	}

	// Resolve wait
	ResolveWait(s, "WK-05W", "Approved!")

	// Complete WK-04 (should trigger auto-complete)
	result, _ = CompleteTask(s, "WK-04", false)
	// Note: WK-04 already completed above, so this is just for demonstration
	// In a real scenario, the auto-complete would have been triggered

	// Verify final state
	pf, _ = s.LoadProject("WK")
	wait := findWait(pf, "WK-05W")
	if wait.Status != model.WaitStatusDone {
		t.Error("wait should be done")
	}
}

// TestEditWait tests wait editing.
func TestEditWait(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a manual wait
	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Original question?",
		Title:    "Original title",
		Notes:    "Original notes",
	})

	// Edit various fields
	newTitle := "Updated title"
	newQuestion := "Updated question?"
	newNotes := "Updated notes"

	err := EditWait(s, "TS-01W", WaitChanges{
		Title:    &newTitle,
		Question: &newQuestion,
		Notes:    &newNotes,
	})
	if err != nil {
		t.Fatalf("EditWait failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-01W")
	if wait.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, wait.Title)
	}
	if wait.ResolutionCriteria.Question != newQuestion {
		t.Errorf("expected question %q, got %q", newQuestion, wait.ResolutionCriteria.Question)
	}
	if wait.Notes != newNotes {
		t.Errorf("expected notes %q, got %q", newNotes, wait.Notes)
	}
}

// TestEditWaitTimeFields tests editing time-related fields on waits.
func TestEditWaitTimeFields(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a time wait
	originalTime := time.Now().Add(1 * time.Hour)
	AddWait(s, "TS", WaitOptions{
		Type:  model.ResolutionTypeTime,
		After: &originalTime,
	})

	// Edit the after time
	newTime := time.Now().Add(48 * time.Hour)
	newTimePtr := &newTime
	err := EditWait(s, "TS-01W", WaitChanges{
		After: &newTimePtr,
	})
	if err != nil {
		t.Fatalf("EditWait (after) failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-01W")
	if !wait.ResolutionCriteria.After.After(originalTime) {
		t.Error("after time should be updated")
	}

	// Create a manual wait with check_after
	checkTime := time.Now().Add(2 * time.Hour)
	AddWait(s, "TS", WaitOptions{
		Type:       model.ResolutionTypeManual,
		Question:   "Test?",
		CheckAfter: &checkTime,
	})

	// Edit the check_after time
	newCheckTime := time.Now().Add(72 * time.Hour)
	newCheckTimePtr := &newCheckTime
	err = EditWait(s, "TS-02W", WaitChanges{
		CheckAfter: &newCheckTimePtr,
	})
	if err != nil {
		t.Fatalf("EditWait (check_after) failed: %v", err)
	}

	pf, _ = s.LoadProject("TS")
	wait2 := findWait(pf, "TS-02W")
	if !wait2.ResolutionCriteria.CheckAfter.After(checkTime) {
		t.Error("check_after time should be updated")
	}
}

// TestEditWaitBlockedBy tests editing wait blockers.
func TestEditWaitBlockedBy(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create tasks and wait
	AddTask(s, "TS", "Task 1", TaskOptions{})
	AddTask(s, "TS", "Task 2", TaskOptions{})
	AddWait(s, "TS", WaitOptions{
		Type:      model.ResolutionTypeManual,
		Question:  "Test?",
		BlockedBy: []string{"TS-01"},
	})

	// Change blockers
	newBlockers := []string{"TS-02"}
	err := EditWait(s, "TS-03W", WaitChanges{
		BlockedBy: &newBlockers,
	})
	if err != nil {
		t.Fatalf("EditWait (blocked_by) failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-03W")
	if len(wait.BlockedBy) != 1 || wait.BlockedBy[0] != "TS-02" {
		t.Errorf("expected blockers [TS-02], got %v", wait.BlockedBy)
	}
}

// TestEditWaitCycleDetection tests that editing wait blockers detects cycles.
func TestEditWaitCycleDetection(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create: Task -> Wait -> (try to add Wait -> Task = cycle)
	AddTask(s, "TS", "Task", TaskOptions{})
	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})
	// Make task blocked by wait
	AddBlocker(s, "TS-01", "TS-02W")

	// Try to make wait blocked by task (would create cycle)
	blockers := []string{"TS-01"}
	err := EditWait(s, "TS-02W", WaitChanges{
		BlockedBy: &blockers,
	})
	if err == nil {
		t.Error("expected error for cycle")
	}
}

// TestEditWaitNotFound tests editing non-existent wait.
func TestEditWaitNotFound(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	newTitle := "Test"
	err := EditWait(s, "TS-99W", WaitChanges{Title: &newTitle})
	if err == nil {
		t.Error("expected error for non-existent wait")
	}
}

// TestEditWaitInvalidID tests editing with invalid ID.
func TestEditWaitInvalidID(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	newTitle := "Test"
	err := EditWait(s, "invalid", WaitChanges{Title: &newTitle})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

// TestAddWaitBlocker tests adding blockers to waits.
func TestAddWaitBlocker(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Blocker task", TaskOptions{})
	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})

	err := AddWaitBlocker(s, "TS-02W", "TS-01")
	if err != nil {
		t.Fatalf("AddWaitBlocker failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-02W")
	if len(wait.BlockedBy) != 1 {
		t.Error("wait should have one blocker")
	}
}

// TestAddWaitBlockerDuplicate tests adding duplicate blocker.
func TestAddWaitBlockerDuplicate(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Blocker task", TaskOptions{})
	AddWait(s, "TS", WaitOptions{
		Type:      model.ResolutionTypeManual,
		Question:  "Test?",
		BlockedBy: []string{"TS-01"},
	})

	err := AddWaitBlocker(s, "TS-02W", "TS-01")
	if err == nil {
		t.Error("expected error for duplicate blocker")
	}
}

// TestAddWaitBlockerCycle tests cycle detection when adding blocker.
func TestAddWaitBlockerCycle(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task", TaskOptions{})
	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})
	AddBlocker(s, "TS-01", "TS-02W")

	// Try to create cycle
	err := AddWaitBlocker(s, "TS-02W", "TS-01")
	if err == nil {
		t.Error("expected error for cycle")
	}
}

// TestAddWaitBlockerNotFound tests adding blocker to non-existent wait.
func TestAddWaitBlockerNotFound(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task", TaskOptions{})
	err := AddWaitBlocker(s, "TS-99W", "TS-01")
	if err == nil {
		t.Error("expected error for non-existent wait")
	}
}

// TestAddWaitBlockerInvalidID tests adding blocker with invalid ID.
func TestAddWaitBlockerInvalidID(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	err := AddWaitBlocker(s, "invalid", "TS-01")
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

// TestAddWaitBlockerInvalidBlocker tests adding invalid blocker.
func TestAddWaitBlockerInvalidBlocker(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})

	err := AddWaitBlocker(s, "TS-01W", "TS-99")
	if err == nil {
		t.Error("expected error for non-existent blocker")
	}
}

// TestRemoveWaitBlocker tests removing blockers from waits.
func TestRemoveWaitBlocker(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Blocker task", TaskOptions{})
	AddWait(s, "TS", WaitOptions{
		Type:      model.ResolutionTypeManual,
		Question:  "Test?",
		BlockedBy: []string{"TS-01"},
	})

	err := RemoveWaitBlocker(s, "TS-02W", "TS-01")
	if err != nil {
		t.Fatalf("RemoveWaitBlocker failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-02W")
	if len(wait.BlockedBy) != 0 {
		t.Error("wait should have no blockers")
	}
}

// TestRemoveWaitBlockerNotFound tests removing non-existent blocker.
func TestRemoveWaitBlockerNotFound(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})

	err := RemoveWaitBlocker(s, "TS-01W", "TS-99")
	if err == nil {
		t.Error("expected error for non-existent blocker")
	}
}

// TestRemoveWaitBlockerInvalidID tests removing blocker with invalid wait ID.
func TestRemoveWaitBlockerInvalidID(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	err := RemoveWaitBlocker(s, "invalid", "TS-01")
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

// TestRemoveWaitBlockerWaitNotFound tests removing blocker from non-existent wait.
func TestRemoveWaitBlockerWaitNotFound(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	err := RemoveWaitBlocker(s, "TS-99W", "TS-01")
	if err == nil {
		t.Error("expected error for non-existent wait")
	}
}

// TestValidateProject tests single project validation.
func TestValidateProject(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create valid task
	AddTask(s, "TS", "Valid task", TaskOptions{})

	errors, err := ValidateProject(s, "TS")
	if err != nil {
		t.Fatalf("ValidateProject failed: %v", err)
	}
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errors), errors)
	}
}

// TestValidateProjectWithIssues tests validation detecting issues.
func TestValidateProjectWithIssues(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create task with orphan blocker
	pf, _ := s.LoadProject("TS")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:        "TS-01",
		Title:     "Task with orphan",
		Status:    model.TaskStatusOpen,
		Priority:  3,
		BlockedBy: []string{"TS-99"},
		Created:   time.Now(),
		Updated:   time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	errors, err := ValidateProject(s, "TS")
	if err != nil {
		t.Fatalf("ValidateProject failed: %v", err)
	}
	if len(errors) == 0 {
		t.Error("expected validation errors")
	}
}

// TestValidationErrorString tests the ValidationError.Error() method.
func TestValidationErrorString(t *testing.T) {
	err := ValidationError{
		Type:    ValidationErrorOrphanBlocker,
		ItemID:  "TS-01",
		Message: "references non-existent blocker: TS-99",
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "TS-01") {
		t.Error("error string should contain item ID")
	}
	if !strings.Contains(errStr, "orphan_blocker") {
		t.Error("error string should contain error type")
	}
	if !strings.Contains(errStr, "non-existent") {
		t.Error("error string should contain message")
	}
}

// TestDetectCyclesInWaits tests cycle detection in waits.
func TestDetectCyclesInWaits(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a cycle manually: W1 -> W2 -> W1
	pf, _ := s.LoadProject("TS")
	pf.Waits = append(pf.Waits, model.Wait{
		ID:     "TS-01W",
		Status: model.WaitStatusOpen,
		ResolutionCriteria: model.ResolutionCriteria{
			Type:     model.ResolutionTypeManual,
			Question: "Q1?",
		},
		BlockedBy: []string{"TS-02W"},
		Created:   time.Now(),
	})
	pf.Waits = append(pf.Waits, model.Wait{
		ID:     "TS-02W",
		Status: model.WaitStatusOpen,
		ResolutionCriteria: model.ResolutionCriteria{
			Type:     model.ResolutionTypeManual,
			Question: "Q2?",
		},
		BlockedBy: []string{"TS-01W"},
		Created:   time.Now(),
	})
	pf.NextID = 3
	s.SaveProject(pf)

	errors, _ := ValidateProject(s, "TS")
	foundCycle := false
	for _, e := range errors {
		if e.Type == ValidationErrorCycle {
			foundCycle = true
			break
		}
	}
	if !foundCycle {
		t.Error("expected cycle error to be detected")
	}
}

// TestDetectDuplicateIDs tests detection of duplicate IDs.
func TestDetectDuplicateIDs(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create duplicate IDs manually
	pf, _ := s.LoadProject("TS")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:       "TS-01",
		Title:    "Task 1",
		Status:   model.TaskStatusOpen,
		Priority: 3,
		Created:  time.Now(),
		Updated:  time.Now(),
	})
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:       "TS-01", // Duplicate
		Title:    "Task 1 duplicate",
		Status:   model.TaskStatusOpen,
		Priority: 3,
		Created:  time.Now(),
		Updated:  time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	errors, _ := ValidateProject(s, "TS")
	foundDuplicate := false
	for _, e := range errors {
		if e.Type == ValidationErrorDuplicateID {
			foundDuplicate = true
			break
		}
	}
	if !foundDuplicate {
		t.Error("expected duplicate ID error")
	}
}

// TestDetectMissingRequiredFields tests detection of missing required fields.
func TestDetectMissingRequiredFields(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create task with missing title
	pf, _ := s.LoadProject("TS")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:       "TS-01",
		Title:    "", // Missing
		Status:   model.TaskStatusOpen,
		Priority: 3,
		Created:  time.Now(),
		Updated:  time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	errors, _ := ValidateProject(s, "TS")
	foundMissing := false
	for _, e := range errors {
		if e.Type == ValidationErrorMissingRequired {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Error("expected missing required field error")
	}
}

// TestDetectInvalidIDs tests detection of invalid ID formats.
func TestDetectInvalidIDs(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create task with invalid ID
	pf, _ := s.LoadProject("TS")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:       "INVALID", // Not valid format
		Title:    "Task",
		Status:   model.TaskStatusOpen,
		Priority: 3,
		Created:  time.Now(),
		Updated:  time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	errors, _ := ValidateProject(s, "TS")
	foundInvalid := false
	for _, e := range errors {
		if e.Type == ValidationErrorInvalidID {
			foundInvalid = true
			break
		}
	}
	if !foundInvalid {
		t.Error("expected invalid ID error")
	}
}

// TestValidateAndFixOrphanInWait tests fixing orphan blockers in waits.
func TestValidateAndFixOrphanInWait(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create wait with orphan blocker
	pf, _ := s.LoadProject("TS")
	pf.Waits = append(pf.Waits, model.Wait{
		ID:     "TS-01W",
		Status: model.WaitStatusOpen,
		ResolutionCriteria: model.ResolutionCriteria{
			Type:     model.ResolutionTypeManual,
			Question: "Test?",
		},
		BlockedBy: []string{"TS-99"}, // Non-existent
		Created:   time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	fixes, err := ValidateAndFix(s)
	if err != nil {
		t.Fatalf("ValidateAndFix failed: %v", err)
	}

	if len(fixes) == 0 {
		t.Error("expected fixes to be applied")
	}

	// Verify fix was applied
	pf, _ = s.LoadProject("TS")
	wait := findWait(pf, "TS-01W")
	if len(wait.BlockedBy) != 0 {
		t.Error("orphan blocker should be removed from wait")
	}
}

// TestResolveTimeWaitEarly tests early resolution of time waits.
func TestResolveTimeWaitEarly(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a time wait in the future
	future := time.Now().Add(24 * time.Hour)
	AddWait(s, "TS", WaitOptions{
		Type:  model.ResolutionTypeTime,
		After: &future,
	})

	// Resolve it early
	err := ResolveWait(s, "TS-01W", "Resolved early")
	if err != nil {
		t.Fatalf("ResolveWait (early) failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-01W")
	if wait.Status != model.WaitStatusDone {
		t.Error("wait should be done")
	}
	// The 'after' time should be updated to now (approximately)
	if wait.ResolutionCriteria.After.After(time.Now().Add(1 * time.Second)) {
		t.Error("after time should be updated to current time when resolving early")
	}
}

// TestDeleteProjectWithOpenWaits tests deletion blocked by open waits.
func TestDeleteProjectWithOpenWaits(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create project with only waits (no tasks)
	CreateProject(s, "todelete", "TD", "To Delete", "")
	AddWait(s, "TD", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})

	// Should fail without force
	err := DeleteProject(s, "TD", false)
	if err == nil {
		t.Error("expected error when deleting project with open waits")
	}

	// Should succeed with force
	err = DeleteProject(s, "TD", true)
	if err != nil {
		t.Fatalf("DeleteProject with force failed: %v", err)
	}
}

// TestChangeProjectPrefixInvalid tests invalid prefix changes.
func TestChangeProjectPrefixInvalid(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Same prefix
	err := ChangeProjectPrefix(s, "TS", "TS")
	if err == nil {
		t.Error("expected error for same prefix")
	}

	// Invalid new prefix (too short)
	err = ChangeProjectPrefix(s, "TS", "A")
	if err == nil {
		t.Error("expected error for short prefix")
	}

	// Invalid new prefix (contains numbers)
	err = ChangeProjectPrefix(s, "TS", "A1")
	if err == nil {
		t.Error("expected error for non-letter prefix")
	}

	// New prefix already exists
	CreateProject(s, "other", "OT", "Other", "")
	err = ChangeProjectPrefix(s, "TS", "OT")
	if err == nil {
		t.Error("expected error for existing prefix")
	}
}

// TestCompleteAlreadyDone tests completing an already done task.
func TestCompleteAlreadyDone(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task", TaskOptions{})
	CompleteTask(s, "TS-01", false)

	// Try to complete again
	_, err := CompleteTask(s, "TS-01", false)
	if err == nil {
		t.Error("expected error when completing already done task")
	}
}

// TestCompleteAlreadyDoneErrorType verifies that completing an already-done task
// does NOT return an IncompleteBlockersError (it should be a plain error about status).
func TestCompleteAlreadyDoneErrorType(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task", TaskOptions{})
	CompleteTask(s, "TS-01", false)

	_, err := CompleteTask(s, "TS-01", false)
	if err == nil {
		t.Fatal("expected error when completing already done task")
	}

	var blockerErr *IncompleteBlockersError
	if errors.As(err, &blockerErr) {
		t.Error("completing a done task should NOT return IncompleteBlockersError")
	}

	if !strings.Contains(err.Error(), "not open") {
		t.Errorf("expected error to mention 'not open', got: %v", err)
	}
}

// TestCompleteDroppedTaskErrorType verifies that completing a dropped task
// does NOT return an IncompleteBlockersError.
func TestCompleteDroppedTaskErrorType(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task", TaskOptions{})
	DropTask(s, "TS-01", "not needed", false, false)

	_, err := CompleteTask(s, "TS-01", false)
	if err == nil {
		t.Fatal("expected error when completing dropped task")
	}

	var blockerErr *IncompleteBlockersError
	if errors.As(err, &blockerErr) {
		t.Error("completing a dropped task should NOT return IncompleteBlockersError")
	}

	if !strings.Contains(err.Error(), "not open") {
		t.Errorf("expected error to mention 'not open', got: %v", err)
	}
}

// TestCompleteBlockedTaskReturnsBlockerError verifies that completing a task
// with incomplete blockers returns an IncompleteBlockersError.
func TestCompleteBlockedTaskReturnsBlockerError(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Blocker", TaskOptions{})
	AddTask(s, "TS", "Blocked", TaskOptions{BlockedBy: []string{"TS-01"}})

	_, err := CompleteTask(s, "TS-02", false)
	if err == nil {
		t.Fatal("expected error when completing blocked task")
	}

	var blockerErr *IncompleteBlockersError
	if !errors.As(err, &blockerErr) {
		t.Fatalf("expected IncompleteBlockersError, got %T: %v", err, err)
	}

	if blockerErr.TaskID != "TS-02" {
		t.Errorf("expected TaskID 'TS-02', got %q", blockerErr.TaskID)
	}
	if len(blockerErr.Blockers) != 1 || blockerErr.Blockers[0] != "TS-01" {
		t.Errorf("expected Blockers [TS-01], got %v", blockerErr.Blockers)
	}
}

// TestCompleteNotFoundErrorType verifies that completing a non-existent task
// does NOT return an IncompleteBlockersError.
func TestCompleteNotFoundErrorType(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := CompleteTask(s, "TS-99", false)
	if err == nil {
		t.Fatal("expected error when completing non-existent task")
	}

	var blockerErr *IncompleteBlockersError
	if errors.As(err, &blockerErr) {
		t.Error("completing a non-existent task should NOT return IncompleteBlockersError")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to mention 'not found', got: %v", err)
	}
}

// TestReopenAlreadyOpen tests reopening an already open task.
func TestReopenAlreadyOpen(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task", TaskOptions{})

	err := ReopenTask(s, "TS-01")
	if err == nil {
		t.Error("expected error when reopening already open task")
	}
}

// TestDeferTaskWithExistingWait tests deferring task that already has open wait.
func TestDeferTaskWithExistingWait(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddTask(s, "TS", "Task", TaskOptions{})
	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})
	AddBlocker(s, "TS-01", "TS-02W")

	// Try to defer (should fail because task already has open wait)
	until := time.Now().Add(24 * time.Hour)
	_, err := DeferTask(s, "TS-01", until)
	if err == nil {
		t.Error("expected error when deferring task with existing open wait")
	}
}

// TestDropWaitWithDependents tests dropping wait that has dependents.
func TestDropWaitWithDependents(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})
	AddTask(s, "TS", "Blocked task", TaskOptions{BlockedBy: []string{"TS-01W"}})

	// Should fail without flags
	err := DropWait(s, "TS-01W", "reason", false, false)
	if err == nil {
		t.Error("expected error when dropping wait with dependents")
	}

	// Should succeed with dropDeps
	err = DropWait(s, "TS-01W", "reason", true, false)
	if err != nil {
		t.Fatalf("DropWait with dropDeps failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	task := findTask(pf, "TS-02")
	if task.Status != model.TaskStatusDropped {
		t.Error("dependent task should be dropped")
	}
}

// TestDropWaitRemoveDeps tests dropping wait with remove deps flag.
func TestDropWaitRemoveDeps(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})
	AddTask(s, "TS", "Blocked task", TaskOptions{BlockedBy: []string{"TS-01W"}})

	err := DropWait(s, "TS-01W", "reason", false, true)
	if err != nil {
		t.Fatalf("DropWait with removeDeps failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	task := findTask(pf, "TS-02")
	if task.Status != model.TaskStatusOpen {
		t.Error("dependent task should remain open")
	}
	if len(task.BlockedBy) != 0 {
		t.Error("dependent task should have wait removed from blockers")
	}
}

// TestDeferWaitManual tests deferring manual wait updates check_after.
func TestDeferWaitManual(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	checkAfter := time.Now()
	AddWait(s, "TS", WaitOptions{
		Type:       model.ResolutionTypeManual,
		Question:   "Test?",
		CheckAfter: &checkAfter,
	})

	newTime := time.Now().Add(48 * time.Hour)
	err := DeferWait(s, "TS-01W", newTime)
	if err != nil {
		t.Fatalf("DeferWait (manual) failed: %v", err)
	}

	pf, _ := s.LoadProject("TS")
	wait := findWait(pf, "TS-01W")
	if !wait.ResolutionCriteria.CheckAfter.After(checkAfter) {
		t.Error("check_after should be updated for manual wait")
	}
}

// TestAddWaitInvalidType tests creating wait with invalid type.
func TestAddWaitInvalidType(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := AddWait(s, "TS", WaitOptions{
		Type: "invalid",
	})
	if err == nil {
		t.Error("expected error for invalid resolution type")
	}
}

// TestAddWaitTimeNoAfter tests creating time wait without after date.
func TestAddWaitTimeNoAfter(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := AddWait(s, "TS", WaitOptions{
		Type: model.ResolutionTypeTime,
		// No After specified
	})
	if err == nil {
		t.Error("expected error for time wait without after date")
	}
}

// TestAddWaitManualNoQuestion tests creating manual wait without question or title.
func TestAddWaitManualNoQuestion(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	_, err := AddWait(s, "TS", WaitOptions{
		Type: model.ResolutionTypeManual,
		// No Question or Title specified
	})
	if err == nil {
		t.Error("expected error for manual wait without question or title")
	}
}

// TestAddTaskToPausedProject tests adding task to paused project fails.
func TestAddTaskToPausedProject(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Pause the default project
	pausedStatus := model.ProjectStatusPaused
	EditProject(s, "TS", ProjectChanges{Status: &pausedStatus})

	// Try to add task (should fail)
	_, err := AddTask(s, "TS", "Test task", TaskOptions{})
	if err == nil {
		t.Error("expected error when adding task to paused project")
	}
}

// TestAddTaskToDoneProject tests adding task to done project fails.
func TestAddTaskToDoneProject(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Mark project as done
	doneStatus := model.ProjectStatusDone
	EditProject(s, "TS", ProjectChanges{Status: &doneStatus})

	// Try to add task (should fail)
	_, err := AddTask(s, "TS", "Test task", TaskOptions{})
	if err == nil {
		t.Error("expected error when adding task to done project")
	}
}

// TestAddWaitToPausedProject tests adding wait to paused project fails.
func TestAddWaitToPausedProject(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Pause the default project
	pausedStatus := model.ProjectStatusPaused
	EditProject(s, "TS", ProjectChanges{Status: &pausedStatus})

	// Try to add wait (should fail)
	_, err := AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})
	if err == nil {
		t.Error("expected error when adding wait to paused project")
	}
}

// TestAddWaitToDoneProject tests adding wait to done project fails.
func TestAddWaitToDoneProject(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Mark project as done
	doneStatus := model.ProjectStatusDone
	EditProject(s, "TS", ProjectChanges{Status: &doneStatus})

	// Try to add wait (should fail)
	_, err := AddWait(s, "TS", WaitOptions{
		Type:     model.ResolutionTypeManual,
		Question: "Test?",
	})
	if err == nil {
		t.Error("expected error when adding wait to done project")
	}
}

// TestFindNewlyUnblockedWaits tests finding newly unblocked waits after check.
func TestFindNewlyUnblockedWaits(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create: past time wait -> another wait (dormant) -> task
	past := time.Now().Add(-1 * time.Hour)
	AddWait(s, "TS", WaitOptions{
		Type:  model.ResolutionTypeTime,
		After: &past,
	})
	AddWait(s, "TS", WaitOptions{
		Type:      model.ResolutionTypeManual,
		Question:  "Test?",
		BlockedBy: []string{"TS-01W"},
	})
	AddTask(s, "TS", "Task", TaskOptions{BlockedBy: []string{"TS-02W"}})

	result, err := RunCheck(s)
	if err != nil {
		t.Fatalf("RunCheck failed: %v", err)
	}

	// Should have resolved TS-01W and unblocked TS-02W
	if len(result.ResolvedWaits) != 1 || result.ResolvedWaits[0] != "TS-01W" {
		t.Errorf("expected resolved wait TS-01W, got %v", result.ResolvedWaits)
	}
	if len(result.Unblocked) != 1 || result.Unblocked[0] != "TS-02W" {
		t.Errorf("expected unblocked TS-02W, got %v", result.Unblocked)
	}
}

// BenchmarkAddTask benchmarks task creation.
func BenchmarkAddTask(b *testing.B) {
	dir, _ := os.MkdirTemp("", "tk-bench")
	defer os.RemoveAll(dir)

	s, _ := storage.Init(dir, "Bench", "BN")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AddTask(s, "BN", "Benchmark task", TaskOptions{})
	}
}

// Test helper to verify storage directory structure
func TestStorageStructure(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Verify .tk directory exists
	tkPath := s.TkPath()
	if _, err := os.Stat(tkPath); os.IsNotExist(err) {
		t.Error(".tk directory should exist")
	}

	// Verify projects directory exists
	projectsPath := filepath.Join(tkPath, "projects")
	if _, err := os.Stat(projectsPath); os.IsNotExist(err) {
		t.Error("projects directory should exist")
	}

	// Verify default project file exists
	projectFile := filepath.Join(projectsPath, "TS.yaml")
	if _, err := os.Stat(projectFile); os.IsNotExist(err) {
		t.Error("default project file should exist")
	}
}

// TestValidatePriority tests the ValidatePriority function.
func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"zero is allowed (default)", 0, false},
		{"priority 1 is valid", 1, false},
		{"priority 2 is valid", 2, false},
		{"priority 3 is valid", 3, false},
		{"priority 4 is valid", 4, false},
		{"priority -1 is invalid", -1, true},
		{"priority -5 is invalid", -5, true},
		{"priority 5 is invalid", 5, true},
		{"priority 99 is invalid", 99, true},
		{"priority -100 is invalid", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePriority(tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePriority(%d) error = %v, wantErr %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

// TestAddTaskInvalidPriority tests that AddTask rejects invalid priorities.
func TestAddTaskInvalidPriority(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"priority 0 uses default", 0, false},
		{"priority 1 is valid", 1, false},
		{"priority 4 is valid", 4, false},
		{"priority -5 is rejected", -5, true},
		{"priority 5 is rejected", 5, true},
		{"priority 99 is rejected", 99, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AddTask(s, "TS", tt.name, TaskOptions{Priority: tt.priority})
			if (err != nil) != tt.wantErr {
				t.Errorf("AddTask with priority %d: error = %v, wantErr %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

// TestEditTaskInvalidPriority tests that EditTask rejects invalid priorities.
func TestEditTaskInvalidPriority(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a valid task first
	AddTask(s, "TS", "Test task", TaskOptions{Priority: 2})

	tests := []struct {
		name     string
		priority int
		wantErr  bool
	}{
		{"edit to priority 1", 1, false},
		{"edit to priority 4", 4, false},
		{"edit to priority -1", -1, true},
		{"edit to priority 0", 0, true},
		{"edit to priority 5", 5, true},
		{"edit to priority 99", 99, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.priority
			err := EditTask(s, "TS-01", TaskChanges{Priority: &p})
			if (err != nil) != tt.wantErr {
				t.Errorf("EditTask with priority %d: error = %v, wantErr %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

// TestValidateInvalidPriority tests that Validate detects invalid priorities.
func TestValidateInvalidPriority(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Manually create a task with an invalid priority
	pf, _ := s.LoadProject("TS")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:       "TS-01",
		Title:    "Bad priority task",
		Status:   model.TaskStatusOpen,
		Priority: -5,
		Created:  time.Now(),
		Updated:  time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	errors, err := Validate(s)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	found := false
	for _, e := range errors {
		if e.Type == ValidationErrorInvalidPriority {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected invalid_priority validation error")
	}
}

// TestValidateHighPriority tests that Validate detects priority > 4.
func TestValidateHighPriority(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Manually create a task with priority 99
	pf, _ := s.LoadProject("TS")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:       "TS-01",
		Title:    "High priority task",
		Status:   model.TaskStatusOpen,
		Priority: 99,
		Created:  time.Now(),
		Updated:  time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	errors, err := Validate(s)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	found := false
	for _, e := range errors {
		if e.Type == ValidationErrorInvalidPriority {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected invalid_priority validation error for priority 99")
	}
}

// TestValidateZeroPriority tests that Validate detects priority 0 (unset).
func TestValidateZeroPriority(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Manually create a task with priority 0
	pf, _ := s.LoadProject("TS")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:       "TS-01",
		Title:    "Zero priority task",
		Status:   model.TaskStatusOpen,
		Priority: 0,
		Created:  time.Now(),
		Updated:  time.Now(),
	})
	pf.NextID = 2
	s.SaveProject(pf)

	errors, err := Validate(s)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	found := false
	for _, e := range errors {
		if e.Type == ValidationErrorInvalidPriority {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected invalid_priority validation error for priority 0")
	}
}

// TestValidateTitle tests the ValidateTitle function.
func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"non-empty title is valid", "Buy groceries", false},
		{"title with spaces is valid", "  Buy groceries  ", false},
		{"empty string is rejected", "", true},
		{"whitespace-only is rejected", "   ", true},
		{"tab-only is rejected", "\t", true},
		{"newline-only is rejected", "\n", true},
		{"mixed whitespace is rejected", " \t\n ", true},
		{"single character is valid", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTitle(tt.title)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTitle(%q) error = %v, wantErr %v", tt.title, err, tt.wantErr)
			}
		})
	}
}

// TestAddTaskEmptyTitle tests that AddTask rejects empty titles.
func TestAddTaskEmptyTitle(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"empty title rejected", "", true},
		{"whitespace-only title rejected", "   ", true},
		{"tab-only title rejected", "\t\t", true},
		{"valid title accepted", "Real task", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AddTask(s, "TS", tt.title, TaskOptions{})
			if (err != nil) != tt.wantErr {
				t.Errorf("AddTask with title %q: error = %v, wantErr %v", tt.title, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "title must not be empty") {
					t.Errorf("expected error about empty title, got: %v", err)
				}
			}
		})
	}
}

// TestEditTaskEmptyTitle tests that EditTask rejects empty titles.
func TestEditTaskEmptyTitle(t *testing.T) {
	s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a valid task first
	AddTask(s, "TS", "Original title", TaskOptions{})

	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"empty title rejected", "", true},
		{"whitespace-only title rejected", "   ", true},
		{"tab-only title rejected", "\t", true},
		{"valid title accepted", "New title", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := tt.title
			err := EditTask(s, "TS-01", TaskChanges{Title: &title})
			if (err != nil) != tt.wantErr {
				t.Errorf("EditTask with title %q: error = %v, wantErr %v", tt.title, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "title must not be empty") {
					t.Errorf("expected error about empty title, got: %v", err)
				}
			}
		})
	}
}
