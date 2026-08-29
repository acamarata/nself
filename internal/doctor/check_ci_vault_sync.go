package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ciVaultPATs lists the critical CI Personal Access Tokens that must be
// present in vault.env (or the live environment). These are all used by
// GitHub Actions workflows to access cross-repo resources and trigger
// repository dispatch events. A missing PAT causes silent CI failures.
//
//   - NSELF_CI_TOKEN: version-lockstep workflow — contents:read on admin + homebrew-nself
//   - HOMEBREW_TAP_DISPATCH_TOKEN: triggers homebrew-nself formula update on release
//   - ADMIN_REBUILD_TOKEN: triggers admin rebuild after CLI release
//   - GH_BILLING_PAT: used by admin-merge.sh to check Actions billing quota
var ciVaultPATs = []string{
	"NSELF_CI_TOKEN",
	"HOMEBREW_TAP_DISPATCH_TOKEN",
	"ADMIN_REBUILD_TOKEN",
	"GH_BILLING_PAT",
}

// ciVaultInfraTokens lists 180-day infrastructure tokens. Missing values
// generate a warn rather than fail: they are used by plugin services that
// can continue running with stale tokens short-term, but they will break
// on next deploy / key rotation if not present in vault.env.
var ciVaultInfraTokens = []string{
	"CLOUDFLARE_API_KEY",
	"POSTMARK_SERVER_API_TOKEN",
}

// isCIEnvironment reports whether we are running on a CI runner. It mirrors the
// detection used by cmd/commands/browser.go so the two cannot drift.
func isCIEnvironment() bool {
	return os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
}

// fileExists reports whether path exists and is a regular file. A directory at
// the vault path is treated as absent: it cannot be parsed for keys either way.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// vaultEnvPath returns the canonical path to vault.env on the current machine.
// The file lives at ~/.claude/vault.env per GCI Global Credentials Vault.
func vaultEnvPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: try the HOME env var. envFileHasKey will fail gracefully
		// if neither path exists.
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".claude", "vault.env")
}

// CheckCIVaultSync is the CI-VAULT-SYNC-01 deep doctor check.
//
// It verifies that vault.env or the live environment contains all critical
// CI PATs (fail on missing) and 180-day infra tokens (warn on missing).
//
// The check does NOT validate token values — only presence. A token could
// be present but expired; CI-TOKEN-01 handles rotation-age tracking for
// NSELF_CI_TOKEN specifically. CI-VAULT-SYNC-01 covers the broader vault
// coverage question.
func CheckCIVaultSync(projectDir string) CheckResult {
	const name = "CI-VAULT-SYNC-01"
	section := "security"

	vault := vaultEnvPath()

	// vault.env is a developer-workstation artifact (~/.claude/vault.env). A CI
	// runner has no such file by design: workflows receive these PATs as
	// per-workflow GitHub Actions secrets, which are not in the ambient
	// environment of an unrelated job. Running the check there reported CRITICAL
	// on every single run — a gate that could never pass, which is worse than no
	// gate because it trains people to ignore a security failure.
	//
	// Deliberately narrow: only skip when the vault file is genuinely absent. A
	// self-hosted runner that does have one is still checked, so this cannot
	// become a gate that never fails.
	if isCIEnvironment() && !fileExists(vault) {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "skip",
			Message: "CI-VAULT-SYNC-01: skipped — no vault.env on this machine and CI detected. " +
				"Vault sync is a developer-workstation concern; CI receives these PATs as " +
				"GitHub Actions secrets. Run `nself doctor --deep` locally to verify vault coverage.",
		}
	}

	// Collect missing critical PATs.
	var missingPATs []string
	for _, key := range ciVaultPATs {
		if !vaultHasKey(vault, projectDir, key) {
			missingPATs = append(missingPATs, key)
		}
	}

	if len(missingPATs) > 0 {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "fail",
			Message: fmt.Sprintf("CI-VAULT-SYNC-01: critical CI PAT(s) not found in vault.env or environment: %s. "+
				"These tokens are required by GitHub Actions workflows. "+
				"Add them to ~/.claude/vault.env and to the relevant GitHub Actions secrets "+
				"(gh secret set <NAME> --repo nself-org/<repo>).",
				strings.Join(missingPATs, ", ")),
			FixCmd: "gh secret set <TOKEN_NAME> --repo nself-org/cli --body '<value>'",
		}
	}

	// Collect missing infra tokens (warn only).
	var missingInfra []string
	for _, key := range ciVaultInfraTokens {
		if !vaultHasKey(vault, projectDir, key) {
			missingInfra = append(missingInfra, key)
		}
	}

	if len(missingInfra) > 0 {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "warn",
			Message: fmt.Sprintf("CI-VAULT-SYNC-01: 180-day infra token(s) not found in vault.env: %s. "+
				"These are needed at next deploy / rotation cycle. "+
				"Add them to ~/.claude/vault.env before the next rotation window.",
				strings.Join(missingInfra, ", ")),
			FixCmd: "# Edit ~/.claude/vault.env and add the missing token(s)",
		}
	}

	total := len(ciVaultPATs) + len(ciVaultInfraTokens)
	return CheckResult{
		Section: section,
		Name:    name,
		Status:  "pass",
		Message: fmt.Sprintf("CI-VAULT-SYNC-01: all %d CI tokens present in vault.env or environment", total),
	}
}

// vaultHasKey checks for key in vault.env first, then the live environment,
// then the project-local env cascade (.env.secrets, .env.local, .env).
// Returns true as soon as any source has a non-empty value for the key.
func vaultHasKey(vaultPath, projectDir, key string) bool {
	// 1. vault.env (global credential store).
	if envFileHasKey(vaultPath, key) {
		return true
	}

	// 2. Live environment (e.g. CI runner already exports it).
	if os.Getenv(key) != "" {
		return true
	}

	// 3. Project-local env cascade secrets files (not .env.dev — secrets only).
	for _, name := range []string{".env.secrets", ".env.local", ".env"} {
		if envFileHasKey(filepath.Join(projectDir, name), key) {
			return true
		}
	}

	return false
}
