package build

import (
	"bufio"
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
		composePath := filepath.Join(pluginDir, entry.Name(), pluginComposeFilename)
		absPath, err := filepath.Abs(composePath)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			composePaths = append(composePaths, absPath)
		}
	}

	return composePaths, nil
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

// ComputePluginEnvVars returns environment variables needed by plugin compose
// files. These are written to .env.computed so that `docker compose` can
// interpolate them in plugin compose fragments.
//
// Variables returned:
//   - NSELF_PLUGIN_DIR: absolute path to the global plugin directory
//   - DOCKER_NETWORK: the project's Docker network name (from config or
//     computed as {project}_network)
func ComputePluginEnvVars(workdir, pluginDir string) map[string]string {
	vars := make(map[string]string)

	absPluginDir, err := filepath.Abs(pluginDir)
	if err != nil {
		absPluginDir = pluginDir
	}
	vars["NSELF_PLUGIN_DIR"] = absPluginDir

	return vars
}
