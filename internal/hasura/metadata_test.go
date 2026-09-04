package hasura

// Purpose: unit tests for ApplyIfPresent's skip/strict/warn decision tree —
// the actual apply and inconsistency-check calls are faked (applyFn/
// inconsistentFn) so this exercises the logic without a live Hasura.
// Constraints: no network, no docker required.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

func TestApplyIfPresent_SkipsCleanlyWhenNoMetadataDir(t *testing.T) {
	dir := t.TempDir() // no hasura/ subdir at all
	cfg := &config.Config{}
	called := false
	err := applyIfPresent(t.Context(), cfg, dir,
		func(context.Context, *config.Config, string) error { called = true; return nil },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err != nil {
		t.Fatalf("expected nil error on clean skip, got %v", err)
	}
	if called {
		t.Error("apply function must not be called when hasura/metadata is absent")
	}
}

func TestApplyIfPresent_WarnOnlyInDevOnApplyFailure(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	cfg := &config.Config{Env: "dev"}
	err := applyIfPresent(t.Context(), cfg, dir,
		func(context.Context, *config.Config, string) error { return errors.New("boom") },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err != nil {
		t.Errorf("dev (non-strict) apply failure must warn, not fail the caller; got %v", err)
	}
}

func TestApplyIfPresent_StrictInProdOnApplyFailure(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	cfg := &config.Config{Env: "prod"}
	err := applyIfPresent(t.Context(), cfg, dir,
		func(context.Context, *config.Config, string) error { return errors.New("boom") },
		func(context.Context, *config.Config) ([]string, error) { return nil, nil })
	if err == nil {
		t.Error("prod (strict-by-default) apply failure must fail the caller")
	}
}

func TestApplyIfPresent_StrictInStagingOnApplyFailure(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	cfg := &config.Config{Env: "staging"}
	err := applyIfPresent(t.Context(), cfg, dir,
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
	err := applyIfPresent(t.Context(), cfg, dir,
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
	err := applyIfPresent(t.Context(), cfg, dir,
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
	err := applyIfPresent(t.Context(), prodCfg, dir,
		func(context.Context, *config.Config, string) error { return nil },
		func(context.Context, *config.Config) ([]string, error) { return []string{"np_foo_select_perm"}, nil })
	if err == nil {
		t.Error("prod: inconsistent objects after a successful apply must fail the caller")
	}

	// Warn-only (dev): inconsistency is reported but does not fail.
	devCfg := &config.Config{Env: "dev"}
	err = applyIfPresent(t.Context(), devCfg, dir,
		func(context.Context, *config.Config, string) error { return nil },
		func(context.Context, *config.Config) ([]string, error) { return []string{"np_foo_select_perm"}, nil })
	if err != nil {
		t.Errorf("dev: inconsistent objects must warn, not fail; got %v", err)
	}
}

func TestApplyIfPresent_ConsistencyCheckFailureNeverMasksSuccessfulApply(t *testing.T) {
	dir := makeMetadataDirFixture(t)
	cfg := &config.Config{Env: "prod"}
	err := applyIfPresent(t.Context(), cfg, dir,
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
