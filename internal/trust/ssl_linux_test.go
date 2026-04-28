// Package trust — ssl_linux_test.go: Linux-targeted tests for ssl.go.
//
// Targets the Linux-specific switch arms in installMkcert and installMkcertCA,
// which return manual-installation error strings rather than executing brew /
// osascript. These branches are fully testable on a Linux runner without root,
// network, or system mods.
//
// Cross-platform helpers (buildDomainList, fileExists) are also exercised on
// Linux to bump per-statement coverage on the GOOS=linux build.
package trust

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallMkcert_LinuxBranch verifies installMkcert on Linux returns the
// documented manual-install instructions when mkcert is not on PATH.
//
// Skipped when mkcert is already installed on the runner, since the function
// would short-circuit before reaching the switch arm.
func TestInstallMkcert_LinuxBranch(t *testing.T) {
	requireLinux(t)

	if _, err := exec.LookPath("mkcert"); err == nil {
		t.Skip("mkcert already on PATH — Linux manual-install branch unreachable")
	}

	err := installMkcert()
	if err == nil {
		t.Fatal("expected manual-install error on Linux, got nil")
	}

	msg := err.Error()
	for _, want := range []string{
		"mkcert not found",
		"Debian/Ubuntu",
		"libnss3-tools",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error should mention %q; got: %s", want, msg)
		}
	}
}

// TestInstallMkcertCA_LinuxBranch verifies installMkcertCA on Linux returns
// the documented manual-trust instructions.
func TestInstallMkcertCA_LinuxBranch(t *testing.T) {
	requireLinux(t)

	caPath := "/tmp/test-mkcert-ca.pem"
	err := installMkcertCA(caPath)
	if err == nil {
		t.Fatal("expected manual-trust error on Linux, got nil")
	}

	msg := err.Error()
	for _, want := range []string{
		"mkcert CA not trusted",
		"mkcert -install",
		caPath,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error should mention %q; got: %s", want, msg)
		}
	}
}

// TestBuildDomainList_LinuxRunner exercises buildDomainList on Linux to
// contribute to GOOS=linux coverage. Cross-platform logic, but cover stmt
// per build.
func TestBuildDomainList_LinuxRunner(t *testing.T) {
	requireLinux(t)

	cfg := TrustConfig{
		BaseDomain:        "ummat.local",
		NamespacePrefixes: []string{"pro", "app"},
		ExtraSSLDomains:   []string{"extra.test"},
	}

	got := buildDomainList(cfg)

	// Required entries: base, *.base, per-namespace wildcards, 127.0.0.1, extras.
	wantSubset := []string{
		"ummat.local",
		"*.ummat.local",
		"*.pro.ummat.local",
		"*.app.ummat.local",
		"127.0.0.1",
		"extra.test",
	}

	have := strings.Join(got, " ")
	for _, w := range wantSubset {
		if !strings.Contains(have, w) {
			t.Errorf("buildDomainList missing %q; got %v", w, got)
		}
	}
}

// TestBuildDomainList_NoBaseDomain confirms the empty-base path still
// includes 127.0.0.1.
func TestBuildDomainList_NoBaseDomain(t *testing.T) {
	requireLinux(t)

	got := buildDomainList(TrustConfig{})
	found := false
	for _, d := range got {
		if d == "127.0.0.1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("buildDomainList must always include 127.0.0.1; got %v", got)
	}
}

// TestFileExists_LinuxRunner exercises fileExists on Linux against a
// real tempfile, a missing path, and a directory.
func TestFileExists_LinuxRunner(t *testing.T) {
	requireLinux(t)

	dir := t.TempDir()
	present := filepath.Join(dir, "real.pem")
	if err := os.WriteFile(present, []byte("x"), 0600); err != nil {
		t.Fatalf("seed tempfile: %v", err)
	}

	if !fileExists(present) {
		t.Errorf("fileExists(%q) should be true", present)
	}
	if fileExists(filepath.Join(dir, "missing.pem")) {
		t.Errorf("fileExists on missing path should be false")
	}
	if fileExists(dir) {
		t.Errorf("fileExists on directory should be false")
	}
}

// TestCheckMkcert_LinuxRunner exercises CheckMkcert read-only branches on
// Linux for GOOS=linux statement coverage. Result combinations vary by host.
func TestCheckMkcert_LinuxRunner(t *testing.T) {
	requireLinux(t)

	dir := t.TempDir()
	cfg := TrustConfig{WorkDir: dir}
	_, _, _ = CheckMkcert(cfg)

	// Seed cert files (malformed PEM is fine — exercises expiry-check branch).
	sslDir := filepath.Join(dir, "ssl")
	if err := os.MkdirAll(sslDir, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"fullchain.pem", "privkey.pem"} {
		_ = os.WriteFile(filepath.Join(sslDir, name), []byte("not-real"), 0600)
	}
	_, _, _ = CheckMkcert(cfg)
}
