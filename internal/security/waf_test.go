package security

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

// captureHandler is a slog.Handler that records emitted log records for
// assertion in tests. It is safe for use within a single goroutine.
type captureHandler struct {
	records []slog.Record
	level   slog.Level
}

func (h *captureHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(name string) slog.Handler       { return h }

// attrMap extracts all key-value pairs from a slog.Record into a flat map.
func attrMap(r slog.Record) map[string]string {
	m := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.String()
		return true
	})
	return m
}

func TestParseWAFEvent_ValidLine(t *testing.T) {
	line := `[27/Apr/2026:12:34:56 +0000] client=192.168.1.1 path=/api/v1/users rule=942100 severity=CRITICAL action=blocked`

	ev, ok := ParseWAFEvent(line)
	if !ok {
		t.Fatal("expected ParseWAFEvent to return ok=true for valid line")
	}

	if ev.ClientIP != "192.168.1.1" {
		t.Errorf("client_ip: got %q, want %q", ev.ClientIP, "192.168.1.1")
	}
	if ev.Path != "/api/v1/users" {
		t.Errorf("path: got %q, want %q", ev.Path, "/api/v1/users")
	}
	if ev.Rule != "942100" {
		t.Errorf("rule: got %q, want %q", ev.Rule, "942100")
	}
	if ev.Severity != "CRITICAL" {
		t.Errorf("severity: got %q, want %q", ev.Severity, "CRITICAL")
	}
	if ev.Action != "blocked" {
		t.Errorf("action: got %q, want %q", ev.Action, "blocked")
	}
}

func TestParseWAFEvent_EmptyLine(t *testing.T) {
	_, ok := ParseWAFEvent("")
	if ok {
		t.Error("expected ParseWAFEvent to return ok=false for empty line")
	}
}

func TestParseWAFEvent_CommentLine(t *testing.T) {
	_, ok := ParseWAFEvent("# this is a comment")
	if ok {
		t.Error("expected ParseWAFEvent to return ok=false for comment line")
	}
}

func TestParseWAFEvent_NoClientOrPath(t *testing.T) {
	// A line with neither client nor path should be rejected.
	line := `[27/Apr/2026:12:34:56 +0000] rule=942100 severity=WARNING`
	_, ok := ParseWAFEvent(line)
	if ok {
		t.Error("expected ParseWAFEvent to return ok=false when client and path are both missing")
	}
}

func TestParseWAFEvent_PathOnly(t *testing.T) {
	line := `[27/Apr/2026:12:34:56 +0000] path=/admin rule=920350 severity=ERROR action=detected`
	ev, ok := ParseWAFEvent(line)
	if !ok {
		t.Fatal("expected ParseWAFEvent to return ok=true when at least path is present")
	}
	if ev.Path != "/admin" {
		t.Errorf("path: got %q, want /admin", ev.Path)
	}
}

func TestLogWAFBlock_EmitsStructuredFields(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	ctx := context.Background()
	LogWAFBlock(ctx, "10.0.0.1", "/etc/passwd", "930110")

	if len(h.records) != 1 {
		t.Fatalf("expected 1 log record, got %d", len(h.records))
	}

	r := h.records[0]
	if r.Message != "waf_blocked" {
		t.Errorf("message: got %q, want %q", r.Message, "waf_blocked")
	}
	if r.Level != slog.LevelWarn {
		t.Errorf("level: got %v, want Warn", r.Level)
	}

	attrs := attrMap(r)
	if attrs["client_ip"] != "10.0.0.1" {
		t.Errorf("client_ip: got %q, want %q", attrs["client_ip"], "10.0.0.1")
	}
	if attrs["path"] != "/etc/passwd" {
		t.Errorf("path: got %q, want %q", attrs["path"], "/etc/passwd")
	}
	if attrs["rule"] != "930110" {
		t.Errorf("rule: got %q, want %q", attrs["rule"], "930110")
	}
}

func TestLogWAFDetect_EmitsInfoLevel(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	LogWAFDetect(context.Background(), "172.16.0.5", "/login", "941100")

	if len(h.records) != 1 {
		t.Fatalf("expected 1 log record, got %d", len(h.records))
	}
	r := h.records[0]
	if r.Message != "waf_detected" {
		t.Errorf("message: got %q, want %q", r.Message, "waf_detected")
	}
	if r.Level != slog.LevelInfo {
		t.Errorf("level: got %v, want Info", r.Level)
	}

	attrs := attrMap(r)
	if attrs["client_ip"] != "172.16.0.5" {
		t.Errorf("client_ip: got %q", attrs["client_ip"])
	}
}

func TestLogWAFModeChange_EmitsOldAndNewMode(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	LogWAFModeChange(context.Background(), WAFModeDetection, WAFModeBlocking)

	if len(h.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(h.records))
	}
	attrs := attrMap(h.records[0])
	if attrs["old_mode"] != "detection" {
		t.Errorf("old_mode: got %q, want detection", attrs["old_mode"])
	}
	if attrs["new_mode"] != "blocking" {
		t.Errorf("new_mode: got %q, want blocking", attrs["new_mode"])
	}
}

func TestScanWAFAuditLog_ParsesMultipleEvents(t *testing.T) {
	log := strings.Join([]string{
		`[27/Apr/2026:10:00:00 +0000] client=1.2.3.4 path=/api/data rule=920350 severity=WARNING action=detected`,
		`# comment line`,
		``,
		`[27/Apr/2026:10:01:00 +0000] client=5.6.7.8 path=/etc/shadow rule=930120 severity=CRITICAL action=blocked`,
	}, "\n")

	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	events, unparseable, err := ScanWAFAuditLog(context.Background(), bytes.NewBufferString(log))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	if unparseable != 0 {
		t.Errorf("expected 0 unparseable lines, got %d", unparseable)
	}

	// Verify structured fields on the blocked event.
	blocked := events[1]
	if blocked.Action != "blocked" {
		t.Errorf("second event action: got %q, want blocked", blocked.Action)
	}
	if blocked.Rule != "930120" {
		t.Errorf("second event rule: got %q, want 930120", blocked.Rule)
	}

	// Verify that a waf_event record was emitted for each parsed line.
	wafEventCount := 0
	for _, r := range h.records {
		if r.Message == "waf_event" {
			wafEventCount++
		}
	}
	if wafEventCount != 2 {
		t.Errorf("expected 2 waf_event records, got %d", wafEventCount)
	}
}

func TestScanWAFAuditLog_CountsUnparseableLines(t *testing.T) {
	// Mix one valid event with one malformed line (no client/path).
	log := strings.Join([]string{
		`[27/Apr/2026:10:00:00 +0000] client=1.2.3.4 path=/login rule=941100 severity=ERROR action=detected`,
		`this is not a waf log line at all and has no key=value pairs that match`,
	}, "\n")

	orig := slog.Default()
	slog.SetDefault(slog.New(&captureHandler{level: slog.LevelDebug}))
	t.Cleanup(func() { slog.SetDefault(orig) })

	events, unparseable, err := ScanWAFAuditLog(context.Background(), bytes.NewBufferString(log))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if unparseable != 1 {
		t.Errorf("expected 1 unparseable line, got %d", unparseable)
	}
}
