package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProject(t *testing.T) {
	// Create a temporary test file
	content := `id: backyard
prefix: BY
name: Backyard Redo
description: Replace turf with gravel
status: active
next_id: 24
created: 2025-12-02T10:30:00Z
tasks:
  - id: BY-01
    title: Get paper bags
    status: done
    priority: 2
    tags: [shopping]
    created: 2025-12-02T10:30:00Z
    updated: 2025-12-02T10:30:00Z
    done_at: 2025-12-02T11:00:00Z
  - id: BY-02
    title: Fill bags with weeds
    status: open
    priority: 2
    blocked_by: [BY-01]
    notes: |
      Make sure bags are sturdy.
      Don't overfill them.
    created: 2025-12-02T10:35:00Z
    updated: 2025-12-02T10:35:00Z
waits:
  - id: BY-03W
    status: open
    resolution_criteria:
      type: manual
      question: Did the landscape fabric arrive from Home Depot?
    notes: Ordered standard shipping, tracking #12345
    created: 2025-12-06T14:22:00Z
`

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "BY.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	// Load the project
	pf, err := LoadProject(path)
	require.NoError(t, err)

	// Verify project metadata
	assert.Equal(t, "backyard", pf.ID)
	assert.Equal(t, "BY", pf.Prefix)
	assert.Equal(t, "Backyard Redo", pf.Name)
	assert.Equal(t, "Replace turf with gravel", pf.Description)
	assert.Equal(t, ProjectStatusActive, pf.Status)
	assert.Equal(t, 24, pf.NextID)

	// Verify tasks
	require.Len(t, pf.Tasks, 2)
	assert.Equal(t, "BY-01", pf.Tasks[0].ID)
	assert.Equal(t, "Get paper bags", pf.Tasks[0].Title)
	assert.Equal(t, TaskStatusDone, pf.Tasks[0].Status)
	assert.Equal(t, []string{"shopping"}, pf.Tasks[0].Tags)

	assert.Equal(t, "BY-02", pf.Tasks[1].ID)
	assert.Equal(t, []string{"BY-01"}, pf.Tasks[1].BlockedBy)
	assert.Contains(t, pf.Tasks[1].Notes, "Make sure bags are sturdy")

	// Verify waits
	require.Len(t, pf.Waits, 1)
	assert.Equal(t, "BY-03W", pf.Waits[0].ID)
	assert.Equal(t, WaitStatusOpen, pf.Waits[0].Status)
	assert.Equal(t, ResolutionTypeManual, pf.Waits[0].ResolutionCriteria.Type)
	assert.Equal(t, "Did the landscape fabric arrive from Home Depot?", pf.Waits[0].ResolutionCriteria.Question)
}

func TestLoadProject_FileNotFound(t *testing.T) {
	_, err := LoadProject("/nonexistent/path/BY.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestLoadProject_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.yaml")
	require.NoError(t, os.WriteFile(path, []byte("invalid: [unclosed"), 0644))

	_, err := LoadProject(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestSaveProject_RoundTrip(t *testing.T) {
	now := time.Date(2025, 12, 2, 10, 30, 0, 0, time.UTC)
	later := now.Add(time.Hour)
	dueDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	original := &ProjectFile{
		Project: Project{
			ID:          "backyard",
			Prefix:      "BY",
			Name:        "Backyard Redo",
			Description: "Replace turf with gravel",
			Status:      ProjectStatusActive,
			NextID:      24,
			Created:     now,
		},
		Tasks: []Task{
			{
				ID:       "BY-01",
				Title:    "Get paper bags",
				Status:   TaskStatusDone,
				Priority: 2,
				Tags:     []string{"shopping"},
				Created:  now,
				Updated:  now,
				DoneAt:   &later,
			},
			{
				ID:        "BY-02",
				Title:     "Fill bags with weeds",
				Status:    TaskStatusOpen,
				Priority:  2,
				BlockedBy: []string{"BY-01"},
				Notes:     "Make sure bags are sturdy.\nDon't overfill them.",
				Created:   now,
				Updated:   now,
				DueDate:   &dueDate,
			},
		},
		Waits: []Wait{
			{
				ID:     "BY-03W",
				Status: WaitStatusOpen,
				ResolutionCriteria: ResolutionCriteria{
					Type:     ResolutionTypeManual,
					Question: "Did the landscape fabric arrive?",
				},
				Notes:   "Tracking #12345",
				Created: now,
			},
		},
	}

	// Save and reload
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "BY.yaml")
	require.NoError(t, SaveProject(path, original))

	loaded, err := LoadProject(path)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.ID, loaded.ID)
	assert.Equal(t, original.Prefix, loaded.Prefix)
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Description, loaded.Description)
	assert.Equal(t, original.Status, loaded.Status)
	assert.Equal(t, original.NextID, loaded.NextID)

	require.Len(t, loaded.Tasks, 2)
	assert.Equal(t, original.Tasks[0].ID, loaded.Tasks[0].ID)
	assert.Equal(t, original.Tasks[0].Title, loaded.Tasks[0].Title)
	assert.Equal(t, original.Tasks[0].Status, loaded.Tasks[0].Status)
	assert.Equal(t, original.Tasks[0].Tags, loaded.Tasks[0].Tags)
	require.NotNil(t, loaded.Tasks[0].DoneAt)

	assert.Equal(t, original.Tasks[1].ID, loaded.Tasks[1].ID)
	assert.Equal(t, original.Tasks[1].BlockedBy, loaded.Tasks[1].BlockedBy)
	assert.Equal(t, original.Tasks[1].Notes, loaded.Tasks[1].Notes)
	require.NotNil(t, loaded.Tasks[1].DueDate)

	require.Len(t, loaded.Waits, 1)
	assert.Equal(t, original.Waits[0].ID, loaded.Waits[0].ID)
	assert.Equal(t, original.Waits[0].ResolutionCriteria.Type, loaded.Waits[0].ResolutionCriteria.Type)
	assert.Equal(t, original.Waits[0].ResolutionCriteria.Question, loaded.Waits[0].ResolutionCriteria.Question)
}

func TestSaveProject_SortOrder(t *testing.T) {
	now := time.Date(2025, 12, 2, 10, 30, 0, 0, time.UTC)

	// Create project with tasks out of order
	pf := &ProjectFile{
		Project: Project{
			ID:      "test",
			Prefix:  "TS",
			Name:    "Test",
			Status:  ProjectStatusActive,
			NextID:  11,
			Created: now,
		},
		Tasks: []Task{
			{ID: "TS-10", Title: "Third", Status: TaskStatusOpen, Priority: 3, Created: now, Updated: now},
			{ID: "TS-02", Title: "Second", Status: TaskStatusOpen, Priority: 3, Created: now, Updated: now},
			{ID: "TS-01", Title: "First", Status: TaskStatusOpen, Priority: 3, Created: now, Updated: now},
		},
		Waits: []Wait{
			{ID: "TS-05W", Status: WaitStatusOpen, ResolutionCriteria: ResolutionCriteria{Type: ResolutionTypeManual, Question: "Q2?"}, Created: now},
			{ID: "TS-02W", Status: WaitStatusOpen, ResolutionCriteria: ResolutionCriteria{Type: ResolutionTypeManual, Question: "Q1?"}, Created: now},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "TS.yaml")
	require.NoError(t, SaveProject(path, pf))

	// Read raw file to verify order
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	// Tasks should be sorted: TS-01, TS-02, TS-10
	idx01 := strings.Index(content, "TS-01")
	idx02 := strings.Index(content, "TS-02")
	idx10 := strings.Index(content, "TS-10")
	assert.True(t, idx01 < idx02, "TS-01 should appear before TS-02")
	assert.True(t, idx02 < idx10, "TS-02 should appear before TS-10")

	// Waits should be sorted: TS-02W, TS-05W
	idx02w := strings.Index(content, "TS-02W")
	idx05w := strings.Index(content, "TS-05W")
	assert.True(t, idx02w < idx05w, "TS-02W should appear before TS-05W")
}

func TestSaveProject_NullFieldsOmitted(t *testing.T) {
	now := time.Date(2025, 12, 2, 10, 30, 0, 0, time.UTC)

	// Create task with minimal fields (many nulls)
	pf := &ProjectFile{
		Project: Project{
			ID:      "test",
			Prefix:  "TS",
			Name:    "Test",
			Status:  ProjectStatusActive,
			NextID:  2,
			Created: now,
		},
		Tasks: []Task{
			{
				ID:       "TS-01",
				Title:    "Minimal task",
				Status:   TaskStatusOpen,
				Priority: 3,
				Created:  now,
				Updated:  now,
				// All optional fields are empty/nil
			},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "TS.yaml")
	require.NoError(t, SaveProject(path, pf))

	// Read raw file to verify null fields are omitted
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	// These fields should NOT appear since they're empty/nil
	assert.NotContains(t, content, "tags:")
	assert.NotContains(t, content, "blocked_by:")
	assert.NotContains(t, content, "notes:")
	assert.NotContains(t, content, "assignee:")
	assert.NotContains(t, content, "due_date:")
	assert.NotContains(t, content, "done_at:")
	assert.NotContains(t, content, "dropped_at:")
	assert.NotContains(t, content, "drop_reason:")
	assert.NotContains(t, content, "auto_complete:") // false is the default, omitted

	// These fields SHOULD appear
	assert.Contains(t, content, "id: TS-01")
	assert.Contains(t, content, "title: Minimal task")
	assert.Contains(t, content, "status: open")
	assert.Contains(t, content, "priority: 3")
}

func TestSaveProject_MultilineBlockScalar(t *testing.T) {
	now := time.Date(2025, 12, 2, 10, 30, 0, 0, time.UTC)

	pf := &ProjectFile{
		Project: Project{
			ID:      "test",
			Prefix:  "TS",
			Name:    "Test",
			Status:  ProjectStatusActive,
			NextID:  2,
			Created: now,
		},
		Tasks: []Task{
			{
				ID:       "TS-01",
				Title:    "Task with notes",
				Status:   TaskStatusOpen,
				Priority: 3,
				Notes:    "Line one.\nLine two.\nLine three.",
				Created:  now,
				Updated:  now,
			},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "TS.yaml")
	require.NoError(t, SaveProject(path, pf))

	// Read raw file to verify block scalar style
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	// Should use literal block scalar style (|) for multi-line strings
	assert.Contains(t, content, "notes: |")
}

func TestSaveProject_TimeWait(t *testing.T) {
	now := time.Date(2025, 12, 2, 10, 30, 0, 0, time.UTC)
	afterTime := time.Date(2026, 1, 15, 23, 59, 59, 0, time.UTC)

	pf := &ProjectFile{
		Project: Project{
			ID:      "test",
			Prefix:  "TS",
			Name:    "Test",
			Status:  ProjectStatusActive,
			NextID:  2,
			Created: now,
		},
		Waits: []Wait{
			{
				ID:     "TS-01W",
				Status: WaitStatusOpen,
				ResolutionCriteria: ResolutionCriteria{
					Type:  ResolutionTypeTime,
					After: &afterTime,
				},
				Created: now,
			},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "TS.yaml")
	require.NoError(t, SaveProject(path, pf))

	// Reload and verify
	loaded, err := LoadProject(path)
	require.NoError(t, err)
	require.Len(t, loaded.Waits, 1)
	assert.Equal(t, ResolutionTypeTime, loaded.Waits[0].ResolutionCriteria.Type)
	require.NotNil(t, loaded.Waits[0].ResolutionCriteria.After)
}

func TestSaveProject_TaskWithAutoComplete(t *testing.T) {
	now := time.Date(2025, 12, 2, 10, 30, 0, 0, time.UTC)

	pf := &ProjectFile{
		Project: Project{
			ID:      "test",
			Prefix:  "TS",
			Name:    "Test",
			Status:  ProjectStatusActive,
			NextID:  2,
			Created: now,
		},
		Tasks: []Task{
			{
				ID:           "TS-01",
				Title:        "Auto-completing task",
				Status:       TaskStatusOpen,
				Priority:     3,
				AutoComplete: true,
				Created:      now,
				Updated:      now,
			},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "TS.yaml")
	require.NoError(t, SaveProject(path, pf))

	// Verify auto_complete appears in output
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "auto_complete: true")

	// Reload and verify
	loaded, err := LoadProject(path)
	require.NoError(t, err)
	require.Len(t, loaded.Tasks, 1)
	assert.True(t, loaded.Tasks[0].AutoComplete)
}

func TestSaveProject_EmptyLists(t *testing.T) {
	now := time.Date(2025, 12, 2, 10, 30, 0, 0, time.UTC)

	pf := &ProjectFile{
		Project: Project{
			ID:      "test",
			Prefix:  "TS",
			Name:    "Test",
			Status:  ProjectStatusActive,
			NextID:  1,
			Created: now,
		},
		Tasks: []Task{},
		Waits: []Wait{},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "TS.yaml")
	require.NoError(t, SaveProject(path, pf))

	// Read raw file - empty lists should be omitted
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.NotContains(t, content, "tasks:")
	assert.NotContains(t, content, "waits:")

	// Reload and verify
	loaded, err := LoadProject(path)
	require.NoError(t, err)
	assert.Len(t, loaded.Tasks, 0)
	assert.Len(t, loaded.Waits, 0)
}

func TestSaveProject_WaitWithBlockedByAndCheckAfter(t *testing.T) {
	now := time.Date(2025, 12, 2, 10, 30, 0, 0, time.UTC)
	checkAfter := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	pf := &ProjectFile{
		Project: Project{
			ID:      "test",
			Prefix:  "TS",
			Name:    "Test",
			Status:  ProjectStatusActive,
			NextID:  10,
			Created: now,
		},
		Tasks: []Task{
			{
				ID:       "TS-05",
				Title:    "Order PCBs",
				Status:   TaskStatusDone,
				Priority: 2,
				Created:  now,
				Updated:  now,
			},
		},
		Waits: []Wait{
			{
				ID:     "TS-08W",
				Title:  "Prototype PCBs",
				Status: WaitStatusOpen,
				ResolutionCriteria: ResolutionCriteria{
					Type:       ResolutionTypeManual,
					Question:   "Did the prototype PCBs arrive?",
					CheckAfter: &checkAfter,
				},
				BlockedBy: []string{"TS-05"},
				Notes:     "Should take 2-3 weeks",
				Created:   now,
			},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "TS.yaml")
	require.NoError(t, SaveProject(path, pf))

	// Reload and verify all fields
	loaded, err := LoadProject(path)
	require.NoError(t, err)

	require.Len(t, loaded.Waits, 1)
	w := loaded.Waits[0]
	assert.Equal(t, "TS-08W", w.ID)
	assert.Equal(t, "Prototype PCBs", w.Title)
	assert.Equal(t, WaitStatusOpen, w.Status)
	assert.Equal(t, ResolutionTypeManual, w.ResolutionCriteria.Type)
	assert.Equal(t, "Did the prototype PCBs arrive?", w.ResolutionCriteria.Question)
	require.NotNil(t, w.ResolutionCriteria.CheckAfter)
	assert.Equal(t, []string{"TS-05"}, w.BlockedBy)
	assert.Equal(t, "Should take 2-3 weeks", w.Notes)

	// Verify raw YAML contains expected fields
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "blocked_by:")
	assert.Contains(t, content, "check_after:")
}

func TestWait_DisplayText(t *testing.T) {
	afterTime := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		wait Wait
		want string
	}{
		{
			name: "with title",
			wait: Wait{
				Title: "Custom Title",
				ResolutionCriteria: ResolutionCriteria{
					Type:     ResolutionTypeManual,
					Question: "Some question?",
				},
			},
			want: "Custom Title",
		},
		{
			name: "manual without title",
			wait: Wait{
				ResolutionCriteria: ResolutionCriteria{
					Type:     ResolutionTypeManual,
					Question: "Did it arrive?",
				},
			},
			want: "Did it arrive?",
		},
		{
			name: "time without title",
			wait: Wait{
				ResolutionCriteria: ResolutionCriteria{
					Type:  ResolutionTypeTime,
					After: &afterTime,
				},
			},
			want: "Until 2026-01-15",
		},
		{
			name: "time wait with nil After returns empty",
			wait: Wait{
				ResolutionCriteria: ResolutionCriteria{
					Type:  ResolutionTypeTime,
					After: nil,
				},
			},
			want: "",
		},
		{
			name: "unknown type returns empty",
			wait: Wait{
				ResolutionCriteria: ResolutionCriteria{
					Type: "unknown",
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.wait.DisplayText())
		})
	}
}
