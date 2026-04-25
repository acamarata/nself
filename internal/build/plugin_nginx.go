package build

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/config"
)

// checkPluginRouteConflict checks if a plugin's nginx route config conflicts
// with an existing config in the sites directory or a core service route.
func checkPluginRouteConflict(workdir string, pluginName string, destName string) error {
	sitesDir := filepath.Join(workdir, "nginx", "sites")

	// Check if dest file already exists (not written by us in this run).
	destPath := filepath.Join(sitesDir, destName)
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("nginx route conflict: %s already exists in %s", destName, sitesDir)
	}

	// Check core service routes — files without the plugin prefix.
	coreConf := filepath.Join(sitesDir, pluginName+".conf")
	if _, err := os.Stat(coreConf); err == nil {
		return fmt.Errorf("nginx route conflict: %s.conf already exists as a core service route", pluginName)
	}

	return nil
}

// InjectPluginNginxRoutes scans pluginDir for installed plugins that ship
// nginx route configs (in a nginx/ subdirectory) and copies them into the
// project's nginx/sites/ directory after templating config variables.
//
// Returns the number of config files injected.
// Plugins without a nginx/ directory are silently skipped.
func InjectPluginNginxRoutes(workdir, pluginDir string, cfg *config.Config) (int, error) {
	if pluginDir == "" {
		pluginDir = cfg.PluginSystem.Dir
	}
	if pluginDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return 0, fmt.Errorf("resolving home directory: %w", err)
		}
		pluginDir = filepath.Join(home, ".nself", "plugins")
	}

	// If the plugin directory does not exist, there is nothing to inject.
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return 0, nil
	}

	// Reuse the shared template variable map.
	vars := buildTemplateVars(workdir, cfg)

	// Scan each plugin directory for nginx/*.conf files.
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return 0, fmt.Errorf("reading plugin directory %s: %w", pluginDir, err)
	}

	count := 0
	sitesDir := filepath.Join(workdir, "nginx", "sites")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginName := entry.Name()
		nginxDir := filepath.Join(pluginDir, pluginName, "nginx")

		// Skip plugins without an nginx/ directory.
		if _, err := os.Stat(nginxDir); os.IsNotExist(err) {
			continue
		}

		// Glob for .conf files inside the plugin's nginx/ directory.
		pattern := filepath.Join(nginxDir, "*.conf")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return count, fmt.Errorf("globbing %s: %w", pattern, err)
		}

		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				return count, fmt.Errorf("reading %s: %w", match, err)
			}

			// Template the config content with project values.
			// Reuses renderTemplate from plugin_configs.go.
			rendered := renderTemplate(string(content), vars)

			// Write to nginx/sites/{pluginname}-{filename} to avoid conflicts.
			filename := filepath.Base(match)
			destName := pluginName + "-" + filename
			destPath := filepath.Join(sitesDir, destName)

			// Warn on route conflicts but do not block — plugin may be
			// replacing its own config on update.
			if err := checkPluginRouteConflict(workdir, pluginName, destName); err != nil {
				slog.Warn("plugin nginx route conflict — proceeding anyway", "plugin", pluginName, "err", err)
			}

			if err := os.WriteFile(destPath, []byte(rendered), 0644); err != nil {
				return count, fmt.Errorf("writing %s: %w", destPath, err)
			}
			count++
		}
	}

	return count, nil
}
