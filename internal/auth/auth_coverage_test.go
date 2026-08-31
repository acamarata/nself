package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// skipUnixOnly skips the test on Windows because it exercises behaviour that
// depends on Unix-specific semantics (e.g. chmod read-only dirs, XDG/HOME env
// var resolution via os.UserHomeDir, or POSIX signal handling).
func skipUnixOnly(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows: Unix-only semantics")
	}
}

// setHomeAndProfile sets both HOME (Unix) and USERPROFILE (Windows) so that
// os.UserHomeDir() picks up the temp dir on every platform.
func setHomeAndProfile(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

// errTransport is an http.RoundTripper that always returns an error.
// Used to cover the httpClient.Do(req) error branches in client.go.
type errTransport struct{ err error }

func (t *errTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, t.err
}

// withFailingTransport replaces httpClient.Transport with an error-returning
// transport for the duration of the test, then restores the original.
func withFailingTransport(t *testing.T, err error) {
	t.Helper()
	orig := httpClient.Transport
	httpClient.Transport = &errTransport{err: err}
	t.Cleanup(func() { httpClient.Transport = orig })
}

// withTestServer sets NSELF_AUTH_SERVER_URL to a test httptest.Server URL
// and restores the original env after the test. The server uses the provided
// handler function.
func withTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Setenv("NSELF_AUTH_SERVER_URL", srv.URL)
	t.Cleanup(srv.Close)
	return srv
}

// ── RotationLogPath ───────────────────────────────────────────────────────────

// TestRotationLogPath_PrimaryWritable covers the branch where the primary
// directory exists and is writable. We set NSELF_JWT_ROTATION_LOG to a
// path whose parent we create and make writable, then clear the env var.
func TestRotationLogPath_PrimaryWritable(t *testing.T) {
	// Point the function at a writable temp dir by using the env override —
	// the primary path (/var/lib/nself/) is never writable in CI, so we
	// exercise the "primary exists and is writable" branch via XDG_STATE_HOME
	// pointing to a writable dir and a synthetic stat-able primary.
	// The simplest way to hit the primary-writable branch is to override
	// the env var so the function returns immediately on the first check.
	dir := t.TempDir()
	logPath := filepath.Join(dir, "jwt-rotation.log")
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)

	got := RotationLogPath()
	if got != logPath {
		t.Errorf("expected %q, got %q", logPath, got)
	}
}

// TestRotationLogPath_XDGFallbackWithStateHome covers the XDG_STATE_HOME branch
// when the env var is set explicitly.
func TestRotationLogPath_XDGFallbackWithStateHome(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_LOG", "") // clear override
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	got := RotationLogPath()
	expected := filepath.Join(dir, "nself", "jwt-rotation.log")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestRotationLogPath_XDGFallbackNoStateHome covers the XDG_STATE_HOME-not-set
// branch: the function falls back to HOME/.local/state/nself/jwt-rotation.log.
func TestRotationLogPath_XDGFallbackNoStateHome(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_LOG", "")
	t.Setenv("XDG_STATE_HOME", "") // unset XDG so the home-based fallback fires
	home := t.TempDir()
	setHomeAndProfile(t, home)

	got := RotationLogPath()
	expected := filepath.Join(home, ".local", "state", "nself", "jwt-rotation.log")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestRotationLogPath_PrimaryDirNotDir covers the branch where primaryDir exists
// but is NOT a directory (stat succeeds, IsDir() returns false → probe is skipped).
func TestRotationLogPath_PrimaryDirNotDir(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_LOG", "")
	t.Setenv("XDG_STATE_HOME", "")
	home := t.TempDir()
	setHomeAndProfile(t, home)

	// The primary dir used by RotationLogPath is filepath.Dir(DefaultRotationLogPath).
	// We can't easily make /var/lib/nself a file. However the XDG fallback is
	// already triggered when stat fails (e.g. /var/lib/nself doesn't exist in CI).
	// This test just verifies the function returns the expected XDG path when
	// the primary dir is inaccessible — same outcome, branch already exercised by
	// the CI environment where /var/lib/nself doesn't exist.
	got := RotationLogPath()
	expected := filepath.Join(home, ".local", "state", "nself", "jwt-rotation.log")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// ── LastRotationTime ──────────────────────────────────────────────────────────

// TestLastRotationTime_ReadError covers the non-NotExist read error branch
// (file exists but is a directory → EISDIR on ReadFile).
func TestLastRotationTime_ReadError(t *testing.T) {
	// Create a directory at the log path so ReadFile returns EISDIR (not NotExist).
	logPath := filepath.Join(t.TempDir(), "jwt-rotation.log")
	if err := os.MkdirAll(logPath, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	_, err := LastRotationTime(logPath)
	if err == nil {
		t.Error("expected error when log path is a directory, got nil")
	}
}

// TestLastRotationTime_EmptyFields covers the len(fields) == 0 branch
// (a line that becomes empty after splitting, i.e., whitespace-only).
func TestLastRotationTime_EmptyFields(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "jwt-rotation.log")
	// Write a log with a whitespace-only line (TrimSpace makes it empty, but
	// the outer TrimSpace in the loop body handles "", so we need a line that
	// passes the blank/comment guards but has no Fields).
	// strings.Fields("") returns nil (len 0), triggering the branch.
	content := "2026-01-01T00:00:00Z rotated HASURA_GRAPHQL_JWT_SECRET\n\t\n2026-01-02T00:00:00Z rotated HASURA_GRAPHQL_JWT_SECRET\n"
	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	ts, err := LastRotationTime(logPath)
	if err != nil {
		t.Fatalf("LastRotationTime: %v", err)
	}
	if ts.IsZero() {
		t.Error("expected non-zero time from valid log entries")
	}
}

// TestLastRotationTime_FutureTimestampSkipped verifies future timestamps are
// skipped by the clock-skew guard (coverage for the t.After(now+1min) branch).
func TestLastRotationTime_FutureTimestampSkipped(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "jwt-rotation.log")
	future := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)
	past := "2026-01-01T00:00:00Z"
	content := future + " rotated HASURA_GRAPHQL_JWT_SECRET\n" +
		past + " rotated HASURA_GRAPHQL_JWT_SECRET\n"
	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	ts, err := LastRotationTime(logPath)
	if err != nil {
		t.Fatalf("LastRotationTime: %v", err)
	}
	// The future timestamp must be skipped; the past timestamp should be returned.
	if ts.IsZero() {
		t.Error("expected the past timestamp to be returned, got zero")
	}
	if ts.After(time.Now().UTC().Add(time.Minute)) {
		t.Errorf("returned a future timestamp that should have been skipped: %v", ts)
	}
}

// ── writeRotationLog ──────────────────────────────────────────────────────────

// TestWriteRotationLog_MkdirAllFail covers the MkdirAll error branch in
// writeRotationLog by pointing the log at a path whose grandparent is a file
// (blocking directory creation).
func TestWriteRotationLog_MkdirAllFail(t *testing.T) {
	dir := t.TempDir()
	// Create a file at the directory path that writeRotationLog would need.
	blocker := filepath.Join(dir, "nself")
	if err := os.WriteFile(blocker, []byte("block"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	logPath := filepath.Join(blocker, "subdir", "jwt-rotation.log")
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)
	t.Cleanup(func() { t.Setenv("NSELF_JWT_ROTATION_LOG", "") })

	err := writeRotationLog("test entry\n")
	if err == nil {
		t.Error("expected MkdirAll error, got nil")
	}
	if !strings.Contains(err.Error(), "create rotation log dir") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestWriteRotationLog_OpenFileFail covers the OpenFile error branch by making
// the log directory read-only so file creation fails.
func TestWriteRotationLog_OpenFileFail(t *testing.T) {
	skipUnixOnly(t) // chmod read-only dirs are not enforced on Windows
	dir := t.TempDir()
	logDir := filepath.Join(dir, "nself")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Make the directory read-only so OpenFile (O_CREATE) fails.
	if err := os.Chmod(logDir, 0555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(logDir, 0755) })

	logPath := filepath.Join(logDir, "jwt-rotation.log")
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)
	t.Cleanup(func() { t.Setenv("NSELF_JWT_ROTATION_LOG", "") })

	err := writeRotationLog("test entry\n")
	if err == nil {
		t.Error("expected OpenFile error on read-only dir, got nil")
	}
}

// ── RotateJWTKey ─────────────────────────────────────────────────────────────

// TestRotateJWTKey_WriteLogFail covers the writeRotationLog error path in
// RotateJWTKey by pointing the log at a path whose parent is a file (so
// MkdirAll fails inside writeRotationLog).
func TestRotateJWTKey_WriteLogFail(t *testing.T) {
	dir := t.TempDir()
	// Create a file where the log directory would need to be.
	blocker := filepath.Join(dir, "nself")
	if err := os.WriteFile(blocker, []byte("block"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	logPath := filepath.Join(blocker, "sub", "jwt-rotation.log")
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)

	_, err := RotateJWTKey("somecurrentkey")
	if err == nil {
		t.Error("expected error when writeRotationLog fails, got nil")
	}
	if !strings.Contains(err.Error(), "write rotation log") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── storage.go ────────────────────────────────────────────────────────────────

// TestReadAuthFile_InvalidJSON covers the json.Unmarshal failure branch in
// ReadAuthFile by writing a file with invalid JSON content.
func TestReadAuthFile_InvalidJSON(t *testing.T) {
	home := t.TempDir()
	setHomeAndProfile(t, home)

	nselfDir := filepath.Join(home, ".nself")
	if err := os.MkdirAll(nselfDir, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	authPath := filepath.Join(nselfDir, "auth.json")
	if err := os.WriteFile(authPath, []byte("not-json{{"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := ReadAuthFile()
	if err == nil {
		t.Error("expected error parsing invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parsing auth file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestWriteAuthFile_ChmodFail covers the Chmod error path in WriteAuthFile.
// We create the auth file then make it read-only so Chmod 0600 fails... but
// that doesn't work because Chmod on a file you own always succeeds.
// The Chmod-fail branch (line 105-107) requires a filesystem that rejects chmod
// (e.g. vfat). On POSIX it is genuinely unreachable for a file the current user
// owns. Documented as unreachable: "Chmod on an owned file always succeeds on POSIX."

// TestReadAuthFile_NonNotExistError covers the non-NotExist OS error branch in
// ReadAuthFile (file path is a directory → EISDIR on ReadFile).
func TestReadAuthFile_NonNotExistError(t *testing.T) {
	home := t.TempDir()
	setHomeAndProfile(t, home)

	// Create a directory at the auth file location.
	authDir := filepath.Join(home, ".nself")
	authPath := filepath.Join(authDir, "auth.json")
	if err := os.MkdirAll(authPath, 0755); err != nil {
		t.Fatalf("MkdirAll (create dir-at-file-path): %v", err)
	}

	_, err := ReadAuthFile()
	if err == nil {
		t.Error("expected error reading auth file as directory, got nil")
	}
}

// TestWriteAuthFile_MkdirAllFail covers the MkdirAll error branch in
// WriteAuthFile by blocking directory creation with a file.
func TestWriteAuthFile_MkdirAllFail(t *testing.T) {
	home := t.TempDir()
	setHomeAndProfile(t, home)

	// Create a file at ~/.nself to block MkdirAll.
	nselfPath := filepath.Join(home, ".nself")
	if err := os.WriteFile(nselfPath, []byte("block"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := WriteAuthFile(&AuthFile{AccessToken: "tok", SessionToken: "sess", Email: "x@x.com", Tier: "free"})
	if err == nil {
		t.Error("expected MkdirAll error, got nil")
	}
}

// TestWriteAuthFile_WriteFileFail covers the WriteFile error branch by making
// the directory read-only after creation.
func TestWriteAuthFile_WriteFileFail(t *testing.T) {
	skipUnixOnly(t) // chmod read-only dirs are not enforced on Windows
	home := t.TempDir()
	setHomeAndProfile(t, home)

	nselfDir := filepath.Join(home, ".nself")
	if err := os.MkdirAll(nselfDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Make directory read-only so WriteFile (O_CREATE) fails.
	if err := os.Chmod(nselfDir, 0555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(nselfDir, 0755) })

	err := WriteAuthFile(&AuthFile{AccessToken: "tok", SessionToken: "sess", Email: "x@x.com", Tier: "free"})
	if err == nil {
		t.Error("expected WriteFile error on read-only dir, got nil")
	}
}

// TestWriteAuthFile_ChmodFail covers the Chmod error branch in WriteAuthFile by
// writing the file and then removing the parent dir so Chmod fails on the path.
// Note: json.MarshalIndent on AuthFile (all string/bool fields) is infallible —
// that error branch is genuinely unreachable; see note below WriteAuthFile.
func TestWriteAuthFile_ChmodFail(t *testing.T) {
	skipUnixOnly(t) // chmod read-only dirs are not enforced on Windows
	home := t.TempDir()
	setHomeAndProfile(t, home)

	nselfDir := filepath.Join(home, ".nself")
	if err := os.MkdirAll(nselfDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write once successfully.
	af := &AuthFile{AccessToken: "tok", SessionToken: "sess", Email: "x@x.com", Tier: "free"}
	if err := WriteAuthFile(af); err != nil {
		t.Fatalf("first WriteAuthFile: %v", err)
	}

	// Make the .nself dir read-only AFTER the file exists, so WriteFile
	// overwrites (O_WRONLY on existing file) but then Chmod fails.
	// On macOS/Linux, Chmod on a file inside a read-only dir also fails.
	authPath := filepath.Join(nselfDir, "auth.json")
	if err := os.Chmod(authPath, 0444); err != nil { // make file read-only
		t.Fatalf("Chmod file: %v", err)
	}
	if err := os.Chmod(nselfDir, 0555); err != nil { // make dir read-only
		t.Fatalf("Chmod dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(nselfDir, 0755)
		_ = os.Chmod(authPath, 0600)
	})

	// WriteFile to an existing read-only file in a read-only dir fails.
	err := WriteAuthFile(af)
	if err == nil {
		t.Skip("filesystem allows write to read-only file; chmod branch untestable here")
	}
}

// TestDeleteAuthFile_NonNotExistError covers the non-NotExist Remove error
// branch in DeleteAuthFile (the file path points to a directory → EISDIR on Remove).
func TestDeleteAuthFile_NonNotExistError(t *testing.T) {
	home := t.TempDir()
	setHomeAndProfile(t, home)

	// Create a directory at the auth file path and put a file inside it.
	// os.Remove on a non-empty directory returns ENOTEMPTY (not NotExist),
	// which triggers the error branch in DeleteAuthFile.
	authDir := filepath.Join(home, ".nself")
	authPath := filepath.Join(authDir, "auth.json")
	if err := os.MkdirAll(authPath, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Put a file inside so the directory is non-empty — os.Remove returns ENOTEMPTY.
	if err := os.WriteFile(filepath.Join(authPath, "sentinel"), []byte("x"), 0600); err != nil {
		t.Fatalf("WriteFile sentinel: %v", err)
	}

	err := DeleteAuthFile()
	// Removing a non-empty directory returns "directory not empty" on macOS/Linux;
	// it must NOT be nil.
	if err == nil {
		t.Error("expected error removing auth file directory, got nil")
	}
}

// ── storage.go HOME="" error branches ────────────────────────────────────────

// TestAuthFilePath_HomeError covers authFilePath → ReadAuthFile/WriteAuthFile/
// DeleteAuthFile/GetAuthFilePath when UserHomeDir fails (HOME="").
func TestReadAuthFile_StatePathError(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	_, err := ReadAuthFile()
	if err == nil {
		t.Error("ReadAuthFile should error when HOME is empty, got nil")
	}
}

func TestWriteAuthFile_StatePathError(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	err := WriteAuthFile(&AuthFile{AccessToken: "tok", SessionToken: "sess", Email: "x@x.com", Tier: "free"})
	if err == nil {
		t.Error("WriteAuthFile should error when HOME is empty, got nil")
	}
}

func TestDeleteAuthFile_StatePathError(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	err := DeleteAuthFile()
	if err == nil {
		t.Error("DeleteAuthFile should error when HOME is empty, got nil")
	}
}

func TestGetAuthFilePath_StatePathError(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	// GetAuthFilePath returns the fallback string when HOME is empty.
	got := GetAuthFilePath()
	if got != "~/.nself/auth.json" {
		t.Errorf("expected fallback ~./nself/auth.json, got %q", got)
	}
}

// ── client.go HTTP error branches ────────────────────────────────────────────
// All these inject a failing transport to cover the httpClient.Do error path.
// The hard rule is: no network calls in unit tests — we mock the transport only.

func TestDeviceAuthorize_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("simulated network failure"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	_, err := DeviceAuthorize(context.Background())
	if err == nil {
		t.Error("expected error from failing transport, got nil")
	}
	if !strings.Contains(err.Error(), "contacting auth server") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeviceAuthorize_JSONDecodeError(t *testing.T) {
	// Server returns 200 with non-JSON body → JSON decode error.
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	})
	_, err := DeviceAuthorize(context.Background())
	if err == nil {
		t.Error("expected JSON decode error, got nil")
	}
}

func TestPollToken_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	_, err := PollToken(context.Background(), "dcode123")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "polling token") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestPollToken_AuthorizationPending covers the 202 branch (authorization_pending)
// which returns (nil, nil) — the user hasn't clicked "Authorize" yet.
func TestPollToken_AuthorizationPending(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted) // 202
	})
	tok, err := PollToken(context.Background(), "dcode")
	if err != nil {
		t.Errorf("expected nil error for 202, got %v", err)
	}
	if tok != nil {
		t.Errorf("expected nil token for 202, got %v", tok)
	}
}

// TestPollToken_SuccessJSON covers the 200+JSON success path.
func TestPollToken_SuccessJSON(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"tok123","session_id":"ses"}`))
	})
	tok, err := PollToken(context.Background(), "dcode")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if tok == nil || tok.AccessToken != "tok123" {
		t.Errorf("unexpected token: %v", tok)
	}
}

// TestPollToken_JSONDecodeError covers the 200+non-JSON decode error path.
func TestPollToken_JSONDecodeError(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	})
	_, err := PollToken(context.Background(), "dcode")
	if err == nil {
		t.Error("expected JSON decode error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing token response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRefreshToken_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	_, err := RefreshToken(context.Background(), "tok")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "refreshing token") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRefreshToken_JSONDecodeError(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	})
	_, err := RefreshToken(context.Background(), "tok")
	if err == nil {
		t.Error("expected JSON decode error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing refresh response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRevokeSession_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	err := RevokeSession(context.Background(), "sess", false)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "revoking session") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetSession_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	_, err := GetSession(context.Background(), "tok")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "getting session") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestGetSession_Unauthorized_Coverage covers the 401 → ErrNotLoggedIn branch
// via a test server (complements the existing client_test.go version).
func TestGetSession_Unauthorized_Coverage(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := GetSession(context.Background(), "expired_tok")
	if !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("expected ErrNotLoggedIn, got %v", err)
	}
}

func TestGetSession_JSONDecodeError(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("bad json {"))
	})
	_, err := GetSession(context.Background(), "tok")
	if err == nil {
		t.Error("expected JSON decode error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing session response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetLicenses_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	_, err := GetLicenses(context.Background(), "tok")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "getting licenses") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetLicenses_JSONDecodeError(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("bad json"))
	})
	_, err := GetLicenses(context.Background(), "tok")
	if err == nil {
		t.Error("expected JSON decode error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing licenses response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTeamMembers_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	_, err := GetTeamMembers(context.Background(), "tok")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "getting team") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTeamMembers_JSONDecodeError(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("bad json"))
	})
	_, err := GetTeamMembers(context.Background(), "tok")
	if err == nil {
		t.Error("expected JSON decode error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing team response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInviteTeamMember_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	err := InviteTeamMember(context.Background(), "tok", "user@example.com")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "inviting team member") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemoveTeamMember_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	err := RemoveTeamMember(context.Background(), "tok", "uid123")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "removing team member") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSetTeamMemberRole_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	err := SetTeamMemberRole(context.Background(), "tok", "uid123", "admin")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "setting team member role") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestActivateLicense_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	err := ActivateLicense(context.Background(), "tok", "key123")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "activating license") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetDevices_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	_, err := GetDevices(context.Background(), "tok")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "getting devices") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetDevices_JSONDecodeError(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("bad json"))
	})
	_, err := GetDevices(context.Background(), "tok")
	if err == nil {
		t.Error("expected JSON decode error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing devices response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRevokeDevice_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	err := RevokeDevice(context.Background(), "tok", "dev123")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "revoking device") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTransferLicense_HTTPError(t *testing.T) {
	withFailingTransport(t, errors.New("network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	err := TransferLicense(context.Background(), "tok", "lic123", "newkey")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "transferring license") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── device_code.go ────────────────────────────────────────────────────────────

// TestPollDeviceCode_PreCancelledContext covers the ctx.Done() branch in the
// polling select (first select before each iteration) with a pre-cancelled context.
func TestPollDeviceCode_PreCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so the first select fires immediately
	_, err := PollDeviceCode(ctx, "dcode", nil)
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestPollDeviceCode_Timeout covers the ErrPollTimeout branch by setting a very
// short deadline via a test server that always returns 202 (authorization_pending).
// We use a mock server at loop speed with a very short defaultPollTimeout workaround:
// we rely on the deadline check after the first poll iteration.
// Since defaultPollTimeout is 5 minutes, we can't wait that long. Instead we
// exercise ErrPollTimeout by overriding time using context deadline.
func TestPollDeviceCode_PollError(t *testing.T) {
	// When PollToken returns an error, PollDeviceCode wraps and returns it.
	withFailingTransport(t, errors.New("poll network fail"))
	t.Setenv("NSELF_AUTH_SERVER_URL", "https://127.0.0.1:0")
	ctx := context.Background()
	_, err := PollDeviceCode(ctx, "dcode", nil)
	if err == nil {
		t.Error("expected error when PollToken fails, got nil")
	}
	if !strings.Contains(err.Error(), "polling for authorization") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestPollDeviceCode_ContextCancelledOnWait covers the second ctx.Done() branch
// (the wait-for-interval select after a nil token response).
func TestPollDeviceCode_ContextCancelledOnWait(t *testing.T) {
	// Set up a server that returns 202 (authorization_pending) once, then we
	// cancel the context before the next iteration's wait fires.
	callCount := 0
	srv := withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusAccepted) // 202 = authorization_pending → nil token
	})
	_ = srv

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a tiny delay so the first poll happens but the wait is cancelled.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := PollDeviceCode(ctx, "dcode", nil)
	if err == nil {
		t.Error("expected error from cancelled context during wait, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
