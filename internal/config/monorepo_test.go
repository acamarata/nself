package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectMonorepoRoot_BackendDir(t *testing.T) {
	// Create a temp directory that looks like a monorepo root with backend/.env
	tmp := t.TempDir()
	backendDir := filepath.Join(tmp, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("creating backend dir: %v", err)
	}
	envFile := filepath.Join(backendDir, ".env")
	if err := os.WriteFile(envFile, []byte("PROJECT_NAME=test\n"), 0o644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	got := DetectMonorepoRoot(tmp)
	if got != backendDir {
		t.Errorf("DetectMonorepoRoot(%q) = %q, want %q", tmp, got, backendDir)
	}
}

func TestDetectMonorepoRoot_DotBackendDir(t *testing.T) {
	// Create a temp directory that looks like a consumer app root with .backend/.env
	tmp := t.TempDir()
	backendDir := filepath.Join(tmp, ".backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("creating .backend dir: %v", err)
	}
	envFile := filepath.Join(backendDir, ".env")
	if err := os.WriteFile(envFile, []byte("PROJECT_NAME=test\n"), 0o644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	got := DetectMonorepoRoot(tmp)
	if got != backendDir {
		t.Errorf("DetectMonorepoRoot(%q) = %q, want %q", tmp, got, backendDir)
	}
}

func TestDetectMonorepoRoot_NoBackend(t *testing.T) {
	// A plain directory with no backend sub-directory returns "".
	tmp := t.TempDir()

	got := DetectMonorepoRoot(tmp)
	if got != "" {
		t.Errorf("DetectMonorepoRoot(%q) = %q, want empty string", tmp, got)
	}
}

func TestDetectMonorepoRoot_BackendDirWithoutEnv(t *testing.T) {
	// backend/ dir exists but has no .env file — should not be detected.
	tmp := t.TempDir()
	backendDir := filepath.Join(tmp, "backend")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("creating backend dir: %v", err)
	}
	// Intentionally no .env file.

	got := DetectMonorepoRoot(tmp)
	if got != "" {
		t.Errorf("DetectMonorepoRoot(%q) = %q, want empty string", tmp, got)
	}
}

func TestDetectMonorepoRoot_PrefersBackendOverDotBackend(t *testing.T) {
	// When both backend/.env and .backend/.env exist, "backend" is preferred
	// because it is listed first in backendDirCandidates.
	tmp := t.TempDir()

	for _, dir := range []string{"backend", ".backend"} {
		d := filepath.Join(tmp, dir)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("creating %s dir: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(d, ".env"), []byte("PROJECT_NAME=test\n"), 0o644); err != nil {
			t.Fatalf("writing %s/.env: %v", dir, err)
		}
	}

	want := filepath.Join(tmp, "backend")
	got := DetectMonorepoRoot(tmp)
	if got != want {
		t.Errorf("DetectMonorepoRoot(%q) = %q, want %q", tmp, got, want)
	}
}
