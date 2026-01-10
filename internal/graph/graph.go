// Package graph provides dependency graph operations for tk.
package graph

import (
	"sort"

	"github.com/jacksmith/tk/internal/model"
)

// Graph represents a dependency graph of tasks and waits.
// Edges go from blocked items to their blockers (e.g., if A is blocked by B,
// there's an edge A → B in the blockedBy direction).
type Graph struct {
	// blockedBy maps a node ID to the IDs that block it (direct blockers)
	blockedBy map[string][]string
	// blocking maps a node ID to the IDs it directly blocks (reverse edges)
	blocking map[string][]string
	// nodes tracks all known node IDs for validation
	nodes map[string]bool
}

// BuildGraph constructs a dependency graph from a project file.
// Both tasks and waits are included as nodes, with edges representing
// blocked_by relationships.
func BuildGraph(p *model.ProjectFile) *Graph {
	g := &Graph{
		blockedBy: make(map[string][]string),
		blocking:  make(map[string][]string),
		nodes:     make(map[string]bool),
	}

	// Add all tasks as nodes
	for _, t := range p.Tasks {
		g.nodes[t.ID] = true
		if len(t.BlockedBy) > 0 {
			g.blockedBy[t.ID] = append([]string{}, t.BlockedBy...)
			// Build reverse edges
			for _, blockerID := range t.BlockedBy {
				g.blocking[blockerID] = append(g.blocking[blockerID], t.ID)
			}
		}
	}

	// Add all waits as nodes
	for _, w := range p.Waits {
		g.nodes[w.ID] = true
		if len(w.BlockedBy) > 0 {
			g.blockedBy[w.ID] = append([]string{}, w.BlockedBy...)
			// Build reverse edges
			for _, blockerID := range w.BlockedBy {
				g.blocking[blockerID] = append(g.blocking[blockerID], w.ID)
			}
		}
	}

	return g
}

// BlockedBy returns the direct blockers of the given node.
// Returns empty slice if the node doesn't exist or has no blockers.
func (g *Graph) BlockedBy(id string) []string {
	blockers := g.blockedBy[id]
	if blockers == nil {
		return []string{}
	}
	// Return a copy to prevent mutation
	result := make([]string, len(blockers))
	copy(result, blockers)
	return result
}

// Blocking returns the nodes directly blocked by the given node.
// Returns empty slice if the node doesn't exist or blocks nothing.
func (g *Graph) Blocking(id string) []string {
	blocked := g.blocking[id]
	if blocked == nil {
		return []string{}
	}
	// Return a copy to prevent mutation
	result := make([]string, len(blocked))
	copy(result, blocked)
	return result
}

// TransitiveBlockedBy returns all transitive blockers of the given node.
// This includes direct blockers and their blockers recursively.
// Returns empty slice if the node doesn't exist or has no blockers.
// The result is sorted for deterministic output.
func (g *Graph) TransitiveBlockedBy(id string) []string {
	visited := make(map[string]bool)
	g.collectBlockedBy(id, visited)
	// Remove the starting node if it was visited (shouldn't be for acyclic graph)
	delete(visited, id)

	result := make([]string, 0, len(visited))
	for nodeID := range visited {
		result = append(result, nodeID)
	}
	sort.Strings(result)
	return result
}

// collectBlockedBy is a recursive helper for TransitiveBlockedBy.
func (g *Graph) collectBlockedBy(id string, visited map[string]bool) {
	for _, blockerID := range g.blockedBy[id] {
		if !visited[blockerID] {
			visited[blockerID] = true
			g.collectBlockedBy(blockerID, visited)
		}
	}
}

// TransitiveBlocking returns all nodes transitively blocked by the given node.
// This includes nodes directly blocked and nodes they block recursively.
// Returns empty slice if the node doesn't exist or blocks nothing.
// The result is sorted for deterministic output.
func (g *Graph) TransitiveBlocking(id string) []string {
	visited := make(map[string]bool)
	g.collectBlocking(id, visited)
	// Remove the starting node if it was visited (shouldn't be for acyclic graph)
	delete(visited, id)

	result := make([]string, 0, len(visited))
	for nodeID := range visited {
		result = append(result, nodeID)
	}
	sort.Strings(result)
	return result
}

// collectBlocking is a recursive helper for TransitiveBlocking.
func (g *Graph) collectBlocking(id string, visited map[string]bool) {
	for _, blockedID := range g.blocking[id] {
		if !visited[blockedID] {
			visited[blockedID] = true
			g.collectBlocking(blockedID, visited)
		}
	}
}

// HasNode returns true if the node exists in the graph.
func (g *Graph) HasNode(id string) bool {
	return g.nodes[id]
}

// Nodes returns all node IDs in the graph (sorted for deterministic output).
func (g *Graph) Nodes() []string {
	result := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
