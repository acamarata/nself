package seed

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// EnvMatrix maps seed files to the environments they should run in.
// Key: environment name (dev, staging, prod, test, _common)
// Value: list of seed file names for that environment
type EnvMatrix map[string][]string

// BuildEnvMatrix reads the seed directory and builds the environment matrix.
// Reads subdirectory names as environment names.
// Fixtures directory is excluded (fixtures are run explicitly).
func BuildEnvMatrix(projectDir string) (EnvMatrix, error) {
	base := SeedDir(projectDir)
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return EnvMatrix{}, nil
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("reading seed dir %s: %w", base, err)
	}

	matrix := EnvMatrix{}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() == "fixtures" {
			continue
		}

		dirPath := filepath.Join(base, e.Name())
		files, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, fmt.Errorf("reading env dir %s: %w", dirPath, err)
		}

		var names []string
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
				names = append(names, f.Name())
			}
		}
		sort.Strings(names)
		matrix[e.Name()] = names
	}

	return matrix, nil
}

// FilterForEnv returns the seeds that should run in the given environment.
// Seeds in _common run in all environments.
// Seeds in {env} only run in that environment.
// Seeds in fixtures/ are excluded (run explicitly via fixture commands).
func FilterForEnv(seeds []SeedFile, env string) []SeedFile {
	var result []SeedFile
	for _, s := range seeds {
		// Exclude fixtures entirely.
		if strings.HasPrefix(s.Env, "fixture:") {
			continue
		}
		// _common seeds run everywhere.
		if s.Env == "_common" {
			result = append(result, s)
			continue
		}
		// Env-specific seeds run only in their environment.
		if s.Env == env {
			result = append(result, s)
		}
	}
	return result
}

// PrintMatrix prints the env matrix in a human-readable table format.
func PrintMatrix(matrix EnvMatrix) string {
	if len(matrix) == 0 {
		return "No seed environments found.\n"
	}

	// Sort environment names for deterministic output, _common first.
	envs := make([]string, 0, len(matrix))
	for env := range matrix {
		envs = append(envs, env)
	}
	sort.Slice(envs, func(i, j int) bool {
		if envs[i] == "_common" {
			return true
		}
		if envs[j] == "_common" {
			return false
		}
		return envs[i] < envs[j]
	})

	var sb strings.Builder
	for _, env := range envs {
		files := matrix[env]
		fmt.Fprintf(&sb, "[%s] (%d file(s))\n", env, len(files))
		for _, f := range files {
			fmt.Fprintf(&sb, "  %s\n", f)
		}
	}

	return sb.String()
}
