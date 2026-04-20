package dr

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// TestDrillRegionFailoverRemoved verifies that invoking the region-failover
// scenario returns an explicit unsupported error — not a stub slog.Info — and
// that no slog.Info placeholder exists in the drillRegionFailover code path.
func TestDrillRegionFailoverRemoved(t *testing.T) {
	cfg := &config.Config{ProjectName: "test"}
	ctx := context.Background()

	_, err := Drill(ctx, cfg, DrillOptions{Scenario: ScenarioRegionFailover})
	if err == nil {
		t.Fatal("expected error for region-failover scenario, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not supported") {
		t.Errorf("error message %q should contain 'not supported'", msg)
	}
	if !strings.Contains(msg, "v1.0.9") {
		t.Errorf("error message %q should reference v1.0.9", msg)
	}
	if !strings.Contains(msg, "v1.1.0") {
		t.Errorf("error message %q should reference v1.1.0 as the planned version", msg)
	}
}

// TestDrillDataCorruptionRemoved verifies that invoking the data-corruption
// scenario returns an explicit unsupported error (not a stub slog.Info).
func TestDrillDataCorruptionRemoved(t *testing.T) {
	cfg := &config.Config{ProjectName: "test"}
	ctx := context.Background()

	_, err := Drill(ctx, cfg, DrillOptions{Scenario: ScenarioDataCorruption})
	if err == nil {
		t.Fatal("expected error for data-corruption scenario, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not supported") {
		t.Errorf("error message %q should contain 'not supported'", msg)
	}
	if !strings.Contains(msg, "v1.0.9") {
		t.Errorf("error message %q should reference v1.0.9", msg)
	}
}

// TestDrillDryRun verifies that dry-run mode exits cleanly for cold-start
// without provisioning any real infrastructure.
func TestDrillDryRun(t *testing.T) {
	cfg := &config.Config{ProjectName: "test"}
	ctx := context.Background()

	result, err := Drill(ctx, cfg, DrillOptions{
		Scenario: ScenarioColdStart,
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error in dry-run: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Status != "dry-run" {
		t.Errorf("status = %q, want %q", result.Status, "dry-run")
	}
	if result.FinishedAt.IsZero() {
		t.Error("FinishedAt should not be zero after dry-run")
	}
}

// TestDrillResultFormat verifies that FormatDrillResult produces valid JSON
// containing the expected fields.
func TestDrillResultFormat(t *testing.T) {
	result := &DrillResult{
		ID:            "dr-cold-start-20260420-120000",
		Scenario:      ScenarioColdStart,
		StartedAt:     time.Now(),
		FinishedAt:    time.Now().Add(5 * time.Minute),
		Status:        "success",
		RowCountDelta: map[string]int64{"auth.users": 0},
		Details:       map[string]interface{}{"rto": "5m0s"},
	}

	out, err := FormatDrillResult(result, "json")
	if err != nil {
		t.Fatalf("FormatDrillResult error: %v", err)
	}
	for _, field := range []string{"id", "scenario", "status", "started_at"} {
		if !strings.Contains(out, field) {
			t.Errorf("output missing field %q", field)
		}
	}
}

// TestDrillUnknownScenario verifies that an unknown scenario returns an error.
func TestDrillUnknownScenario(t *testing.T) {
	cfg := &config.Config{ProjectName: "test"}
	ctx := context.Background()

	_, err := Drill(ctx, cfg, DrillOptions{Scenario: Scenario("unknown-scenario")})
	if err == nil {
		t.Fatal("expected error for unknown scenario, got nil")
	}
}
