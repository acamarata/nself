package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"nself/internal/build"
	"nself/internal/config"
	"nself/internal/migration"
	"nself/internal/ui"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compose your infrastructure from .env",
	Long: `Generate docker-compose.yml, nginx configs, and SSL certificates
from your .env configuration.

The build pipeline:
  1. Load and validate .env cascade
  2. Generate SSL certificates (mkcert or self-signed)
  3. Generate nginx reverse-proxy configuration
  4. Generate docker-compose.yml with all enabled services`,
	RunE: runBuild,
}

func init() {
	buildCmd.Flags().BoolP("force", "f", false, "Force rebuild all components")
	buildCmd.Flags().Bool("no-cache", false, "Disable build cache")
	buildCmd.Flags().BoolP("verbose", "v", false, "Show environment cascade")
	buildCmd.Flags().Bool("debug", false, "Enable debug mode")
	buildCmd.Flags().Bool("security-report", false, "Generate security analysis")
	buildCmd.Flags().Bool("allow-insecure", false, "Allow insecure config (dev only)")
	buildCmd.Flags().Bool("check", false, "Validate only, don't build")
	buildCmd.Flags().BoolP("quiet", "q", false, "Suppress non-error output (for CI use)")
	buildCmd.Flags().Bool("no-monorepo", false, "Disable automatic monorepo backend detection")
	buildCmd.Flags().Bool("no-migration-check", false, "Skip v1 artifact detection (for automation/CI)")

	RootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	noCache, _ := cmd.Flags().GetBool("no-cache")
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")
	securityReport, _ := cmd.Flags().GetBool("security-report")
	allowInsecure, _ := cmd.Flags().GetBool("allow-insecure")
	check, _ := cmd.Flags().GetBool("check")
	quiet, _ := cmd.Flags().GetBool("quiet")
	noMonorepo, _ := cmd.Flags().GetBool("no-monorepo")
	noMigrationCheck, _ := cmd.Flags().GetBool("no-migration-check")

	if !quiet {
		ui.CommandHeader("nself build", "Generate project infrastructure")
	}

	if debug {
		_ = os.Setenv("DEBUG", "true")
	}

	if allowInsecure && !quiet {
		ui.Warn("Running with --allow-insecure: security checks relaxed")
	}

	cwd, err := os.Getwd()
	if err != nil {
		ui.Error("Failed to determine working directory")
		return fmt.Errorf("getting working directory: %w", err)
	}

	// ── Monorepo detection ────────────────────────────────────────────────
	// Check before FindNSelfRoot so that users running nself build from the
	// monorepo root (e.g. the nself-web Turborepo) are redirected into the
	// correct backend sub-directory automatically.
	if !noMonorepo {
		if backendRoot := config.DetectMonorepoRoot(cwd); backendRoot != "" {
			if !quiet {
				fmt.Printf("→ Detected monorepo layout. Using %s as project root.\n", filepath.Base(backendRoot))
			}
			cwd = backendRoot
		}
	}

	workdir, err := config.FindNSelfRoot(cwd)
	if err != nil {
		return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
	}

	if verbose && !quiet {
		ui.Info(fmt.Sprintf("Working directory: %s", workdir))
		ui.Info(fmt.Sprintf("Force: %t | No-cache: %t | Check: %t", force, noCache, check))
	}

	// ── v1 artifact detection ─────────────────────────────────────────────
	if !noMigrationCheck && !quiet {
		if artifacts := migration.Detect(workdir); len(artifacts) > 0 {
			ui.Warn("v1 artifacts detected. Run `nself migrate run` before building to ensure compatibility.")
			fmt.Println("Continuing with build...")
		}
	}

	opts := build.BuildOptions{
		Force:          force || noCache,
		Verbose:        verbose,
		Check:          check,
		SecurityReport: securityReport,
	}

	result, err := build.Build(workdir, opts)
	if err != nil {
		ui.Error(fmt.Sprintf("Build failed: %v", err))
		return err
	}

	if !quiet {
		if check {
			ui.Success(fmt.Sprintf("Configuration valid for project %q", result.ProjectName))
			return nil
		}

		// SSL CA trust status line.
		if result.CAInstalled {
			ui.Success("mkcert CA trusted")
		} else if result.CAManualCmd != "" {
			ui.Warn(fmt.Sprintf("Add CA manually: %s", result.CAManualCmd))
		}

		// /etc/hosts status line.
		if result.HostsAdded > 0 {
			ui.Success(fmt.Sprintf("Added %d domain(s) to /etc/hosts", result.HostsAdded))
		} else if result.HostsManualNote != "" {
			ui.Warn(fmt.Sprintf("Could not update /etc/hosts automatically.\n%s", result.HostsManualNote))
		}

		items := []string{
			fmt.Sprintf("Project: %s", result.ProjectName),
			fmt.Sprintf("Compose: %s", result.ComposeFile),
			fmt.Sprintf("Nginx:   %s", result.NginxConfig),
			fmt.Sprintf("SSL:     %d certificate(s)", result.SSLCerts),
			fmt.Sprintf("Files:   %d generated", result.FilesGenerated),
			fmt.Sprintf("Time:    %s", result.Duration.Round(1e6)),
		}
		ui.SummaryBox("Build Complete", items)

		ui.Info("Next step: nself start")
	}

	return nil
}
