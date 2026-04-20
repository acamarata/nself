// Package auth — unit tests for credential storage.
// Tests use a temp dir to avoid touching ~/.nself.
package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// withTempHome overrides HOME to an isolated temp dir for the duration of the test.
// It restores the original HOME on cleanup.
func withTempHome(t *testing.T) (tempDir string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// On macOS os.UserHomeDir() reads $HOME first.
	return dir
}

// ─── WriteAuthFile / ReadAuthFile round-trip ──────────────────────────────────

func TestWriteReadAuthFile_RoundTrip(t *testing.T) {
	withTempHome(t)

	want := &AuthFile{
		AccessToken:  "jwt.access.token",
		SessionToken: "opaque-session",
		Email:        "user@example.com",
		Tier:         "basic",
		DisplayName:  "Test User",
		Bundles:      []string{"nChat", "nClaw"},
		ExpiresAt:    "2027-01-01T00:00:00Z",
	}

	if err := WriteAuthFile(want); err != nil {
		t.Fatalf("WriteAuthFile: %v", err)
	}

	got, err := ReadAuthFile()
	if err != nil {
		t.Fatalf("ReadAuthFile: %v", err)
	}

	if got.AccessToken != want.AccessToken {
		t.Errorf("AccessToken: got %q, want %q", got.AccessToken, want.AccessToken)
	}
	if got.Email != want.Email {
		t.Errorf("Email: got %q, want %q", got.Email, want.Email)
	}
	if got.Tier != want.Tier {
		t.Errorf("Tier: got %q, want %q", got.Tier, want.Tier)
	}
	if len(got.Bundles) != 2 {
		t.Errorf("Bundles len: got %d, want 2", len(got.Bundles))
	}
}

// ─── File permissions ─────────────────────────────────────────────────────────

func TestWriteAuthFile_Permissions(t *testing.T) {
	withTempHome(t)

	af := &AuthFile{
		AccessToken: "jwt.token",
		Email:       "perm@example.com",
	}

	if err := WriteAuthFile(af); err != nil {
		t.Fatalf("WriteAuthFile: %v", err)
	}

	path, err := authFilePath()
	if err != nil {
		t.Fatalf("authFilePath: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}

	// Security invariant: auth.json must be 0600 only.
	const want = os.FileMode(0600)
	if info.Mode().Perm() != want {
		t.Errorf("permissions: got %04o, want %04o", info.Mode().Perm(), want)
	}
}

func TestWriteAuthFile_DirectoryPermissions(t *testing.T) {
	withTempHome(t)

	if err := WriteAuthFile(&AuthFile{AccessToken: "tok"}); err != nil {
		t.Fatalf("WriteAuthFile: %v", err)
	}

	path, _ := authFilePath()
	dir := filepath.Dir(path)

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat %s: %v", dir, err)
	}

	// Security invariant: ~/.nself/ must be 0700.
	const want = os.FileMode(0700)
	if info.Mode().Perm() != want {
		t.Errorf("dir permissions: got %04o, want %04o", info.Mode().Perm(), want)
	}
}

// ─── ReadAuthFile — missing file ──────────────────────────────────────────────

func TestReadAuthFile_NotFound(t *testing.T) {
	withTempHome(t)

	_, err := ReadAuthFile()
	if err == nil {
		t.Fatal("expected ErrNotLoggedIn, got nil")
	}
	if !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("expected ErrNotLoggedIn, got %v", err)
	}
}

// ─── ReadAuthFile — empty access token ────────────────────────────────────────

func TestReadAuthFile_EmptyToken(t *testing.T) {
	home := withTempHome(t)

	// Write a file with no access token.
	dir := filepath.Join(home, ".nself")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"email":"x@y.com"}`), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := ReadAuthFile()
	if !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("expected ErrNotLoggedIn for empty token, got %v", err)
	}
}

// ─── ReadAuthFile — malformed JSON ────────────────────────────────────────────

func TestReadAuthFile_Malformed(t *testing.T) {
	home := withTempHome(t)

	dir := filepath.Join(home, ".nself")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{bad json}`), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := ReadAuthFile()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	// Should NOT be ErrNotLoggedIn — it's a parse failure, not absence.
	if errors.Is(err, ErrNotLoggedIn) {
		t.Error("expected parse error, not ErrNotLoggedIn")
	}
}

// ─── DeleteAuthFile ───────────────────────────────────────────────────────────

func TestDeleteAuthFile_Success(t *testing.T) {
	withTempHome(t)

	af := &AuthFile{AccessToken: "tok"}
	if err := WriteAuthFile(af); err != nil {
		t.Fatalf("WriteAuthFile: %v", err)
	}

	if err := DeleteAuthFile(); err != nil {
		t.Fatalf("DeleteAuthFile: %v", err)
	}

	// File must be gone.
	path, _ := authFilePath()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestDeleteAuthFile_Idempotent(t *testing.T) {
	withTempHome(t)

	// Delete when no file exists — must return nil.
	if err := DeleteAuthFile(); err != nil {
		t.Fatalf("DeleteAuthFile on missing file: %v", err)
	}
}

// ─── IsLoggedIn ───────────────────────────────────────────────────────────────

func TestIsLoggedIn_True(t *testing.T) {
	withTempHome(t)

	if err := WriteAuthFile(&AuthFile{AccessToken: "tok"}); err != nil {
		t.Fatalf("WriteAuthFile: %v", err)
	}

	if !IsLoggedIn() {
		t.Error("expected IsLoggedIn()=true")
	}
}

func TestIsLoggedIn_False_NoFile(t *testing.T) {
	withTempHome(t)

	if IsLoggedIn() {
		t.Error("expected IsLoggedIn()=false for missing file")
	}
}

func TestIsLoggedIn_False_EmptyToken(t *testing.T) {
	home := withTempHome(t)

	dir := filepath.Join(home, ".nself")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"email":"x@y.com"}`), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if IsLoggedIn() {
		t.Error("expected IsLoggedIn()=false for empty token")
	}
}

// ─── GetAuthFilePath ──────────────────────────────────────────────────────────

func TestGetAuthFilePath_ContainsNself(t *testing.T) {
	withTempHome(t)

	p := GetAuthFilePath()
	if p == "" {
		t.Error("expected non-empty path")
	}
	// Must contain .nself and auth.json.
	for _, part := range []string{".nself", "auth.json"} {
		if !containsString(p, part) {
			t.Errorf("path %q missing %q", p, part)
		}
	}
}

// ─── WriteAuthFile idempotency (overwrite) ────────────────────────────────────

func TestWriteAuthFile_Overwrite(t *testing.T) {
	withTempHome(t)

	first := &AuthFile{AccessToken: "first.token", Email: "a@b.com"}
	if err := WriteAuthFile(first); err != nil {
		t.Fatalf("first write: %v", err)
	}

	second := &AuthFile{AccessToken: "second.token", Email: "c@d.com"}
	if err := WriteAuthFile(second); err != nil {
		t.Fatalf("second write: %v", err)
	}

	got, err := ReadAuthFile()
	if err != nil {
		t.Fatalf("ReadAuthFile: %v", err)
	}
	if got.AccessToken != "second.token" {
		t.Errorf("expected second.token, got %q", got.AccessToken)
	}
}
