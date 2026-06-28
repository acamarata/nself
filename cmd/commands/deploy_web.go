// Package commands implements CLI commands for the nSelf binary.
// This file adds 'nself deploy web' which builds Vercel/web apps locally and
// deploys the prebuilt output — eliminating Vercel Build CPU charges (~$43/mo).
//
// Why prebuilt: 'vercel deploy --prebuilt' skips remote build compute entirely;
// Vercel just serves the already-built .vercel/output directory. Local or
// ops-box builds are free; Vercel Build CPU is not.
//
// Usage:
//
//	nself deploy web [app…]          # build+deploy all apps (or listed subset)
//	nself deploy web --prod          # promote to production
//	nself deploy web nchat org       # subset by name
//	nself deploy web --dry-run       # print commands without running
//	nself deploy web --token=<tok>   # explicit Vercel token (else VERCEL_TOKEN)
package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

// webApps is the canonical list of deployable web apps in the nSelf Turborepo.
// Source of truth: web/pnpm-workspace.yaml (apps/* + named roots).
// Update this list when apps are added or removed from the monorepo.
var webApps = []string{
	"org",
	"docs",
	"nchat",
	"nclaw",
	"ntv",
	"clawde",
	"nfamily",
	"ntask",
	"ntask-marketing",
	"cloud",
	"base",
	"install",
	"status",
	"nsentry",
}

// deployWebCmd is the 'nself deploy web' subcommand.
var deployWebCmd = &cobra.Command{
	Use:   "web [app...]",
	Short: "Build web apps locally and deploy prebuilt output to Vercel",
	Long: `Build one or more web apps locally (vercel build) then deploy the
prebuilt output (vercel deploy --prebuilt), eliminating Vercel Build CPU charges.

Running 'vercel build' locally produces a .vercel/output directory.
'vercel deploy --prebuilt' uploads that directory directly — Vercel does
no build compute, so you pay only for hosting, not CPU minutes.

Apps are deployed sequentially. Pass app names to target a subset.

Examples:
  nself deploy web                # build+deploy all 14 apps
  nself deploy web org docs       # subset: org + docs only
  nself deploy web --prod         # deploy to production
  nself deploy web --dry-run      # print commands, do nothing
  nself deploy web nchat --prod   # prod deploy for nchat only

Environment:
  VERCEL_TOKEN    Vercel API token (required; also accepted via --token)
  VERCEL_ORG_ID   Vercel org/team ID (optional; read from .vercel/project.json)
  VERCEL_PROJECT_ID  Vercel project ID (optional; read from .vercel/project.json)`,
	Args: cobra.ArbitraryArgs,
	RunE: runDeployWeb,
}

func init() {
	f := deployWebCmd.Flags()
	f.Bool("prod", false, "Promote deploy to production (adds --prod to 'vercel deploy')")
	f.Bool("dry-run", false, "Print the vercel commands without executing")
	f.String("token", "", "Vercel API token (overrides VERCEL_TOKEN env var)")
	f.String("web-dir", "", "Path to the web Turborepo root (default: auto-detected sibling '../web')")

	deployCmd.AddCommand(deployWebCmd)
}

// runDeployWeb is the RunE handler for 'nself deploy web'.
func runDeployWeb(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// ── flags ────────────────────────────────────────────────────────────────
	prod, _ := cmd.Flags().GetBool("prod")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	token, _ := cmd.Flags().GetString("token")
	webDirFlag, _ := cmd.Flags().GetString("web-dir")

	// ── vercel CLI check ─────────────────────────────────────────────────────
	if _, err := exec.LookPath("vercel"); err != nil {
		ui.UXError(
			"'vercel' CLI not found in PATH",
			"nself deploy web requires the Vercel CLI to build and deploy.",
			[]string{
				"Install globally: pnpm add -g vercel",
				"Or: npm install -g vercel",
				"Then re-run: nself deploy web",
			},
		)
		return fmt.Errorf("vercel CLI not found: %w", err)
	}

	// ── token resolution ─────────────────────────────────────────────────────
	if token == "" {
		token = os.Getenv("VERCEL_TOKEN")
	}
	if token == "" {
		ui.UXError(
			"Vercel token not set",
			"A Vercel API token is required to deploy.",
			[]string{
				"Set VERCEL_TOKEN in your environment or vault",
				"Or pass: nself deploy web --token=<token>",
			},
		)
		return fmt.Errorf("VERCEL_TOKEN is empty and --token was not provided")
	}

	// ── web root resolution ───────────────────────────────────────────────────
	webRoot, err := resolveWebRoot(webDirFlag)
	if err != nil {
		return err
	}

	// ── target app list ───────────────────────────────────────────────────────
	targets, err := resolveWebApps(args, webRoot)
	if err != nil {
		return err
	}

	// ── header ────────────────────────────────────────────────────────────────
	env := "preview"
	if prod {
		env = "production"
	}
	ui.CommandHeader(
		"Deploy Web (prebuilt)",
		fmt.Sprintf("%d apps → Vercel %s | local build, zero remote CPU", len(targets), env),
	)
	if dryRun {
		ui.Warn("dry-run: commands will be printed but not executed")
	}

	// ── per-app deploy loop ───────────────────────────────────────────────────
	failed := []string{}
	for i, app := range targets {
		ui.Step(i+1, len(targets), app)
		if err := deployWebApp(ctx, deployWebAppParams{
			app:     app,
			webRoot: webRoot,
			token:   token,
			prod:    prod,
			dryRun:  dryRun,
		}); err != nil {
			ui.Error(fmt.Sprintf("%s: %v", app, err))
			failed = append(failed, app)
			// continue to next app instead of hard-abort
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("deploy failed for apps: %s", strings.Join(failed, ", "))
	}
	if !dryRun {
		ui.Success(fmt.Sprintf("All %d apps deployed to Vercel (%s) — no remote build compute used", len(targets), env))
	}
	return nil
}

// deployWebAppParams groups parameters for a single-app prebuilt deploy.
type deployWebAppParams struct {
	app     string
	webRoot string
	token   string
	prod    bool
	dryRun  bool
}

// deployWebApp runs the three-step prebuilt pipeline for one app:
//
//  1. vercel pull   — sync .vercel/project.json + env files from Vercel
//  2. vercel build  — local build → .vercel/output (no remote compute)
//  3. vercel deploy --prebuilt [--prod] — upload .vercel/output only
func deployWebApp(ctx context.Context, p deployWebAppParams) error {
	appDir, err := resolveAppDir(p.webRoot, p.app)
	if err != nil {
		return err
	}

	steps := []struct {
		label string
		args  []string
	}{
		{
			"pull env",
			[]string{"vercel", "pull", "--yes", "--token=" + p.token},
		},
		{
			"build (local)",
			[]string{"vercel", "build"},
		},
		{
			"deploy prebuilt",
			buildDeployArgs(p.token, p.prod),
		},
	}

	for _, s := range steps {
		if p.dryRun {
			ui.Dimmed(fmt.Sprintf("  [dry-run] cd %s && %s", appDir, strings.Join(s.args, " ")))
			continue
		}
		ui.Info(fmt.Sprintf("  %s: %s", s.label, strings.Join(s.args, " ")))
		if err := runCmd(ctx, appDir, s.args[0], s.args[1:]...); err != nil {
			return fmt.Errorf("step %q: %w", s.label, err)
		}
	}
	if !p.dryRun {
		ui.Success(fmt.Sprintf("  %s deployed (prebuilt)", p.app))
	}
	return nil
}

// buildDeployArgs constructs the 'vercel deploy --prebuilt' arguments.
// --prebuilt is always set — this is the core of the cost-elimination strategy.
func buildDeployArgs(token string, prod bool) []string {
	args := []string{"vercel", "deploy", "--prebuilt", "--token=" + token}
	if prod {
		args = append(args, "--prod")
	}
	return args
}

// runCmd executes a command in the given directory, streaming stdout/stderr.
func runCmd(ctx context.Context, dir, name string, args ...string) error {
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = os.Environ()
	return c.Run()
}

// resolveWebRoot finds the web Turborepo root.
// Priority: --web-dir flag → sibling ../web of the nSelf project root → cwd/web.
func resolveWebRoot(flagVal string) (string, error) {
	if flagVal != "" {
		abs, err := filepath.Abs(flagVal)
		if err != nil {
			return "", fmt.Errorf("resolving --web-dir: %w", err)
		}
		if err := expectDir(abs); err != nil {
			return "", fmt.Errorf("--web-dir %q: %w", flagVal, err)
		}
		return abs, nil
	}

	// Try sibling directory of the nSelf project root.
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	candidates := []string{
		filepath.Join(cwd, "..", "web"),
		filepath.Join(cwd, "web"),
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if err := expectDir(abs); err == nil {
			// Confirm it looks like the nSelf web monorepo.
			if _, err := os.Stat(filepath.Join(abs, "pnpm-workspace.yaml")); err == nil {
				return abs, nil
			}
		}
	}
	return "", fmt.Errorf("web Turborepo root not found; pass --web-dir=<path>")
}

// resolveWebApps returns the list of app names to deploy.
// If args is non-empty, it validates each against the known list.
// If args is empty, all webApps are returned.
func resolveWebApps(args []string, webRoot string) ([]string, error) {
	if len(args) == 0 {
		return webApps, nil
	}

	known := make(map[string]bool, len(webApps))
	for _, a := range webApps {
		known[a] = true
	}

	out := make([]string, 0, len(args))
	for _, a := range args {
		if !known[a] {
			return nil, fmt.Errorf("unknown app %q; known apps: %s", a, strings.Join(webApps, ", "))
		}
		out = append(out, a)
	}
	return out, nil
}

// resolveAppDir returns the absolute directory for an app within the web root.
// Apps may live at <webRoot>/<app> or <webRoot>/apps/<app>.
func resolveAppDir(webRoot, app string) (string, error) {
	candidates := []string{
		filepath.Join(webRoot, app),
		filepath.Join(webRoot, "apps", app),
	}
	for _, c := range candidates {
		if err := expectDir(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("app directory not found for %q (tried %s and %s)",
		app, candidates[0], candidates[1])
}

// expectDir returns an error if path does not exist or is not a directory.
func expectDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}
