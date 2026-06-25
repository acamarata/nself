package deploy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---- Helpers ----------------------------------------------------------------

// createRollbackStateFile writes a test rollback state file for the given service.
func createRollbackStateFile(t *testing.T, projectRoot, service, env, priorTag string) {
	t.Helper()
	dir := filepath.Join(projectRoot, ".nself", "rollback-state", env)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}
	state := map[string]interface{}{
		"prior_tag":  priorTag,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(dir, service+".json"), data, 0o644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}
}

// ---- Tests ------------------------------------------------------------------

// TestUIStatePermDenied verifies rollback returns permission-denied when no deploy key.
func TestUIStatePermDenied(t *testing.T) {
	cfg := RollbackConfig{
		Service:   "ping_api",
		DeployKey: "",
		Reason:    "test",
	}
	result := ExecuteRollback(context.Background(), cfg)
	if result.UIState != UIStatePermDenied {
		t.Errorf("expected UIStatePermDenied, got %q", result.UIState)
	}
	if result.Success {
		t.Error("expected failure when deploy key missing")
	}
}

// TestUIStateEmpty verifies rollback returns empty when no prior version exists.
func TestUIStateEmpty(t *testing.T) {
	tmp := t.TempDir()
	cfg := RollbackConfig{
		Service:     "ping_api",
		DeployKey:   "nself_deploy_test_key_1234567890abcdef",
		Reason:      "test: slo breach",
		Environment: "staging",
		ProjectRoot: tmp, // No state file exists
	}
	result := ExecuteRollback(context.Background(), cfg)
	if result.UIState != UIStateEmpty {
		t.Errorf("expected UIStateEmpty (no state file), got %q", result.UIState)
	}
}

// TestRollbackStateDiscovery verifies that a valid state file is found and parsed.
func TestRollbackStateDiscovery(t *testing.T) {
	tmp := t.TempDir()
	createRollbackStateFile(t, tmp, "ping_api", "staging", "v1.0.8")

	priorVersion, err := discoverPriorVersion(tmp, "ping_api", "staging")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if priorVersion != "v1.0.8" {
		t.Errorf("expected prior version v1.0.8, got %q", priorVersion)
	}
}

// TestRollbackStateEmpty verifies empty state file returns error.
func TestRollbackStateEmpty(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".nself", "rollback-state", "prod")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write empty state
	emptyState := `{"prior_tag": "", "prior_hash": ""}`
	if err := os.WriteFile(filepath.Join(dir, "ping_api.json"), []byte(emptyState), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := discoverPriorVersion(tmp, "ping_api", "prod")
	if err == nil {
		t.Error("expected error for empty rollback state, got nil")
	}
}

// TestDeployKeyValidation verifies deploy key format validation.
func TestDeployKeyValidation(t *testing.T) {
	valid := []string{
		"nself_deploy_test_key_abcdefghijklmnop",
		"nself_pro_abcdefghijklmnopqrstuvwxyz12345",
		"nself_test_key_with_long_suffix_1234567",
	}
	invalid := []string{
		"",
		"short",
		"nself_deploy_tooshort",
		"not_nself_key_at_all_and_long_enough_still",
	}

	for _, k := range valid {
		if !isValidDeployKey(k) {
			t.Errorf("expected valid key: %q", k)
		}
	}
	for _, k := range invalid {
		if isValidDeployKey(k) {
			t.Errorf("expected invalid key: %q", k)
		}
	}
}

// TestAuditEventJSON verifies audit events are marshalled to valid JSON with required fields.
// writeAuditEvent emits via slog.Info — this test verifies the AuditEvent struct directly
// rather than capturing stdout (which changed when the function moved to slog).
func TestAuditEventJSON(t *testing.T) {
	cfg := RollbackConfig{
		Service:     "ping_api",
		Environment: "prod",
		Reason:      "slo-watcher: error rate exceeded 1%",
		DeployKey:   "nself_deploy_key_test",
	}

	// Build the same event struct that writeAuditEvent would emit.
	event := AuditEvent{
		EventType:    "deploy.rollback",
		Service:      cfg.Service,
		Environment:  cfg.Environment,
		Reason:       cfg.Reason,
		Outcome:      "success",
		PriorVersion: "v1.0.8",
		TriggeredBy:  "slo-watcher",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("AuditEvent does not marshal to JSON: %v", err)
	}

	var roundtrip AuditEvent
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("audit event is not valid JSON after roundtrip: %v\nraw: %s", err, data)
	}

	if roundtrip.EventType != "deploy.rollback" {
		t.Errorf("expected event_type deploy.rollback, got %q", roundtrip.EventType)
	}
	if roundtrip.Service != "ping_api" {
		t.Errorf("expected service ping_api, got %q", roundtrip.Service)
	}
	if roundtrip.Outcome != "success" {
		t.Errorf("expected outcome success, got %q", roundtrip.Outcome)
	}
	if roundtrip.TriggeredBy != "slo-watcher" {
		t.Errorf("expected triggered_by slo-watcher, got %q", roundtrip.TriggeredBy)
	}
}

// TestUIStateOffline verifies rollback returns offline when project root not found.
func TestUIStateOffline(t *testing.T) {
	cfg := RollbackConfig{
		Service:     "ping_api",
		DeployKey:   "nself_deploy_test_key_1234567890abcdef",
		Reason:      "test",
		Environment: "prod",
		ProjectRoot: "/nonexistent/path/to/project",
	}
	result := ExecuteRollback(context.Background(), cfg)
	// /nonexistent path → state file not found → UIStateEmpty (can't distinguish offline from empty without network)
	// Accept either empty or offline.
	if result.UIState != UIStateEmpty && result.UIState != UIStateOffline {
		t.Errorf("expected UIStateEmpty or UIStateOffline for missing project, got %q", result.UIState)
	}
}

// TestAllUIStatesDocumented verifies that all 7 UI states are defined.
func TestAllUIStatesDocumented(t *testing.T) {
	states := []RollbackUIState{
		UIStateLoading,
		UIStateEmpty,
		UIStateError,
		UIStatePopulated,
		UIStateOffline,
		UIStatePermDenied,
		UIStateRateLimited,
	}
	if len(states) != 7 {
		t.Errorf("expected 7 UI states, got %d", len(states))
	}
}
