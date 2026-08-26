package bluegreen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── GenerateNginxUpstream ────────────────────────────────────────────────────

func TestGenerateNginxUpstream_canary10(t *testing.T) {
	cfg := DeployConfig{
		BluePortOffset:  0,
		GreenPortOffset: 100,
	}
	got := GenerateNginxUpstream(cfg, 10)

	// Blue at base port with weight 90.
	if !strings.Contains(got, "127.0.0.1:8080 weight=90") {
		t.Errorf("expected blue 8080 weight=90, got:\n%s", got)
	}
	// Green at base+100 with weight 10.
	if !strings.Contains(got, "127.0.0.1:8180 weight=10") {
		t.Errorf("expected green 8180 weight=10, got:\n%s", got)
	}
}

func TestGenerateNginxUpstream_fullFlipToGreen(t *testing.T) {
	cfg := DeployConfig{
		BluePortOffset:  0,
		GreenPortOffset: 100,
	}
	got := GenerateNginxUpstream(cfg, 100)

	// Green at full weight.
	if !strings.Contains(got, "127.0.0.1:8180 weight=100") {
		t.Errorf("expected green 8180 weight=100, got:\n%s", got)
	}
	// Blue should be a backup entry (weight=0 backup).
	if !strings.Contains(got, "8080 weight=0 backup") {
		t.Errorf("expected blue as backup, got:\n%s", got)
	}
}

func TestGenerateNginxUpstream_allBlue(t *testing.T) {
	cfg := DeployConfig{
		BluePortOffset:  0,
		GreenPortOffset: 100,
	}
	got := GenerateNginxUpstream(cfg, 0)

	// Blue at full weight.
	if !strings.Contains(got, "127.0.0.1:8080 weight=100") {
		t.Errorf("expected blue 8080 weight=100, got:\n%s", got)
	}
	// Green should be a backup entry.
	if !strings.Contains(got, "8180 weight=0 backup") {
		t.Errorf("expected green as backup, got:\n%s", got)
	}
}

func TestGenerateNginxUpstream_canary25(t *testing.T) {
	cfg := DeployConfig{
		BluePortOffset:  0,
		GreenPortOffset: 100,
	}
	got := GenerateNginxUpstream(cfg, 25)

	if !strings.Contains(got, "weight=75") {
		t.Errorf("expected blue weight=75, got:\n%s", got)
	}
	if !strings.Contains(got, "weight=25") {
		t.Errorf("expected green weight=25, got:\n%s", got)
	}
}

func TestGenerateNginxUpstream_wrapsInUpstreamBlock(t *testing.T) {
	cfg := DeployConfig{BluePortOffset: 0, GreenPortOffset: 100}
	got := GenerateNginxUpstream(cfg, 10)

	if !strings.HasPrefix(got, "upstream nself_api {") {
		t.Errorf("expected upstream block, got:\n%s", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), "}") {
		t.Errorf("expected closing brace, got:\n%s", got)
	}
}

// ── CheckMigrationCompatibility ──────────────────────────────────────────────

func TestCheckMigrationCompatibility_noMigrationsDir(t *testing.T) {
	tmp := t.TempDir()
	result := CheckMigrationCompatibility(tmp)
	if !result.Compatible {
		t.Errorf("expected compatible when no migrations dir, got: %v", result)
	}
}

func TestCheckMigrationCompatibility_safeOnly(t *testing.T) {
	tmp := t.TempDir()
	migrDir := filepath.Join(tmp, "migrations")
	if err := os.Mkdir(migrDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write a benign migration.
	if err := os.WriteFile(filepath.Join(migrDir, "0001_add_users_table.sql"), []byte("CREATE TABLE users (id uuid);"), 0644); err != nil {
		t.Fatal(err)
	}

	result := CheckMigrationCompatibility(tmp)
	if !result.Compatible {
		t.Errorf("expected compatible with safe migration, got: %+v", result)
	}
}

func TestCheckMigrationCompatibility_dropColumnFlagged(t *testing.T) {
	tmp := t.TempDir()
	migrDir := filepath.Join(tmp, "migrations")
	if err := os.Mkdir(migrDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Migration filename signals an incompatible schema change.
	if err := os.WriteFile(filepath.Join(migrDir, "0002_drop_column_old_name.sql"), []byte("ALTER TABLE users DROP COLUMN old_name;"), 0644); err != nil {
		t.Fatal(err)
	}

	result := CheckMigrationCompatibility(tmp)
	if result.Compatible {
		t.Errorf("expected incompatible when drop_column migration present")
	}
	if len(result.IncompatibleFiles) == 0 {
		t.Errorf("expected IncompatibleFiles to be set")
	}
	if result.Reason == "" {
		t.Errorf("expected Reason to be set")
	}
}

func TestCheckMigrationCompatibility_dropTableFlagged(t *testing.T) {
	tmp := t.TempDir()
	migrDir := filepath.Join(tmp, "migrations")
	if err := os.Mkdir(migrDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migrDir, "0003_drop_table_legacy.sql"), []byte("DROP TABLE legacy;"), 0644); err != nil {
		t.Fatal(err)
	}

	result := CheckMigrationCompatibility(tmp)
	if result.Compatible {
		t.Errorf("expected incompatible for drop_table migration")
	}
}

// ── DeployState persistence ───────────────────────────────────────────────────

func TestSaveAndLoadState(t *testing.T) {
	tmp := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	state := DeployState{
		Active:        EnvBlue,
		BlueVersion:   "v1.0.9",
		GreenVersion:  "v1.1.0",
		CanaryPercent: 10,
		LastDeploy:    now,
	}

	if err := saveState(tmp, state); err != nil {
		t.Fatalf("saveState failed: %v", err)
	}

	loaded, err := loadState(tmp)
	if err != nil {
		t.Fatalf("loadState failed: %v", err)
	}

	if loaded.Active != state.Active {
		t.Errorf("Active: got %q, want %q", loaded.Active, state.Active)
	}
	if loaded.BlueVersion != state.BlueVersion {
		t.Errorf("BlueVersion: got %q, want %q", loaded.BlueVersion, state.BlueVersion)
	}
	if loaded.CanaryPercent != state.CanaryPercent {
		t.Errorf("CanaryPercent: got %d, want %d", loaded.CanaryPercent, state.CanaryPercent)
	}
}

func TestLoadState_missingFile(t *testing.T) {
	tmp := t.TempDir()
	state, err := loadState(tmp)
	if err != nil {
		t.Fatalf("expected nil error for missing state file, got: %v", err)
	}
	// Default active is blue.
	if state.Active != EnvBlue {
		t.Errorf("expected default active=blue, got %q", state.Active)
	}
}

func TestSwapState(t *testing.T) {
	tmp := t.TempDir()
	initial := DeployState{
		Active:        EnvBlue,
		BlueVersion:   "v1.0.9",
		GreenVersion:  "v1.1.0",
		CanaryPercent: 50,
	}
	if err := saveState(tmp, initial); err != nil {
		t.Fatal(err)
	}

	if err := swapState(tmp); err != nil {
		t.Fatalf("swapState failed: %v", err)
	}

	state, err := loadState(tmp)
	if err != nil {
		t.Fatal(err)
	}

	// After swap: active=blue (re-labelled), blue holds the former green version,
	// green is cleared, canary reset.
	if state.Active != EnvBlue {
		t.Errorf("expected active=blue after swap, got %q", state.Active)
	}
	if state.BlueVersion != "v1.1.0" {
		t.Errorf("expected BlueVersion=v1.1.0, got %q", state.BlueVersion)
	}
	if state.GreenVersion != "" {
		t.Errorf("expected GreenVersion cleared, got %q", state.GreenVersion)
	}
	if state.CanaryPercent != 0 {
		t.Errorf("expected CanaryPercent=0, got %d", state.CanaryPercent)
	}
}

// ── DryRun ───────────────────────────────────────────────────────────────────

func TestDeploy_dryRun(t *testing.T) {
	cfg := DeployConfig{
		ProjectRoot:   t.TempDir(),
		CanaryPercent: 10,
		DryRun:        true,
	}

	result := Deploy(nil, cfg) //nolint:staticcheck // nil ctx is fine for dry-run
	if !result.Success {
		t.Errorf("expected dry-run success, got error: %s", result.Error)
	}
	if len(result.Steps) == 0 {
		t.Errorf("expected steps in dry-run result")
	}
	// All steps should be pending.
	for _, step := range result.Steps {
		if !strings.Contains(step.Status, "dry-run") {
			t.Errorf("expected dry-run status for step %q, got %q", step.Name, step.Status)
		}
	}
}

func TestDeploy_dryRunSkipCanary(t *testing.T) {
	cfg := DeployConfig{
		ProjectRoot: t.TempDir(),
		SkipCanary:  true,
		DryRun:      true,
	}

	result := Deploy(nil, cfg) //nolint:staticcheck // nil ctx is fine for dry-run
	if !result.Success {
		t.Errorf("expected dry-run success: %s", result.Error)
	}
	// No canary step in output.
	for _, step := range result.Steps {
		if strings.Contains(strings.ToLower(step.Name), "canary") {
			t.Errorf("expected no canary steps when SkipCanary=true, got: %q", step.Name)
		}
	}
}

// ── applyDefaults ────────────────────────────────────────────────────────────

func TestApplyDefaults(t *testing.T) {
	cfg := DeployConfig{}
	applyDefaults(&cfg)

	if cfg.SoakMinutes != 5 {
		t.Errorf("expected SoakMinutes=5, got %d", cfg.SoakMinutes)
	}
	if cfg.ErrorThresholdPct != 1.0 {
		t.Errorf("expected ErrorThresholdPct=1.0, got %f", cfg.ErrorThresholdPct)
	}
	if cfg.HealthTimeoutSec != 30 {
		t.Errorf("expected HealthTimeoutSec=30, got %d", cfg.HealthTimeoutSec)
	}
	if cfg.GreenPortOffset != DefaultGreenPortOffset {
		t.Errorf("expected GreenPortOffset=%d, got %d", DefaultGreenPortOffset, cfg.GreenPortOffset)
	}
}

// ── Rollback dry-run ──────────────────────────────────────────────────────────

func TestRollback_dryRun(t *testing.T) {
	cfg := DeployConfig{
		ProjectRoot: t.TempDir(),
		DryRun:      true,
	}
	result := Rollback(nil, cfg) //nolint:staticcheck // nil ctx is fine for dry-run
	if !result.Success {
		t.Errorf("expected dry-run rollback success, got: %s", result.Error)
	}
}

// ── StateFile JSON shape ──────────────────────────────────────────────────────

func TestDeployState_jsonShape(t *testing.T) {
	now := time.Now().UTC()
	s := DeployState{
		Active:        "blue",
		BlueVersion:   "v1.0.9",
		GreenVersion:  "v1.1.0",
		CanaryPercent: 10,
		LastDeploy:    now,
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{"active", "blue_version", "green_version", "canary_percent", "last_deploy"}
	for _, key := range requiredKeys {
		if _, ok := out[key]; !ok {
			t.Errorf("missing JSON key: %s", key)
		}
	}
}

// ── composeProjectName ────────────────────────────────────────────────────────

func TestComposeProjectName(t *testing.T) {
	if got := composeProjectName("blue"); got != "nself-blue" {
		t.Errorf("got %q, want %q", got, "nself-blue")
	}
	if got := composeProjectName("green"); got != "nself-green" {
		t.Errorf("got %q, want %q", got, "nself-green")
	}
}

// ── parseContainerStatusOutput (issue #270) ─────────────────────────────────
//
// The soak gate used to derive the entire error rate from {{.Health}}
// alone, which is only populated for containers that declare a Docker
// healthcheck (postgres, hasura). Every other service hit `continue` before
// `total++`, so it was dropped from the denominator instead of being judged.
// These tests assert both the error rate AND total (the denominator),
// because the bug was a denominator bug: a crash-looping nginx with no
// healthcheck reported 0.00% because it was never counted at all.

func TestParseContainerStatusOutput_table(t *testing.T) {
	cases := []struct {
		name      string
		out       string
		wantRate  float64
		wantTotal int
	}{
		{
			name:      "running with no healthcheck counts healthy",
			out:       "green_nginx\trunning\t",
			wantRate:  0.0,
			wantTotal: 1,
		},
		{
			name:      "running with healthy healthcheck",
			out:       "green_postgres\trunning\thealthy",
			wantRate:  0.0,
			wantTotal: 1,
		},
		{
			name:      "running with unhealthy healthcheck",
			out:       "green_hasura\trunning\tunhealthy",
			wantRate:  100.0,
			wantTotal: 1,
		},
		{
			name:      "restarting with no healthcheck counts unhealthy",
			out:       "green_nginx\trestarting\t",
			wantRate:  100.0,
			wantTotal: 1,
		},
		{
			name:      "exited with no healthcheck counts unhealthy",
			out:       "green_auth\texited\t",
			wantRate:  100.0,
			wantTotal: 1,
		},
		{
			name:      "no containers reported",
			out:       "",
			wantRate:  0.0,
			wantTotal: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rate, total := parseContainerStatusOutput(tc.out)
			if total != tc.wantTotal {
				t.Errorf("total = %d, want %d", total, tc.wantTotal)
			}
			if rate != tc.wantRate {
				t.Errorf("rate = %.2f, want %.2f", rate, tc.wantRate)
			}
		})
	}
}

// TestParseContainerStatusOutput_issue270Scenario reproduces the #270
// failure scenario directly: nginx and auth declare no healthcheck (empty
// Health) and are down, while postgres and hasura declare a healthcheck and
// report healthy. Before the fix, nginx/auth had no Health value so they
// were dropped from the denominator before anything was counted, leaving
// only the two genuinely-healthy services and reporting a clean 0.00% no
// matter how broken the rest of the environment was.
func TestParseContainerStatusOutput_issue270Scenario(t *testing.T) {
	out := strings.Join([]string{
		"nself-green_nginx\trestarting\t",
		"nself-green_auth\texited\t",
		"nself-green_postgres\trunning\thealthy",
		"nself-green_hasura\trunning\thealthy",
	}, "\n")

	rate, total := parseContainerStatusOutput(out)
	if total != 4 {
		t.Fatalf("total = %d, want 4 (nginx/auth must not be dropped from the denominator)", total)
	}
	if rate == 0.0 {
		t.Fatalf("rate = 0.00%%, want > 0%% — a crash-looping nginx and a dead auth service must not report a clean soak")
	}
	if rate != 50.0 {
		t.Errorf("rate = %.2f, want 50.00 (2 of 4 containers down)", rate)
	}
}

// ── allContainersReady (waitForHealth, issue #270 counterpart) ─────────────
//
// waitForHealth used to require every docker compose ps line to literally
// contain the substring "healthy", so a container with no Docker
// healthcheck (empty Health: nginx, auth, any CS_N custom service) could
// never satisfy readiness: the opposite failure mode from the soak gate's
// fail-open bug, on the same underlying question of what counts as healthy.

func TestAllContainersReady_table(t *testing.T) {
	cases := []struct {
		name      string
		out       string
		wantReady bool
		wantTotal int
	}{
		{
			name:      "running with no healthcheck is ready",
			out:       "green_nginx	running	",
			wantReady: true,
			wantTotal: 1,
		},
		{
			name:      "running with healthy healthcheck is ready",
			out:       "green_postgres	running	healthy",
			wantReady: true,
			wantTotal: 1,
		},
		{
			name:      "starting is not ready",
			out:       "green_hasura	running	starting",
			wantReady: false,
			wantTotal: 1,
		},
		{
			name:      "unhealthy is not ready",
			out:       "green_hasura	running	unhealthy",
			wantReady: false,
			wantTotal: 1,
		},
		{
			name:      "restarting is not ready",
			out:       "green_nginx	restarting	",
			wantReady: false,
			wantTotal: 1,
		},
		{
			name:      "exited is not ready",
			out:       "green_auth	exited	",
			wantReady: false,
			wantTotal: 1,
		},
		{
			name:      "no containers reported is not ready",
			out:       "",
			wantReady: false,
			wantTotal: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ready, total := allContainersReady(tc.out)
			if total != tc.wantTotal {
				t.Errorf("total = %d, want %d", total, tc.wantTotal)
			}
			if ready != tc.wantReady {
				t.Errorf("ready = %v, want %v", ready, tc.wantReady)
			}
		})
	}
}

// TestAllContainersReady_issue270Scenario mirrors
// TestParseContainerStatusOutput_issue270Scenario but for the readiness
// side: nginx and auth declare no healthcheck and ARE actually running
// fine (unlike the soak scenario, where they were down) alongside a
// healthy postgres and hasura. Before this fix, nginx/auth's empty Health
// field meant their docker compose ps line never contained "healthy", so
// this reported not-ready forever even though every container was fine.
func TestAllContainersReady_issue270Scenario(t *testing.T) {
	out := strings.Join([]string{
		"nself-green_nginx\trunning\t",
		"nself-green_auth\trunning\t",
		"nself-green_postgres\trunning\thealthy",
		"nself-green_hasura\trunning\thealthy",
	}, "\n")

	ready, total := allContainersReady(out)
	if total != 4 {
		t.Fatalf("total = %d, want 4", total)
	}
	if !ready {
		t.Fatalf("ready = false, want true: nginx/auth running with no healthcheck must count as ready")
	}
}
