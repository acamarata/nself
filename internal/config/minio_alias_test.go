package config

// Purpose: Regression tests for the MINIO_ACCESS_KEY/MINIO_SECRET_KEY alias
//          (gap #2) and the MinIO-enabled intent inference (gap #8).
//
// Inputs:  Simulated .env cascade values via t.Setenv.
// Outputs: none (t.Error/t.Fatal on assertion failure).
//
// Constraints: Pure env var manipulation; no filesystem access, matching
//              loader_test.go's style.
//
// SPORT: cli/internal/config — gap #2/#8 fix (env-schema parity ticket).

import (
	"os"
	"testing"
)

// clearMinioEnv resets every MinIO-related env var so each test starts from
// a clean slate regardless of ambient environment state or test order.
//
// Uses os.Unsetenv (not t.Setenv(k, "")) deliberately: setting a var to the
// empty string still makes os.LookupEnv report it as present, which would
// incorrectly trip the "explicitly set" checks in parseEnvToConfig's MinIO
// intent inference (gap #8). A truly absent var — the real-world case for
// every key this helper resets — must make LookupEnv report ok=false.
func clearMinioEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"MINIO_ENABLED", "STORAGE_ENABLED",
		"MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD",
		"MINIO_ACCESS_KEY", "MINIO_SECRET_KEY",
		"S3_ACCESS_KEY", "S3_SECRET_KEY",
	} {
		orig, had := os.LookupEnv(k)
		_ = os.Unsetenv(k)
		if had {
			t.Cleanup(func(k, orig string) func() {
				return func() { _ = os.Setenv(k, orig) }
			}(k, orig))
		} else {
			t.Cleanup(func(k string) func() {
				return func() { _ = os.Unsetenv(k) }
			}(k))
		}
	}
}

// ── Gap #2: MINIO_ACCESS_KEY/MINIO_SECRET_KEY alias ─────────────────────────

// TestMinioAlias_AccessKeyMapsToRootUser verifies MINIO_ACCESS_KEY is used as
// MINIO_ROOT_USER when MINIO_ROOT_USER itself is not set.
func TestMinioAlias_AccessKeyMapsToRootUser(t *testing.T) {
	clearMinioEnv(t)
	t.Setenv("MINIO_ACCESS_KEY", "aliased-access-key")
	t.Setenv("MINIO_SECRET_KEY", "aliased-secret-key")

	cfg := parseEnvToConfig()
	if cfg.Minio.RootUser != "aliased-access-key" {
		t.Errorf("Minio.RootUser = %q, want %q (aliased from MINIO_ACCESS_KEY)", cfg.Minio.RootUser, "aliased-access-key")
	}
	if cfg.Minio.RootPassword != "aliased-secret-key" {
		t.Errorf("Minio.RootPassword = %q, want %q (aliased from MINIO_SECRET_KEY)", cfg.Minio.RootPassword, "aliased-secret-key")
	}
}

// TestMinioAlias_RootUserWinsOverAlias verifies MINIO_ROOT_USER/PASSWORD take
// priority when both the canonical vars and the alias vars are set — back
// compat with anyone already using ROOT_* directly.
func TestMinioAlias_RootUserWinsOverAlias(t *testing.T) {
	clearMinioEnv(t)
	t.Setenv("MINIO_ROOT_USER", "canonical-user")
	t.Setenv("MINIO_ROOT_PASSWORD", "canonical-pass")
	t.Setenv("MINIO_ACCESS_KEY", "alias-user")
	t.Setenv("MINIO_SECRET_KEY", "alias-pass")

	cfg := parseEnvToConfig()
	if cfg.Minio.RootUser != "canonical-user" {
		t.Errorf("Minio.RootUser = %q, want %q (MINIO_ROOT_USER must win)", cfg.Minio.RootUser, "canonical-user")
	}
	if cfg.Minio.RootPassword != "canonical-pass" {
		t.Errorf("Minio.RootPassword = %q, want %q (MINIO_ROOT_PASSWORD must win)", cfg.Minio.RootPassword, "canonical-pass")
	}
}

// TestMinioAlias_NoAliasSet_EmptyRootCredentials verifies that with nothing
// set, RootUser/RootPassword remain empty (ApplyDefaults fills defaults
// later) — the alias must not invent values from thin air.
func TestMinioAlias_NoAliasSet_EmptyRootCredentials(t *testing.T) {
	clearMinioEnv(t)

	cfg := parseEnvToConfig()
	if cfg.Minio.RootUser != "" {
		t.Errorf("Minio.RootUser = %q, want empty when nothing set", cfg.Minio.RootUser)
	}
	if cfg.Minio.RootPassword != "" {
		t.Errorf("Minio.RootPassword = %q, want empty when nothing set", cfg.Minio.RootPassword)
	}
}

// ── Gap #8: MinIO-enabled intent inference ──────────────────────────────────

// TestMinioIntent_AccessKeySetWithoutEnabledFlag_InfersEnabled is the primary
// gap #8 regression: an app (e.g. ntask/backend/.env.example) that declares a
// full MinIO credential surface (MINIO_ACCESS_KEY et al.) but never sets
// MINIO_ENABLED/STORAGE_ENABLED must still get the MinIO service generated.
func TestMinioIntent_AccessKeySetWithoutEnabledFlag_InfersEnabled(t *testing.T) {
	clearMinioEnv(t)
	t.Setenv("MINIO_ACCESS_KEY", "minioaccesskey")
	t.Setenv("MINIO_SECRET_KEY", "miniosecretkey")

	cfg := parseEnvToConfig()
	if !cfg.Minio.Enabled {
		t.Error("Minio.Enabled = false, want true when MINIO_ACCESS_KEY is explicitly set")
	}
}

// TestMinioIntent_S3KeysSetWithoutEnabledFlag_InfersEnabled verifies the same
// inference applies for the S3_ACCESS_KEY/S3_SECRET_KEY naming.
func TestMinioIntent_S3KeysSetWithoutEnabledFlag_InfersEnabled(t *testing.T) {
	clearMinioEnv(t)
	t.Setenv("S3_ACCESS_KEY", "s3key")
	t.Setenv("S3_SECRET_KEY", "s3secret")

	cfg := parseEnvToConfig()
	if !cfg.Minio.Enabled {
		t.Error("Minio.Enabled = false, want true when S3_ACCESS_KEY is explicitly set")
	}
}

// TestMinioIntent_NoCredentials_StaysDisabled verifies backward compatibility:
// a minimal .env with no MinIO vars at all must NOT enable MinIO by
// surprise — this is the default behavior every existing deployment
// (ummat, unity) relies on.
func TestMinioIntent_NoCredentials_StaysDisabled(t *testing.T) {
	clearMinioEnv(t)

	cfg := parseEnvToConfig()
	if cfg.Minio.Enabled {
		t.Error("Minio.Enabled = true, want false when no MinIO vars are set at all")
	}
}

// TestMinioIntent_ExplicitlyDisabled_NotOverridden verifies that a user who
// explicitly sets MINIO_ENABLED=false (while also setting credentials, e.g.
// for a future/staged rollout) is honored — the intent inference only fires
// when the ENABLED vars are completely absent, never overriding an explicit
// false.
func TestMinioIntent_ExplicitlyDisabled_NotOverridden(t *testing.T) {
	clearMinioEnv(t)
	t.Setenv("MINIO_ENABLED", "false")
	t.Setenv("MINIO_ACCESS_KEY", "minioaccesskey")
	t.Setenv("MINIO_SECRET_KEY", "miniosecretkey")

	cfg := parseEnvToConfig()
	if cfg.Minio.Enabled {
		t.Error("Minio.Enabled = true, want false — explicit MINIO_ENABLED=false must not be overridden by credential presence")
	}
}
