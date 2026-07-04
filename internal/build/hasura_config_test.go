package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// TestRenderHasuraCLIConfig_UsesResolvedValues verifies the rendered
// config.yaml reflects the actual resolved Hasura port and admin_secret
// (gap #10: this replaces a hardcoded placeholder that silently drifted
// from the real project config).
func TestRenderHasuraCLIConfig_UsesResolvedValues(t *testing.T) {
	cfg := &config.Config{}
	cfg.Hasura.Port = 8080
	cfg.Hasura.AdminSecret = "test-admin-secret"

	out := RenderHasuraCLIConfig(cfg)
	content := string(out)

	if !strings.Contains(content, "endpoint: http://localhost:8080") {
		t.Errorf("expected endpoint to use the resolved Hasura port; got:\n%s", content)
	}
	if !strings.Contains(content, `admin_secret: "test-admin-secret"`) {
		t.Errorf("expected admin_secret to use the resolved value; got:\n%s", content)
	}
	if !strings.Contains(content, "version: 3") {
		t.Errorf("expected Hasura CLI config schema version 3; got:\n%s", content)
	}
}

// TestRenderHasuraCLIConfig_DefaultsPortWhenZero verifies the fallback to
// port 8080 when cfg.Hasura.Port is unset, matching hasuraMetadataURL's
// existing fallback convention (internal/database/hasura.go).
func TestRenderHasuraCLIConfig_DefaultsPortWhenZero(t *testing.T) {
	cfg := &config.Config{}
	out := RenderHasuraCLIConfig(cfg)
	if !strings.Contains(string(out), "http://localhost:8080") {
		t.Errorf("expected default port 8080 fallback; got:\n%s", out)
	}
}

// TestRenderHasuraCLIConfig_UsesFunctionsPortForActions verifies the actions
// handler base URL points at the functions service port when configured,
// rather than always pointing back at Hasura's own port.
func TestRenderHasuraCLIConfig_UsesFunctionsPortForActions(t *testing.T) {
	cfg := &config.Config{}
	cfg.Hasura.Port = 8080
	cfg.Functions.Port = 3008

	out := RenderHasuraCLIConfig(cfg)
	if !strings.Contains(string(out), "handler_webhook_baseurl: http://localhost:3008") {
		t.Errorf("expected actions handler_webhook_baseurl to use FUNCTIONS_PORT; got:\n%s", out)
	}
}

// TestWriteHasuraCLIConfig_WritesFileWithRestrictivePerms verifies the file
// is written to {workdir}/hasura/config.yaml with 0600 permissions (it
// contains admin_secret) and that the directory is created if absent.
func TestWriteHasuraCLIConfig_WritesFileWithRestrictivePerms(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Hasura.Port = 8080
	cfg.Hasura.AdminSecret = "s3cr3t"

	n, err := WriteHasuraCLIConfig(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 file written, got %d", n)
	}

	path := filepath.Join(dir, "hasura", "config.yaml")
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatalf("expected hasura/config.yaml to exist: %v", statErr)
	}
	// Unix permission bits don't map onto NTFS ACLs — Windows reports
	// 0666/0444 — so the 0600 assertion only holds on POSIX systems.
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm != 0o600 {
		t.Errorf("expected 0600 permissions (file contains admin_secret), got %o", perm)
	}
}

// TestWriteHasuraCLIConfig_IdempotentOverwrite verifies re-running the
// writer (as `nself build` does on every rebuild) produces byte-identical
// output for unchanged config, and correctly reflects a changed value on
// the next run — i.e. it isn't a one-time scaffold that then goes stale.
func TestWriteHasuraCLIConfig_IdempotentOverwrite(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Hasura.Port = 8080
	cfg.Hasura.AdminSecret = "first-secret"

	if _, err := WriteHasuraCLIConfig(dir, cfg); err != nil {
		t.Fatalf("first write: %v", err)
	}
	path := filepath.Join(dir, "hasura", "config.yaml")
	first, _ := os.ReadFile(path)

	// Re-running with identical config must produce byte-identical output.
	if _, err := WriteHasuraCLIConfig(dir, cfg); err != nil {
		t.Fatalf("second write: %v", err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Errorf("expected idempotent output for unchanged config;\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	// Changing the admin secret (e.g. rotated in .env) and rebuilding must
	// update the file rather than leaving a stale value on disk — this is
	// the actual bug being fixed (gap #10: a static, drifting placeholder).
	cfg.Hasura.AdminSecret = "rotated-secret"
	if _, err := WriteHasuraCLIConfig(dir, cfg); err != nil {
		t.Fatalf("third write: %v", err)
	}
	third, _ := os.ReadFile(path)
	if !strings.Contains(string(third), "rotated-secret") {
		t.Errorf("expected rotated admin_secret to be reflected on rebuild; got:\n%s", third)
	}
	if strings.Contains(string(third), "first-secret") {
		t.Errorf("did not expect the stale admin_secret to survive a rebuild; got:\n%s", third)
	}
}
