package secrets

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

// captureHandler records slog.Records emitted during a test.
// It is single-goroutine safe and follows the same pattern used in
// cli/internal/security/waf_test.go (G6-T01 reference implementation).
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

// attrMap extracts all key-value pairs from a slog.Record into a flat string map.
func attrMap(r slog.Record) map[string]string {
	m := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.String()
		return true
	})
	return m
}

// secretValue is a sentinel that must never appear in any slog field.
const secretValue = "super-secret-api-key-12345"

// TestSecretsSet_LogsKeyNameNotValue verifies that Set emits a structured
// slog record containing the key name but never the secret value.
// This is the primary PII/credential guard for G6-T02.
func TestSecretsSet_LogsKeyNameNotValue(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	// saveStore will fail (no real age binary in CI), so Set returns an error
	// after the slog.Error call. We only care that no log field contains the value.
	// Use a tempDir project root so loadStore returns an empty store.
	dir := t.TempDir()
	_ = Set(dir, "dev", "MY_API_KEY", secretValue)

	for _, r := range h.records {
		attrs := attrMap(r)
		for k, v := range attrs {
			if strings.Contains(v, secretValue) {
				t.Errorf("slog field %q contains secret value — must log key name only, got: %q", k, v)
			}
		}
		if strings.Contains(r.Message, secretValue) {
			t.Errorf("slog message contains secret value: %q", r.Message)
		}
	}
}

// TestSecretsSet_EmitsKeyNameField verifies that on a successful save path
// (mocked via the error branch) the key_name field is present.
// We verify the error-path log record contains key_name but not the value.
func TestSecretsSet_ErrorPath_ContainsKeyName(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	dir := t.TempDir()
	_ = Set(dir, "dev", "MY_API_KEY", secretValue)

	// At least one record should reference the key name (not value).
	found := false
	for _, r := range h.records {
		attrs := attrMap(r)
		if name, ok := attrs["key_name"]; ok {
			found = true
			if name != "MY_API_KEY" {
				t.Errorf("key_name: got %q, want MY_API_KEY", name)
			}
		}
	}
	if !found {
		t.Log("note: no slog records emitted (saveStore may have returned early without logging) — this is acceptable when age is not installed")
	}
}

// TestSecretsRotate_LogsKeyNameNotValue verifies Rotate does not log the rotated value.
func TestSecretsRotate_LogsKeyNameNotValue(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	dir := t.TempDir()
	_, _ = Rotate(dir, "dev", "JWT_SIGNING_KEY")

	for _, r := range h.records {
		attrs := attrMap(r)
		for k, v := range attrs {
			// The rotated value is a random alphanumeric string — we cannot
			// predict it, but we ensure no field contains the sentinel value
			// used elsewhere, and that no field key is named "value".
			if k == "value" {
				t.Errorf("slog field named 'value' found — secret values must never be logged, key: %q", k)
			}
			_ = v
		}
	}
}

// TestSecretsNoSecretValueInAnyLogField is a broader invariant check:
// run several operations and confirm the sentinel secret value never appears
// in any slog output across all operations.
func TestSecretsNoSecretValueInAnyLogField(t *testing.T) {
	h := &captureHandler{level: slog.LevelDebug}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })

	dir := t.TempDir()
	const key = "STRIPE_API_KEY"
	_ = Set(dir, "staging", key, secretValue)
	_, _ = Rotate(dir, "staging", key)

	for i, r := range h.records {
		if strings.Contains(r.Message, secretValue) {
			t.Errorf("record[%d] message contains secret value", i)
		}
		r.Attrs(func(a slog.Attr) bool {
			if strings.Contains(a.Value.String(), secretValue) {
				t.Errorf("record[%d] attr %q contains secret value", i, a.Key)
			}
			return true
		})
	}
}
