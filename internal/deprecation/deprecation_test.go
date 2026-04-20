package deprecation_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/deprecation"
)

// writeRegistry creates a temporary registry YAML file and returns its path.
func writeRegistry(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write registry: %v", err)
	}
	return path
}

func TestLoadRegistry_EmptyItems(t *testing.T) {
	path := writeRegistry(t, "items: []\n")
	reg, err := deprecation.LoadRegistry(path)
	if err != nil {
		t.Fatalf("expected no error for empty registry, got: %v", err)
	}
	if reg.IsDeprecated("nself anything") {
		t.Error("empty registry should return false for IsDeprecated")
	}
}

func TestLoadRegistry_MissingFile(t *testing.T) {
	reg, err := deprecation.LoadRegistry("/no/such/file.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	// Registry must still be usable (not nil)
	if reg == nil {
		t.Fatal("registry must not be nil even on load error")
	}
	if reg.IsDeprecated("anything") {
		t.Error("empty registry must not mark anything as deprecated")
	}
}

func TestLoadRegistry_WithEntry(t *testing.T) {
	yaml := `items:
  - name: "nself old-cmd"
    type: command
    since: "v1.0.9"
    replacement: "nself new-cmd"
    docs_url: "https://docs.nself.org/migrate"
    phase: 1
`
	path := writeRegistry(t, yaml)
	reg, err := deprecation.LoadRegistry(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reg.IsDeprecated("nself old-cmd") {
		t.Error("expected 'nself old-cmd' to be deprecated")
	}
	if reg.IsDeprecated("nself other-cmd") {
		t.Error("'nself other-cmd' should not be deprecated")
	}
}

func TestWarnFormat_Phase1(t *testing.T) {
	yaml := `items:
  - name: "nself old-cmd"
    type: command
    since: "v1.0.9"
    replacement: "nself new-cmd"
    docs_url: "https://docs.nself.org/migrate"
    phase: 1
`
	path := writeRegistry(t, yaml)
	reg, _ := deprecation.LoadRegistry(path)
	item, _ := reg.Lookup("nself old-cmd")

	var buf bytes.Buffer
	reg.Warn(&buf, item)
	out := buf.String()

	if !strings.Contains(out, "[DEPRECATED]") {
		t.Errorf("warning must contain [DEPRECATED], got: %q", out)
	}
	if !strings.Contains(out, "nself old-cmd") {
		t.Errorf("warning must contain command name, got: %q", out)
	}
	if !strings.Contains(out, "v1.0.9") {
		t.Errorf("warning must contain since version, got: %q", out)
	}
	if !strings.Contains(out, "nself new-cmd") {
		t.Errorf("warning must contain replacement, got: %q", out)
	}
	if !strings.Contains(out, "https://docs.nself.org/migrate") {
		t.Errorf("warning must contain docs URL, got: %q", out)
	}
	if strings.Contains(out, "Will be removed") {
		t.Errorf("phase 1 warning must not mention removal, got: %q", out)
	}
}

func TestWarnFormat_Phase2(t *testing.T) {
	yaml := `items:
  - name: "nself old-cmd"
    type: command
    since: "v1.0.9"
    replacement: "nself new-cmd"
    docs_url: "https://docs.nself.org/migrate"
    phase: 2
`
	path := writeRegistry(t, yaml)
	reg, _ := deprecation.LoadRegistry(path)
	item, _ := reg.Lookup("nself old-cmd")

	var buf bytes.Buffer
	reg.Warn(&buf, item)
	out := buf.String()

	if !strings.Contains(out, "Will be removed") {
		t.Errorf("phase 2 warning must mention removal, got: %q", out)
	}
}

func TestWarnWritesToStderr(t *testing.T) {
	// Verify Warn() accepts os.Stderr without panic — io.Writer interface test.
	var buf bytes.Buffer
	deprecation.Warn(&buf, "nself x", "v1.0.0", "nself y", "https://docs.nself.org")
	if buf.Len() == 0 {
		t.Error("Warn() must write output to the provided writer")
	}
}

func TestLookup_NotFound(t *testing.T) {
	path := writeRegistry(t, "items: []\n")
	reg, _ := deprecation.LoadRegistry(path)
	_, ok := reg.Lookup("nself missing")
	if ok {
		t.Error("Lookup must return false for unknown entries")
	}
}

func TestPackageLevelWarnFormat(t *testing.T) {
	var buf bytes.Buffer
	deprecation.Warn(&buf, "nself old", "v1.1.0", "nself new", "https://docs.nself.org/migrate")
	out := buf.String()
	if !strings.Contains(out, "[DEPRECATED]") {
		t.Errorf("package-level Warn must contain [DEPRECATED], got: %q", out)
	}
}
