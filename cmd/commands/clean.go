package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nself/internal/config"

	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove generated artifacts (docker-compose.yml, nginx configs, build cache)",
	Long: `Clean removes generated artifacts that are recreated by 'nself build'.

Removed:
  - docker-compose.yml
  - nginx/sites/ configuration files
  - .nself/cache/ (build cache)
  - Docker build cache (exec.cachemount type)

Preserved:
  - .env and all .env.* variants
  - Docker volumes and container data
  - User-managed files

No confirmation required — this operation is non-destructive (data is preserved).
Run 'nself build' after clean to regenerate all artifacts.`,
	RunE: runClean,
}

func init() {
	RootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Find project root (best effort — clean can still work without one)
	projectDir := cwd
	if root, findErr := config.FindNSelfRoot(cwd); findErr == nil {
		projectDir = root
	}

	var removed []string

	// 1. docker-compose.yml
	composePath := filepath.Join(projectDir, "docker-compose.yml")
	if _, statErr := os.Stat(composePath); statErr == nil {
		if removeErr := os.Remove(composePath); removeErr == nil {
			removed = append(removed, "docker-compose.yml")
		} else {
			fmt.Fprintf(os.Stderr, "Warning: could not remove docker-compose.yml: %v\n", removeErr)
		}
	}

	// 2. nginx/sites/ configuration files
	nginxSitesDir := filepath.Join(projectDir, "nginx", "sites")
	if info, statErr := os.Stat(nginxSitesDir); statErr == nil && info.IsDir() {
		entries, readErr := os.ReadDir(nginxSitesDir)
		if readErr == nil {
			for _, entry := range entries {
				name := entry.Name()
				if strings.HasSuffix(name, ".conf") {
					entryPath := filepath.Join(nginxSitesDir, name)
					if removeErr := os.Remove(entryPath); removeErr == nil {
						removed = append(removed, filepath.Join("nginx", "sites", name))
					}
				}
			}
		}
	}

	// 3. .nself/cache/ directory
	nSelfCacheDir := filepath.Join(projectDir, ".nself", "cache")
	if _, statErr := os.Stat(nSelfCacheDir); statErr == nil {
		if removeErr := os.RemoveAll(nSelfCacheDir); removeErr == nil {
			removed = append(removed, ".nself/cache/")
		} else {
			fmt.Fprintf(os.Stderr, "Warning: could not remove .nself/cache/: %v\n", removeErr)
		}
	}

	// 4. Docker build cache (non-fatal if docker not available)
	fmt.Println("Pruning Docker build cache...")
	pruneCmd := exec.Command("docker", "builder", "prune",
		"--filter", "type=exec.cachemount",
		"--force",
	)
	pruneOut, pruneErr := pruneCmd.CombinedOutput()
	if pruneErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: docker builder prune failed (Docker may not be running): %v\n", pruneErr)
	} else {
		trimmed := strings.TrimSpace(string(pruneOut))
		if trimmed != "" {
			fmt.Println(trimmed)
		}
	}

	// Summary
	fmt.Println()
	if len(removed) == 0 {
		fmt.Println("Nothing to remove — already clean.")
	} else {
		fmt.Printf("Removed %d file(s):\n", len(removed))
		for _, r := range removed {
			fmt.Printf("  - %s\n", r)
		}
		fmt.Println("\nRun 'nself build' to regenerate.")
	}

	return nil
}
