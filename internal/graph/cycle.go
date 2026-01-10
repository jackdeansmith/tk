package graph

// CheckCycle determines if adding an edge from `from` to `to` would create a cycle.
// The edge represents "from is blocked by to" (from → to in graph terms).
//
// Returns nil if no cycle would be created.
// Returns the cycle path if a cycle would be created, starting and ending
// with the same node to show the cycle.
//
// Example: if BY-05 → BY-03 → BY-07 exists (BY-05 blocked by BY-03, BY-03 blocked by BY-07),
// then CheckCycle("BY-07", "BY-05") would return ["BY-05", "BY-03", "BY-07", "BY-05"]
// because adding BY-07 → BY-05 would complete a cycle.
func (g *Graph) CheckCycle(from, to string) []string {
	// Self-reference is always a cycle
	if from == to {
		return []string{from, from}
	}

	// Check if we can reach `from` starting from `to` by following blockedBy edges.
	// If so, adding from → to would create a cycle.
	path := g.findPath(to, from)
	if path == nil {
		return nil
	}

	// Found a path from `to` to `from`. The cycle would be:
	// to → ... → from → to (after adding the new edge)
	// Return the cycle starting from `to`
	return append(path, to)
}

// findPath returns the path from start to target following blockedBy edges,
// or nil if no path exists. Uses DFS.
func (g *Graph) findPath(start, target string) []string {
	visited := make(map[string]bool)
	path := []string{}
	if g.dfsPath(start, target, visited, &path) {
		return path
	}
	return nil
}

// dfsPath performs DFS to find a path from current to target.
// Returns true if a path is found, with the path stored in *path.
func (g *Graph) dfsPath(current, target string, visited map[string]bool, path *[]string) bool {
	if current == target {
		*path = append(*path, current)
		return true
	}

	if visited[current] {
		return false
	}
	visited[current] = true

	*path = append(*path, current)

	for _, blockerID := range g.blockedBy[current] {
		if g.dfsPath(blockerID, target, visited, path) {
			return true
		}
	}

	// Backtrack
	*path = (*path)[:len(*path)-1]
	return false
}

// WouldCreateCycle is a convenience method that returns just a boolean.
// Use CheckCycle if you need the cycle path for error messages.
func (g *Graph) WouldCreateCycle(from, to string) bool {
	return g.CheckCycle(from, to) != nil
}

// AddEdge temporarily adds an edge for validation purposes.
// This modifies the graph in place. Use with caution.
// Returns a function to remove the edge.
func (g *Graph) AddEdge(from, to string) func() {
	g.blockedBy[from] = append(g.blockedBy[from], to)
	g.blocking[to] = append(g.blocking[to], from)

	return func() {
		// Remove the edge
		g.blockedBy[from] = removeString(g.blockedBy[from], to)
		g.blocking[to] = removeString(g.blocking[to], from)
	}
}

// removeString removes the first occurrence of s from slice.
func removeString(slice []string, s string) []string {
	for i, v := range slice {
		if v == s {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
