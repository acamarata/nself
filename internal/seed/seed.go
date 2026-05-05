// Package seed implements the DB seeding fixtures system for nSelf projects.
// Seeds are SQL files organized by environment and fixture type, with support
// for idempotent and destructive headers, dependency ordering, and verification.
package seed

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Layout:
//   db/seeds/
//     _common/        — run in all environments
//     dev/            — dev-only seeds
//     staging/        — staging-only seeds
//     test/           — test-only seeds
//     fixtures/
//       demo/         — demo data fixture
//       load-test/    — load-test fixture

// SeedHeader directives parsed from SQL comments.
const (
	HeaderIdempotent  = "-- +seed:idempotent"
	HeaderDestructive = "-- +seed:destructive"
	HeaderDependsOn   = "-- +seed:depends-on "
)

// SeedFile describes a single seed SQL file with parsed metadata.
type SeedFile struct {
	Path        string
	Name        string
	Env         string // _common, dev, staging, prod, or fixture name
	Idempotent  bool
	Destructive bool
	DependsOn   []string
}

// SeedDir returns the base seeds directory for a project.
func SeedDir(projectDir string) string {
	return filepath.Join(projectDir, "db", "seeds")
}

// FixtureDir returns the directory for a specific fixture.
func FixtureDir(projectDir, fixture string) string {
	return filepath.Join(SeedDir(projectDir), "fixtures", fixture)
}

// ListSeeds returns all available seed files grouped by category.
func ListSeeds(projectDir string) ([]SeedFile, error) {
	base := SeedDir(projectDir)
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return nil, nil
	}

	var seeds []SeedFile
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("read seeds dir: %w", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirName := e.Name()
		dirPath := filepath.Join(base, dirName)

		if dirName == "fixtures" {
			// Scan fixture subdirectories.
			fixtures, _ := os.ReadDir(dirPath)
			for _, f := range fixtures {
				if !f.IsDir() {
					continue
				}
				fixFiles, err := scanSeedDir(filepath.Join(dirPath, f.Name()), "fixture:"+f.Name())
				if err != nil {
					return nil, err
				}
				seeds = append(seeds, fixFiles...)
			}
		} else {
			dirFiles, err := scanSeedDir(dirPath, dirName)
			if err != nil {
				return nil, err
			}
			seeds = append(seeds, dirFiles...)
		}
	}

	return seeds, nil
}

// ListFixtures returns available fixture names.
func ListFixtures(projectDir string) ([]string, error) {
	dir := filepath.Join(SeedDir(projectDir), "fixtures")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read fixtures dir: %w", err)
	}

	var fixtures []string
	for _, e := range entries {
		if e.IsDir() {
			fixtures = append(fixtures, e.Name())
		}
	}
	return fixtures, nil
}

// CollectForRun returns the ordered list of seed files to execute for a given
// environment and optional fixture. Order: _common first, then env-specific,
// then fixture if specified.
func CollectForRun(projectDir, env, fixture string) ([]SeedFile, error) {
	base := SeedDir(projectDir)
	var all []SeedFile

	// 1. _common seeds
	commonDir := filepath.Join(base, "_common")
	if _, err := os.Stat(commonDir); err == nil {
		common, err := scanSeedDir(commonDir, "_common")
		if err != nil {
			return nil, err
		}
		all = append(all, common...)
	}

	// 2. Environment-specific seeds
	if env != "" {
		envDir := filepath.Join(base, env)
		if _, err := os.Stat(envDir); err == nil {
			envSeeds, err := scanSeedDir(envDir, env)
			if err != nil {
				return nil, err
			}
			all = append(all, envSeeds...)
		}
	}

	// 3. Fixture seeds
	if fixture != "" {
		fixDir := FixtureDir(projectDir, fixture)
		if _, err := os.Stat(fixDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("fixture %q not found at %s", fixture, fixDir)
		}
		fixSeeds, err := scanSeedDir(fixDir, "fixture:"+fixture)
		if err != nil {
			return nil, err
		}
		all = append(all, fixSeeds...)
	}

	// Sort by dependency order within each group.
	all = topoSort(all)

	return all, nil
}

// ParseHeader reads a seed file and extracts its metadata headers.
func ParseHeader(path string) (SeedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SeedFile{}, fmt.Errorf("read seed %s: %w", path, err)
	}

	sf := SeedFile{
		Path: path,
		Name: filepath.Base(path),
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "--") {
			break // headers must be at the top
		}
		switch {
		case trimmed == HeaderIdempotent:
			sf.Idempotent = true
		case trimmed == HeaderDestructive:
			sf.Destructive = true
		case strings.HasPrefix(trimmed, HeaderDependsOn):
			dep := strings.TrimSpace(strings.TrimPrefix(trimmed, HeaderDependsOn))
			if dep != "" {
				sf.DependsOn = append(sf.DependsOn, dep)
			}
		}
	}
	return sf, nil
}

// DependencyGraph returns the seed files with their dependency relationships
// as a printable adjacency list.
func DependencyGraph(projectDir string) ([]GraphNode, error) {
	seeds, err := ListSeeds(projectDir)
	if err != nil {
		return nil, err
	}

	var nodes []GraphNode
	for _, s := range seeds {
		nodes = append(nodes, GraphNode{
			Name:      s.Name,
			Env:       s.Env,
			DependsOn: s.DependsOn,
		})
	}
	return nodes, nil
}

// GraphNode represents a seed file in the dependency graph.
type GraphNode struct {
	Name      string
	Env       string
	DependsOn []string
}

// scanSeedDir reads all .sql files in a directory and parses their headers.
func scanSeedDir(dir, env string) ([]SeedFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read seed dir %s: %w", dir, err)
	}

	var files []SeedFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		sf, err := ParseHeader(path)
		if err != nil {
			return nil, err
		}
		sf.Env = env
		files = append(files, sf)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	return files, nil
}

// topoSort does a simple topological sort of seed files by their DependsOn field.
// Files without dependencies come first. Falls back to name order on cycles.
func topoSort(seeds []SeedFile) []SeedFile {
	if len(seeds) == 0 {
		return seeds
	}

	byName := make(map[string]int)
	for i, s := range seeds {
		byName[s.Name] = i
	}

	visited := make(map[int]bool)
	var result []SeedFile
	var visit func(i int)
	visiting := make(map[int]bool)

	visit = func(i int) {
		if visited[i] || visiting[i] {
			return
		}
		visiting[i] = true
		for _, dep := range seeds[i].DependsOn {
			if j, ok := byName[dep]; ok {
				visit(j)
			}
		}
		visiting[i] = false
		visited[i] = true
		result = append(result, seeds[i])
	}

	for i := range seeds {
		visit(i)
	}
	return result
}
