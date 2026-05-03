package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCheckJWTRotation_NoJWTSecret verifies JWT-ROT-01 fails when there is
// no JWT secret configured at all (delegates to CheckJWTSecretPresent).
func TestCheckJWTRotation_NoJWTSecret(t *testing.T) {
	dir := t.TempDir()
	// No env files at all — secret is missing.
	result := CheckJWTRotation(dir)
	if result.Status != "fail" {
		t.Errorf("CheckJWTRotation with missing secret: status = %q, want fail", result.Status)
	}
	if result.Name != "JWT-ROT-01" {
		t.Errorf("CheckJWTRotation Name = %q, want JWT-ROT-01", result.Name)
	}
	if result.Section != "security" {
		t.Errorf("CheckJWTRotation Section = %q, want security", result.Section)
	}
}

// TestCheckJWTRotation_NoLog verifies JWT-ROT-01 warns when the rotation log
// does not exist (key is set but was never rotated).
func TestCheckJWTRotation_NoLog(t *testing.T) {
	dir := t.TempDir()

	// Write JWT secret so the secret-present check passes.
	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// Point rotation log to a non-existent path.
	logPath := filepath.Join(dir, "does-not-exist", "jwt-rotation.log")
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)

	result := CheckJWTRotation(dir)
	if result.Status != "warn" {
		t.Errorf("CheckJWTRotation no log: status = %q, want warn", result.Status)
	}
	if result.FixCmd != "nself self-heal --jwt" {
		t.Errorf("CheckJWTRotation no log FixCmd = %q, want 'nself self-heal --jwt'", result.FixCmd)
	}
}

// TestCheckJWTRotation_EmptyLog verifies JWT-ROT-01 warns when the log exists
// but has no recorded rotation entries.
func TestCheckJWTRotation_EmptyLog(t *testing.T) {
	dir := t.TempDir()

	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(dir, "jwt-rotation.log")
	if err := os.WriteFile(logPath, []byte("# no entries yet\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)

	result := CheckJWTRotation(dir)
	if result.Status != "warn" {
		t.Errorf("CheckJWTRotation empty log: status = %q, want warn", result.Status)
	}
}

// TestCheckJWTRotation_WithinWindow verifies JWT-ROT-01 passes when the last
// rotation is recent (within the configured window).
func TestCheckJWTRotation_WithinWindow(t *testing.T) {
	dir := t.TempDir()

	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// Write a log entry dated 10 days ago.
	recentTS := time.Now().UTC().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	logEntry := fmt.Sprintf("%s rotated HASURA_GRAPHQL_JWT_SECRET (grace period ends ...)\n", recentTS)
	logPath := filepath.Join(dir, "jwt-rotation.log")
	if err := os.WriteFile(logPath, []byte(logEntry), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "90")

	result := CheckJWTRotation(dir)
	if result.Status != "pass" {
		t.Errorf("CheckJWTRotation within window: status = %q, want pass (msg=%q)", result.Status, result.Message)
	}
}

// TestCheckJWTRotation_ExpiredWindow verifies JWT-ROT-01 warns when the last
// rotation exceeds the configured window.
func TestCheckJWTRotation_ExpiredWindow(t *testing.T) {
	dir := t.TempDir()

	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// Log entry dated 100 days ago — past the 90-day window.
	oldTS := time.Now().UTC().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	logEntry := fmt.Sprintf("%s rotated HASURA_GRAPHQL_JWT_SECRET (grace period ends ...)\n", oldTS)
	logPath := filepath.Join(dir, "jwt-rotation.log")
	if err := os.WriteFile(logPath, []byte(logEntry), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "90")

	result := CheckJWTRotation(dir)
	if result.Status != "warn" {
		t.Errorf("CheckJWTRotation expired window: status = %q, want warn", result.Status)
	}
	if result.FixCmd != "nself self-heal --jwt" {
		t.Errorf("CheckJWTRotation expired window FixCmd = %q, want 'nself self-heal --jwt'", result.FixCmd)
	}
}

// TestCheckJWTRotation_CustomWindow verifies JWT-ROT-01 respects a custom window.
func TestCheckJWTRotation_CustomWindow(t *testing.T) {
	dir := t.TempDir()

	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// Log entry dated 25 days ago — within 90d but past a custom 20d window.
	recentTS := time.Now().UTC().Add(-25 * 24 * time.Hour).Format(time.RFC3339)
	logEntry := fmt.Sprintf("%s rotated HASURA_GRAPHQL_JWT_SECRET (grace period ends ...)\n", recentTS)
	logPath := filepath.Join(dir, "jwt-rotation.log")
	if err := os.WriteFile(logPath, []byte(logEntry), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "20")

	result := CheckJWTRotation(dir)
	if result.Status != "warn" {
		t.Errorf("CheckJWTRotation custom window (20d, age 25d): status = %q, want warn", result.Status)
	}
}

// TestCheckJWTRotation_SectionAndName verifies the section and name fields are
// always set correctly regardless of the outcome.
func TestCheckJWTRotation_SectionAndName(t *testing.T) {
	dir := t.TempDir()

	result := CheckJWTRotation(dir)
	if result.Section != "security" {
		t.Errorf("CheckJWTRotation Section = %q, want security", result.Section)
	}
	if result.Name != "JWT-ROT-01" {
		t.Errorf("CheckJWTRotation Name = %q, want JWT-ROT-01", result.Name)
	}
}

// TestCheckJWTRotation_HardDaysEscalateToFail verifies that JWT-ROT-01 returns
// fail (not warn) when age exceeds NSELF_JWT_ROTATION_HARD_DAYS.
func TestCheckJWTRotation_HardDaysEscalateToFail(t *testing.T) {
	dir := t.TempDir()

	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// Log entry dated 200 days ago — past even the hard limit.
	oldTS := time.Now().UTC().Add(-200 * 24 * time.Hour).Format(time.RFC3339)
	logEntry := fmt.Sprintf("%s rotated HASURA_GRAPHQL_JWT_SECRET (grace period ends ...)\n", oldTS)
	logPath := filepath.Join(dir, "jwt-rotation.log")
	if err := os.WriteFile(logPath, []byte(logEntry), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "90")
	t.Setenv("NSELF_JWT_ROTATION_HARD_DAYS", "180")

	result := CheckJWTRotation(dir)
	if result.Status != "fail" {
		t.Errorf("CheckJWTRotation hard limit exceeded: status = %q, want fail", result.Status)
	}
}

// TestCheckJWTRotation_HardDaysDefault verifies the default hard limit is 2×
// the window (180 days when window=90).
func TestCheckJWTRotation_HardDaysDefault(t *testing.T) {
	dir := t.TempDir()

	content := `HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"abc"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	// 100 days ago: past window (90d) but not hard limit (180d) — should warn, not fail.
	ts := time.Now().UTC().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	logEntry := fmt.Sprintf("%s rotated HASURA_GRAPHQL_JWT_SECRET (grace period ends ...)\n", ts)
	logPath := filepath.Join(dir, "jwt-rotation.log")
	if err := os.WriteFile(logPath, []byte(logEntry), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NSELF_JWT_ROTATION_LOG", logPath)
	t.Setenv("NSELF_JWT_ROTATION_WINDOW_DAYS", "90")
	t.Setenv("NSELF_JWT_ROTATION_HARD_DAYS", "")

	result := CheckJWTRotation(dir)
	if result.Status != "warn" {
		t.Errorf("CheckJWTRotation within hard limit (default 2×): status = %q, want warn", result.Status)
	}
}
