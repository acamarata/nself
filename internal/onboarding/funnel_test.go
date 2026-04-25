package onboarding

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupConfigDir creates a temporary ~/.config/nself-like directory and returns
// a cleanup function. It patches nSelfConfigDir via the tmpDir mechanism.
func setupConfigDir(t *testing.T) (dir string, cleanup func()) {
	t.Helper()
	dir = t.TempDir()
	// Override the config dir for the duration of the test by monkey-patching
	// the package-level helper via the test-only override (see state_test_hook.go).
	origDir := testConfigDirOverride
	testConfigDirOverride = dir
	return dir, func() {
		testConfigDirOverride = origDir
	}
}

// ── Stage 1: Install ─────────────────────────────────────────────────────────

func TestCheckStage1Install_AlwaysPasses(t *testing.T) {
	result := checkStage1Install()
	if result.Status != StatusPass {
		t.Errorf("stage 1 should always pass, got %s: %s", result.Status, result.Message)
	}
	if result.Metadata["version"] == nil {
		t.Error("stage 1 should include version in metadata")
	}
	if result.Metadata["os"] == nil || result.Metadata["arch"] == nil {
		t.Error("stage 1 should include os and arch in metadata")
	}
}

// ── Stage 2: Activation ──────────────────────────────────────────────────────

func TestCheckStage2Activation_NoProjects(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	_ = dir // no projects.json written
	result := checkStage2Activation(false)
	if result.Status != StatusFail {
		t.Errorf("expected fail when no projects.json, got %s", result.Status)
	}
	if result.Remediation == "" {
		t.Error("expected remediation hint")
	}
}

func TestCheckStage2Activation_WithProjects(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	writeFile(t, filepath.Join(dir, "projects.json"), `[{"name":"myproject"}]`)
	result := checkStage2Activation(false)
	if result.Status != StatusPass {
		t.Errorf("expected pass with projects, got %s: %s", result.Status, result.Message)
	}
	if result.Metadata["projects_count"].(int) != 1 {
		t.Error("expected projects_count=1")
	}
}

func TestCheckStage2Activation_Skipped(t *testing.T) {
	result := checkStage2Activation(true)
	if result.Status != StatusSkipped {
		t.Errorf("expected skipped when prior stage failed, got %s", result.Status)
	}
}

// ── Stage 3: First-use ───────────────────────────────────────────────────────

func TestCheckStage3FirstUse_NoTimestamp(t *testing.T) {
	_, cleanup := setupConfigDir(t)
	defer cleanup()
	result := checkStage3FirstUse(false, StageResult{Status: StatusPass})
	if result.Status != StatusFail {
		t.Errorf("expected fail with no timestamp file, got %s", result.Status)
	}
}

func TestCheckStage3FirstUse_WithTimestamp(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	ts := time.Now().Add(-72 * time.Hour).UTC().Format(time.RFC3339)
	writeFile(t, filepath.Join(dir, "last-start.timestamp"), ts)
	result := checkStage3FirstUse(false, StageResult{Status: StatusPass})
	if result.Status != StatusPass {
		t.Errorf("expected pass with timestamp file, got %s: %s", result.Status, result.Message)
	}
}

func TestCheckStage3FirstUse_Skipped(t *testing.T) {
	result := checkStage3FirstUse(true, StageResult{})
	if result.Status != StatusSkipped {
		t.Errorf("expected skipped, got %s", result.Status)
	}
}

// ── Stage 4: First-plugin ────────────────────────────────────────────────────

func TestCheckStage4FirstPlugin_NoPlugins(t *testing.T) {
	_, cleanup := setupConfigDir(t)
	defer cleanup()
	result := checkStage4FirstPlugin(false)
	if result.Status != StatusFail {
		t.Errorf("expected fail with no plugin-state.yaml, got %s", result.Status)
	}
}

func TestCheckStage4FirstPlugin_WithPlugin(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	writeFile(t, filepath.Join(dir, "plugin-state.yaml"), "plugins:\n  - name: ai\n    tier: pro\n")
	result := checkStage4FirstPlugin(false)
	if result.Status != StatusPass {
		t.Errorf("expected pass with plugin-state.yaml, got %s: %s", result.Status, result.Message)
	}
	if result.Metadata["plugin_name"] != "ai" {
		t.Errorf("expected plugin_name=ai, got %v", result.Metadata["plugin_name"])
	}
}

// ── Stage 5: First-value ─────────────────────────────────────────────────────

func TestCheckStage5FirstValue_FileAbsent(t *testing.T) {
	_, cleanup := setupConfigDir(t)
	defer cleanup()
	result := checkStage5FirstValue(false)
	if result.Status != StatusUnknown {
		t.Errorf("expected unknown when query-count absent, got %s", result.Status)
	}
}

func TestCheckStage5FirstValue_ZeroCount(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	writeFile(t, filepath.Join(dir, "query-count"), "0")
	result := checkStage5FirstValue(false)
	if result.Status != StatusFail {
		t.Errorf("expected fail with 0 queries, got %s", result.Status)
	}
}

func TestCheckStage5FirstValue_NonZeroCount(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	writeFile(t, filepath.Join(dir, "query-count"), "42")
	result := checkStage5FirstValue(false)
	if result.Status != StatusPass {
		t.Errorf("expected pass with 42 queries, got %s: %s", result.Status, result.Message)
	}
}

// ── Stage 6: Habit ───────────────────────────────────────────────────────────

func TestCheckStage6Habit_NoLog(t *testing.T) {
	_, cleanup := setupConfigDir(t)
	defer cleanup()
	result := checkStage6Habit(false)
	if result.Status != StatusFail {
		t.Errorf("expected fail with no usage.log, got %s", result.Status)
	}
}

func TestCheckStage6Habit_InsufficientStarts(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	ts := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	writeFile(t, filepath.Join(dir, "usage.log"), ts+"\n"+ts+"\n")
	result := checkStage6Habit(false)
	if result.Status != StatusFail {
		t.Errorf("expected fail with 2 starts, got %s", result.Status)
	}
}

func TestCheckStage6Habit_SufficientStarts(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	ts := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	lines := ts + "\n" + ts + "\n" + ts + "\n"
	writeFile(t, filepath.Join(dir, "usage.log"), lines)
	result := checkStage6Habit(false)
	if result.Status != StatusPass {
		t.Errorf("expected pass with 3 recent starts, got %s: %s", result.Status, result.Message)
	}
}

func TestCheckStage6Habit_OldStartsFiltered(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	old := time.Now().Add(-10 * 24 * time.Hour).UTC().Format(time.RFC3339)
	writeFile(t, filepath.Join(dir, "usage.log"), old+"\n"+old+"\n"+old+"\n"+old+"\n")
	result := checkStage6Habit(false)
	if result.Status != StatusFail {
		t.Errorf("expected fail when all starts are >7 days old, got %s", result.Status)
	}
}

// ── Full funnel integration ──────────────────────────────────────────────────

func TestRunFunnel_NewInstall(t *testing.T) {
	_, cleanup := setupConfigDir(t)
	defer cleanup()
	// No state files: stages 2-6 should fail/skip after stage 1 passes
	report := RunFunnel()
	if report.Stages[0].Status != StatusPass {
		t.Errorf("stage 1 should pass, got %s", report.Stages[0].Status)
	}
	if report.Stages[1].Status != StatusFail {
		t.Errorf("stage 2 should fail (no projects), got %s", report.Stages[1].Status)
	}
	for i := 2; i < 6; i++ {
		if report.Stages[i].Status != StatusSkipped {
			t.Errorf("stage %d should be skipped, got %s", i+1, report.Stages[i].Status)
		}
	}
	if report.Position != 1 {
		t.Errorf("funnel position should be 1, got %d", report.Position)
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writeFile %s: %v", path, err)
	}
}
