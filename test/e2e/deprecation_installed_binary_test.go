// Package e2e — deprecation_installed_binary_test.go
//
// CLI-R03 regression guard (P6-E2-W1-S1-T2): resolveRegistryPath() used to
// look for internal/deprecation/registry.yaml beside the executable, which
// never exists after `make install`/brew (the yaml source only lives in the
// dev tree). The fix embeds the registry via `//go:embed registry.yaml`
// (internal/deprecation/embedded.go) so it travels inside the binary.
//
// internal/deprecation/embedded_test.go already proves byte-identical
// embed-vs-disk equality in-process, but nothing previously simulated a real
// `make install` + copy-away scenario: a binary built once, then copied to a
// second, completely empty directory with no source tree — and therefore no
// registry.yaml — anywhere nearby. This is that black-box proof.
//
// This test is fully headless: it builds the CLI, copies it to an isolated
// temp dir, and runs it there, asserting a clean exit with no
// registry-file-not-found error in stderr.
package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDeprecationInstalledBinary builds the nself CLI, copies the resulting
// binary into a second, empty temp directory with no adjacent
// registry.yaml or source tree, runs it there, and asserts the deprecation
// registry lookup does not fail with a missing-file error — proving the
// go:embed fix travels with a copied-away, installed-style binary.
func TestDeprecationInstalledBinary(t *testing.T) {
	buildDir := t.TempDir()
	// Windows decides what is executable by the .exe extension, not a mode bit,
	// so a binary written without it cannot be exec'd there. See the repo rules
	// on Windows-portable tests.
	bin := filepath.Join(buildDir, "nself-copy-test"+exeSuffix())

	buildCmd := exec.Command("go", "build", "-mod=vendor", "-o", bin, "../../cmd/nself")
	buildCmd.Dir = mustGetwdT2(t)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("building nself: %v\n%s", err, out)
	}

	// Copy the binary to a second, unrelated empty directory. This is the
	// exact scenario CLI-R03 described: no source tree, no
	// internal/deprecation/registry.yaml anywhere on disk nearby — only the
	// compiled binary itself.
	emptyDir := t.TempDir()
	copiedBin := filepath.Join(emptyDir, "nself"+exeSuffix())
	copyFile(t, bin, copiedBin)
	if err := os.Chmod(copiedBin, 0o755); err != nil {
		t.Fatalf("chmod copied binary: %v", err)
	}

	// Run a command that exercises the deprecation checker (any real
	// command does, since it is wired at the root command layer via
	// cmd/commands/root.go's deprecationRegistry). "version" is side-effect
	// free and always available.
	runCmd := exec.Command(copiedBin, "version")
	runCmd.Dir = emptyDir
	// HOME/USERPROFILE point at a scratch dir too, so the binary cannot
	// accidentally find a registry.yaml via an unrelated project or dotfile
	// lookup rooted at the real developer home directory.
	scratchHome := t.TempDir()
	runCmd.Env = append(os.Environ(),
		"HOME="+scratchHome,
		"USERPROFILE="+scratchHome,
	)
	out, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running copied binary failed: %v\noutput:\n%s", err, out)
	}

	output := string(out)
	for _, bad := range []string{
		"registry.yaml not found",
		"no such file or directory",
		"registry.yaml: no such file",
	} {
		if strings.Contains(output, bad) {
			t.Errorf("copied-binary run referenced a missing registry file (%q) in output:\n%s", bad, output)
		}
	}
}

func mustGetwdT2(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatalf("writing %s: %v", dst, err)
	}
}

// exeSuffix returns the extension an executable needs on this platform.
// Windows resolves executability from the extension, so a test that builds a
// binary and then runs it must add .exe or exec fails with "file does not
// exist" on that platform only.
func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
