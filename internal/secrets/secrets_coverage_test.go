// Package secrets — secrets_coverage_test.go covers branches in rotation.go and
// secrets.go that are reachable without the age/age-keygen external binaries.
//
// Functions requiring age (EnsureAgeInstalled, Init, Get, List, loadStore when
// file exists, saveStore, DecryptForDeploy, decryptWithKey, Rekey, LintSecrets,
// Audit, RotateDualWindow, RetireOldKey, VerifySecretExists, Rotate) are 0% and
// genuinely untestable without the binaries present in CI.
// unreachable in test context: see UNREACHABLE comments below.
package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ageKeyPath — env var override branch
// ---------------------------------------------------------------------------

// TestAgeKeyPath_EnvOverride verifies the SECRETS_AGE_KEY_PATH override is
// returned directly without calling UserHomeDir.
func TestAgeKeyPath_EnvOverride(t *testing.T) {
	const want = "/custom/path/age-key.txt"
	t.Setenv("SECRETS_AGE_KEY_PATH", want)

	got, err := ageKeyPath()
	if err != nil {
		t.Fatalf("ageKeyPath with env override: unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("ageKeyPath: got %q, want %q", got, want)
	}
}

// TestAgeKeyPath_DefaultPath verifies the default path is under ~/.config/nself
// when the env var is not set.
func TestAgeKeyPath_DefaultPath(t *testing.T) {
	t.Setenv("SECRETS_AGE_KEY_PATH", "") // ensure cleared

	got, err := ageKeyPath()
	if err != nil {
		t.Fatalf("ageKeyPath default: unexpected error: %v", err)
	}
	if !strings.HasSuffix(got, filepath.Join(".config", "nself", "age-key.txt")) {
		t.Errorf("ageKeyPath default: got %q, want suffix .config/nself/age-key.txt", got)
	}
}

// ---------------------------------------------------------------------------
// envFileName — empty env → "dev" branch
// ---------------------------------------------------------------------------

// TestEnvFileName_EmptyIsdev verifies empty string returns "dev.age".
func TestEnvFileName_EmptyIsDev(t *testing.T) {
	got := envFileName("")
	if got != "dev.age" {
		t.Errorf("envFileName(''): got %q, want %q", got, "dev.age")
	}
}

// TestEnvFileName_ExplicitEnv verifies non-empty env is returned as "<env>.age".
func TestEnvFileName_ExplicitEnv(t *testing.T) {
	cases := []struct{ in, want string }{
		{"prod", "prod.age"},
		{"staging", "staging.age"},
		{"dev", "dev.age"},
	}
	for _, tc := range cases {
		got := envFileName(tc.in)
		if got != tc.want {
			t.Errorf("envFileName(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// GetPublicKey — branches: file read error, no public key line, found line
// ---------------------------------------------------------------------------

// TestGetPublicKey_ReadError verifies error when the file does not exist.
func TestGetPublicKey_ReadError(t *testing.T) {
	_, err := GetPublicKey("/nonexistent/path/age-key.txt")
	if err == nil {
		t.Fatal("GetPublicKey on nonexistent file: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading key file") {
		t.Errorf("GetPublicKey error: got %q, want 'reading key file' in message", err.Error())
	}
}

// TestGetPublicKey_NoPublicKeyLine verifies error when the file contains no
// "# public key:" comment.
func TestGetPublicKey_NoPublicKeyLine(t *testing.T) {
	f := filepath.Join(t.TempDir(), "age-key.txt")
	if err := os.WriteFile(f, []byte("# created: 2024-01-01\nAGE-SECRET-KEY-1FAKE\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := GetPublicKey(f)
	if err == nil {
		t.Fatal("GetPublicKey without public key line: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no public key found") {
		t.Errorf("GetPublicKey error: got %q, want 'no public key found'", err.Error())
	}
}

// TestGetPublicKey_FoundLine verifies the public key is extracted correctly.
func TestGetPublicKey_FoundLine(t *testing.T) {
	const wantKey = "age1abcdef1234567890"
	content := "# created: 2024-01-01\n# public key: " + wantKey + "\nAGE-SECRET-KEY-1FAKE\n"
	f := filepath.Join(t.TempDir(), "age-key.txt")
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := GetPublicKey(f)
	if err != nil {
		t.Fatalf("GetPublicKey with valid file: unexpected error: %v", err)
	}
	if got != wantKey {
		t.Errorf("GetPublicKey: got %q, want %q", got, wantKey)
	}
}

// ---------------------------------------------------------------------------
// LoadRotationState — non-NotExist error and JSON parse error branches
// ---------------------------------------------------------------------------

// TestLoadRotationState_ReadError verifies "reading rotation state" error when
// the path is a directory (EISDIR makes ReadFile fail with non-NotExist error).
func TestLoadRotationState_ReadError(t *testing.T) {
	dir := t.TempDir()
	// Make the rotation-state.json path a directory so ReadFile returns EISDIR.
	statePath := rotationStatePath(dir)
	if err := os.MkdirAll(statePath, 0755); err != nil {
		t.Fatalf("MkdirAll to create EISDIR blocker: %v", err)
	}

	_, err := LoadRotationState(dir)
	if err == nil {
		t.Fatal("LoadRotationState with EISDIR: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading rotation state") {
		t.Errorf("LoadRotationState error: got %q, want 'reading rotation state'", err.Error())
	}
}

// TestLoadRotationState_ParseError verifies "parsing rotation state" error when
// the file contains invalid JSON.
func TestLoadRotationState_ParseError(t *testing.T) {
	dir := t.TempDir()
	statePath := rotationStatePath(dir)
	if err := os.MkdirAll(filepath.Dir(statePath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte("not valid json {{{"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadRotationState(dir)
	if err == nil {
		t.Fatal("LoadRotationState with bad JSON: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing rotation state") {
		t.Errorf("LoadRotationState error: got %q, want 'parsing rotation state'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// SaveRotationState — MkdirAll fail branch
// ---------------------------------------------------------------------------

// TestSaveRotationState_MkdirAllFail verifies error propagation when MkdirAll
// cannot create the directory because a regular file blocks the path.
func TestSaveRotationState_MkdirAllFail(t *testing.T) {
	dir := t.TempDir()
	// Write a regular file at the path where MkdirAll needs to create a dir.
	secretsDirPath := filepath.Join(dir, SecretsDir)
	if err := os.WriteFile(secretsDirPath, []byte("blocker"), 0600); err != nil {
		t.Fatalf("creating blocker file: %v", err)
	}

	state := &RotationState{Schedules: DefaultSchedules()}
	err := SaveRotationState(dir, state)
	if err == nil {
		t.Fatal("SaveRotationState with blocked dir: expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// LoadRotationLog — non-NotExist error and JSON parse error branches
// ---------------------------------------------------------------------------

// TestLoadRotationLog_ReadError verifies "reading rotation log" error when the
// path is a directory (EISDIR).
func TestLoadRotationLog_ReadError(t *testing.T) {
	dir := t.TempDir()
	logPath := rotationLogPath(dir)
	if err := os.MkdirAll(logPath, 0755); err != nil {
		t.Fatalf("MkdirAll for EISDIR: %v", err)
	}

	_, err := LoadRotationLog(dir)
	if err == nil {
		t.Fatal("LoadRotationLog with EISDIR: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading rotation log") {
		t.Errorf("LoadRotationLog error: got %q, want 'reading rotation log'", err.Error())
	}
}

// TestLoadRotationLog_ParseError verifies "parsing rotation log" error when the
// file contains invalid JSON.
func TestLoadRotationLog_ParseError(t *testing.T) {
	dir := t.TempDir()
	logPath := rotationLogPath(dir)
	if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("{bad json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadRotationLog(dir)
	if err == nil {
		t.Fatal("LoadRotationLog with bad JSON: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing rotation log") {
		t.Errorf("LoadRotationLog error: got %q, want 'parsing rotation log'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// AppendRotationEvent — MkdirAll fail branch
// ---------------------------------------------------------------------------

// TestAppendRotationEvent_MkdirAllFail verifies error propagation when
// MkdirAll is blocked by a regular file at the target directory path.
func TestAppendRotationEvent_MkdirAllFail(t *testing.T) {
	dir := t.TempDir()
	// Block the secrets dir path with a regular file.
	secretsDirPath := filepath.Join(dir, SecretsDir)
	if err := os.WriteFile(secretsDirPath, []byte("blocker"), 0600); err != nil {
		t.Fatalf("creating blocker file: %v", err)
	}

	err := AppendRotationEvent(dir, "JWT_SIGNING_KEY", "success", "test")
	if err == nil {
		t.Fatal("AppendRotationEvent with blocked dir: expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// CheckSchedule — branch coverage for schedule check states
// ---------------------------------------------------------------------------

// TestCheckSchedule_EmptyDir runs CheckSchedule against a fresh temp dir.
// Exercises: LoadRotationState with no file → returns default schedules → checks them.
func TestCheckSchedule_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := CheckSchedule(dir)
	if err != nil {
		t.Fatalf("CheckSchedule on empty dir: %v", err)
	}
	// DefaultSchedules returns non-empty list; every result must have a status.
	for _, r := range results {
		if r.Status == "" {
			t.Errorf("CheckSchedule: entry %q has empty status", r.SecretName)
		}
	}
}

// TestCheckSchedule_LoadError verifies error propagation from LoadRotationState.
func TestCheckSchedule_LoadError(t *testing.T) {
	dir := t.TempDir()
	// Make rotation state path a directory so ReadFile fails.
	statePath := rotationStatePath(dir)
	if err := os.MkdirAll(statePath, 0755); err != nil {
		t.Fatal(err)
	}
	_, err := CheckSchedule(dir)
	if err == nil {
		t.Fatal("CheckSchedule with bad state file: expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// InitSchedules — error propagation from LoadRotationState
// ---------------------------------------------------------------------------

// TestInitSchedules_LoadStateError verifies error propagation when state cannot
// be loaded.
func TestInitSchedules_LoadStateError(t *testing.T) {
	dir := t.TempDir()
	statePath := rotationStatePath(dir)
	if err := os.MkdirAll(statePath, 0755); err != nil {
		t.Fatal(err)
	}
	err := InitSchedules(dir)
	if err == nil {
		t.Fatal("InitSchedules with bad state: expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// UNREACHABLE documentation
// ---------------------------------------------------------------------------
//
// The following functions are 0% covered and genuinely unreachable without
// the age/age-keygen external binaries present in the test environment:
//
// unreachable: EnsureAgeInstalled — calls exec.LookPath("age") + exec.LookPath("age-keygen")
// unreachable: Init — calls EnsureAgeInstalled first; needs age-keygen to generate a keypair
// unreachable: Get — calls loadStore which runs age --decrypt; no age binary in CI
// unreachable: List — calls loadStore; same reason
// unreachable: Audit — calls loadStore; same reason
// unreachable: DecryptForDeploy — calls decryptWithKey which runs age --decrypt
// unreachable: decryptWithKey — runs age --decrypt subprocess
// unreachable: Rekey — calls EnsureAgeInstalled + age --keygen + age --rekey
// unreachable: LintSecrets — calls loadStore; same reason
// unreachable: loadStore (file-exists branch) — requires age --decrypt to succeed
// unreachable: saveStore — requires age --encrypt subprocess
// unreachable: RotateDualWindow — calls loadStore; needs age binary
// unreachable: RetireOldKey — calls loadStore; needs age binary
// unreachable: VerifySecretExists — calls loadStore; needs age binary
// unreachable: Rotate — calls loadStore; needs age binary
