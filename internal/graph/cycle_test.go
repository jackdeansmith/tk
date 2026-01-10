package graph

import (
	"testing"

	"github.com/jacksmith/tk/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCheckCycle_SelfReference(t *testing.T) {
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

	// Self-reference should be detected as cycle
	cycle := g.CheckCycle("TS-01", "TS-01")
	assert.Equal(t, []string{"TS-01", "TS-01"}, cycle)
}

func TestCheckCycle_SimpleCycle(t *testing.T) {
	// A → B exists, adding B → A would create cycle
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

	// Adding TS-01 → TS-02 (TS-01 blocked by TS-02) would create cycle
	cycle := g.CheckCycle("TS-01", "TS-02")
	assert.Equal(t, []string{"TS-02", "TS-01", "TS-02"}, cycle)

	// But adding TS-03 → TS-01 would be fine (if TS-03 existed)
	assert.Nil(t, g.CheckCycle("TS-03", "TS-01"))
}

func TestCheckCycle_TransitiveCycle(t *testing.T) {
	// A → B → C exists, adding C → A would create cycle
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
			makeTask("TS-03", "TS-02"), // TS-03 blocked by TS-02
		},
	}

	g := BuildGraph(p)

	// Adding TS-01 → TS-03 (TS-01 blocked by TS-03) would create cycle
	cycle := g.CheckCycle("TS-01", "TS-03")
	assert.Equal(t, []string{"TS-03", "TS-02", "TS-01", "TS-03"}, cycle)
}

func TestCheckCycle_SeparateSubgraph(t *testing.T) {
	// A → B exists, C → D exists (separate)
	// Adding D → C would create cycle in C-D subgraph
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
			makeTask("TS-03"),
			makeTask("TS-04", "TS-03"), // TS-04 blocked by TS-03
		},
	}

	g := BuildGraph(p)

	// Adding TS-03 → TS-04 (TS-03 blocked by TS-04) would create cycle
	cycle := g.CheckCycle("TS-03", "TS-04")
	assert.Equal(t, []string{"TS-04", "TS-03", "TS-04"}, cycle)

	// But adding TS-01 → TS-03 would be fine (connects subgraphs)
	assert.Nil(t, g.CheckCycle("TS-01", "TS-03"))
}

func TestCheckCycle_NoCycleNewEdge(t *testing.T) {
	// A → B exists, adding A → C is fine
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
			makeTask("TS-03"),
		},
	}

	g := BuildGraph(p)

	// Adding TS-02 → TS-03 (TS-02 also blocked by TS-03) is fine
	assert.Nil(t, g.CheckCycle("TS-02", "TS-03"))

	// Adding TS-03 → TS-01 is fine
	assert.Nil(t, g.CheckCycle("TS-03", "TS-01"))
}

func TestCheckCycle_MixedTaskAndWait(t *testing.T) {
	// Task → Wait → Task cycle
	// TS-03 (task) → TS-01W (wait) → TS-01 (task) exists
	// Adding TS-01 → TS-03 would create cycle
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("TS-01"),
			makeTask("TS-03", "TS-01W"), // Task blocked by wait
		},
		Waits: []model.Wait{
			makeWait("TS-01W", "TS-01"), // Wait blocked by task
		},
	}

	g := BuildGraph(p)

	// Adding TS-01 → TS-03 (TS-01 blocked by TS-03) would create cycle
	cycle := g.CheckCycle("TS-01", "TS-03")
	assert.Equal(t, []string{"TS-03", "TS-01W", "TS-01", "TS-03"}, cycle)
}

func TestCheckCycle_WaitToWaitCycle(t *testing.T) {
	// W1 → W2 exists, adding W2 → W1 would create cycle
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
		Waits: []model.Wait{
			makeWait("TS-01W"),
			makeWait("TS-02W", "TS-01W"), // TS-02W blocked by TS-01W
		},
	}

	g := BuildGraph(p)

	// Adding TS-01W → TS-02W would create cycle
	cycle := g.CheckCycle("TS-01W", "TS-02W")
	assert.Equal(t, []string{"TS-02W", "TS-01W", "TS-02W"}, cycle)
}

func TestCheckCycle_LongChain(t *testing.T) {
	// TS-05 → TS-04 → TS-03 → TS-02 → TS-01
	// Adding TS-01 → TS-05 would create cycle
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
			makeTask("TS-05", "TS-04"),
		},
	}

	g := BuildGraph(p)

	cycle := g.CheckCycle("TS-01", "TS-05")
	assert.Equal(t, []string{"TS-05", "TS-04", "TS-03", "TS-02", "TS-01", "TS-05"}, cycle)
}

func TestCheckCycle_DiamondNoCycle(t *testing.T) {
	// Diamond doesn't create cycle:
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

	// Adding TS-05 → TS-04 is fine
	assert.Nil(t, g.CheckCycle("TS-05", "TS-04"))

	// Adding TS-01 → TS-04 would create cycle
	cycle := g.CheckCycle("TS-01", "TS-04")
	assert.NotNil(t, cycle)
	// The exact path depends on DFS order, but it should start and end with TS-04
	assert.Equal(t, "TS-04", cycle[0])
	assert.Equal(t, "TS-04", cycle[len(cycle)-1])
}

func TestWouldCreateCycle(t *testing.T) {
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

	assert.True(t, g.WouldCreateCycle("TS-01", "TS-02"))
	assert.True(t, g.WouldCreateCycle("TS-01", "TS-01"))
	assert.False(t, g.WouldCreateCycle("TS-02", "TS-03"))
}

func TestAddEdge(t *testing.T) {
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
	}

	g := BuildGraph(p)

	// Initially no edges
	assert.Empty(t, g.BlockedBy("TS-02"))

	// Add edge: TS-02 blocked by TS-01
	remove := g.AddEdge("TS-02", "TS-01")
	assert.Equal(t, []string{"TS-01"}, g.BlockedBy("TS-02"))
	assert.Equal(t, []string{"TS-02"}, g.Blocking("TS-01"))

	// Remove edge
	remove()
	assert.Empty(t, g.BlockedBy("TS-02"))
	assert.Empty(t, g.Blocking("TS-01"))
}

func TestAddEdge_WithCycleCheck(t *testing.T) {
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

	// Before adding, check if it would create a cycle
	if g.WouldCreateCycle("TS-01", "TS-02") {
		// This would create a cycle, so don't add
		assert.Empty(t, g.BlockedBy("TS-01"))
	} else {
		// Safe to add
		g.AddEdge("TS-01", "TS-02")
	}

	// TS-01 should still have no blockers (we didn't add the edge)
	assert.Empty(t, g.BlockedBy("TS-01"))

	// But we can add TS-03 → TS-02
	if !g.WouldCreateCycle("TS-03", "TS-02") {
		g.AddEdge("TS-03", "TS-02")
		assert.Equal(t, []string{"TS-02"}, g.BlockedBy("TS-03"))
	}
}

func TestAddEdge_RemovalIdempotent(t *testing.T) {
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
		},
	}

	g := BuildGraph(p)

	// Add edge
	remove := g.AddEdge("TS-02", "TS-01")
	assert.Equal(t, []string{"TS-01"}, g.BlockedBy("TS-02"))

	// Remove edge first time
	remove()
	assert.Empty(t, g.BlockedBy("TS-02"))

	// Remove edge second time (idempotent - should not panic or error)
	remove()
	assert.Empty(t, g.BlockedBy("TS-02"))
}

func TestCheckCycle_EmptyGraph(t *testing.T) {
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "TS",
			Name:   "Test Project",
			Status: model.ProjectStatusActive,
		},
	}

	g := BuildGraph(p)

	// No cycle possible in empty graph for non-existent nodes
	assert.Nil(t, g.CheckCycle("TS-01", "TS-02"))

	// Self-reference is still a cycle
	assert.NotNil(t, g.CheckCycle("TS-01", "TS-01"))
}

func TestCheckCycle_SpecExample(t *testing.T) {
	// From the spec: adding BY-07 → BY-05 when BY-05 → BY-03 → BY-07 exists
	// This means:
	// - BY-05 blocked by BY-03
	// - BY-03 blocked by BY-07
	// Adding: BY-07 blocked by BY-05
	p := &model.ProjectFile{
		Project: model.Project{
			ID:     "test",
			Prefix: "BY",
			Name:   "Backyard",
			Status: model.ProjectStatusActive,
		},
		Tasks: []model.Task{
			makeTask("BY-03", "BY-07"), // BY-03 blocked by BY-07
			makeTask("BY-05", "BY-03"), // BY-05 blocked by BY-03
			makeTask("BY-07"),
		},
	}

	g := BuildGraph(p)

	// Adding BY-07 → BY-05 (BY-07 blocked by BY-05) would create cycle
	cycle := g.CheckCycle("BY-07", "BY-05")

	// Expected: ["BY-05", "BY-03", "BY-07", "BY-05"]
	assert.Equal(t, []string{"BY-05", "BY-03", "BY-07", "BY-05"}, cycle)
}
