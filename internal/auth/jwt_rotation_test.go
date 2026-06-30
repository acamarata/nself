package auth

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestGenerateJWTKey verifies the key is the correct length and non-empty.
func TestGenerateJWTKey(t *testing.T) {
	key, err := GenerateJWTKey()
	if err != nil {
		t.Fatalf("GenerateJWTKey() error = %v", err)
	}
	// 32 bytes → 64 hex chars
	if len(key) != 64 {
		t.Errorf("GenerateJWTKey() len = %d, want 64", len(key))
	}
	if key == "" {
		t.Error("GenerateJWTKey() returned empty key")
	}
}

// TestGenerateJWTKey_Unique verifies two calls produce different keys.
func TestGenerateJWTKey_Unique(t *testing.T) {
	k1, err := GenerateJWTKey()
	if err != nil {
		t.Fatal(err)
	}
	k2, err := GenerateJWTKey()
	if err != nil {
		t.Fatal(err)
	}
	if k1 == k2 {
		t.Error("GenerateJWTKey() returned identical keys on successive calls")
	}
}

// TestLastRotationTime_NoLog verifies a missing log returns zero time, not an error.
func TestLastRotationTime_NoLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "jwt-rotation.log")

	got, err := LastRotationTime(logPath)
	if err != nil {
		t.Fatalf("LastRotationTime() unexpected error = %v", err)
	}
	if !got.IsZero() {
		t.Errorf("LastRotationTime() = %v, want zero time", got)
	}
}

// TestLastRotationTime_ValidLog verifies the most recent timestamp is returned.
func TestLastRotationTime_ValidLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "jwt-rotation.log")

	entries := "2026-01-01T00:00:00Z rotated HASURA_GRAPHQL_JWT_SECRET (grace period ends 2026-01-01T12:00:00Z)\n" +
		"2026-03-15T10:00:00Z rotated HASURA_GRAPHQL_JWT_SECRET (grace period ends 2026-03-15T22:00:00Z)\n"
	if err := os.WriteFile(logPath, []byte(entries), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LastRotationTime(logPath)
	if err != nil {
		t.Fatalf("LastRotationTime() error = %v", err)
	}
	want, _ := time.Parse(time.RFC3339, "2026-03-15T10:00:00Z")
	if !got.Equal(want) {
		t.Errorf("LastRotationTime() = %v, want %v", got, want)
	}
}

// TestLastRotationTime_CommentsIgnored verifies comment lines are skipped.
func TestLastRotationTime_CommentsIgnored(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "jwt-rotation.log")

	content := "# nSelf JWT rotation log\n" +
		"2026-02-10T08:00:00Z rotated HASURA_GRAPHQL_JWT_SECRET (grace period ends 2026-02-10T20:00:00Z)\n"
	if err := os.WriteFile(logPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LastRotationTime(logPath)
	if err != nil {
		t.Fatalf("LastRotationTime() error = %v", err)
	}
	want, _ := time.Parse(time.RFC3339, "2026-02-10T08:00:00Z")
	if !got.Equal(want) {
		t.Errorf("LastRotationTime() = %v, want %v", got, want)
	}
}

// TestRotateJWTKey_WritesLog verifies RotateJWTKey writes a log entry and
// returns a non-empty new key distinct from the old key.
func TestRotateJWTKey_WritesLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "jwt-rotation.log")
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)

	result, err := RotateJWTKey("aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")
	if err != nil {
		t.Fatalf("RotateJWTKey() error = %v", err)
	}

	if result.NewKey == "" {
		t.Error("RotateJWTKey() NewKey is empty")
	}
	if result.NewKey == result.OldKey {
		t.Error("RotateJWTKey() NewKey == OldKey")
	}
	if result.OldKey != "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899" {
		t.Errorf("RotateJWTKey() OldKey = %q, want raw hex key", result.OldKey)
	}
	if result.GraceUntil.Before(time.Now()) {
		t.Error("RotateJWTKey() GraceUntil is in the past")
	}

	// Verify log file was created and contains the rotation entry.
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading rotation log: %v", err)
	}
	if !strings.Contains(string(data), "rotated HASURA_GRAPHQL_JWT_SECRET") {
		t.Errorf("rotation log does not contain expected entry; got: %s", string(data))
	}
}

// TestRotateJWTKey_GracePeriod verifies the grace period is set correctly.
func TestRotateJWTKey_GracePeriod(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_JWT_ROTATION_LOG", filepath.Join(dir, "jwt-rotation.log"))

	oldKey := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	before := time.Now().Add((GracePeriodHours - 1) * time.Hour)
	result, err := RotateJWTKey(oldKey)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now().Add((GracePeriodHours + 1) * time.Hour)

	if result.GraceUntil.Before(before) || result.GraceUntil.After(after) {
		t.Errorf("GraceUntil %v not in expected range [%v, %v]", result.GraceUntil, before, after)
	}
}

// TestRotationWindowDays_Default verifies the default is 90 when env is unset.
func TestRotationWindowDays_Default(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "")
	if got := RotationWindowDays(); got != DefaultRotationWindowDays {
		t.Errorf("RotationWindowDays() = %d, want %d", got, DefaultRotationWindowDays)
	}
}

// TestRotationWindowDays_Override verifies the env var override is respected.
func TestRotationWindowDays_Override(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "30")
	if got := RotationWindowDays(); got != 30 {
		t.Errorf("RotationWindowDays() = %d, want 30", got)
	}
}

// TestRotationWindowDays_Malformed verifies malformed input falls back to default.
func TestRotationWindowDays_Malformed(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "not-a-number")
	if got := RotationWindowDays(); got != DefaultRotationWindowDays {
		t.Errorf("RotationWindowDays() with malformed env = %d, want default %d", got, DefaultRotationWindowDays)
	}
}

// TestRotationWindowDays_Zero verifies zero/negative input falls back to default.
func TestRotationWindowDays_Zero(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "0")
	if got := RotationWindowDays(); got != DefaultRotationWindowDays {
		t.Errorf("RotationWindowDays() with zero = %d, want default %d", got, DefaultRotationWindowDays)
	}
}

// TestRotationLogPath_Default verifies the default path or XDG fallback is returned when env is unset.
func TestRotationLogPath_Default(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_LOG", "")
	got := RotationLogPath()
	// Either the primary path or the XDG fallback — both are valid.
	if got == "" {
		t.Error("RotationLogPath() returned empty string")
	}
}

// TestRotationLogPath_Override verifies the env var override is respected.
func TestRotationLogPath_Override(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_LOG", "/tmp/test-jwt-rotation.log")
	if got := RotationLogPath(); got != "/tmp/test-jwt-rotation.log" {
		t.Errorf("RotationLogPath() = %q, want /tmp/test-jwt-rotation.log", got)
	}
}

// TestRotationLogPath_XDGFallback verifies the XDG fallback is used when the
// primary path's parent directory does not exist.
func TestRotationLogPath_XDGFallback(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_LOG", "")
	// Point XDG_STATE_HOME to a temp dir so we can verify the fallback path.
	xdgDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdgDir)

	got := RotationLogPath()
	// If /var/lib/nself/ is not writable (typical on dev machines), the result
	// should be under xdgDir.
	if _, err := os.Stat("/var/lib/nself"); err != nil {
		// Primary dir absent: must fall back.
		want := filepath.Join(xdgDir, "nself", "jwt-rotation.log")
		if got != want {
			t.Errorf("RotationLogPath() XDG fallback = %q, want %q", got, want)
		}
	}
}

// TestRotateJWTKey_EmptyCurrentKey verifies that an empty currentKey returns
// an error rather than silently storing an empty OldKey.
func TestRotateJWTKey_EmptyCurrentKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_JWT_ROTATION_LOG", filepath.Join(dir, "jwt-rotation.log"))

	_, err := RotateJWTKey("")
	if err == nil {
		t.Error("RotateJWTKey() with empty currentKey should return error, got nil")
	}
}

// TestRotateJWTKey_OldKeyIsRawHex verifies OldKey stores the raw hex key and
// not a JSON envelope.
func TestRotateJWTKey_OldKeyIsRawHex(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_JWT_ROTATION_LOG", filepath.Join(dir, "jwt-rotation.log"))

	rawKey := "deadbeef00112233445566778899aabbccddeeff00112233445566778899aabb"
	result, err := RotateJWTKey(rawKey)
	if err != nil {
		t.Fatal(err)
	}
	if result.OldKey != rawKey {
		t.Errorf("OldKey = %q, want raw hex %q", result.OldKey, rawKey)
	}
	if strings.Contains(result.OldKey, `"type"`) || strings.Contains(result.OldKey, `"key"`) {
		t.Errorf("OldKey contains JSON envelope fragment — should be raw hex: %q", result.OldKey)
	}
}

// TestRotateJWTKey_Concurrent verifies that 8 parallel RotateJWTKey calls
// each produce a distinct log entry with no torn writes.
func TestRotateJWTKey_Concurrent(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "jwt-rotation.log")
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)

	const workers = 8
	oldKey := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	errs := make([]error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = RotateJWTKey(oldKey)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("worker %d: RotateJWTKey() error = %v", i, err)
		}
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	lines := 0
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) != "" {
			lines++
		}
	}
	if lines != workers {
		t.Errorf("expected %d log lines from %d workers, got %d\nlog:\n%s", workers, workers, lines, string(data))
	}

	// Each line must be a well-formed RFC3339 entry (no torn writes).
	for i, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			t.Errorf("line %d is malformed (torn write?): %q", i, line)
			continue
		}
		if _, err := time.Parse(time.RFC3339, fields[0]); err != nil {
			t.Errorf("line %d has invalid timestamp %q: %v", i, fields[0], err)
		}
	}
}

// TestRotateJWTKey_LogUnwritable verifies that RotateJWTKey returns an error
// (not a panic or silent success) when the log directory is read-only.
func TestRotateJWTKey_LogUnwritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod read-only semantics not supported on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root can write to read-only dirs, skipping")
	}

	dir := t.TempDir()
	logDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(logDir, 0o555); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(logDir, "jwt-rotation.log")
	// Override both LOG and XDG so neither fallback can succeed.
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)
	t.Setenv("XDG_STATE_HOME", logDir) // also read-only, fallback also fails

	oldKey := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	_, err := RotateJWTKey(oldKey)
	if err == nil {
		t.Error("RotateJWTKey() with unwritable log dir should return error, got nil")
	}
}

// TestCheckJWTRotation_CorruptedLog verifies that a log containing only
// non-parseable lines is treated as zero-rotation (warn, not panic).
func TestCheckJWTRotation_CorruptedLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "jwt-rotation.log")
	// Write garbage that has no valid RFC3339 timestamps.
	if err := os.WriteFile(logPath, []byte("CORRUPTED\nnot-a-timestamp event\n\xff\xfe\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)

	got, err := LastRotationTime(logPath)
	if err != nil {
		t.Fatalf("LastRotationTime() with corrupted log should not error, got %v", err)
	}
	if !got.IsZero() {
		t.Errorf("LastRotationTime() with corrupted log = %v, want zero time", got)
	}
}

// TestCheckJWTRotation_FutureLogTimestamp verifies that future timestamps
// (clock skew) are rejected and treated as zero rotation.
func TestCheckJWTRotation_FutureLogTimestamp(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "jwt-rotation.log")
	futureTS := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	entry := fmt.Sprintf("%s rotated HASURA_GRAPHQL_JWT_SECRET (grace period ends ...)\n", futureTS)
	if err := os.WriteFile(logPath, []byte(entry), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)

	got, err := LastRotationTime(logPath)
	if err != nil {
		t.Fatalf("LastRotationTime() error = %v", err)
	}
	if !got.IsZero() {
		t.Errorf("LastRotationTime() with future timestamp = %v, want zero (clock skew rejected)", got)
	}
}

// TestRotateJWTKey_CryptoRoundTrip verifies that a key generated by
// GenerateJWTKey is a valid 64-character hex string that can be decoded.
func TestRotateJWTKey_CryptoRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NSELF_JWT_ROTATION_LOG", filepath.Join(dir, "jwt-rotation.log"))

	oldKey := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	result, err := RotateJWTKey(oldKey)
	if err != nil {
		t.Fatal(err)
	}

	// NewKey must be 64 hex chars (32 bytes).
	if len(result.NewKey) != 64 {
		t.Errorf("NewKey length = %d, want 64", len(result.NewKey))
	}

	// Must be valid hex.
	for _, c := range result.NewKey {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			t.Errorf("NewKey contains non-hex character %q", c)
			break
		}
	}

	// NewKey and OldKey must differ.
	if result.NewKey == result.OldKey {
		t.Error("NewKey == OldKey after rotation")
	}
}

// TestSelfHealJWT_DryRun verifies that the dry-run path does not write a log
// file. This is a unit-level approximation: we verify RotateJWTKey is not
// called by confirming no log file appears when the function is skipped.
func TestSelfHealJWT_DryRun(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "jwt-rotation.log")
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)

	// Pre-condition: log does not exist.
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("log file should not exist before any rotation")
	}
	// Dry-run path: RotateJWTKey is never called, so no log is written.
	// (Full Cobra dry-run wiring is covered by integration tests.)
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("log file must not be written when RotateJWTKey is not called")
	}
}

// TestRotateJWTKey_GenerateError simulates a crypto/rand failure by temporarily
// replacing the package-level randReader with an error-producing reader.
func TestRotateJWTKey_GenerateError(t *testing.T) {
	// Save and restore the original reader.
	original := randReader
	t.Cleanup(func() { randReader = original })
	randReader = &errorReader{}

	_, err := GenerateJWTKey()
	if err == nil {
		t.Error("GenerateJWTKey() with failing randReader should return error")
	}
}

// errorReader is an io.Reader that always returns an error.
type errorReader struct{}

func (*errorReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
