package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nself-org/cli/internal/build"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/docker"
	"github.com/nself-org/cli/internal/lifecycle"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:     "stop [SERVICES...]",
	Aliases: []string{"down"},
	Short:   "Gracefully shut down all services or specific services",
	Long: `Stop shuts down the nSelf stack. By default it performs a graceful stop
(sending SIGTERM and waiting for the timeout) before running compose down.

If --volumes or --rmi is specified, the graceful stop phase is skipped and
compose down runs immediately with the requested cleanup flags.`,
	RunE: runStop,
}

func init() {
	stopCmd.Flags().BoolP("volumes", "v", false, "Remove volumes (WARNING: deletes data)")
	stopCmd.Flags().Bool("rmi", false, "Remove Docker images")
	stopCmd.Flags().Bool("remove-images", false, "Alias for --rmi")
	stopCmd.Flags().Bool("remove-orphans", false, "Remove orphaned containers")
	stopCmd.Flags().Int("graceful", 30, "Graceful shutdown timeout in seconds")
	stopCmd.Flags().Bool("verbose", false, "Show detailed output")
	stopCmd.Flags().Bool("no-monorepo", false, "Disable automatic monorepo backend detection")

	RootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	volumes, _ := cmd.Flags().GetBool("volumes")
	rmi, _ := cmd.Flags().GetBool("rmi")
	removeImages, _ := cmd.Flags().GetBool("remove-images")
	removeOrphans, _ := cmd.Flags().GetBool("remove-orphans")
	gracefulTimeout, _ := cmd.Flags().GetInt("graceful")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// --remove-images is an alias for --rmi
	if removeImages {
		rmi = true
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	lifecycle.TrapSignals(ctx, cancel, func() {}, 10*time.Second)

	// Locate the nself project root (handles monorepo .backend/ layout).
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine working directory: %w", err)
	}

	// ── Monorepo detection ────────────────────────────────────────────────
	// Check before FindNSelfRoot so that users running nself stop from the
	// monorepo root (e.g. the nself-web Turborepo) are redirected into the
	// correct backend sub-directory automatically.
	noMonorepo, _ := cmd.Flags().GetBool("no-monorepo")
	if !noMonorepo {
		if backendRoot := config.DetectMonorepoRoot(cwd); backendRoot != "" {
			fmt.Printf("→ Detected monorepo layout. Using %s as project root.\n", filepath.Base(backendRoot))
			cwd = backendRoot
		}
	}

	workdir, err := config.FindNSelfRoot(cwd)
	if err != nil {
		return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
	}

	// Verify docker-compose.yml exists.
	composePath := workdir + "/docker-compose.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("no docker-compose.yml found in %s — run 'nself build' first", workdir)
	}

	// Read compose file manifest for multi-file support.
	composeFiles, err := build.ReadComposeManifest(workdir)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: could not read compose manifest: %v (using defaults)\n", err)
		}
		composeFiles = nil
	}

	compose := docker.NewCompose(composeFiles...)

	// Check running containers.
	containers, err := compose.ComposePs(ctx, workdir)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: could not list containers: %v\n", err)
		}
	}

	if len(containers) == 0 && len(args) == 0 {
		fmt.Println("No running containers found.")
		if volumes || rmi || removeOrphans {
			fmt.Println("Running cleanup with requested flags...")
		} else {
			return nil
		}
	}

	// If specific services are requested, stop only those.
	if len(args) > 0 {
		if verbose {
			fmt.Printf("Stopping services: %v\n", args)
		}
		// Use ComposeStop for specific services (graceful stop with timeout).
		// ComposeStop is not available, so build args via the compose instance's
		// Run method. The -f flags are already embedded in buildBaseArgs via
		// ComposeRestart/ComposeDown, but for raw "stop" we need to construct
		// them ourselves. Use the compose base prefix helper.
		stopArgs := composeBaseArgs(composeFiles)
		stopArgs = append(stopArgs, "stop", "-t", fmt.Sprintf("%d", gracefulTimeout))
		stopArgs = append(stopArgs, args...)
		if err := compose.Run(ctx, workdir, stopArgs...); err != nil {
			return fmt.Errorf("failed to stop services: %w", err)
		}
		fmt.Printf("Stopped %d service(s).\n", len(args))
		return nil
	}

	destructive := volumes || rmi

	// If NOT removing volumes/images, do a graceful stop first.
	if !destructive {
		if verbose {
			fmt.Printf("Graceful stop (timeout: %ds)...\n", gracefulTimeout)
		}
		stopArgs := composeBaseArgs(composeFiles)
		stopArgs = append(stopArgs, "stop", "-t", fmt.Sprintf("%d", gracefulTimeout))
		if err := compose.Run(ctx, workdir, stopArgs...); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: graceful stop returned error: %v\n", err)
		}
	} else if verbose {
		fmt.Println("Skipping graceful stop (destructive flags set).")
	}

	// Build compose down options.
	downOpts := docker.DownOptions{
		RemoveVolumes: volumes,
		RemoveOrphans: removeOrphans,
		Timeout:       gracefulTimeout,
	}

	if verbose {
		fmt.Println("Running compose down...")
		if volumes {
			fmt.Println("  --volumes: removing volumes")
		}
		if rmi {
			fmt.Println("  --rmi: removing images")
		}
		if removeOrphans {
			fmt.Println("  --remove-orphans: cleaning orphans")
		}
	}

	// ComposeDown handles volumes, orphans, and timeout.
	// For --rmi we need to pass it separately since DownOptions doesn't have it.
	if rmi {
		downArgs := composeBaseArgs(composeFiles)
		downArgs = append(downArgs, "down")
		if volumes {
			downArgs = append(downArgs, "-v")
		}
		downArgs = append(downArgs, "--rmi", "all")
		if removeOrphans {
			downArgs = append(downArgs, "--remove-orphans")
		}
		if gracefulTimeout > 0 {
			downArgs = append(downArgs, "-t", fmt.Sprintf("%d", gracefulTimeout))
		}
		if err := compose.Run(ctx, workdir, downArgs...); err != nil {
			return fmt.Errorf("compose down failed: %w", err)
		}
	} else {
		if err := compose.ComposeDown(ctx, workdir, downOpts); err != nil {
			return fmt.Errorf("compose down failed: %w", err)
		}
	}

	// Summary.
	runningCount := len(containers)
	fmt.Printf("Stopped %d container(s).", runningCount)
	if volumes {
		fmt.Print(" Volumes removed.")
	}
	if rmi {
		fmt.Print(" Images removed.")
	}
	fmt.Println()

	return nil
}

// composeBaseArgs builds the docker compose base arguments including -f flags
// for multi-file support. This mirrors docker.Compose.buildBaseArgs but is
// needed for raw Run calls that bypass the typed Compose methods.
func composeBaseArgs(files []string) []string {
	args := []string{"compose"}
	for _, f := range files {
		args = append(args, "-f", f)
	}
	return args
}
