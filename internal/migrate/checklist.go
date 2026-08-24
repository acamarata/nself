package migrate

// Purpose: the printed migration checklist and automated migration path (RunAuto, backupBashEraArtifacts) for the v0.9.9 Bash-era migration, built on the detection helpers in from_bash.go.
// Inputs: a project directory and the BashDetectResult produced by DetectBashEra.
// Outputs: an ordered []MigrationStep for display, or an AutoMigrateResult after applying automatic steps.
// Constraints: split out of from_bash.go as a pure move (CLI-R12); no behavior change.

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MigrationStep is a step in the printed migration checklist.
type MigrationStep struct {
	// Number is the 1-based step index.
	Number int
	// Title is a short description of the step.
	Title string
	// Commands are the exact shell commands to run, ready to copy-paste.
	Commands []string
	// Notes is optional additional context.
	Notes string
	// P98Scope marks steps handled by the automated migration path.
	P98Scope bool
}

// MigrationChecklist returns the full ordered list of steps to migrate a
// v0.9.9 Bash-era project to the current nSelf CLI.
//
// Steps are always returned in full, regardless of which artifacts were
// detected. The caller should present them to the operator even when some
// steps are already complete — idempotency is noted per step.
func MigrationChecklist(projectDir string) []MigrationStep {
	return []MigrationStep{
		{
			Number: 1,
			Title:  "Install the current nSelf CLI",
			Commands: []string{
				"brew tap nself-org/nself",
				"brew install nself",
				"nself --version  # verify",
			},
			Notes: "If you installed via Homebrew previously, run: brew upgrade nself",
		},
		{
			Number: 2,
			Title:  "Stop running containers",
			Commands: []string{
				"docker compose down  # or: docker-compose down",
			},
			Notes: "The v0.9.9 compose file may not be recognized by nself stop. Use docker compose directly.",
		},
		{
			Number: 3,
			Title:  "Back up your v0.9.9 config",
			Commands: []string{
				fmt.Sprintf("cp -r %s %s.bak.$(date +%%Y%%m%%d)", projectDir, projectDir),
				"# Or at minimum:",
				"cp .nself/config.sh .nself/config.sh.bak",
				"cp docker-compose.yml docker-compose.yml.bak",
			},
			Notes: "Keep the backup until you have verified the migrated stack is healthy.",
		},
		{
			Number: 4,
			Title:  "Create a .env.dev file from your Bash-era config",
			Commands: []string{
				"# Extract key values from .nself/config.sh and write them to .env.dev.",
				"# Example (adjust variable names to match your config.sh):",
				"grep -E '^(PROJECT_NAME|BASE_DOMAIN|POSTGRES_|HASURA_|AUTH_)' .nself/config.sh \\",
				"  | sed 's/export //' >> .env.dev",
			},
			Notes: "The Go CLI uses .env.dev → .env.local → .env.prod → .env.secrets → .env.computed. " +
				"See: https://github.com/nself-org/cli/wiki/Env-Cascade",
		},
		{
			Number: 5,
			Title:  "Initialize the project with the current CLI",
			Commands: []string{
				"nself init   # creates required directories and .env scaffold if missing",
			},
			Notes: "nself init is idempotent — safe to run even if .env.dev already exists.",
		},
		{
			Number: 6,
			Title:  "Rebuild with the current CLI",
			Commands: []string{
				"nself build",
			},
			Notes: "This regenerates docker-compose.yml, nginx configs, and SSL certificates. " +
				"Your old docker-compose.yml.bak is still available if you need to compare.",
		},
		{
			Number: 7,
			Title:  "Start the stack",
			Commands: []string{
				"nself start",
			},
		},
		{
			Number: 8,
			Title:  "Verify the stack is healthy",
			Commands: []string{
				"nself status",
				"nself doctor",
			},
			Notes: "nself doctor checks SSL, ports, Hasura connectivity, and plugin states.",
		},
		{
			Number: 9,
			Title:  "Remove v0.9.9 artifacts (once healthy)",
			Commands: []string{
				"rm -f nself.sh",
				"rm -f .nself/config.sh",
				"# Keep docker-compose.yml.bak and .nself/config.sh.bak for now",
			},
		},
		{
			Number:   10,
			Title:    "Automated config conversion (P98 scope)",
			Commands: []string{"# Coming in a future release: nself migrate from-bash --auto --convert-config"},
			Notes:    "Automated parsing of .nself/config.sh and direct conversion to .env.dev is planned for P98.",
			P98Scope: true,
		},
	}
}

// AutoMigrateResult summarises what RunAuto accomplished.
type AutoMigrateResult struct {
	// BackupDir is the path where the backup was written, or empty if skipped.
	BackupDir string
	// ActionsPerformed lists the steps that were executed.
	ActionsPerformed []string
	// Skipped lists steps that were skipped (already done or not auto-migratable).
	Skipped []string
}

// RunAuto performs the automatable migration steps for a v0.9.9 Bash-era
// project. It only acts on artifacts marked AutoMigratable=true in the
// detection result.
//
// RunAuto is idempotent: re-running on an already-migrated project detects
// no artifacts and returns immediately with nothing done.
//
// The caller is responsible for prompting the operator for confirmation before
// calling RunAuto. RunAuto itself does not prompt.
func RunAuto(projectDir string, result BashDetectResult) (*AutoMigrateResult, error) {
	out := &AutoMigrateResult{}

	if result.AlreadyMigrated {
		out.Skipped = append(out.Skipped, "no v0.9.9 artifacts found — already migrated")
		return out, nil
	}

	// Step 1: create a timestamped backup of automatable artifacts.
	backupDir, err := backupBashEraArtifacts(projectDir, result.Artifacts)
	if err != nil {
		return nil, fmt.Errorf("backing up v0.9.9 artifacts: %w", err)
	}
	out.BackupDir = backupDir
	out.ActionsPerformed = append(out.ActionsPerformed, fmt.Sprintf("created backup at %s", backupDir))

	// Step 2: remove nself.sh (auto-migratable — replaced by the CLI binary).
	nselfSh := filepath.Join(projectDir, "nself.sh")
	if _, statErr := os.Stat(nselfSh); statErr == nil {
		if err := os.Remove(nselfSh); err != nil {
			return nil, fmt.Errorf("removing nself.sh: %w", err)
		}
		out.ActionsPerformed = append(out.ActionsPerformed, "removed nself.sh (backed up)")
	}

	// Non-auto-migratable steps are recorded as skipped with guidance.
	out.Skipped = append(out.Skipped,
		"docker-compose.yml: review and run 'nself build' to regenerate",
		".envrc: review manually, copy vars to .env.dev, then remove",
		"config.sh vars: copy to .env.dev manually (see checklist step 4)",
	)

	return out, nil
}

// backupBashEraArtifacts copies auto-migratable artifacts to a timestamped
// backup directory inside .nself/bash-migration-backup/<timestamp>/.
// Returns the backup directory path.
func backupBashEraArtifacts(projectDir string, artifacts []BashEraArtifact) (string, error) {
	ts := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(projectDir, ".nself", "bash-migration-backup", ts)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}

	for _, a := range artifacts {
		if !a.AutoMigratable {
			continue
		}
		src := filepath.Join(projectDir, a.Path)
		data, err := os.ReadFile(src)
		if err != nil {
			// File may not exist (e.g. already removed). Skip silently.
			continue
		}
		// Preserve the relative path structure inside the backup directory.
		dest := filepath.Join(backupDir, a.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return "", fmt.Errorf("creating backup subdir for %s: %w", a.Path, err)
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return "", fmt.Errorf("writing backup for %s: %w", a.Path, err)
		}
	}

	return backupDir, nil
}
