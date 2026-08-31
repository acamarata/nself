package commands

// start_checks.go — pre-flight gate checks for `nself start`: the second
// (post-FindNSelfRoot) v0.9 legacy-project gate, opt-in AI auto-install, the
// auto-rebuild-if-stale check, and the docker-compose.yml existence check.
// Split out of start.go (T-P6-E2-W1-S1-T3) for 300-line compliance.
// Inputs:  the relevant runStart locals (ctx, projectDir, allowLegacy, opts).
// Outputs: error where the original inline code returned one; the AI
//          auto-install check is a pure side effect with no return value.
// Constraints: pure move, same checks/output/errors/order, no behavior change.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/build"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/migration"
	"github.com/nself-org/cli/internal/ui"
)

// checkLegacyProjectGate re-checks projectDir for v0.9 legacy artifacts
// (S60-T02). Requires >=2 of 5 heuristics to trigger (prevents false
// positives); a single artifact warns but proceeds. >=2 artifacts fail
// unless allowLegacy is set.
func checkLegacyProjectGate(projectDir string, allowLegacy bool) error {
	if count, names := migration.CheckLegacyProject(projectDir); count >= migration.DetectionThreshold {
		if allowLegacy {
			ui.Warn(fmt.Sprintf("WARNING: v0.9 project detected (%d artifact(s): %s). Proceeding due to --allow-legacy (not recommended).", count, strings.Join(names, ", ")))
		} else {
			ui.Error(fmt.Sprintf("v0.9 project detected. Found %d legacy artifact(s): %s", count, strings.Join(names, ", ")))
			fmt.Fprintln(os.Stderr, "Run `nself migrate` first. See https://nself.org/docs/migrate/from-v0.9")
			return fmt.Errorf("v0.9 project detected — run `nself migrate` first")
		}
	} else if count == 1 {
		ui.Warn(fmt.Sprintf("One possible v0.9 artifact found (%s). Proceeding — run `nself migrate` if this is a v0.9 project.", names[0]))
	}
	return nil
}

// autoInstallAIIfNeeded runs `doctor --ai --yes --skip-pool` to get Ollama
// running before the stack boots, when AI_AUTO_INSTALL is enabled (default)
// and NSELF_MASTER_SECRET is present (T-05-05).
//
// CLI-R18: the AI config block (and NSELF_MASTER_SECRET) used to live in a
// dedicated .env.ai file, and its mere existence gated this block. .env.ai
// is now folded into .env.secrets, so the gate is NSELF_MASTER_SECRET
// presence in the resolved environment — the same signal ("zero-config AI
// was set up for this project"), just read from the cascade instead of the
// filesystem.
func autoInstallAIIfNeeded(ctx context.Context) {
	if aiAutoInstall := os.Getenv("AI_AUTO_INSTALL"); aiAutoInstall == "" || strings.EqualFold(aiAutoInstall, "true") {
		if os.Getenv("NSELF_MASTER_SECRET") != "" {
			if !ollamaHealthy(ctx) {
				ui.Info("AI_AUTO_INSTALL: setting up local AI...")
				_ = runDoctorAI(ctx, doctorAIFlags{
					yes:        true,
					skipPool:   true,
					skipOllama: false,
					headless:   false,
					jsonOut:    false,
				})
			}
		}
	}
}

// autoRebuildIfNeeded runs an automatic `nself build` before the
// docker-compose.yml check when the resolved config has drifted from the
// generated compose file. Must run before checkComposeFileExists because
// build creates the file.
func autoRebuildIfNeeded(projectDir string, opts startOpts) error {
	needsRebuild, err := build.NeedsRebuild(projectDir)
	if err != nil {
		return fmt.Errorf("checking build state: %w", err)
	}
	if needsRebuild {
		ui.Info("Configuration changed — rebuilding before start...")
		if _, err := build.Build(projectDir, build.BuildOptions{Profile: opts.profile}); err != nil {
			return fmt.Errorf("auto-build failed: %w", err)
		}
		ui.Success("Build completed")
	}
	return nil
}

// checkComposeFileExists validates that docker-compose.yml exists in
// projectDir (Step 1 of the start sequence).
func checkComposeFileExists(projectDir string) error {
	composePath := filepath.Join(projectDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		ui.UXError(
			"docker-compose.yml not found",
			fmt.Sprintf("Looked in %s", projectDir),
			[]string{
				"Run 'nself build' to generate your compose configuration",
				"Make sure you are in the correct project directory",
			},
		)
		return fmt.Errorf("docker-compose.yml not found in %s: %w", projectDir, errs.ErrComposeNotFound)
	}
	return nil
}
