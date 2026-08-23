package commands

// Purpose: helpers for "nself deploy web": the per-app deploy params/routine,
// arg building, subprocess running, and web-root/app-dir resolution. Inputs
// are a context, deployWebAppParams, or path/app-name strings; outputs are a
// deployed app, resolved paths, or an error.
// Constraints: split out of deploy_web.go (CLI-R12) as a pure move, no behavior change.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/ui"
)

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
