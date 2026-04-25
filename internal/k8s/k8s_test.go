package k8s

import (
	"errors"
	"testing"
)

// TestErrHelmNotFound verifies the sentinel error has a useful message.
func TestErrHelmNotFound(t *testing.T) {
	if ErrHelmNotFound == nil {
		t.Fatal("ErrHelmNotFound is nil")
	}
	msg := ErrHelmNotFound.Error()
	if msg == "" {
		t.Error("ErrHelmNotFound.Error() returned empty string")
	}
	// Ensure it names "helm" so the user knows what to install.
	if !containsStr(msg, "helm") {
		t.Errorf("ErrHelmNotFound.Error() = %q: should mention helm", msg)
	}
}

// TestHelmBinaryNotInPath verifies helmBinary returns ErrHelmNotFound when
// helm is not on PATH. This test sets PATH to an empty dir so the lookup fails.
func TestHelmBinaryNotInPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir — no binaries
	_, err := helmBinary()
	if err == nil {
		t.Fatal("helmBinary: expected error when helm not on PATH, got nil")
	}
	if !errors.Is(err, ErrHelmNotFound) {
		t.Errorf("helmBinary: got %v, want ErrHelmNotFound", err)
	}
}

// TestConstants verifies exported constants have the expected values.
func TestConstants(t *testing.T) {
	if HelmReleaseName != "nself" {
		t.Errorf("HelmReleaseName = %q, want %q", HelmReleaseName, "nself")
	}
	if ChartRepo == "" {
		t.Error("ChartRepo must not be empty")
	}
	if !containsStr(ChartRepo, "nself") {
		t.Errorf("ChartRepo = %q: expected to contain 'nself'", ChartRepo)
	}
}

// TestInstallOptionsDefaults verifies zero-value InstallOptions struct.
func TestInstallOptionsDefaults(t *testing.T) {
	opts := InstallOptions{}
	if opts.ReleaseName != "" {
		t.Errorf("InstallOptions.ReleaseName default should be empty, got %q", opts.ReleaseName)
	}
}

// TestInstallErrorsWithoutHelm verifies Install propagates ErrHelmNotFound
// when helm is not on PATH.
func TestInstallErrorsWithoutHelm(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := Install(t.Context(), InstallOptions{Domain: "test.local"})
	if err == nil {
		t.Fatal("Install: expected error when helm not on PATH")
	}
	if !errors.Is(err, ErrHelmNotFound) {
		t.Errorf("Install: got %v, want ErrHelmNotFound", err)
	}
}

// TestUpgradeErrorsWithoutHelm verifies Upgrade propagates ErrHelmNotFound.
func TestUpgradeErrorsWithoutHelm(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := Upgrade(t.Context(), InstallOptions{Domain: "test.local"})
	if err == nil {
		t.Fatal("Upgrade: expected error when helm not on PATH")
	}
	if !errors.Is(err, ErrHelmNotFound) {
		t.Errorf("Upgrade: got %v, want ErrHelmNotFound", err)
	}
}

// containsStr is a tiny helper to avoid importing strings in test file.
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || searchStr(s, sub))
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
