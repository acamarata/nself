package commands

// Purpose: Dependency-graph resolution/printing for plugin installs, split
// out of plugin.go (CLI-R12 Batch B mechanical file-size split). Backs the
// `nself plugin install --show-graph` and `--preview` flags — both resolve
// the full dependency tree without installing anything.
// Inputs: an install context, the requested plugin names, the registry URL,
// and (for preview) whether optional dependencies should be included.
// Outputs: a topologically sorted install-order printout (--show-graph) or a
// tree-formatted dependency preview (--preview) on stdout; errors on missing
// plugins or dependency cycles.
// Constraints: pure move, no behavior change. Called only from
// runPluginInstall in plugin_install.go.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/plugin"
)

// runPluginInstallShowGraph resolves plugin dependencies, detects cycles,
// builds a DAG, topologically sorts it, and prints the install order with depth.
// S3.T14: --show-graph flag for `nself plugin install --bundle`.
func runPluginInstallShowGraph(ctx context.Context, pluginNames []string, registryURL string) error {
	cacheDir := os.Getenv("NSELF_PLUGIN_CACHE")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			cacheDir = "/tmp/.nself/cache/plugins"
		} else {
			cacheDir = home + "/.nself/cache/plugins"
		}
	}

	reg, err := plugin.FetchRegistry(ctx, registryURL, cacheDir)
	if err != nil {
		return fmt.Errorf("fetching registry: %w", err)
	}

	// Build manifest lookup
	byName := make(map[string]*plugin.PluginManifest, len(reg.Plugins))
	for i := range reg.Plugins {
		byName[strings.ToLower(reg.Plugins[i].Name)] = &reg.Plugins[i]
	}

	// Collect all plugins + deps via DFS, detect cycles
	depGraph := make(map[string][]string)
	visited := make(map[string]bool)
	var visited_stack []string

	var collectAll func(string) error
	collectAll = func(name string) error {
		lname := strings.ToLower(name)
		if visited[lname] {
			return nil
		}

		// Check for cycle: is lname in the current recursion stack?
		for _, v := range visited_stack {
			if v == lname {
				return fmt.Errorf("dependency cycle detected: %s -> ... -> %s", name, name)
			}
		}

		visited_stack = append(visited_stack, lname)
		defer func() { visited_stack = visited_stack[:len(visited_stack)-1] }()

		m, found := byName[lname]
		if !found {
			return fmt.Errorf("plugin %q not found", name)
		}

		depGraph[lname] = m.Dependencies
		for _, dep := range m.Dependencies {
			if err := collectAll(dep); err != nil {
				return err
			}
		}

		visited[lname] = true
		return nil
	}

	// Collect from all requested plugins
	for _, name := range pluginNames {
		if err := collectAll(name); err != nil {
			return err
		}
	}

	// Topological sort (Kahn's algorithm)
	inDeg := make(map[string]int)
	for node := range depGraph {
		if inDeg[node] == 0 {
			inDeg[node] = 0
		}
	}
	for node := range depGraph {
		for _, dep := range depGraph[node] {
			inDeg[dep]++
		}
	}

	queue := []string{}
	for node, deg := range inDeg {
		if deg == 0 {
			queue = append(queue, node)
		}
	}
	sort.Strings(queue)

	var topoOrder []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		topoOrder = append(topoOrder, node)

		neighbors := depGraph[node]
		sort.Strings(neighbors)
		for _, neighbor := range neighbors {
			inDeg[neighbor]--
			if inDeg[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue)
			}
		}
	}

	// Print install order with depth
	depth := make(map[string]int)
	for _, node := range topoOrder {
		maxDepth := 0
		for _, dep := range depGraph[node] {
			if depth[dep] > maxDepth {
				maxDepth = depth[dep]
			}
		}
		depth[node] = maxDepth + 1
	}

	fmt.Println("Install order (topologically sorted):")
	for _, name := range topoOrder {
		indent := strings.Repeat("  ", depth[name]-1)
		deps := depGraph[name]
		depStr := ""
		if len(deps) > 0 {
			depStr = fmt.Sprintf(" → %s", strings.Join(deps, ", "))
		}
		fmt.Printf("%s├─ %s (depth %d)%s\n", indent, name, depth[name], depStr)
	}

	return nil
}

// runPluginInstallPreview resolves and prints the full dependency tree for the
// requested plugins without performing any installation. Called when
// `nself plugin install --preview` is set.
//
// Output format:
//
//	Installing <name> will also install:
//	  ├── ai (required)
//	  ├── mux (required)
//	  │   └── ai (already required)
//	  └── notify (required)
//	Optional dependencies (not installed):
//	  ├── voice
//	  └── browser
//	Use --with-optional to include these.
func runPluginInstallPreview(ctx context.Context, names []string, registryURL string, withOptional bool) error {
	cacheDir := os.Getenv("NSELF_PLUGIN_CACHE")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			cacheDir = "/tmp/.nself/cache/plugins"
		} else {
			cacheDir = home + "/.nself/cache/plugins"
		}
	}

	reg, err := plugin.FetchRegistry(ctx, registryURL, cacheDir)
	if err != nil {
		return fmt.Errorf("fetching registry: %w", err)
	}

	// Build a lookup map: plugin name → manifest.
	byName := make(map[string]*plugin.PluginManifest, len(reg.Plugins))
	for i := range reg.Plugins {
		byName[strings.ToLower(reg.Plugins[i].Name)] = &reg.Plugins[i]
	}

	// For each requested plugin, print the dep tree.
	for _, name := range names {
		lname := strings.ToLower(name)
		manifest, ok := byName[lname]
		if !ok {
			fmt.Printf("Plugin %q not found in registry.\n", name)
			continue
		}

		// Traverse required deps, collecting tree lines.
		seen := map[string]bool{lname: true}
		expanded := map[string]bool{}
		var requiredLines []string

		var collectDeps func(m *plugin.PluginManifest, childPrefix string)
		collectDeps = func(m *plugin.PluginManifest, childPrefix string) {
			deps := m.Dependencies
			for i, dep := range deps {
				dl := strings.ToLower(dep)
				isLast := i == len(deps)-1
				connector := "├──"
				nextChildPrefix := childPrefix + "│   "
				if isLast {
					connector = "└──"
					nextChildPrefix = childPrefix + "    "
				}
				note := ""
				alreadySeen := seen[dl]
				if alreadySeen {
					note = " (already required)"
				} else {
					seen[dl] = true
				}
				requiredLines = append(requiredLines, fmt.Sprintf("%s%s %s (required)%s", childPrefix, connector, dep, note))
				// Recurse into this dep's own deps if not yet expanded.
				if !alreadySeen && !expanded[dl] {
					if child, ok2 := byName[dl]; ok2 {
						expanded[dl] = true
						collectDeps(child, nextChildPrefix)
					}
				}
			}
		}

		fmt.Printf("Installing %s will also install:\n", manifest.Name)
		if len(manifest.Dependencies) == 0 {
			fmt.Println("  (no required dependencies)")
		} else {
			collectDeps(manifest, "  ")
			for _, l := range requiredLines {
				fmt.Println(l)
			}
		}

		// Optional deps.
		if len(manifest.OptionalDependencies) > 0 {
			fmt.Println("Optional dependencies (not installed):")
			for i, dep := range manifest.OptionalDependencies {
				connector := "├──"
				if i == len(manifest.OptionalDependencies)-1 {
					connector = "└──"
				}
				fmt.Printf("  %s %s\n", connector, dep)
			}
			if !withOptional {
				fmt.Println("  Use --with-optional to include these.")
			}
		}
		fmt.Println()
	}

	return nil
}
