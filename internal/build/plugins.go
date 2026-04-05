package build

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// composeManifestFile is the path relative to workdir where the ordered list
// of compose files (base + plugins) is recorded. Start/stop/restart commands
// read this file to build the `-f` flag list for docker compose.
const composeManifestFile = ".nself/compose-files.txt"

// pluginComposeFilename is the well-known name for a plugin's Docker Compose
// fragment. Plugins that only contribute background processes (no containers)
// will not have this file and are silently skipped.
const pluginComposeFilename = "docker-compose.plugin.yml"

// DefaultPluginDir returns the default global plugin installation directory
// (~/.nself/plugins). Falls back to /tmp/.nself/plugins when the home
// directory cannot be determined.
func DefaultPluginDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "plugins")
	}
	return filepath.Join(home, ".nself", "plugins")
}

// DiscoverPluginComposeFiles scans pluginDir for installed plugins that
// contain a docker-compose.plugin.yml file. It returns absolute paths to
// each discovered compose file, sorted by plugin directory name for
// deterministic ordering. Plugins without a compose file are silently
// skipped (they are background-process plugins, not compose plugins).
func DiscoverPluginComposeFiles(workdir, pluginDir string) ([]string, error) {
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading plugin directory %s: %w", pluginDir, err)
	}

	var composePaths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip disabled plugins (those with a .disabled marker file).
		disabledPath := filepath.Join(pluginDir, entry.Name(), ".disabled")
		if _, err := os.Stat(disabledPath); err == nil {
			continue
		}
		composePath := filepath.Join(pluginDir, entry.Name(), pluginComposeFilename)
		absPath, err := filepath.Abs(composePath)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err != nil {
			continue
		}

		// Normalize stale Dockerfile references in place. Installed compose
		// files may predate the Rust→Go migration and reference "Dockerfile.go"
		// or "Dockerfile.golang" that no longer exist. Fix them so `nself build`
		// always produces a working docker-compose manifest without requiring
		// manual docker-compose.override.yml edits.
		if content, readErr := os.ReadFile(absPath); readErr == nil {
			normalized := normalizeComposeDockerfile(content, pluginDir, entry.Name())
			if !bytes.Equal(normalized, content) {
				// Write the corrected file back so the manifest references a valid compose.
				_ = os.WriteFile(absPath, normalized, 0644)
			}
		}

		composePaths = append(composePaths, absPath)
	}

	return composePaths, nil
}

// canonicalDockerfile returns the correct Dockerfile name for a plugin.
// Plugins that have been migrated from Rust to Go ship a single "Dockerfile"
// (Go multi-stage). Legacy names "Dockerfile.go" and "Dockerfile.golang" were
// used during the transition period. This function normalises to "Dockerfile"
// whenever that file actually exists in the plugin directory, regardless of
// what the installed docker-compose.plugin.yml references.
func canonicalDockerfile(pluginDir, pluginName string) string {
	canonical := filepath.Join(pluginDir, pluginName, "Dockerfile")
	if _, err := os.Stat(canonical); err == nil {
		return "Dockerfile"
	}
	return ""
}

// normalizeComposeDockerfile rewrites a plugin compose YAML in-memory so that
// any "dockerfile:" directive that references a non-existent file is corrected
// to point to "Dockerfile" when a canonical Dockerfile exists in the plugin dir.
// Returns the (possibly unchanged) content.
func normalizeComposeDockerfile(content []byte, pluginDir, pluginName string) []byte {
	canonical := canonicalDockerfile(pluginDir, pluginName)
	if canonical == "" {
		return content // no canonical Dockerfile found — leave as-is
	}

	// Legacy dockerfile names produced during the Rust→Go migration.
	legacy := []string{"Dockerfile.go", "Dockerfile.golang", "Dockerfile.rust"}

	for _, old := range legacy {
		// Only replace when the referenced file does NOT actually exist, to
		// avoid clobbering plugins that legitimately ship multiple Dockerfiles.
		oldPath := filepath.Join(pluginDir, pluginName, old)
		if _, err := os.Stat(oldPath); err == nil {
			continue // file exists — keep the reference as authored
		}
		needle := []byte("dockerfile: " + old)
		replacement := []byte("dockerfile: " + canonical)
		content = bytes.ReplaceAll(content, needle, replacement)
	}
	return content
}

// WriteComposeManifest writes .nself/compose-files.txt with one compose file
// path per line. The first line is always the base docker-compose.yml
// (absolute path). Subsequent lines are absolute paths to plugin compose
// files. The .nself/ directory is created if it does not exist.
func WriteComposeManifest(workdir string, baseCompose string, pluginFiles []string) error {
	absBase, err := filepath.Abs(baseCompose)
	if err != nil {
		return fmt.Errorf("resolving base compose path: %w", err)
	}

	manifestPath := filepath.Join(workdir, composeManifestFile)
	manifestDir := filepath.Dir(manifestPath)
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", manifestDir, err)
	}

	var lines []string
	lines = append(lines, absBase)
	lines = append(lines, pluginFiles...)

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing compose manifest: %w", err)
	}

	return nil
}

// ReadComposeManifest reads .nself/compose-files.txt and returns the ordered
// list of compose file paths. If the manifest does not exist, it returns a
// single-element slice with "docker-compose.yml" (relative, base only) so
// callers always get a usable default. Lines pointing to files that no longer
// exist on disk (e.g. a plugin was uninstalled) are silently skipped.
func ReadComposeManifest(workdir string) ([]string, error) {
	manifestPath := filepath.Join(workdir, composeManifestFile)

	f, err := os.Open(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{"docker-compose.yml"}, nil
		}
		return nil, fmt.Errorf("opening compose manifest: %w", err)
	}
	defer f.Close()

	var paths []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Skip lines pointing to files that no longer exist.
		if _, err := os.Stat(line); err != nil {
			continue
		}
		paths = append(paths, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading compose manifest: %w", err)
	}

	// If the manifest was empty or all entries were stale, fall back to base.
	if len(paths) == 0 {
		return []string{"docker-compose.yml"}, nil
	}

	return paths, nil
}

// pluginManifestMinimal holds the fields we need from plugin.json during build.
type pluginManifestMinimal struct {
	Name                 string   `json:"name"`
	Port                 int      `json:"port"`
	Language             string   `json:"language"`
	Dependencies         []string `json:"dependencies"`
	OptionalDependencies []string `json:"optionalDependencies"`
}

// readPluginManifest reads the plugin.json from a plugin directory.
// Returns nil (not an error) if the file is absent or unparseable — callers
// should skip plugins whose manifests cannot be read.
func readPluginManifest(pluginDir, name string) *pluginManifestMinimal {
	path := filepath.Join(pluginDir, name, "plugin.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var m pluginManifestMinimal
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return &m
}

// ComputePluginEnvVars returns environment variables needed by plugin compose
// files. These are written to .env.computed so that `docker compose` can
// interpolate them in plugin compose fragments.
//
// Variables returned:
//   - NSELF_PLUGIN_DIR: absolute path to the global plugin directory
//   - PLUGIN_{NAME}_INTERNAL_URL: http://plugin-{name}:{port} for every
//     declared dependency (required + optional) of every installed plugin.
//     Only wired when the dependency plugin is also installed.
func ComputePluginEnvVars(workdir, pluginDir string) map[string]string {
	vars := make(map[string]string)

	absPluginDir, err := filepath.Abs(pluginDir)
	if err != nil {
		absPluginDir = pluginDir
	}
	vars["NSELF_PLUGIN_DIR"] = absPluginDir

	// Build a port map for all installed plugins so dependency resolution is O(1).
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return vars
	}

	portByName := make(map[string]int)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m := readPluginManifest(pluginDir, entry.Name())
		if m != nil && m.Port > 0 {
			portByName[m.Name] = m.Port
			if portByName[entry.Name()] == 0 {
				portByName[entry.Name()] = m.Port // also index by dir name
			}
		}
	}

	// For each installed Go plugin, inject PLUGIN_{DEP}_INTERNAL_URL for every
	// declared dependency that is also installed. Only Go plugins communicate
	// with sibling plugins over HTTP — non-Go plugins use other IPC mechanisms.
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m := readPluginManifest(pluginDir, entry.Name())
		if m == nil || m.Language != "go" {
			continue
		}
		allDeps := append(m.Dependencies, m.OptionalDependencies...)
		for _, dep := range allDeps {
			port, ok := portByName[dep]
			if !ok || port == 0 {
				continue // dep not installed or has no port
			}
			key := "PLUGIN_" + strings.ToUpper(strings.ReplaceAll(dep, "-", "_")) + "_INTERNAL_URL"
			vars[key] = fmt.Sprintf("http://plugin-%s:%d", dep, port)
		}
	}

	return vars
}
