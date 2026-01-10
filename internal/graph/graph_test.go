package graph

import (
	"testing"
	"time"

	"github.com/jacksmith/tk/internal/model"
	"github.com/stretchr/testify/assert"
)

// Helper to create a minimal task for testing
func makeTask(id string, blockedBy ...string) model.Task {
	return model.Task{
		ID:        id,
		Title:     "Task " + id,
		Status:    model.TaskStatusOpen,
		Priority:  2,
		BlockedBy: blockedBy,
		Created:   time.Now(),
		Updated:   time.Now(),
	}
}

// Helper to create a minimal wait for testing
func makeWait(id string, blockedBy ...string) model.Wait {
	return model.Wait{
		ID:        id,
		Status:    model.WaitStatusOpen,
		BlockedBy: blockedBy,
		ResolutionCriteria: model.ResolutionCriteria{
			Type:     model.ResolutionTypeManual,
			Question: "Question for " + id,
		},
		Created: time.Now(),
	}
}

func TestBuildGraph_EmptyProject(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{},
		Waits: []model.Wait{},
	}

	g := BuildGraph(p)

	assert.Empty(t, g.Nodes(), "empty project should have no nodes")
}

func TestBuildGraph_TasksWithNoDependencies(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02"),
			makeTask("TS-03"),
		},
		Waits: []model.Wait{},
	}

	g := BuildGraph(p)

	assert.ElementsMatch(t, []string{"TS-01", "TS-02", "TS-03"}, g.Nodes())
	assert.True(t, g.HasNode("TS-01"))
	assert.True(t, g.HasNode("TS-02"))
	assert.True(t, g.HasNode("TS-03"))
	assert.False(t, g.HasNode("TS-99"))

	// No edges
	assert.Empty(t, g.BlockedBy("TS-01"))
	assert.Empty(t, g.BlockedBy("TS-02"))
	assert.Empty(t, g.Blocking("TS-01"))
}

func TestBuildGraph_TaskBlockedByTask(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02", "TS-01"), // TS-02 blocked by TS-01
		},
	}

	g := BuildGraph(p)

	assert.Equal(t, []string{"TS-01"}, g.BlockedBy("TS-02"))
	assert.Equal(t, []string{"TS-02"}, g.Blocking("TS-01"))
	assert.Empty(t, g.BlockedBy("TS-01"))
	assert.Empty(t, g.Blocking("TS-02"))
}

func TestBuildGraph_TaskBlockedByWait(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01", "TS-01W"), // TS-01 blocked by wait TS-01W
		},
		Waits: []model.Wait{
			makeWait("TS-01W"),
		},
	}

	g := BuildGraph(p)

	assert.Equal(t, []string{"TS-01W"}, g.BlockedBy("TS-01"))
	assert.Equal(t, []string{"TS-01"}, g.Blocking("TS-01W"))
}

func TestBuildGraph_WaitBlockedByTask(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
		},
		Waits: []model.Wait{
			makeWait("TS-01W", "TS-01"), // Wait blocked by task
		},
	}

	g := BuildGraph(p)

	assert.Equal(t, []string{"TS-01"}, g.BlockedBy("TS-01W"))
	assert.Equal(t, []string{"TS-01W"}, g.Blocking("TS-01"))
}

func TestBuildGraph_WaitBlockedByWait(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Waits: []model.Wait{
			makeWait("TS-01W"),
			makeWait("TS-02W", "TS-01W"), // Wait blocked by another wait
		},
	}

	g := BuildGraph(p)

	assert.Equal(t, []string{"TS-01W"}, g.BlockedBy("TS-02W"))
	assert.Equal(t, []string{"TS-02W"}, g.Blocking("TS-01W"))
}

func TestBlockedBy_ReturnsOnlyDirectBlockers(t *testing.T) {
	// TS-03 → TS-02 → TS-01 (TS-03 blocked by TS-02, TS-02 blocked by TS-01)
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02", "TS-01"),
			makeTask("TS-03", "TS-02"),
		},
	}

	g := BuildGraph(p)

	// BlockedBy should return only direct blockers
	assert.Equal(t, []string{"TS-02"}, g.BlockedBy("TS-03"))
	assert.Equal(t, []string{"TS-01"}, g.BlockedBy("TS-02"))
	assert.Empty(t, g.BlockedBy("TS-01"))
}

func TestBlocking_ReturnsOnlyDirectDependents(t *testing.T) {
	// TS-03 → TS-02 → TS-01
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02", "TS-01"),
			makeTask("TS-03", "TS-02"),
		},
	}

	g := BuildGraph(p)

	// Blocking should return only direct dependents
	assert.Equal(t, []string{"TS-02"}, g.Blocking("TS-01"))
	assert.Equal(t, []string{"TS-03"}, g.Blocking("TS-02"))
	assert.Empty(t, g.Blocking("TS-03"))
}

func TestTransitiveBlockedBy(t *testing.T) {
	// TS-04 → TS-03 → TS-02 → TS-01
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02", "TS-01"),
			makeTask("TS-03", "TS-02"),
			makeTask("TS-04", "TS-03"),
		},
	}

	g := BuildGraph(p)

	// TransitiveBlockedBy should follow full chain
	assert.ElementsMatch(t, []string{"TS-01", "TS-02", "TS-03"}, g.TransitiveBlockedBy("TS-04"))
	assert.ElementsMatch(t, []string{"TS-01", "TS-02"}, g.TransitiveBlockedBy("TS-03"))
	assert.ElementsMatch(t, []string{"TS-01"}, g.TransitiveBlockedBy("TS-02"))
	assert.Empty(t, g.TransitiveBlockedBy("TS-01"))
}

func TestTransitiveBlocking(t *testing.T) {
	// TS-04 → TS-03 → TS-02 → TS-01
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02", "TS-01"),
			makeTask("TS-03", "TS-02"),
			makeTask("TS-04", "TS-03"),
		},
	}

	g := BuildGraph(p)

	// TransitiveBlocking should follow full chain
	assert.ElementsMatch(t, []string{"TS-02", "TS-03", "TS-04"}, g.TransitiveBlocking("TS-01"))
	assert.ElementsMatch(t, []string{"TS-03", "TS-04"}, g.TransitiveBlocking("TS-02"))
	assert.ElementsMatch(t, []string{"TS-04"}, g.TransitiveBlocking("TS-03"))
	assert.Empty(t, g.TransitiveBlocking("TS-04"))
}

func TestTransitiveQueries_DiamondGraph(t *testing.T) {
	// Diamond dependency:
	//     TS-01
	//    /      \
	// TS-02    TS-03
	//    \      /
	//     TS-04
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02", "TS-01"),
			makeTask("TS-03", "TS-01"),
			makeTask("TS-04", "TS-02", "TS-03"),
		},
	}

	g := BuildGraph(p)

	// TS-04 should have all three as transitive blockers
	assert.ElementsMatch(t, []string{"TS-01", "TS-02", "TS-03"}, g.TransitiveBlockedBy("TS-04"))

	// TS-01 should transitively block all three
	assert.ElementsMatch(t, []string{"TS-02", "TS-03", "TS-04"}, g.TransitiveBlocking("TS-01"))
}

func TestQueries_NonExistentID(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
		},
	}

	g := BuildGraph(p)

	// Queries on non-existent ID should return empty, not error
	assert.Empty(t, g.BlockedBy("TS-99"))
	assert.Empty(t, g.Blocking("TS-99"))
	assert.Empty(t, g.TransitiveBlockedBy("TS-99"))
	assert.Empty(t, g.TransitiveBlocking("TS-99"))
	assert.False(t, g.HasNode("TS-99"))
}

func TestBlockedBy_ReturnsCopy(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02", "TS-01"),
		},
	}

	g := BuildGraph(p)

	// Modifying the returned slice should not affect the graph
	blockers := g.BlockedBy("TS-02")
	blockers[0] = "MODIFIED"

	assert.Equal(t, []string{"TS-01"}, g.BlockedBy("TS-02"))
}

func TestBlocking_ReturnsCopy(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02", "TS-01"),
		},
	}

	g := BuildGraph(p)

	// Modifying the returned slice should not affect the graph
	blocking := g.Blocking("TS-01")
	blocking[0] = "MODIFIED"

	assert.Equal(t, []string{"TS-02"}, g.Blocking("TS-01"))
}

func TestMixedTasksAndWaits(t *testing.T) {
	// Complex graph with tasks and waits:
	// TS-03 (task) → TS-01W (wait) → TS-01 (task)
	// TS-04 (task) → TS-02W (wait) → TS-02 (task)
	// TS-05 (task) → TS-03, TS-04
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02"),
			makeTask("TS-03", "TS-01W"),
			makeTask("TS-04", "TS-02W"),
			makeTask("TS-05", "TS-03", "TS-04"),
		},
		Waits: []model.Wait{
			makeWait("TS-01W", "TS-01"),
			makeWait("TS-02W", "TS-02"),
		},
	}

	g := BuildGraph(p)

	// Check direct relationships
	assert.ElementsMatch(t, []string{"TS-03", "TS-04"}, g.BlockedBy("TS-05"))
	assert.Equal(t, []string{"TS-01W"}, g.BlockedBy("TS-03"))
	assert.Equal(t, []string{"TS-01"}, g.BlockedBy("TS-01W"))

	// Check transitive relationships
	blockers := g.TransitiveBlockedBy("TS-05")
	assert.ElementsMatch(t, []string{"TS-01", "TS-02", "TS-01W", "TS-02W", "TS-03", "TS-04"}, blockers)

	// Check what TS-01 transitively blocks
	blocking := g.TransitiveBlocking("TS-01")
	assert.ElementsMatch(t, []string{"TS-01W", "TS-03", "TS-05"}, blocking)
}

func TestNodes_Sorted(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-03"),
			makeTask("TS-01"),
			makeTask("TS-02"),
		},
	}

	g := BuildGraph(p)

	// Nodes should be sorted
	assert.Equal(t, []string{"TS-01", "TS-02", "TS-03"}, g.Nodes())
}

func TestTransitiveBlockedBy_Sorted(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02"),
			makeTask("TS-03"),
			makeTask("TS-04", "TS-01", "TS-02", "TS-03"),
		},
	}

	g := BuildGraph(p)

	// Result should be sorted
	assert.Equal(t, []string{"TS-01", "TS-02", "TS-03"}, g.TransitiveBlockedBy("TS-04"))
}

func TestTransitiveBlocking_Sorted(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-02", "TS-01"),
			makeTask("TS-03", "TS-01"),
			makeTask("TS-04", "TS-01"),
		},
	}

	g := BuildGraph(p)

	// Result should be sorted
	assert.Equal(t, []string{"TS-02", "TS-03", "TS-04"}, g.TransitiveBlocking("TS-01"))
}
