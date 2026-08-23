package commands

// Purpose: runDeployWeb, the RunE for "nself deploy web". Inputs are the
// cobra command/args (app names, --prod, --dry-run, --token); outputs are
// the built/deployed web apps or an error.
// Constraints: split out of deploy_web.go (CLI-R12) as a pure move, no behavior change.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

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
