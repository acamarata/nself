package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteAIConfig_CreatesSecretsFile verifies that writeAIConfig creates
// .env.secrets with the AI block and a generated NSELF_MASTER_SECRET when no
// .env.secrets exists yet (CLI-R18: .env.ai folded into .env.secrets).
func TestWriteAIConfig_CreatesSecretsFile(t *testing.T) {
	dir := t.TempDir()

	ok, err := writeAIConfig(dir)
	if err != nil {
		t.Fatalf("writeAIConfig() error: %v", err)
	}
	if !ok {
		t.Fatal("writeAIConfig() ok = false, want true on first write")
	}

	path := filepath.Join(dir, ".env.secrets")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading .env.secrets: %v", err)
	}
	if !strings.Contains(string(data), "NSELF_MASTER_SECRET=") {
		t.Error(".env.secrets missing NSELF_MASTER_SECRET")
	}
	if !strings.Contains(string(data), "AI_PROFILE=auto") {
		t.Error(".env.secrets missing AI_PROFILE")
	}
}

// TestWriteAIConfig_Permissions0600 verifies the P15-class regression guard:
// .env.secrets must always be written 0600, never world/group readable.
func TestWriteAIConfig_Permissions0600(t *testing.T) {
	dir := t.TempDir()

	if _, err := writeAIConfig(dir); err != nil {
		t.Fatalf("writeAIConfig() error: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".env.secrets"))
	if err != nil {
		t.Fatalf("stat .env.secrets: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf(".env.secrets perm = %o, want 0600", perm)
	}
}

// TestWriteAIConfig_AppendsToExistingSecretsFile verifies that a .env.secrets
// file already created by --full init (e.g. holding POSTGRES_PASSWORD) gets
// the AI block appended rather than overwritten.
func TestWriteAIConfig_AppendsToExistingSecretsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.secrets")
	existing := "POSTGRES_PASSWORD=already-here\n"
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatalf("seeding .env.secrets: %v", err)
	}

	ok, err := writeAIConfig(dir)
	if err != nil {
		t.Fatalf("writeAIConfig() error: %v", err)
	}
	if !ok {
		t.Fatal("writeAIConfig() ok = false, want true when NSELF_MASTER_SECRET absent")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading .env.secrets: %v", err)
	}
	if !strings.Contains(string(data), "POSTGRES_PASSWORD=already-here") {
		t.Error("existing .env.secrets content was clobbered")
	}
	if !strings.Contains(string(data), "NSELF_MASTER_SECRET=") {
		t.Error("AI block was not appended")
	}
}

// TestWriteAIConfig_NeverRegeneratesExistingMasterSecret is the core
// anti-clobber guarantee: once NSELF_MASTER_SECRET is on disk, re-running
// writeAIConfig (e.g. on `nself init` re-run, or repeated builds) must never
// replace it — doing so would make previously-encrypted material unreadable.
func TestWriteAIConfig_NeverRegeneratesExistingMasterSecret(t *testing.T) {
	dir := t.TempDir()

	if _, err := writeAIConfig(dir); err != nil {
		t.Fatalf("first writeAIConfig() error: %v", err)
	}
	path := filepath.Join(dir, ".env.secrets")
	firstContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading .env.secrets: %v", err)
	}

	ok, err := writeAIConfig(dir)
	if err != nil {
		t.Fatalf("second writeAIConfig() error: %v", err)
	}
	if ok {
		t.Error("writeAIConfig() ok = true on second call, want false (must not rewrite)")
	}

	secondContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading .env.secrets after second call: %v", err)
	}
	if string(firstContent) != string(secondContent) {
		t.Error(".env.secrets content changed on second writeAIConfig() call — master secret must never rotate")
	}
}
