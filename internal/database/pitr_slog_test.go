package database

import (
	"context"
	"log/slog"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// captureHandler records slog.Records for assertion in tests.
// Matches the pattern established in cli/internal/security/waf_test.go (G6-T01).
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
func (h *captureHandler) WithGroup(name string) slog.Handler        { return h }

// attrMap extracts key-value pairs from a slog.Record.
func attrMap(r slog.Record) map[string]string {
	m := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.String()
		return true
	})
	return m
}

// TestWritePITRConfig_EmitsStructuredLog verifies WritePITRConfig emits a
// slog record with the expected structured fields on success.
func TestWritePITRConfig_EmitsStructuredLog(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	dir := t.TempDir()
	cfg := &config.Config{}
	pitrCfg := PITRConfig{ArchiveDir: "/var/lib/postgresql/wal_archive"}

	err := WritePITRConfig(cfg, dir, pitrCfg)
	if err != nil {
		t.Fatalf("WritePITRConfig: %v", err)
	}

	if len(h.records) == 0 {
		t.Fatal("expected at least one slog record, got none")
	}

	// Find the success record.
	var found bool
	for _, r := range h.records {
		if r.Message == "pitr_config_written" {
			found = true
			attrs := attrMap(r)
			if attrs["backup_type"] != "wal_archive" {
				t.Errorf("backup_type: got %q, want wal_archive", attrs["backup_type"])
			}
			if attrs["path"] == "" {
				t.Error("path field must not be empty")
			}
			break
		}
	}
	if !found {
		t.Errorf("expected slog record with message 'pitr_config_written', got messages: %v", recordMessages(h.records))
	}
}

// TestWritePITRConfig_StructuredFieldsPresent verifies that backup_type and path
// fields are present and meaningful in the emitted log record.
func TestWritePITRConfig_StructuredFieldsPresent(t *testing.T) {
	h := &captureHandler{level: slog.LevelInfo}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	dir := t.TempDir()
	cfg := &config.Config{}
	pitrCfg := PITRConfig{ArchiveDir: "/custom/wal/dir"}

	if err := WritePITRConfig(cfg, dir, pitrCfg); err != nil {
		t.Fatalf("WritePITRConfig: %v", err)
	}

	for _, r := range h.records {
		if r.Message == "pitr_config_written" {
			attrs := attrMap(r)
			if _, ok := attrs["path"]; !ok {
				t.Error("expected 'path' field in pitr_config_written record")
			}
			if _, ok := attrs["backup_type"]; !ok {
				t.Error("expected 'backup_type' field in pitr_config_written record")
			}
			return
		}
	}
	t.Error("pitr_config_written record not found")
}

// TestPITRRestore_EmitsStartLog verifies that PITRRestore emits a start log
// record with the target timestamp before attempting docker operations.
func TestPITRRestore_EmitsStartLog(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	cfg := &config.Config{}
	targetTime := "2026-01-15T14:30:00Z"

	// PITRRestore will fail (no Docker in unit test env) but the start log
	// must be emitted before the first docker call.
	_ = PITRRestore(context.Background(), cfg, targetTime)

	var found bool
	for _, r := range h.records {
		if r.Message == "pitr_restore_started" {
			found = true
			attrs := attrMap(r)
			if attrs["timestamp"] != targetTime {
				t.Errorf("timestamp: got %q, want %q", attrs["timestamp"], targetTime)
			}
			if attrs["backup_type"] != "pitr" {
				t.Errorf("backup_type: got %q, want pitr", attrs["backup_type"])
			}
			break
		}
	}
	if !found {
		t.Errorf("expected pitr_restore_started log record; got: %v", recordMessages(h.records))
	}
}

// TestPITRRestore_InvalidTargetTime_NoLog verifies that validation errors
// return early without emitting any slog records.
func TestPITRRestore_InvalidTargetTime_NoLog(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	cfg := &config.Config{}
	err := PITRRestore(context.Background(), cfg, "not-a-valid-time")
	if err == nil {
		t.Fatal("expected error for invalid targetTime")
	}
	if len(h.records) != 0 {
		t.Errorf("expected no slog records for validation failure, got %d", len(h.records))
	}
}

// TestPITRRestore_EmptyTargetTime_NoLog verifies empty targetTime returns error
// without emitting any log records.
func TestPITRRestore_EmptyTargetTime_NoLog(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	cfg := &config.Config{}
	err := PITRRestore(context.Background(), cfg, "")
	if err == nil {
		t.Fatal("expected error for empty targetTime")
	}
	if len(h.records) != 0 {
		t.Errorf("expected no slog records for empty targetTime, got %d", len(h.records))
	}
}

// recordMessages is a test helper that extracts messages from a slice of records.
func recordMessages(rs []slog.Record) []string {
	msgs := make([]string, len(rs))
	for i, r := range rs {
		msgs[i] = r.Message
	}
	return msgs
}
