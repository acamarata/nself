package deprecation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmbeddedRegistryIsNotEmpty proves the registry YAML is compiled into the
// binary. Before CLI-R03 the registry was read from a path next to the
// executable, which never exists for an installed binary, so this is the exact
// regression guard for "deprecation warnings silently disabled in production".
func TestEmbeddedRegistryIsNotEmpty(t *testing.T) {
	if len(registryYAML) == 0 {
		t.Fatal("embedded registry.yaml is empty — go:embed directive is broken")
	}

	reg, err := LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("LoadEmbeddedRegistry: %v", err)
	}
	if reg.Len() == 0 {
		t.Fatal("embedded registry parsed to zero entries")
	}
}

// TestEmbeddedRegistryMatchesFile keeps the compiled-in copy honest: the bytes
// in the binary must equal the bytes in the repository file.
func TestEmbeddedRegistryMatchesFile(t *testing.T) {
	onDisk, err := os.ReadFile("registry.yaml")
	if err != nil {
		t.Fatalf("read registry.yaml: %v", err)
	}
	if string(onDisk) != string(registryYAML) {
		t.Fatal("embedded registry bytes differ from internal/deprecation/registry.yaml")
	}
}

// TestEmbeddedRegistryWorksFromForeignWorkingDirectory simulates the installed
// binary: the process runs from a directory with no nSelf source tree in sight.
// The embedded path must still resolve every entry.
func TestEmbeddedRegistryWorksFromForeignWorkingDirectory(t *testing.T) {
	before, err := LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("LoadEmbeddedRegistry: %v", err)
	}

	empty := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	if err := os.Chdir(empty); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Sanity: the old path-based resolution would find nothing here.
	if _, err := os.Stat(filepath.Join("internal", "deprecation", "registry.yaml")); err == nil {
		t.Fatal("temp dir unexpectedly contains a source tree")
	}

	after, err := LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("LoadEmbeddedRegistry from empty dir: %v", err)
	}
	if after.Len() != before.Len() {
		t.Fatalf("entry count changed with working directory: %d != %d", after.Len(), before.Len())
	}
	for _, name := range before.Names() {
		if !after.IsDeprecated(name) {
			t.Fatalf("entry %q lost when running outside the source tree", name)
		}
	}
}

// TestRegistryPathEnvOverride verifies the test/preview override, and that a
// broken override degrades to the embedded registry rather than to silence.
func TestRegistryPathEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.yaml")
	body := "items:\n  - name: nself only-in-override\n    type: command\n    since: v9.9.9\n    replacement: nself nothing\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write override: %v", err)
	}

	t.Setenv(RegistryPathEnv, path)
	reg, err := LoadEmbeddedRegistry()
	if err != nil {
		t.Fatalf("override load: %v", err)
	}
	if !reg.IsDeprecated("nself only-in-override") {
		t.Fatal("override registry was not used")
	}

	t.Setenv(RegistryPathEnv, filepath.Join(dir, "does-not-exist.yaml"))
	fallback, err := LoadEmbeddedRegistry()
	if err == nil {
		t.Fatal("expected an error describing the unusable override")
	}
	if !strings.Contains(err.Error(), "using embedded registry") {
		t.Fatalf("unexpected error text: %v", err)
	}
	if fallback.Len() == 0 {
		t.Fatal("broken override disabled the registry instead of falling back")
	}
}
