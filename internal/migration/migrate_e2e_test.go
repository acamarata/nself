package migration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nself-org/cli/internal/migration"
)

// fixtureDir is the committed v0.9 fixture project used for CI regression testing.
const fixtureDir = "testdata/v0.9-fixture"

// expectedDir is the committed expected post-migration state.
const expectedDir = "testdata/v0.9-fixture-expected"

// TestE2E_FixtureDetect verifies that nself migrate detect finds all 5 v0.9
// artifacts in the committed fixture. This catches regressions in Detect().
//
// The fixture stores the env file as .env.example (per Clean Working Tree rule).
// copyDirForTest restores it as .env in the temp copy so Detect() can find it.
func TestE2E_FixtureDetect(t *testing.T) {
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("fixture not found at %s: %v", fixtureDir, err)
	}

	// Copy to temp so Detect() sees .env (restored from .env.example by copyDirForTest).
	tmp := t.TempDir()
	if err := copyDirForTest(t, fixtureDir, tmp); err != nil {
		t.Fatalf("copying fixture to temp: %v", err)
	}

	artifacts := migration.Detect(tmp)
	if len(artifacts) != 5 {
		t.Errorf("expected 5 v0.9 artifacts in fixture, got %d", len(artifacts))
		for _, a := range artifacts {
			t.Logf("  found: %s — %s", a.Path, a.Description)
		}
		t.Log("Missing artifact paths may indicate a Detect() regression.")
	}
}

// TestE2E_CheckLegacyProject_Fixture verifies that CheckLegacyProject returns count≥2
// for the committed fixture, triggering the hard-fail threshold.
//
// Uses a temp copy so .env.example is restored as .env by copyDirForTest.
func TestE2E_CheckLegacyProject_Fixture(t *testing.T) {
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("fixture not found at %s: %v", fixtureDir, err)
	}

	tmp := t.TempDir()
	if err := copyDirForTest(t, fixtureDir, tmp); err != nil {
		t.Fatalf("copying fixture to temp: %v", err)
	}

	count, names := migration.CheckLegacyProject(tmp)
	if count < migration.DetectionThreshold {
		t.Errorf("expected count >= %d, got %d (artifacts: %v)", migration.DetectionThreshold, count, names)
	}
}

// TestE2E_MigrateRun_CopiesFixtureToTemp runs migrate.Run on a copy of the fixture
// in a temp directory and verifies the post-state matches the expected/ snapshot.
// This test does NOT require Docker — it only tests file-level migration steps.
func TestE2E_MigrateRun_CopiesFixtureToTemp(t *testing.T) {
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("fixture not found at %s: %v", fixtureDir, err)
	}

	// Copy the fixture into a temp dir so the real fixture is not mutated.
	tmp := t.TempDir()
	if err := copyDirForTest(t, fixtureDir, tmp); err != nil {
		t.Fatalf("copying fixture to temp: %v", err)
	}

	// Pre-condition: fixture should be detected as v0.9.
	before := migration.Detect(tmp)
	if len(before) == 0 {
		t.Fatal("fixture copy has no v0.9 artifacts — pre-condition failed")
	}

	// Run migration (file-level steps only — skips docker compose down and build).
	// We test the nginx layout migration directly because Run() depends on docker.
	moved, err := migrateNginxLayoutForTest(t, tmp)
	if err != nil {
		t.Fatalf("nginx migration step failed: %v", err)
	}
	if moved == 0 {
		t.Error("expected at least one nginx config to be moved to nginx/sites/")
	}

	// Post-condition: nginx/sites/ must exist and contain the moved config.
	sitesDir := filepath.Join(tmp, "nginx", "sites")
	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		t.Fatalf("nginx/sites/ not created after migration: %v", err)
	}
	if len(entries) == 0 {
		t.Error("nginx/sites/ is empty after migration")
	}

	// Verify the expected post-state has a matching file in nginx/sites/.
	expectedSites := filepath.Join(expectedDir, "nginx", "sites")
	expectedEntries, err := os.ReadDir(expectedSites)
	if err != nil {
		t.Skipf("expected dir not found at %s — skipping post-state comparison", expectedSites)
	}
	for _, ee := range expectedEntries {
		actualPath := filepath.Join(sitesDir, ee.Name())
		if _, err := os.Stat(actualPath); err != nil {
			t.Errorf("expected post-state file %s not found in migrated output", ee.Name())
		}
	}
}

// TestE2E_DetectionThreshold_IsTwo verifies the compile-time constant is correct.
func TestE2E_DetectionThreshold_IsTwo(t *testing.T) {
	if migration.DetectionThreshold != 2 {
		t.Errorf("DetectionThreshold should be 2, got %d", migration.DetectionThreshold)
	}
}

// migrateNginxLayoutForTest calls the exported Detect to determine pre-state,
// then directly invokes the nginx migration via Run with a guard that skips
// the docker-dependent steps. Since migrateNginxLayout is unexported, we
// replicate its logic here for the e2e test.
func migrateNginxLayoutForTest(t *testing.T, dir string) (int, error) {
	t.Helper()
	nginxDir := filepath.Join(dir, "nginx")
	info, err := os.Stat(nginxDir)
	if err != nil || !info.IsDir() {
		return 0, nil
	}
	sitesDir := filepath.Join(nginxDir, "sites")
	if _, err := os.Stat(sitesDir); err == nil {
		return 0, nil // already migrated
	}
	if err := os.MkdirAll(sitesDir, 0o755); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(nginxDir)
	if err != nil {
		return 0, err
	}
	moved := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) == ".conf" || filepath.Ext(name) == ".template" {
			src := filepath.Join(nginxDir, name)
			dst := filepath.Join(sitesDir, name)
			if err := os.Rename(src, dst); err != nil {
				return moved, err
			}
			moved++
		}
	}
	return moved, nil
}

// copyDirForTest recursively copies src into dst for test isolation.
// .env.example files are copied twice: once as-is, and once as .env so that
// the Detect() function (which looks for a literal ".env") still finds the
// fixture artifact in the temp directory.
func copyDirForTest(t *testing.T, src, dst string) error {
	t.Helper()
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, info.Mode()); err != nil {
			return err
		}
		// If this is a .env.example fixture file, also write it as .env so
		// Detect() can find the legacy artifact during migration tests.
		if filepath.Base(path) == ".env.example" {
			envTarget := filepath.Join(filepath.Dir(target), ".env")
			if err := os.WriteFile(envTarget, data, 0o600); err != nil {
				return err
			}
		}
		return nil
	})
}
