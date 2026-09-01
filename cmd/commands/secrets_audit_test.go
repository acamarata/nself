package commands

// secrets_audit_test.go — Unit tests for the `nself secrets` audit-facing
// subcommands in secrets_audit.go: audit, lint, rotation-log, rekey.
// P6-E11-W2-S3-T18: security command test floor (was 0% direct coverage of
// this file's cobra wrappers — the underlying internal/secrets functions
// already had unit tests, but nothing proved the CLI wiring reaches them
// correctly end to end).
//
// Security properties under test:
//   - lint: a plaintext-looking secret committed to a git-tracked file is
//     actually flagged (not silently missed) — this is the command a repo
//     relies on to catch a leaked credential before it merges.
//   - audit: a secret store entry with no creation/rotation timestamp is
//     flagged "high" severity (an untracked secret cannot be proven to
//     comply with the 90-day rotation policy) — AND a freshly-set secret is
//     NOT flagged, so a validator that fires unconditionally would still
//     be caught.
// Inputs: temp git repos / temp project roots.
// Outputs: AuditFinding/LintFinding content, or command errors.
// Constraints: lint needs no `age` binary (plaintext file scan). audit's
// findings-positive case needs a real encrypted store, so it builds one
// directly with the `age`/`age-keygen` binaries and skips if unavailable
// (matching internal/secrets' own documented CI limitation).

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/secrets"
)

// requireAge skips the test if the age/age-keygen binaries are not on PATH.
// Mirrors internal/secrets/secrets_coverage_test.go's documented limitation:
// these paths are genuinely untestable without the binaries.
func requireAge(t *testing.T) {
	t.Helper()
	if err := secrets.EnsureAgeInstalled(); err != nil {
		t.Skip("age/age-keygen not installed — skipping (see coverage.yml, which installs them in CI)")
	}
}

// requireGitleaks skips the test if the gitleaks binary is not on PATH
// (LintSecrets shells out to it; coverage.yml installs it for CI).
func requireGitleaks(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gitleaks"); err != nil {
		t.Skip("gitleaks not installed — skipping (see coverage.yml, which installs it in CI)")
	}
}

// TestSecretsLint_FlagsPlaintextSecretInTrackedFile verifies the real
// security property: `nself secrets lint` must flag a plaintext credential
// committed to a git-tracked file. If this regresses to a no-op, secrets
// leak into git history undetected.
func TestSecretsLint_FlagsPlaintextSecretInTrackedFile(t *testing.T) {
	requireGitleaks(t)
	withProjectRoot(t, func(root string) {
		runGit(t, root, "init", "-q")
		runGit(t, root, "config", "user.email", "test@example.com")
		runGit(t, root, "config", "user.name", "test")

		leaky := filepath.Join(root, "config.yaml")
		// A well-formed GitHub PAT shape (gitleaks' "github-pat" built-in
		// rule): high-entropy, fixed prefix, reliably flagged regardless of
		// gitleaks version-specific rule tuning for other providers.
		const secretLine = `github_token: "ghp_a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6q7R8"`
		if err := os.WriteFile(leaky, []byte("app:\n  name: demo\n"+secretLine+"\n"), 0o644); err != nil {
			t.Fatalf("write leaky file: %v", err)
		}
		runGit(t, root, "add", "config.yaml")
		runGit(t, root, "commit", "-q", "-m", "seed")

		findings, err := secrets.LintSecrets(root)
		if err != nil {
			t.Fatalf("LintSecrets: %v", err)
		}
		if len(findings) == 0 {
			t.Fatal("expected a lint finding for a tracked file containing a GitHub PAT, got none")
		}
		for _, f := range findings {
			if f.Rule == "parse-error" {
				t.Fatalf("gitleaks report failed to parse (regression in the report-path plumbing): %+v", f)
			}
		}

		// The command wrapper must surface the same findings, not swallow them.
		if err := secretsLintCmd.RunE(secretsLintCmd, nil); err != nil {
			t.Errorf("secretsLintCmd.RunE: unexpected error: %v", err)
		}
	})
}

// TestSecretsLint_CleanFile_NoFindings is the negative case: a file with no
// secret-shaped content must NOT be flagged. Without this, a lint rule that
// matches everything would still pass the positive test above.
func TestSecretsLint_CleanFile_NoFindings(t *testing.T) {
	requireGitleaks(t)
	withProjectRoot(t, func(root string) {
		runGit(t, root, "init", "-q")
		runGit(t, root, "config", "user.email", "test@example.com")
		runGit(t, root, "config", "user.name", "test")

		clean := filepath.Join(root, "README.md")
		if err := os.WriteFile(clean, []byte("# Demo\n\nJust documentation, no secrets here.\n"), 0o644); err != nil {
			t.Fatalf("write clean file: %v", err)
		}
		runGit(t, root, "add", "README.md")
		runGit(t, root, "commit", "-q", "-m", "seed")

		findings, err := secrets.LintSecrets(root)
		if err != nil {
			t.Fatalf("LintSecrets: %v", err)
		}
		if len(findings) != 0 {
			t.Errorf("expected 0 findings for a clean file, got %d: %+v", len(findings), findings)
		}
	})
}

// runGit runs a git command in dir, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// buildAgeStore hand-encrypts a SecretStore JSON blob directly with the age
// binaries (bypassing internal/secrets' unexported loadStore/saveStore, which
// this package cannot call) so tests can construct store states — like an
// entry with no timestamps — that the public Set/Rotate API cannot produce.
func buildAgeStore(t *testing.T, root, env string, entries map[string]secrets.SecretEntry) {
	t.Helper()
	keyPath := filepath.Join(root, "age-key.txt")
	if out, err := exec.Command("age-keygen", "-o", keyPath).CombinedOutput(); err != nil {
		t.Fatalf("age-keygen: %v\n%s", err, out)
	}
	t.Setenv("SECRETS_AGE_KEY_PATH", keyPath)

	pubKey, err := secrets.GetPublicKey(keyPath)
	if err != nil {
		t.Fatalf("GetPublicKey: %v", err)
	}

	store := secrets.SecretStore{Secrets: entries, Recipients: []string{pubKey}}
	data, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("marshal store: %v", err)
	}

	secretsDirPath := filepath.Join(root, secrets.SecretsDir)
	if err := os.MkdirAll(secretsDirPath, 0o700); err != nil {
		t.Fatalf("mkdir .secrets: %v", err)
	}
	outPath := filepath.Join(secretsDirPath, env+".age")

	encCmd := exec.Command("age", "--encrypt", "-r", pubKey, "-o", outPath)
	encCmd.Stdin = strings.NewReader(string(data))
	if out, err := encCmd.CombinedOutput(); err != nil {
		t.Fatalf("age --encrypt: %v\n%s", err, out)
	}
}

// TestSecretsAudit_NoTimestamp_FlaggedHighSeverity verifies the real
// security property: a secret entry with neither CreatedAt nor RotatedAt
// set is flagged "high" severity — such an entry cannot be proven to comply
// with the 90-day rotation policy, so it must never audit clean silently.
func TestSecretsAudit_NoTimestamp_FlaggedHighSeverity(t *testing.T) {
	requireAge(t)
	withProjectRoot(t, func(root string) {
		buildAgeStore(t, root, "dev", map[string]secrets.SecretEntry{
			"UNTRACKED_SECRET": {Value: "whatever"}, // CreatedAt/RotatedAt both empty
		})
		secretsEnvFlag = "dev"
		defer func() { secretsEnvFlag = "dev" }()

		findings, err := secrets.Audit(root, "dev")
		if err != nil {
			t.Fatalf("Audit: %v", err)
		}
		found := false
		for _, f := range findings {
			if f.Key == "UNTRACKED_SECRET" {
				found = true
				if f.Severity != "high" {
					t.Errorf("severity = %q, want %q", f.Severity, "high")
				}
			}
		}
		if !found {
			t.Fatal("expected UNTRACKED_SECRET to be flagged for missing timestamps, got no finding")
		}

		if err := secretsAuditCmd.RunE(secretsAuditCmd, nil); err != nil {
			t.Errorf("secretsAuditCmd.RunE: unexpected error: %v", err)
		}
	})
}

// TestSecretsAudit_FreshlyTimestamped_NotFlagged is the negative case: a
// secret with a just-now RotatedAt must NOT appear in findings. Without
// this, an Audit that flags every entry unconditionally would still pass
// the positive test above.
func TestSecretsAudit_FreshlyTimestamped_NotFlagged(t *testing.T) {
	requireAge(t)
	withProjectRoot(t, func(root string) {
		now := time.Now().UTC().Format(time.RFC3339)
		buildAgeStore(t, root, "dev", map[string]secrets.SecretEntry{
			"FRESH_SECRET": {Value: "v", CreatedAt: now, RotatedAt: now},
		})

		findings, err := secrets.Audit(root, "dev")
		if err != nil {
			t.Fatalf("Audit: %v", err)
		}
		for _, f := range findings {
			if f.Key == "FRESH_SECRET" {
				t.Errorf("FRESH_SECRET unexpectedly flagged: %+v", f)
			}
		}
	})
}

// Rekey and decrypt-on-deploy command tests live in secrets_rekey_test.go
// (split out to keep this file under the engineering standard's 300-line
// cap; same package, shares requireAge/buildAgeStore above).
