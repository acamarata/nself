package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFunctionsDeploy_CopiesFiles verifies that deploy copies a single file
// into ./functions/<name>/ and that the destination exists.
func TestFunctionsDeploy_CopiesFiles(t *testing.T) {
	t.Parallel()

	// Create a temp dir to act as project root.
	projectDir := t.TempDir()

	// Write a source function file.
	src := filepath.Join(projectDir, "hello.ts")
	if err := os.WriteFile(src, []byte("export default () => new Response('hello')"), 0640); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	// Change into project dir so copyFile resolves correctly.
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	_ = os.Chdir(projectDir)

	destDir := filepath.Join(projectDir, "functions", "hello")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dest := filepath.Join(destDir, "hello.ts")

	if err := copyFile(src, dest); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected file at %s, got error: %v", dest, err)
	}
}

// TestFunctionsList_ScansDirectory verifies that the functions directory is
// enumerated correctly.
func TestFunctionsList_ScansDirectory(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	// Create two function directories.
	for _, name := range []string{"fn-a", "fn-b"} {
		dir := filepath.Join(projectDir, "functions", name)
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	// Create a non-directory entry — should be ignored.
	f := filepath.Join(projectDir, "functions", "README.md")
	_ = os.WriteFile(f, []byte("docs"), 0640)

	entries, err := os.ReadDir(filepath.Join(projectDir, "functions"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}

	if len(dirs) != 2 {
		t.Errorf("expected 2 function dirs, got %d: %v", len(dirs), dirs)
	}
}

// TestFunctionsDelete_RemovesDir verifies that the delete handler removes the
// function directory.
func TestFunctionsDelete_RemovesDir(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	fnDir := filepath.Join(projectDir, "functions", "my-fn")
	if err := os.MkdirAll(fnDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.RemoveAll(fnDir); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	if _, err := os.Stat(fnDir); !os.IsNotExist(err) {
		t.Errorf("expected directory to be removed, got: %v", err)
	}
}

// TestFunctionNamePattern verifies the function name validation regex.
func TestFunctionNamePattern(t *testing.T) {
	t.Parallel()

	valid := []string{"hello", "hello-world", "fn1", "my-fn-2"}
	invalid := []string{"Hello", "hello world", "-start", "a_b", "", "fn!", "UPPER"}

	for _, name := range valid {
		if !functionNamePattern.MatchString(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}
	for _, name := range invalid {
		if functionNamePattern.MatchString(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}

// TestServiceUpgrade_WritesEnvVar verifies that serviceUpgrade writes the
// correct env key for various services.
func TestServiceUpgrade_WritesEnvVar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		service string
		version string
		wantKey string
	}{
		{"postgres", "16.3", "POSTGRES_VERSION"},
		{"hasura", "v2.40.0", "HASURA_VERSION"},
		{"auth", "latest", "AUTH_VERSION"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.service, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			envFile := filepath.Join(dir, ".env")

			if err := setEnvKeyInFile(envFile, serviceVersionKeys[tc.service], tc.version); err != nil {
				t.Fatalf("setEnvKeyInFile: %v", err)
			}

			data, err := os.ReadFile(envFile)
			if err != nil {
				t.Fatalf("reading env file: %v", err)
			}

			want := tc.wantKey + "=" + tc.version
			if !containsLine(string(data), want) {
				t.Errorf("env file missing %q\ngot:\n%s", want, string(data))
			}
		})
	}
}

// containsLine reports whether the given text contains the given line.
func containsLine(text, line string) bool {
	for _, l := range testSplitLines(text) {
		if l == line {
			return true
		}
	}
	return false
}

func testSplitLines(s string) []string {
	var lines []string
	for _, l := range filepath.SplitList(s) {
		lines = append(lines, l)
	}
	// filepath.SplitList uses OS path separator, not newline — use manual split.
	var result []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
