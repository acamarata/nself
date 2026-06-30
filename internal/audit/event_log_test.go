//go:build !windows

package audit

// Purpose: Tests for security event log writer (Write function).
// Inputs:  Temp directories, fake fields map.
// Outputs: Verifies JSON structure, file permissions, and append behaviour.
// SPORT: security-event-log; ticket P2-E2-W2-S3-T10

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestWrite_CreatesFileWith0600 verifies that Write creates the audit log file
// with 0600 permissions when it does not exist, and that the entry is valid JSON
// containing the required fields: event, timestamp, and all caller-supplied fields.
func TestWrite_CreatesFileWith0600(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := Write("plugin-install-bypass", map[string]string{
		"plugin": "test-plugin",
		"reason": "NSELF_LICENSE_SKIP_VERIFY=1 + NSELF_LICENSE_SKIP_VERIFY_FORCE=1",
		"uid":    "alice",
		"env":    "dev",
	})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	logPath := filepath.Join(home, ".nself", "audit.log")
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("audit.log not created: %v", err)
	}
	// Verify 0600 permissions.
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}

	// Parse the JSON line.
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("open audit.log: %v", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		t.Fatal("audit.log is empty")
	}
	line := sc.Bytes()

	var record map[string]string
	if err := json.Unmarshal(line, &record); err != nil {
		t.Fatalf("audit.log line is not valid JSON: %v — line: %s", err, line)
	}

	for _, key := range []string{"event", "timestamp", "plugin", "reason", "uid", "env"} {
		if _, ok := record[key]; !ok {
			t.Errorf("audit record missing required key %q; got: %v", key, record)
		}
	}
	if record["event"] != "plugin-install-bypass" {
		t.Errorf("expected event=plugin-install-bypass, got %q", record["event"])
	}
	if record["plugin"] != "test-plugin" {
		t.Errorf("expected plugin=test-plugin, got %q", record["plugin"])
	}
	if record["env"] != "dev" {
		t.Errorf("expected env=dev, got %q", record["env"])
	}
}

// TestWrite_AppendsMultipleEntries verifies that repeated Write calls append
// distinct JSON lines (one per line, no truncation).
func TestWrite_AppendsMultipleEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	for i := 0; i < 3; i++ {
		if err := Write("test-event", map[string]string{"i": string(rune('0' + i))}); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}

	logPath := filepath.Join(home, ".nself", "audit.log")
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	lineCount := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if sc.Text() == "" {
			continue
		}
		lineCount++
		var rec map[string]string
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Errorf("line %d is not valid JSON: %v", lineCount, err)
		}
	}
	if lineCount != 3 {
		t.Fatalf("expected 3 log lines, got %d", lineCount)
	}
}
