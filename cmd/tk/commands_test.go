package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestStorage creates a temporary .tk directory with test data.
func setupTestStorage(t *testing.T) (string, *storage.Storage, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "tk-test-*")
	require.NoError(t, err)

	// Change to temp directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Initialize storage
	s, err := storage.Init(tmpDir, "Test Project", "TP")
	require.NoError(t, err)

	cleanup := func() {
		os.Chdir(origDir)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, s, cleanup
}

// setupTestStorageWithData creates a test storage with sample tasks and waits.
func setupTestStorageWithData(t *testing.T) (string, *storage.Storage, func()) {
	tmpDir, s, cleanup := setupTestStorage(t)

	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(7 * 24 * time.Hour)

	pf, err := s.LoadProject("TP")
	require.NoError(t, err)

	// Add tasks
	pf.Tasks = []model.Task{
		{
			ID:       "TP-01",
			Title:    "Ready task",
			Status:   model.TaskStatusOpen,
			Priority: 1,
			Tags:     []string{"urgent"},
			Created:  now,
			Updated:  now,
		},
		{
			ID:        "TP-02",
			Title:     "Blocked task",
			Status:    model.TaskStatusOpen,
			Priority:  2,
			BlockedBy: []string{"TP-01"},
			Tags:      []string{"feature"},
			Created:   now,
			Updated:   now,
		},
		{
			ID:        "TP-03",
			Title:     "Waiting task",
			Status:    model.TaskStatusOpen,
			Priority:  3,
			BlockedBy: []string{"TP-01W"},
			Created:   now,
			Updated:   now,
		},
		{
			ID:       "TP-04",
			Title:    "Done task",
			Status:   model.TaskStatusDone,
			Priority: 2,
			Created:  past,
			Updated:  now,
			DoneAt:   &now,
		},
		{
			ID:       "TP-05",
			Title:    "Task with notes about gravel",
			Status:   model.TaskStatusOpen,
			Priority: 4,
			Notes:    "Need to order gravel for the project",
			Created:  now,
			Updated:  now,
		},
	}

	// Add waits
	pf.Waits = []model.Wait{
		{
			ID:     "TP-01W",
			Status: model.WaitStatusOpen,
			ResolutionCriteria: model.ResolutionCriteria{
				Type:     model.ResolutionTypeManual,
				Question: "Did the package arrive?",
			},
			Created: now,
		},
		{
			ID:     "TP-02W",
			Status: model.WaitStatusOpen,
			ResolutionCriteria: model.ResolutionCriteria{
				Type:  model.ResolutionTypeTime,
				After: &future,
			},
			Created: now,
		},
	}

	pf.NextID = 6

	err = s.SaveProject(pf)
	require.NoError(t, err)

	return tmpDir, s, cleanup
}

func TestListCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	tests := []struct {
		name     string
		flags    func()
		contains []string
		excludes []string
	}{
		{
			name:  "default lists open tasks",
			flags: func() {},
			contains: []string{
				"TP-01", "Ready task",
				"TP-02", "Blocked task",
				"TP-03", "Waiting task",
				"TP-05",
			},
			excludes: []string{"TP-04", "Done task"},
		},
		{
			name:     "ready filter",
			flags:    func() { listReady = true },
			contains: []string{"TP-01", "TP-05"},
			excludes: []string{"TP-02", "TP-03", "TP-04"},
		},
		{
			name:     "blocked filter",
			flags:    func() { listBlocked = true },
			contains: []string{"TP-02"},
			excludes: []string{"TP-01", "TP-03", "TP-05"},
		},
		{
			name:     "waiting filter",
			flags:    func() { listWaiting = true },
			contains: []string{"TP-03"},
			excludes: []string{"TP-01", "TP-02", "TP-05"},
		},
		{
			name:     "done filter",
			flags:    func() { listDone = true },
			contains: []string{"TP-04", "Done task"},
			excludes: []string{"TP-01", "TP-02", "TP-03"},
		},
		{
			name:     "tag filter",
			flags:    func() { listTags = []string{"urgent"} },
			contains: []string{"TP-01"},
			excludes: []string{"TP-02", "TP-03", "TP-05"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			listProject = ""
			listReady = false
			listBlocked = false
			listWaiting = false
			listDone = false
			listDropped = false
			listAll = false
			listPriority = 0
			listP1 = false
			listP2 = false
			listP3 = false
			listP4 = false
			listTags = nil
			listOverdue = false

			// Apply test-specific flags
			tt.flags()

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := runList(nil, nil)

			w.Close()
			var buf bytes.Buffer
			buf.ReadFrom(r)
			os.Stdout = old

			output := buf.String()

			assert.NoError(t, err)
			for _, s := range tt.contains {
				assert.Contains(t, output, s, "expected output to contain %q", s)
			}
			for _, s := range tt.excludes {
				assert.NotContains(t, output, s, "expected output to not contain %q", s)
			}
		})
	}
}

func TestWaitsCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	waitsProject = ""
	waitsActionable = false
	waitsDormant = false
	waitsDone = false
	waitsDropped = false
	waitsAll = false

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWaits(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01W")
	assert.Contains(t, output, "Did the package arrive?")
	assert.Contains(t, output, "TP-02W")
}

func TestWaitsActionableFilter(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	waitsProject = ""
	waitsActionable = true
	waitsDormant = false
	waitsDone = false
	waitsDropped = false
	waitsAll = false

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWaits(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	// TP-01W is actionable (manual wait with no check_after)
	assert.Contains(t, output, "TP-01W")
	// TP-02W is pending (time wait with future date)
	assert.NotContains(t, output, "TP-02W")
}

func TestFindCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	findProject = ""

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runFind(nil, []string{"gravel"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-05")
	assert.Contains(t, output, "Task with notes about gravel")
}

func TestShowTaskCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"TP-02"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-02: Blocked task")
	assert.Contains(t, output, "Status:")
	assert.Contains(t, output, "blocked")
	assert.Contains(t, output, "Blocked by:")
	assert.Contains(t, output, "TP-01")
}

func TestShowWaitCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"TP-01W"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01W")
	assert.Contains(t, output, "Did the package arrive?")
	assert.Contains(t, output, "Type:")
	assert.Contains(t, output, "manual")
}

func TestProjectCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProject(nil, []string{"TP"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP: Test Project")
	assert.Contains(t, output, "open")
	assert.Contains(t, output, "ready")
	assert.Contains(t, output, "blocked")
	assert.Contains(t, output, "waiting")
	assert.Contains(t, output, "done")
}

func TestProjectsCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	projectsAll = false

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjects(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP")
	assert.Contains(t, output, "Test Project")
}

func TestGraphCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	graphProject = ""

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runGraph(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "digraph tk")
	assert.Contains(t, output, "TP-01")
	assert.Contains(t, output, "TP-02")
	assert.Contains(t, output, "TP-01W")
	assert.Contains(t, output, "shape=diamond") // waits are diamonds
	assert.Contains(t, output, "->")            // edges
}

func TestValidateCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	validateFix = false

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runValidate(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "No issues found")
}

func TestInitCommand(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tk-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Reset flags
	initName = ""
	initPrefix = ""

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runInit(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Initialized tk")
	assert.Contains(t, output, "Default")
	assert.Contains(t, output, "DF")

	// Verify .tk directory was created
	_, err = os.Stat(filepath.Join(tmpDir, ".tk"))
	assert.NoError(t, err)
}

func TestInitWithCustomOptions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tk-init-custom-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Set custom flags
	initName = "My Tasks"
	initPrefix = "MT"

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runInit(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "My Tasks")
	assert.Contains(t, output, "MT")
}

func TestReadyCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset list flags
	listProject = ""
	listReady = false
	listBlocked = false
	listWaiting = false
	listDone = false
	listDropped = false
	listAll = false
	listPriority = 0
	listTags = nil

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runReady(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01")
	assert.Contains(t, output, "TP-05")
	assert.NotContains(t, output, "TP-02")
	assert.NotContains(t, output, "TP-03")
}

func TestWaitingCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset waits flags
	waitsProject = ""
	waitsActionable = false
	waitsDormant = false
	waitsDone = false
	waitsDropped = false
	waitsAll = false

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWaiting(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01W")
	// Time wait should not show in actionable
	assert.NotContains(t, output, "TP-02W")
}

func TestShowNonExistentTask(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	err := runShow(nil, []string{"TP-99"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestShowInvalidID(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	err := runShow(nil, []string{"INVALID"})
	assert.Error(t, err)
}

func TestProjectNotFound(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	err := runProject(nil, []string{"NONEXISTENT"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListWithMultipleTags(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Add a task with multiple tags
	pf, _ := s.LoadProject("TP")
	pf.Tasks = append(pf.Tasks, model.Task{
		ID:       "TP-06",
		Title:    "Multi-tag task",
		Status:   model.TaskStatusOpen,
		Priority: 2,
		Tags:     []string{"urgent", "feature"},
		Created:  time.Now(),
		Updated:  time.Now(),
	})
	pf.NextID = 7
	s.SaveProject(pf)

	// Reset flags
	listProject = ""
	listReady = false
	listBlocked = false
	listWaiting = false
	listDone = false
	listDropped = false
	listAll = false
	listPriority = 0
	listTags = []string{"urgent", "feature"}

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	// Only TP-06 has both tags
	assert.Contains(t, output, "TP-06")
	// TP-01 only has "urgent", not "feature"
	assert.NotContains(t, output, "TP-01")
}

func TestFindNoResults(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	findProject = ""

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runFind(nil, []string{"nonexistent-query-xyz"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "No results found")
}

func TestFindWaitQuestion(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	findProject = ""

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runFind(nil, []string{"package"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Waits:")
	assert.Contains(t, output, "TP-01W")
}

func TestListByPriorityShorthand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	listProject = ""
	listReady = false
	listBlocked = false
	listWaiting = false
	listDone = false
	listDropped = false
	listAll = false
	listPriority = 0
	listP1 = true
	listP2 = false
	listP3 = false
	listP4 = false
	listTags = nil

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runList(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	// Only TP-01 has priority 1
	assert.Contains(t, output, "TP-01")
	assert.NotContains(t, output, "TP-02")
	assert.NotContains(t, output, "TP-03")
	assert.NotContains(t, output, "TP-05")
}

func TestCaseInsensitiveIDLookup(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Test lowercase ID
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShow(nil, []string{"tp-01"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Ready task")
}

func TestGraphDOTFormat(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	graphProject = ""

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runGraph(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)

	// Verify DOT structure
	assert.True(t, strings.HasPrefix(output, "digraph tk {"))
	assert.Contains(t, output, "rankdir=LR")
	assert.Contains(t, output, "node [shape=box]")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(output), "}"))

	// Verify edges are correct direction (blocker -> blocked)
	assert.Contains(t, output, `"TP-01" -> "TP-02"`)
	assert.Contains(t, output, `"TP-01W" -> "TP-03"`)

	// Verify wait edges are dashed
	assert.Contains(t, output, "[style=dashed]")
}

// ============= Phase 8 Write Command Tests =============

func TestAddCommand(t *testing.T) {
	_, s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Reset flags
	addProject = "TP"
	addPriority = 2
	addTags = []string{"test"}
	addNotes = ""
	addAssignee = ""
	addDueDate = ""
	addAutoComplete = false
	addBlockedBy = ""

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runAdd(nil, []string{"Test task"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01")
	assert.Contains(t, output, "Test task")

	// Verify task was created
	pf, _ := s.LoadProject("TP")
	assert.Equal(t, 1, len(pf.Tasks))
	assert.Equal(t, "Test task", pf.Tasks[0].Title)
	assert.Equal(t, 2, pf.Tasks[0].Priority)
	assert.Contains(t, pf.Tasks[0].Tags, "test")
}

func TestDoneCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	doneForce = false

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDone(nil, []string{"TP-01"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01 done")
	assert.Contains(t, output, "Unblocked")
	assert.Contains(t, output, "TP-02")

	// Verify task status
	pf, _ := s.LoadProject("TP")
	var task *model.Task
	for i := range pf.Tasks {
		if pf.Tasks[i].ID == "TP-01" {
			task = &pf.Tasks[i]
			break
		}
	}
	assert.Equal(t, model.TaskStatusDone, task.Status)
	assert.NotNil(t, task.DoneAt)
}

func TestDoneCommandWithBlockers(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Try to complete a blocked task without force
	doneForce = false

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDone(nil, []string{"TP-02"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	// The command should return an error
	assert.Error(t, err)
	// The output should contain information about blockers
	assert.Contains(t, output, "incomplete blockers")
}

func TestDoneCommandForce(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Complete a blocked task with force
	doneForce = true

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDone(nil, []string{"TP-02"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	assert.NoError(t, err)

	// Verify task status
	pf, _ := s.LoadProject("TP")
	var task *model.Task
	for i := range pf.Tasks {
		if pf.Tasks[i].ID == "TP-02" {
			task = &pf.Tasks[i]
			break
		}
	}
	assert.Equal(t, model.TaskStatusDone, task.Status)
	// Blockers should be removed
	assert.Empty(t, task.BlockedBy)
}

func TestDropCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	dropReason = "Not needed"
	dropDropDeps = false
	dropRemoveDeps = true

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDrop(nil, []string{"TP-01"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01 dropped")

	// Verify task status
	pf, _ := s.LoadProject("TP")
	var task *model.Task
	for i := range pf.Tasks {
		if pf.Tasks[i].ID == "TP-01" {
			task = &pf.Tasks[i]
			break
		}
	}
	assert.Equal(t, model.TaskStatusDropped, task.Status)
	assert.Equal(t, "Not needed", task.DropReason)
}

func TestReopenCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runReopen(nil, []string{"TP-04"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-04 reopened")

	// Verify task status
	pf, _ := s.LoadProject("TP")
	var task *model.Task
	for i := range pf.Tasks {
		if pf.Tasks[i].ID == "TP-04" {
			task = &pf.Tasks[i]
			break
		}
	}
	assert.Equal(t, model.TaskStatusOpen, task.Status)
	assert.Nil(t, task.DoneAt)
}

func TestTagCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runTag(nil, []string{"TP-01", "newtag"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Added tag")
	assert.Contains(t, output, "newtag")

	// Verify tag was added
	pf, _ := s.LoadProject("TP")
	var task *model.Task
	for i := range pf.Tasks {
		if pf.Tasks[i].ID == "TP-01" {
			task = &pf.Tasks[i]
			break
		}
	}
	assert.Contains(t, task.Tags, "newtag")
}

func TestUntagCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUntag(nil, []string{"TP-01", "urgent"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Removed tag")
	assert.Contains(t, output, "urgent")

	// Verify tag was removed
	pf, _ := s.LoadProject("TP")
	var task *model.Task
	for i := range pf.Tasks {
		if pf.Tasks[i].ID == "TP-01" {
			task = &pf.Tasks[i]
			break
		}
	}
	assert.NotContains(t, task.Tags, "urgent")
}

func TestBlockCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	blockBy = "TP-04"

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBlock(nil, []string{"TP-05"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "blocked by")

	// Verify blocker was added
	pf, _ := s.LoadProject("TP")
	var task *model.Task
	for i := range pf.Tasks {
		if pf.Tasks[i].ID == "TP-05" {
			task = &pf.Tasks[i]
			break
		}
	}
	assert.Contains(t, task.BlockedBy, "TP-04")
}

func TestUnblockCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	unblockFrom = "TP-01"

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runUnblock(nil, []string{"TP-02"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "no longer blocked")

	// Verify blocker was removed
	pf, _ := s.LoadProject("TP")
	var task *model.Task
	for i := range pf.Tasks {
		if pf.Tasks[i].ID == "TP-02" {
			task = &pf.Tasks[i]
			break
		}
	}
	assert.NotContains(t, task.BlockedBy, "TP-01")
}

func TestBlockedByCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBlockedBy(nil, []string{"TP-02"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01")
}

func TestBlockingCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runBlocking(nil, []string{"TP-01"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-02")
}

func TestProjectNewCommand(t *testing.T) {
	_, s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Reset flags
	projectNewPrefix = "NP"
	projectNewName = "New Project"
	projectNewDescription = "Test description"

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjectNew(nil, []string{"newproject"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Created project")
	assert.Contains(t, output, "NP")

	// Verify project was created
	pf, err := s.LoadProject("NP")
	assert.NoError(t, err)
	assert.Equal(t, "newproject", pf.ID)
	assert.Equal(t, "New Project", pf.Name)
}

func TestProjectDeleteCommand(t *testing.T) {
	_, s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a second project first
	projectNewPrefix = "DP"
	projectNewName = "To Delete"
	projectNewDescription = ""
	runProjectNew(nil, []string{"todelete"})

	// Reset delete flags
	projectDeleteForce = true

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runProjectDelete(nil, []string{"DP"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Deleted")

	// Verify project was deleted
	_, err = s.LoadProject("DP")
	assert.Error(t, err)
}

func TestCheckCommand(t *testing.T) {
	_, s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create a time wait that has passed
	pf, _ := s.LoadProject("TP")
	past := time.Now().Add(-24 * time.Hour)
	pf.Waits = []model.Wait{
		{
			ID:     "TP-01W",
			Status: model.WaitStatusOpen,
			ResolutionCriteria: model.ResolutionCriteria{
				Type:  model.ResolutionTypeTime,
				After: &past,
			},
			Created: time.Now(),
		},
	}
	pf.NextID = 2
	s.SaveProject(pf)

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runCheck(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Resolved")
	assert.Contains(t, output, "TP-01W")

	// Verify wait was resolved
	pf, _ = s.LoadProject("TP")
	assert.Equal(t, model.WaitStatusDone, pf.Waits[0].Status)
}

func TestWaitAddManualCommand(t *testing.T) {
	_, s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Reset flags
	waitAddProject = "TP"
	waitAddQuestion = "Did the package arrive?"
	waitAddAfter = ""
	waitAddCheckAfter = ""
	waitAddNotes = ""
	waitAddBlockedBy = ""

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWaitAdd(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01W")
	assert.Contains(t, output, "Did the package arrive?")

	// Verify wait was created
	pf, _ := s.LoadProject("TP")
	assert.Equal(t, 1, len(pf.Waits))
	assert.Equal(t, model.ResolutionTypeManual, pf.Waits[0].ResolutionCriteria.Type)
}

func TestWaitAddTimeCommand(t *testing.T) {
	_, s, cleanup := setupTestStorage(t)
	defer cleanup()

	// Reset flags
	waitAddProject = "TP"
	waitAddQuestion = ""
	waitAddAfter = "2030-12-31"
	waitAddCheckAfter = ""
	waitAddNotes = ""
	waitAddBlockedBy = ""

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWaitAdd(nil, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "TP-01W")

	// Verify wait was created
	pf, _ := s.LoadProject("TP")
	assert.Equal(t, 1, len(pf.Waits))
	assert.Equal(t, model.ResolutionTypeTime, pf.Waits[0].ResolutionCriteria.Type)
}

func TestWaitResolveCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	waitResolveAs = "Package arrived"

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWaitResolve(nil, []string{"TP-01W"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "resolved")

	// Verify wait was resolved
	pf, _ := s.LoadProject("TP")
	var wait *model.Wait
	for i := range pf.Waits {
		if pf.Waits[i].ID == "TP-01W" {
			wait = &pf.Waits[i]
			break
		}
	}
	assert.Equal(t, model.WaitStatusDone, wait.Status)
	assert.Equal(t, "Package arrived", wait.Resolution)
}

func TestWaitDropCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	waitDropReason = "Not needed"
	waitDropDropDeps = false
	waitDropRemoveDeps = true

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runWaitDrop(nil, []string{"TP-01W"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "dropped")

	// Verify wait was dropped
	pf, _ := s.LoadProject("TP")
	var wait *model.Wait
	for i := range pf.Waits {
		if pf.Waits[i].ID == "TP-01W" {
			wait = &pf.Waits[i]
			break
		}
	}
	assert.Equal(t, model.WaitStatusDropped, wait.Status)
}

func TestDeferCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Reset flags
	deferDays = 5
	deferUntil = ""

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDefer(nil, []string{"TP-05"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "deferred")
	assert.Contains(t, output, "Created wait")

	// Verify wait was created and linked
	pf, _ := s.LoadProject("TP")

	// Find the task
	var task *model.Task
	for i := range pf.Tasks {
		if pf.Tasks[i].ID == "TP-05" {
			task = &pf.Tasks[i]
			break
		}
	}
	assert.NotEmpty(t, task.BlockedBy)

	// Find the wait
	var found bool
	for _, bid := range task.BlockedBy {
		if strings.HasSuffix(bid, "W") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected task to be blocked by a wait")
}

func TestDumpCommand(t *testing.T) {
	_, _, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDump(nil, []string{"TP"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "# TP: Test Project")
	assert.Contains(t, output, "## Tasks")
	assert.Contains(t, output, "### TP-01: Ready task")
	assert.Contains(t, output, "## Waits")
	assert.Contains(t, output, "### TP-01W")
}

func TestBatchDoneCommand(t *testing.T) {
	_, s, cleanup := setupTestStorageWithData(t)
	defer cleanup()

	// First complete TP-01 to unblock TP-02
	doneForce = false
	runDone(nil, []string{"TP-01"})

	// Now try batch completion
	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runDone(nil, []string{"TP-02", "TP-05"})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old

	output := buf.String()

	// Batch should succeed for both
	assert.NoError(t, err)
	assert.Contains(t, output, "TP-02 done")
	assert.Contains(t, output, "TP-05 done")

	// Verify both are done
	pf, _ := s.LoadProject("TP")
	for _, task := range pf.Tasks {
		if task.ID == "TP-02" || task.ID == "TP-05" {
			assert.Equal(t, model.TaskStatusDone, task.Status)
		}
	}
}
