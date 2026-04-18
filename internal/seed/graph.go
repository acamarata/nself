package seed

import "fmt"

// ResolveDependencyOrder returns seeds in topologically sorted execution order
// (dependencies first). Returns an error if there are circular dependencies.
func ResolveDependencyOrder(seeds []SeedFile) ([]SeedFile, error) {
	return topologicalSort(seeds)
}

// topologicalSort implements Kahn's algorithm for topological sort.
// Returns sorted slice or error on cycle detection.
func topologicalSort(seeds []SeedFile) ([]SeedFile, error) {
	if len(seeds) == 0 {
		return seeds, nil
	}

	// Map seed name → index.
	idx := make(map[string]int, len(seeds))
	for i, s := range seeds {
		idx[s.Name] = i
	}

	// Build in-degree map and adjacency list (dependency → dependents).
	inDegree := make([]int, len(seeds))
	adj := make([][]int, len(seeds)) // adj[i] = list of indices that depend on seeds[i]

	for i, s := range seeds {
		for _, dep := range s.DependsOn {
			j, ok := idx[dep]
			if !ok {
				// Dependency not in set; skip (may be external).
				continue
			}
			adj[j] = append(adj[j], i)
			inDegree[i]++
		}
	}

	// Queue all nodes with in-degree 0.
	queue := make([]int, 0, len(seeds))
	for i, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, i)
		}
	}

	result := make([]SeedFile, 0, len(seeds))
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, seeds[cur])

		for _, neighbor := range adj[cur] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(seeds) {
		// Cycle exists — collect names of seeds still in the cycle.
		inCycle := make([]string, 0)
		for i, deg := range inDegree {
			if deg > 0 {
				inCycle = append(inCycle, seeds[i].Name)
			}
		}
		return nil, fmt.Errorf("circular dependency detected among seeds: %v", inCycle)
	}

	return result, nil
}
