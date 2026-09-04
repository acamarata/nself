package hasura

// Purpose: unit tests for ApplyIfPresent's skip/reachable/strict/warn
// decision tree — the actual apply, inconsistency-check, and reachability
// calls are faked (applyFn/inconsistentFn/reachableFn) so this exercises the
// logic without a live Hasura.
// Constraints: no network, no docker required.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// alwaysReachable is the reachableFn every test below uses except the two
// that specifically exercise the reachability gate itself.
func alwaysReachable(context.Context, *config.Config) bool { return true }

func TestApplyIfPresent_SkipsCleanlyWhenNoMetadataDir(t *testing.T) {
	dir := t.TempDir() // no hasura/ subdir at all
	cfg := &config.Config{}
	called := false
	err := applyIfPresent(t.Context(), cfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { called = true; return nil },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err != nil {
		t.Fatalf("expected nil error on clean skip, got %v", err)
	}
	if called {
		t.Error("apply function must not be called when hasura/metadata is absent")
	}
}

func TestApplyIfPresent_SkipsCleanlyWhenHasuraUnreachable(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	// Strict env (prod) too: unreachable Hasura is "not part of this stack",
	// never a strict-mode failure.
	cfg := &config.Config{Env: "prod"}
	applyCalled := false
	err := applyIfPresent(t.Context(), cfg, dir,
		func(context.Context, *config.Config) bool { return false },
		func(context.Context, *config.Config, string) error { applyCalled = true; return nil },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err != nil {
		t.Fatalf("unreachable hasura must never fail the caller (even in prod), got %v", err)
	}
	if applyCalled {
		t.Error("apply must not be attempted when hasura is unreachable")
	}
}

func TestApplyIfPresent_ProceedsWhenHasuraReachable(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	cfg := &config.Config{Env: "dev"}
	applyCalled := false
	err := applyIfPresent(t.Context(), cfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { applyCalled = true; return nil },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !applyCalled {
		t.Error("apply must be attempted when hasura is reachable")
	}
}

func TestApplyIfPresent_WarnOnlyInDevOnApplyFailure(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	cfg := &config.Config{Env: "dev"}
	err := applyIfPresent(t.Context(), cfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { return errors.New("boom") },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err != nil {
		t.Errorf("dev (non-strict) apply failure must warn, not fail the caller; got %v", err)
	}
}

func TestApplyIfPresent_StrictInProdOnApplyFailure(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	cfg := &config.Config{Env: "prod"}
	err := applyIfPresent(t.Context(), cfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { return errors.New("boom") },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err == nil {
		t.Error("prod (strict-by-default) apply failure must fail the caller")
	}
}

func TestApplyIfPresent_StrictInStagingOnApplyFailure(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	cfg := &config.Config{Env: "staging"}
	err := applyIfPresent(t.Context(), cfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { return errors.New("boom") },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err == nil {
		t.Error("staging (strict-by-default) apply failure must fail the caller")
	}
}

func TestApplyIfPresent_ExplicitStrictOverridesDevDefault(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	t.Setenv("NSELF_HASURA_METADATA_STRICT", "true")
	cfg := &config.Config{Env: "dev"}
	err := applyIfPresent(t.Context(), cfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { return errors.New("boom") },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err == nil {
		t.Error("NSELF_HASURA_METADATA_STRICT=true must fail even in dev")
	}
}

func TestApplyIfPresent_ExplicitFalseOverridesProdDefault(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	t.Setenv("NSELF_HASURA_METADATA_STRICT", "false")
	cfg := &config.Config{Env: "prod"}
	err := applyIfPresent(t.Context(), cfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { return errors.New("boom") },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err != nil {
		t.Errorf("NSELF_HASURA_METADATA_STRICT=false must warn-only even in prod, got %v", err)
	}
}

func TestApplyIfPresent_InconsistentObjectsFailStrictWarnDev(t *testing.T) {
	dir := makeMetadataDirFixture(t)

	// Strict (prod): inconsistency fails the caller.
	prodCfg := &config.Config{Env: "prod"}
	err := applyIfPresent(t.Context(), prodCfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { return nil },
		func(context.Context, *config.Config) ([]string, error) { return []string{"np_foo_select_perm"}, nil })
	if err == nil {
		t.Error("prod: inconsistent objects after a successful apply must fail the caller")
	}

	// Warn-only (dev): inconsistency is reported but does not fail.
	devCfg := &config.Config{Env: "dev"}
	err = applyIfPresent(t.Context(), devCfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { return nil },
		func(context.Context, *config.Config) ([]string, error) { return []string{"np_foo_select_perm"}, nil })
	if err != nil {
		t.Errorf("dev: inconsistent objects must warn, not fail; got %v", err)
	}
}

func TestApplyIfPresent_ConsistencyCheckFailureNeverMasksSuccessfulApply(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	cfg := &config.Config{Env: "prod"}
	err := applyIfPresent(t.Context(), cfg, dir, alwaysReachable,
		func(context.Context, *config.Config, string) error { return nil },
		func(context.Context, *config.Config) ([]string, error) { return nil, errors.New("network blip") })
	if err != nil {
		t.Errorf("a failed consistency check must not fail a successful apply, got %v", err)
	}
}

func TestIsStrict(t *testing.T) {
	cases := []struct {
		name string
		env  string
		envv string
		want bool
	}{
		{"dev default warn", "dev", "", false},
		{"staging default strict", "staging", "", true},
		{"prod default strict", "prod", "", true},
		{"production alias strict", "production", "", true},
		{"explicit true wins over dev", "dev", "true", true},
		{"explicit false wins over prod", "prod", "false", false},
		{"invalid override falls back to default", "prod", "not-a-bool", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envv != "" {
				t.Setenv("NSELF_HASURA_METADATA_STRICT", tc.envv)
			}
			cfg := &config.Config{Env: tc.env}
			if got := IsStrict(cfg); got != tc.want {
				t.Errorf("IsStrict(Env=%q, override=%q) = %v, want %v", tc.env, tc.envv, got, tc.want)
			}
		})
	}
}

// TestIsHasuraReachable_FalseWhenNothingListening exercises the real
// (non-faked) isHasuraReachable against a port nothing is listening on, to
// confirm connection failure — not just "any non-2xx" — is what maps to
// false. A live-Hasura "true" path is covered by the docker-gated
// integration test.
func TestIsHasuraReachable_FalseWhenNothingListening(t *testing.T) {
	cfg := &config.Config{}
	cfg.Hasura.Port = 59999 // high ephemeral port, nothing binds it in CI
	if isHasuraReachable(t.Context(), cfg) {
		t.Error("expected false when nothing is listening on the configured port")
	}
}

// makeMetadataDirFixture creates <tmp>/hasura/metadata/ so ApplyIfPresent's
// existence check passes, without needing real metadata content (apply/
// inconsistent are faked in every test above).
func makeMetadataDirFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hasura", "metadata"), 0o755); err != nil {
		t.Fatalf("mkdir hasura/metadata fixture: %v", err)
	}
	return dir
}
