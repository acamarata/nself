package plugin

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestAuditEventJSONShape — ensures audit event serializes to expected shape.
func TestAuditEventJSONShape(t *testing.T) {
	ev := AuditEvent{
		Op:         AuditOpInstall,
		PluginName: "ai",
		Actor:      "cli:test",
		Version:    "1.0.0",
		Source:     "pro",
		Timestamp:  time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(raw)
	for _, want := range []string{
		`"op":"plugin.install"`,
		`"plugin":"ai"`,
		`"actor":"cli:test"`,
		`"version":"1.0.0"`,
		`"source":"pro"`,
		`"timestamp":"2026-05-04T12:00:00Z"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

// TestAuditOpConstants — guards canonical operation strings.
func TestAuditOpConstants(t *testing.T) {
	cases := map[AuditOp]string{
		AuditOpInstall: "plugin.install",
		AuditOpRemove:  "plugin.remove",
		AuditOpUpdate:  "plugin.update",
		AuditOpDisable: "plugin.disable",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("AuditOp constant drift: got %q want %q", got, want)
		}
	}
}

// TestSQLEscape — verifies single-quote handling.
func TestSQLEscape(t *testing.T) {
	if got := sqlEscape("ai's"); got != "ai''s" {
		t.Errorf("sqlEscape: got %q want %q", got, "ai''s")
	}
	if got := sqlEscape("noquote"); got != "noquote" {
		t.Errorf("sqlEscape: got %q want %q", got, "noquote")
	}
}

// TestCurrentActorFormat — actor returns cli:<user> when USER set.
func TestCurrentActorFormat(t *testing.T) {
	t.Setenv("USER", "alice")
	if got := currentActor(); got != "cli:alice" {
		t.Errorf("currentActor: got %q want cli:alice", got)
	}
}

// TestLogPluginInstallNoDBContext — when cfg is nil/empty, audit is silently skipped (no panic).
func TestLogPluginInstallNoDBContext(t *testing.T) {
	// Should not panic when called with nil config.
	LogPluginInstall(nil, nil, "test", "1.0.0", "free") //nolint:staticcheck // nil ctx intentional
	// Sleep briefly to allow the goroutine to run + return.
	time.Sleep(50 * time.Millisecond)
}
