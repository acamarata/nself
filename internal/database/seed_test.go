package database

import (
	"path/filepath"
	"testing"
)

// ── SeedFilter ────────────────────────────────────────────────────────────────

// TestSeedFilter_UnmentionedRunsInAllEnvs verifies that a file not listed in
// dbEnvSeeds is included regardless of the current environment.
func TestSeedFilter_UnmentionedRunsInAllEnvs(t *testing.T) {
	files := []string{"seeds/base.sql", "seeds/fixtures.sql"}
	dbEnvSeeds := "dev:dev-data.sql" // base.sql and fixtures.sql are not mentioned

	for _, env := range []string{"dev", "staging", "prod"} {
		got := SeedFilter(files, dbEnvSeeds, env)
		if len(got) != 2 {
			t.Errorf("env=%q: expected 2 files (unscoped), got %d: %v", env, len(got), got)
		}
	}
}

// TestSeedFilter_ScopedFileRunsOnlyInMatchingEnv verifies that a file mapped to
// a specific env is included only when currentEnv matches.
func TestSeedFilter_ScopedFileRunsOnlyInMatchingEnv(t *testing.T) {
	files := []string{"seeds/dev-data.sql"}
	dbEnvSeeds := "dev:dev-data.sql"

	got := SeedFilter(files, dbEnvSeeds, "dev")
	if len(got) != 1 {
		t.Errorf("expected 1 file in dev env, got %d: %v", len(got), got)
	}

	got = SeedFilter(files, dbEnvSeeds, "prod")
	if len(got) != 0 {
		t.Errorf("expected 0 files in prod env (scoped to dev), got %d: %v", len(got), got)
	}
}

// TestSeedFilter_MultipleEntriesSelectivelyInclude verifies that when dbEnvSeeds
// maps multiple files to different envs, only the matching-env file is included.
func TestSeedFilter_MultipleEntriesSelectivelyInclude(t *testing.T) {
	files := []string{
		"seeds/dev-fixtures.sql",
		"seeds/staging-fixtures.sql",
		"seeds/base.sql",
	}
	dbEnvSeeds := "dev:dev-fixtures.sql,staging:staging-fixtures.sql"

	gotDev := SeedFilter(files, dbEnvSeeds, "dev")
	wantDevCount := 2 // dev-fixtures.sql + base.sql (unscoped)
	if len(gotDev) != wantDevCount {
		t.Errorf("dev env: expected %d files, got %d: %v", wantDevCount, len(gotDev), gotDev)
	}

	// staging-fixtures.sql should be in dev result only if it was unscoped.
	for _, f := range gotDev {
		if filepath.Base(f) == "staging-fixtures.sql" {
			t.Errorf("dev env must not include staging-fixtures.sql, got: %v", gotDev)
		}
	}

	gotStaging := SeedFilter(files, dbEnvSeeds, "staging")
	wantStagingCount := 2 // staging-fixtures.sql + base.sql (unscoped)
	if len(gotStaging) != wantStagingCount {
		t.Errorf("staging env: expected %d files, got %d: %v", wantStagingCount, len(gotStaging), gotStaging)
	}
}

// TestSeedFilter_EmptyDbEnvSeeds includes all files when the env-seeds mapping
// is an empty string (the feature is disabled).
func TestSeedFilter_EmptyDbEnvSeeds(t *testing.T) {
	files := []string{"seeds/a.sql", "seeds/b.sql", "seeds/c.sql"}

	got := SeedFilter(files, "", "prod")
	if len(got) != 3 {
		t.Errorf("empty dbEnvSeeds: expected all 3 files, got %d: %v", len(got), got)
	}
}

// TestSeedFilter_EmptyFileList returns empty slice when there are no seed files.
func TestSeedFilter_EmptyFileList(t *testing.T) {
	got := SeedFilter(nil, "dev:something.sql", "dev")
	if len(got) != 0 {
		t.Errorf("empty file list: expected 0 files, got %d: %v", len(got), got)
	}
}

// TestSeedFilter_MalformedEntryIgnored verifies that an entry without a colon
// separator is silently skipped and does not affect other entries.
func TestSeedFilter_MalformedEntryIgnored(t *testing.T) {
	files := []string{"seeds/a.sql", "seeds/b.sql"}
	// "noseparator" has no colon — should be ignored.
	dbEnvSeeds := "noseparator,dev:b.sql"

	gotDev := SeedFilter(files, dbEnvSeeds, "dev")
	// a.sql is unscoped (runs everywhere), b.sql is scoped to dev.
	if len(gotDev) != 2 {
		t.Errorf("dev env: expected 2 files (a.sql + b.sql), got %d: %v", len(gotDev), gotDev)
	}

	gotProd := SeedFilter(files, dbEnvSeeds, "prod")
	// a.sql is unscoped; b.sql is scoped to dev only.
	if len(gotProd) != 1 {
		t.Errorf("prod env: expected 1 file (a.sql only), got %d: %v", len(gotProd), gotProd)
	}
	if filepath.Base(gotProd[0]) != "a.sql" {
		t.Errorf("prod env: expected a.sql, got %q", gotProd[0])
	}
}

// TestSeedFilter_PreservesOrder verifies that the output slice preserves the
// input order (SeedFilter must not sort or shuffle files).
func TestSeedFilter_PreservesOrder(t *testing.T) {
	files := []string{
		"seeds/01_base.sql",
		"seeds/02_fixtures.sql",
		"seeds/03_extras.sql",
	}

	got := SeedFilter(files, "", "dev") // no scoping — all three should come back
	for i, f := range files {
		if i >= len(got) || got[i] != f {
			t.Errorf("order mismatch at index %d: got %q, want %q", i, got[i], f)
		}
	}
}

// TestSeedFilter_WhitespaceAroundEntries verifies that leading/trailing
// whitespace in dbEnvSeeds entries is trimmed correctly.
func TestSeedFilter_WhitespaceAroundEntries(t *testing.T) {
	files := []string{"seeds/dev-data.sql"}
	dbEnvSeeds := " dev : dev-data.sql " // spaces around both parts

	got := SeedFilter(files, dbEnvSeeds, "dev")
	if len(got) != 1 {
		t.Errorf("whitespace-padded entry: expected 1 file in dev env, got %d: %v", len(got), got)
	}

	got = SeedFilter(files, dbEnvSeeds, "prod")
	if len(got) != 0 {
		t.Errorf("whitespace-padded entry: expected 0 files in prod env, got %d: %v", len(got), got)
	}
}
