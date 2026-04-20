package installmeta_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nself-org/cli/internal/installmeta"
)

// TestSanitizeRef verifies the ref parameter is sanitized correctly.
// Acceptance criteria: install_source sanitized (no XSS via query param).
func TestSanitizeRef(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hn", "hn"},
		{"hacker-news", "hacker-news"},
		{"GITHUB", "github"},
		{"<script>alert(1)</script>", "scriptalert1script"},
		{"a b c", "abc"},
		{"valid_ref.v2", "valid_ref.v2"},
		// Cap at 64 chars
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaXXXX", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}

	for _, tt := range tests {
		got := installmeta.SanitizeRef(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeRef(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

// TestWriteAndRead verifies that a ref is persisted to install-meta.json and read back correctly.
// Acceptance criteria: ?ref=hn install populates .nself/install-meta.json with install_source: hn.
func TestWriteAndRead(t *testing.T) {
	// Use a temp home dir so we don't pollute the real one
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	if err := installmeta.Write("hn"); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	// Verify file exists
	metaPath := filepath.Join(tmpHome, ".nself", "install-meta.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Fatalf("install-meta.json not created at %s", metaPath)
	}

	// Read back
	meta, err := installmeta.Read()
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if meta.InstallSource != "hn" {
		t.Errorf("InstallSource = %q; want %q", meta.InstallSource, "hn")
	}
	if meta.SchemaVersion != 1 {
		t.Errorf("SchemaVersion = %d; want 1", meta.SchemaVersion)
	}
	if meta.InstallDate == "" {
		t.Error("InstallDate is empty")
	}
	if meta.CLIVersion == "" {
		t.Error("CLIVersion is empty")
	}
	// P95 note must be present
	if meta.P95Note == "" {
		t.Error("P95Note is empty")
	}

	// InstallSource() helper
	src := installmeta.InstallSource()
	if src != "hn" {
		t.Errorf("InstallSource() = %q; want %q", src, "hn")
	}
}

// TestInstallSourceNoFile verifies InstallSource returns "unknown" when no file exists.
func TestInstallSourceNoFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	src := installmeta.InstallSource()
	if src != "unknown" {
		t.Errorf("InstallSource() with no file = %q; want %q", src, "unknown")
	}
}

// TestNoPII verifies the install-meta.json does not contain PII.
// The schema only stores the sanitized ref, CLI version, schema version, date, and note.
func TestNoPII(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	if err := installmeta.Write("test-ref"); err != nil {
		t.Fatal(err)
	}

	metaPath := filepath.Join(tmpHome, ".nself", "install-meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("cannot read file: %v", err)
	}

	// The file must not contain email addresses, hostnames, or UUIDs
	// (primitive check — real PII audit requires more; this guards accidental additions)
	content := string(data)
	forbiddenPIIIndicators := []string{"@", "192.168.", "10.0.", "172.16."}
	for _, pii := range forbiddenPIIIndicators {
		if contains := len(content) > 0 && containsSubstring(content, pii); contains {
			t.Errorf("install-meta.json contains possible PII indicator %q", pii)
		}
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && containsSubstringRecurse(s, sub)))
}

func containsSubstringRecurse(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
