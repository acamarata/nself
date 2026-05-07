// Package auth — supplemental tests to bring coverage to 80%+ (S10.T08).
// Targets: RotationHardDays (0%), RotationLogPath write-probe, WriteAuthFile
// error paths, GetAuthFilePath fallback, and client error branches not already
// covered in client_test.go.
package auth

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ─── RotationHardDays ─────────────────────────────────────────────────────────

// TestRotationHardDays_Default verifies the default is 2× RotationWindowDays.
func TestRotationHardDays_Default(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_HARD_DAYS", "")
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "")
	want := DefaultRotationWindowDays * DefaultRotationHardMultiplier
	if got := RotationHardDays(); got != want {
		t.Errorf("RotationHardDays() = %d, want %d", got, want)
	}
}

// TestRotationHardDays_Override verifies the env var override is respected.
func TestRotationHardDays_Override(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_HARD_DAYS", "365")
	if got := RotationHardDays(); got != 365 {
		t.Errorf("RotationHardDays() = %d, want 365", got)
	}
}

// TestRotationHardDays_Malformed verifies malformed input falls back to default.
func TestRotationHardDays_Malformed(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_HARD_DAYS", "not-a-number")
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "")
	want := DefaultRotationWindowDays * DefaultRotationHardMultiplier
	if got := RotationHardDays(); got != want {
		t.Errorf("RotationHardDays() with malformed env = %d, want default %d", got, want)
	}
}

// TestRotationHardDays_Zero verifies zero input falls back to default.
func TestRotationHardDays_Zero(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_HARD_DAYS", "0")
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "")
	want := DefaultRotationWindowDays * DefaultRotationHardMultiplier
	if got := RotationHardDays(); got != want {
		t.Errorf("RotationHardDays() with zero = %d, want default %d", got, want)
	}
}

// TestRotationHardDays_Negative verifies negative input falls back to default.
func TestRotationHardDays_Negative(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_HARD_DAYS", "-5")
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "")
	want := DefaultRotationWindowDays * DefaultRotationHardMultiplier
	if got := RotationHardDays(); got != want {
		t.Errorf("RotationHardDays() with negative = %d, want default %d", got, want)
	}
}

// TestRotationHardDays_RespectsWindowOverride verifies RotationHardDays uses
// the effective RotationWindowDays when computing the fallback.
func TestRotationHardDays_RespectsWindowOverride(t *testing.T) {
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "30")
	t.Setenv("NSELF_JWT_ROTATION_HARD_DAYS", "")
	want := 30 * DefaultRotationHardMultiplier
	if got := RotationHardDays(); got != want {
		t.Errorf("RotationHardDays() with window=30 = %d, want %d", got, want)
	}
}

// ─── RotationLogPath — env override used as primary ───────────────────────────

// TestRotationLogPath_EnvPathReturned verifies the env override is returned
// immediately without any directory probe.
func TestRotationLogPath_EnvPathReturned(t *testing.T) {
	customLog := filepath.Join(t.TempDir(), "custom-jwt.log")
	t.Setenv("NSELF_JWT_ROTATION_LOG", customLog)

	got := RotationLogPath()
	if got != customLog {
		t.Errorf("RotationLogPath() = %q, want %q", got, customLog)
	}
}

// TestRotationLogPath_XDGFallback verifies the XDG_STATE_HOME fallback is used
// when the primary path's parent does not exist.
func TestRotationLogPath_XDGFallbackDirMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows path handling differs")
	}
	t.Setenv("NSELF_JWT_ROTATION_LOG", "")
	xdgDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdgDir)

	got := RotationLogPath()
	if got == "" {
		t.Error("RotationLogPath() returned empty string")
	}
	// When /var/lib/nself is absent (dev machine), must use XDG path.
	if _, err := os.Stat("/var/lib/nself"); err != nil {
		want := filepath.Join(xdgDir, "nself", "jwt-rotation.log")
		if got != want {
			t.Errorf("RotationLogPath() XDG fallback = %q, want %q", got, want)
		}
	}
}

// ─── WriteAuthFile error path — read-only parent directory ───────────────────

// TestWriteAuthFile_ReadOnlyDir verifies WriteAuthFile returns an error when
// the parent directory cannot be created (permissions denied).
func TestWriteAuthFile_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("permission test requires non-root Unix")
	}

	// Create a read-only parent so MkdirAll fails for the .nself sub-dir.
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) }) // restore for TempDir cleanup

	t.Setenv("HOME", parent)

	af := &AuthFile{AccessToken: "tok"}
	err := WriteAuthFile(af)
	if err == nil {
		t.Error("WriteAuthFile() in read-only dir should return error, got nil")
	}
}

// ─── DeleteAuthFile error path — unresolvable HOME ───────────────────────────

// TestDeleteAuthFile_HomeError verifies DeleteAuthFile propagates an error
// when authFilePath() cannot determine the home directory. Works on Linux where
// os.UserHomeDir reads $HOME; macOS falls back to getpwuid which usually works.
func TestDeleteAuthFile_HomeError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("HOME empty only causes UserHomeDir failure on Linux")
	}
	t.Setenv("HOME", "")

	err := DeleteAuthFile()
	if err == nil {
		t.Error("DeleteAuthFile() with no HOME should return error")
	}
}

// ─── GetAuthFilePath — fallback literal ──────────────────────────────────────

// TestGetAuthFilePath_Fallback verifies the fallback literal is returned when
// HOME cannot be resolved on Linux.
func TestGetAuthFilePath_Fallback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("HOME empty only causes UserHomeDir failure on Linux")
	}
	t.Setenv("HOME", "")

	got := GetAuthFilePath()
	if got != "~/.nself/auth.json" {
		t.Errorf("GetAuthFilePath() fallback = %q, want ~/.nself/auth.json", got)
	}
}

// ─── parseAPIError — malformed JSON body ─────────────────────────────────────

// TestParseAPIError_MalformedBody verifies parseAPIError falls back to a plain
// error message when the response body is not valid JSON.
func TestParseAPIError_MalformedBody(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway — not JSON"))
	})
	defer cleanup()

	_, err := DeviceAuthorize(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "502") {
		t.Errorf("expected HTTP 502 in error message, got: %v", err)
	}
}

// ─── GetTeamMembers — server error ───────────────────────────────────────────

func TestGetTeamMembers_ServerError(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "upstream_error",
			"message": "team service unavailable",
		})
	})
	defer cleanup()

	_, err := GetTeamMembers(context.Background(), "tok")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── GetDevices — server error ────────────────────────────────────────────────

func TestGetDevices_ServerError(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "service_unavailable",
			"message": "device service down",
		})
	})
	defer cleanup()

	_, err := GetDevices(context.Background(), "tok")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── RefreshToken — server error (additional error code) ─────────────────────

func TestRefreshToken_ServerError(t *testing.T) {
	cleanup := newAuthServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "service_unavailable",
			"message": "auth service offline",
		})
	})
	defer cleanup()

	_, err := RefreshToken(context.Background(), "tok")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
