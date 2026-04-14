package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start development environment",
	Long: `Start the nSelf stack in development mode.

  nself dev         # Start in dev mode
  nself dev --hot   # Start with plugin hot-reload on file changes`,
	RunE: runDev,
}

func init() {
	devCmd.Flags().Bool("hot", false, "Enable plugin hot-reload on source changes")
	RootCmd.AddCommand(devCmd)
}

func runDev(cmd *cobra.Command, args []string) error {
	hot, _ := cmd.Flags().GetBool("hot")

	rawCwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	workdir, err := config.FindNSelfRoot(rawCwd)
	if err != nil {
		return fmt.Errorf("no nself project found. Run 'nself init' first")
	}

	ui.CommandHeader("nSelf Dev", "Development environment")

	// Start the stack
	ui.Info("Starting development stack...")
	startCmd := exec.CommandContext(cmd.Context(), "docker", "compose", "up", "-d")
	startCmd.Dir = workdir
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("starting stack: %w", err)
	}

	ui.Success("Development stack started")

	if !hot {
		return nil
	}

	// Hot-reload mode: watch plugin directories for changes
	ui.Info("Hot-reload enabled. Watching plugin sources for changes...")
	ui.Dimmed("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		ui.Info("Shutting down hot-reload watcher...")
		cancel()
	}()

	return watchAndReload(ctx, workdir)
}

// watchAndReload polls plugin directories for file changes and rebuilds changed plugins.
// Uses polling instead of fsnotify to avoid an external dependency.
func watchAndReload(ctx context.Context, workdir string) error {
	pluginsDir := filepath.Join(workdir, "plugins")
	if _, err := os.Stat(pluginsDir); os.IsNotExist(err) {
		ui.Warn("No plugins/ directory found. Hot-reload has nothing to watch.")
		<-ctx.Done()
		return nil
	}

	// Build initial state: map of file path -> mod time
	state := scanPluginFiles(pluginsDir)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			newState := scanPluginFiles(pluginsDir)
			changed := detectChangedPlugins(state, newState, pluginsDir)
			state = newState

			for _, pluginName := range changed {
				ui.Info(fmt.Sprintf("Change detected in plugin %q, rebuilding...", pluginName))
				if err := rebuildPlugin(ctx, workdir, pluginName); err != nil {
					ui.Warn(fmt.Sprintf("Rebuild failed for %s: %v", pluginName, err))
				} else {
					ui.Success(fmt.Sprintf("Plugin %q reloaded", pluginName))
				}
			}
		}
	}
}

// scanPluginFiles returns a map of file path -> modification time for all files in pluginsDir.
func scanPluginFiles(pluginsDir string) map[string]time.Time {
	state := make(map[string]time.Time)
	_ = filepath.Walk(pluginsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		// Skip hidden files and common non-source directories
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") {
			return nil
		}
		state[path] = info.ModTime()
		return nil
	})
	return state
}

// detectChangedPlugins compares old and new state, returns plugin names that changed.
func detectChangedPlugins(old, new map[string]time.Time, pluginsDir string) []string {
	changedSet := make(map[string]bool)

	for path, newMod := range new {
		oldMod, exists := old[path]
		if !exists || !oldMod.Equal(newMod) {
			pluginName := extractPluginName(path, pluginsDir)
			if pluginName != "" {
				changedSet[pluginName] = true
			}
		}
	}

	var changed []string
	for name := range changedSet {
		changed = append(changed, name)
	}
	return changed
}

// extractPluginName extracts the top-level plugin directory name from a file path.
func extractPluginName(filePath, pluginsDir string) string {
	rel, err := filepath.Rel(pluginsDir, filePath)
	if err != nil {
		return ""
	}
	parts := strings.SplitN(rel, string(filepath.Separator), 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// rebuildPlugin rebuilds a single plugin's container without restarting others.
func rebuildPlugin(ctx context.Context, workdir, pluginName string) error {
	// docker compose up -d --no-deps --build <service>
	rebuildCmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d", "--no-deps", "--build", pluginName)
	rebuildCmd.Dir = workdir
	rebuildCmd.Stdout = os.Stdout
	rebuildCmd.Stderr = os.Stderr
	return rebuildCmd.Run()
}
