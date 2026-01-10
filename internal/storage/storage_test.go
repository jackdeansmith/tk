package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jacksmith/tk/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Run("init in empty directory creates .tk structure", func(t *testing.T) {
		dir := t.TempDir()

		s, err := Init(dir, "", "")
		require.NoError(t, err)
		require.NotNil(t, s)

		// Verify .tk/ directory exists
		tkPath := filepath.Join(dir, ".tk")
		info, err := os.Stat(tkPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify config.yaml exists
		configPath := filepath.Join(tkPath, "config.yaml")
		_, err = os.Stat(configPath)
		require.NoError(t, err)

		// Verify projects/ directory exists
		projectsPath := filepath.Join(tkPath, "projects")
		info, err = os.Stat(projectsPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify default project file exists
		defaultProjectPath := filepath.Join(projectsPath, "DF.yaml")
		_, err = os.Stat(defaultProjectPath)
		require.NoError(t, err)
	})

	t.Run("init with custom name and prefix uses those values", func(t *testing.T) {
		dir := t.TempDir()

		s, err := Init(dir, "My Tasks", "MT")
		require.NoError(t, err)

		// Load and verify the project
		pf, err := s.LoadProject("MT")
		require.NoError(t, err)
		assert.Equal(t, "default", pf.ID)
		assert.Equal(t, "MT", pf.Prefix)
		assert.Equal(t, "My Tasks", pf.Name)
		assert.Equal(t, model.ProjectStatusActive, pf.Status)
		assert.Equal(t, 1, pf.NextID)
	})

	t.Run("init normalizes prefix to uppercase", func(t *testing.T) {
		dir := t.TempDir()

		s, err := Init(dir, "Test", "by")
		require.NoError(t, err)

		// Verify file is uppercase
		projectPath := filepath.Join(dir, ".tk", "projects", "BY.yaml")
		_, err = os.Stat(projectPath)
		require.NoError(t, err)

		// Verify project prefix is uppercase
		pf, err := s.LoadProject("BY")
		require.NoError(t, err)
		assert.Equal(t, "BY", pf.Prefix)
	})

	t.Run("init in directory with existing .tk returns error", func(t *testing.T) {
		dir := t.TempDir()

		// First init succeeds
		_, err := Init(dir, "", "")
		require.NoError(t, err)

		// Second init fails
		_, err = Init(dir, "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("default project has correct fields", func(t *testing.T) {
		dir := t.TempDir()

		s, err := Init(dir, "", "")
		require.NoError(t, err)

		pf, err := s.LoadProject("DF")
		require.NoError(t, err)

		assert.Equal(t, "default", pf.ID)
		assert.Equal(t, "DF", pf.Prefix)
		assert.Equal(t, "Default", pf.Name)
		assert.Equal(t, model.ProjectStatusActive, pf.Status)
		assert.Equal(t, 1, pf.NextID)
		assert.False(t, pf.Created.IsZero())
	})
}

func TestOpen(t *testing.T) {
	t.Run("open existing .tk directory succeeds", func(t *testing.T) {
		dir := t.TempDir()

		// Create .tk/ directory
		_, err := Init(dir, "", "")
		require.NoError(t, err)

		// Open should succeed
		s, err := Open(dir)
		require.NoError(t, err)
		require.NotNil(t, s)
		assert.Equal(t, dir, s.Root())
	})

	t.Run("open directory without .tk returns error", func(t *testing.T) {
		dir := t.TempDir()

		s, err := Open(dir)
		require.Error(t, err)
		assert.Nil(t, s)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("open non-existent directory returns error", func(t *testing.T) {
		s, err := Open("/nonexistent/path")
		require.Error(t, err)
		assert.Nil(t, s)
	})

	t.Run("open when .tk is a file returns error", func(t *testing.T) {
		dir := t.TempDir()

		// Create .tk as a file instead of directory
		tkPath := filepath.Join(dir, ".tk")
		err := os.WriteFile(tkPath, []byte("not a directory"), 0644)
		require.NoError(t, err)

		s, err := Open(dir)
		require.Error(t, err)
		assert.Nil(t, s)
		assert.Contains(t, err.Error(), "not a directory")
	})
}

func TestLoadProject(t *testing.T) {
	t.Run("load existing project by prefix succeeds", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "Test Project", "TP")
		require.NoError(t, err)

		pf, err := s.LoadProject("TP")
		require.NoError(t, err)
		assert.Equal(t, "Test Project", pf.Name)
		assert.Equal(t, "TP", pf.Prefix)
	})

	t.Run("load project is case-insensitive", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "Test Project", "BY")
		require.NoError(t, err)

		// All these should work
		for _, prefix := range []string{"BY", "by", "By", "bY"} {
			pf, err := s.LoadProject(prefix)
			require.NoError(t, err, "prefix %q should load", prefix)
			assert.Equal(t, "BY", pf.Prefix)
		}
	})

	t.Run("load non-existent prefix returns clear error", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		_, err = s.LoadProject("XX")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.Contains(t, err.Error(), "XX")
	})
}

func TestLoadProjectByID(t *testing.T) {
	t.Run("load existing project by ID succeeds", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		pf, err := s.LoadProjectByID("default")
		require.NoError(t, err)
		assert.Equal(t, "default", pf.ID)
	})

	t.Run("load project by ID is case-insensitive", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// All these should work
		for _, id := range []string{"default", "DEFAULT", "Default"} {
			pf, err := s.LoadProjectByID(id)
			require.NoError(t, err, "id %q should load", id)
			assert.Equal(t, "default", pf.ID)
		}
	})

	t.Run("load non-existent ID returns error", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		_, err = s.LoadProjectByID("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("skips malformed project files", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Create a valid second project
		now := time.Now()
		project := &model.ProjectFile{
			Project: model.Project{
				ID:      "myproject",
				Prefix:  "MP",
				Name:    "My Project",
				Status:  model.ProjectStatusActive,
				NextID:  1,
				Created: now,
			},
		}
		err = s.SaveProject(project)
		require.NoError(t, err)

		// Create a malformed yaml file
		badFile := filepath.Join(dir, ".tk", "projects", "BAD.yaml")
		err = os.WriteFile(badFile, []byte("not: [valid yaml"), 0644)
		require.NoError(t, err)

		// Should still find the valid project
		pf, err := s.LoadProjectByID("myproject")
		require.NoError(t, err)
		assert.Equal(t, "myproject", pf.ID)
	})
}

func TestSaveProject(t *testing.T) {
	t.Run("save project creates file at correct path", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Create new project
		now := time.Now()
		newProject := &model.ProjectFile{
			Project: model.Project{
				ID:      "test",
				Prefix:  "TS",
				Name:    "Test Project",
				Status:  model.ProjectStatusActive,
				NextID:  1,
				Created: now,
			},
		}

		err = s.SaveProject(newProject)
		require.NoError(t, err)

		// Verify file exists
		projectPath := filepath.Join(dir, ".tk", "projects", "TS.yaml")
		_, err = os.Stat(projectPath)
		require.NoError(t, err)
	})

	t.Run("save preserves all fields", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		now := time.Now().Truncate(time.Second)
		dueDate := now.Add(24 * time.Hour)
		doneAt := now.Add(time.Hour)

		project := &model.ProjectFile{
			Project: model.Project{
				ID:          "full",
				Prefix:      "FL",
				Name:        "Full Project",
				Description: "A project with all fields",
				Status:      model.ProjectStatusActive,
				NextID:      10,
				Created:     now,
			},
			Tasks: []model.Task{
				{
					ID:           "FL-01",
					Title:        "Test Task",
					Status:       model.TaskStatusDone,
					Priority:     2,
					BlockedBy:    []string{"FL-02"},
					Tags:         []string{"test", "example"},
					Notes:        "Some notes\nwith multiple lines",
					Assignee:     "John",
					DueDate:      &dueDate,
					AutoComplete: true,
					Created:      now,
					Updated:      now,
					DoneAt:       &doneAt,
				},
			},
			Waits: []model.Wait{
				{
					ID:     "FL-01W",
					Title:  "Test Wait",
					Status: model.WaitStatusOpen,
					ResolutionCriteria: model.ResolutionCriteria{
						Type:     model.ResolutionTypeManual,
						Question: "Did it happen?",
					},
					Notes:   "Wait notes",
					Created: now,
				},
			},
		}

		err = s.SaveProject(project)
		require.NoError(t, err)

		// Reload and verify
		loaded, err := s.LoadProject("FL")
		require.NoError(t, err)

		// Verify project fields
		assert.Equal(t, project.ID, loaded.ID)
		assert.Equal(t, project.Prefix, loaded.Prefix)
		assert.Equal(t, project.Name, loaded.Name)
		assert.Equal(t, project.Description, loaded.Description)
		assert.Equal(t, project.Status, loaded.Status)
		assert.Equal(t, project.NextID, loaded.NextID)

		// Verify task fields
		require.Len(t, loaded.Tasks, 1)
		task := loaded.Tasks[0]
		assert.Equal(t, "FL-01", task.ID)
		assert.Equal(t, "Test Task", task.Title)
		assert.Equal(t, model.TaskStatusDone, task.Status)
		assert.Equal(t, 2, task.Priority)
		assert.Equal(t, []string{"FL-02"}, task.BlockedBy)
		assert.Equal(t, []string{"test", "example"}, task.Tags)
		assert.Contains(t, task.Notes, "Some notes")
		assert.Equal(t, "John", task.Assignee)
		assert.NotNil(t, task.DueDate)
		assert.True(t, task.AutoComplete)
		assert.NotNil(t, task.DoneAt)

		// Verify wait fields
		require.Len(t, loaded.Waits, 1)
		wait := loaded.Waits[0]
		assert.Equal(t, "FL-01W", wait.ID)
		assert.Equal(t, "Test Wait", wait.Title)
		assert.Equal(t, model.WaitStatusOpen, wait.Status)
		assert.Equal(t, model.ResolutionTypeManual, wait.ResolutionCriteria.Type)
		assert.Equal(t, "Did it happen?", wait.ResolutionCriteria.Question)
	})
}

func TestListProjects(t *testing.T) {
	t.Run("empty projects directory returns empty list", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Delete the default project
		err = s.DeleteProject("DF")
		require.NoError(t, err)

		prefixes, err := s.ListProjects()
		require.NoError(t, err)
		assert.Empty(t, prefixes)
	})

	t.Run("multiple projects returns all prefixes", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Create additional projects
		now := time.Now()
		for _, prefix := range []string{"AA", "BB", "CC"} {
			project := &model.ProjectFile{
				Project: model.Project{
					ID:      "project-" + prefix,
					Prefix:  prefix,
					Name:    "Project " + prefix,
					Status:  model.ProjectStatusActive,
					NextID:  1,
					Created: now,
				},
			}
			err := s.SaveProject(project)
			require.NoError(t, err)
		}

		prefixes, err := s.ListProjects()
		require.NoError(t, err)

		// Should have DF (default) + AA, BB, CC
		assert.Len(t, prefixes, 4)
		assert.Contains(t, prefixes, "DF")
		assert.Contains(t, prefixes, "AA")
		assert.Contains(t, prefixes, "BB")
		assert.Contains(t, prefixes, "CC")
	})

	t.Run("ignores directories in projects folder", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Create a subdirectory in projects folder
		subdir := filepath.Join(dir, ".tk", "projects", "subdir")
		err = os.Mkdir(subdir, 0755)
		require.NoError(t, err)

		prefixes, err := s.ListProjects()
		require.NoError(t, err)

		// Should only have DF, not "subdir"
		assert.Len(t, prefixes, 1)
		assert.Contains(t, prefixes, "DF")
	})

	t.Run("ignores non-yaml files in projects folder", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Create a non-yaml file in projects folder
		txtFile := filepath.Join(dir, ".tk", "projects", "README.txt")
		err = os.WriteFile(txtFile, []byte("readme"), 0644)
		require.NoError(t, err)

		prefixes, err := s.ListProjects()
		require.NoError(t, err)

		// Should only have DF, not "README"
		assert.Len(t, prefixes, 1)
		assert.Contains(t, prefixes, "DF")
	})
}

func TestDeleteProject(t *testing.T) {
	t.Run("delete existing project succeeds", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Verify project exists
		assert.True(t, s.ProjectExists("DF"))

		// Delete it
		err = s.DeleteProject("DF")
		require.NoError(t, err)

		// Verify it's gone
		assert.False(t, s.ProjectExists("DF"))
	})

	t.Run("delete is case-insensitive", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "BY")
		require.NoError(t, err)

		err = s.DeleteProject("by")
		require.NoError(t, err)

		assert.False(t, s.ProjectExists("BY"))
	})

	t.Run("delete non-existent project returns error", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		err = s.DeleteProject("XX")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestProjectExists(t *testing.T) {
	t.Run("returns true for existing project", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "BY")
		require.NoError(t, err)

		assert.True(t, s.ProjectExists("BY"))
	})

	t.Run("returns false for non-existent project", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		assert.False(t, s.ProjectExists("XX"))
	})

	t.Run("is case-insensitive", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "BY")
		require.NoError(t, err)

		assert.True(t, s.ProjectExists("BY"))
		assert.True(t, s.ProjectExists("by"))
		assert.True(t, s.ProjectExists("By"))
	})
}

func TestStoragePaths(t *testing.T) {
	t.Run("Root returns correct path", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		assert.Equal(t, dir, s.Root())
	})

	t.Run("TkPath returns .tk directory path", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		expected := filepath.Join(dir, ".tk")
		assert.Equal(t, expected, s.TkPath())
	})
}
