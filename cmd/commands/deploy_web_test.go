package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWebAppsCanonical verifies the webApps list is non-empty and contains
// the known core apps; callers rely on this slice as ground truth.
func TestWebAppsCanonical(t *testing.T) {
	t.Parallel()
	if len(webApps) == 0 {
		t.Fatal("webApps must not be empty")
	}
	must := []string{"org", "docs", "nchat", "cloud", "install"}
	for _, a := range must {
		found := false
		for _, w := range webApps {
			if w == a {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("webApps missing expected app %q", a)
		}
	}
}

// TestResolveWebApps_Empty returns all webApps when no args provided.
func TestResolveWebApps_Empty(t *testing.T) {
	t.Parallel()
	got, err := resolveWebApps(nil, "/dummy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(webApps) {
		t.Errorf("got %d apps, want %d", len(got), len(webApps))
	}
}

// TestResolveWebApps_Subset returns only the named apps.
func TestResolveWebApps_Subset(t *testing.T) {
	t.Parallel()
	got, err := resolveWebApps([]string{"org", "nchat"}, "/dummy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d apps, want 2", len(got))
	}
	if got[0] != "org" || got[1] != "nchat" {
		t.Errorf("unexpected order: %v", got)
	}
}

// TestResolveWebApps_Unknown rejects an unrecognised app name.
func TestResolveWebApps_Unknown(t *testing.T) {
	t.Parallel()
	_, err := resolveWebApps([]string{"nonexistent-app"}, "/dummy")
	if err == nil {
		t.Fatal("expected error for unknown app, got nil")
	}
}

// TestBuildDeployArgs_Preview returns --prebuilt without --prod.
func TestBuildDeployArgs_Preview(t *testing.T) {
	t.Parallel()
	args := buildDeployArgs("tok123", false)
	if !sliceHas(args, "--prebuilt") {
		t.Error("--prebuilt must always be present")
	}
	if sliceHas(args, "--prod") {
		t.Error("--prod must not be present for preview deploy")
	}
}

// TestBuildDeployArgs_Prod includes both --prebuilt and --prod.
func TestBuildDeployArgs_Prod(t *testing.T) {
	t.Parallel()
	args := buildDeployArgs("tok123", true)
	if !sliceHas(args, "--prebuilt") {
		t.Error("--prebuilt must always be present")
	}
	if !sliceHas(args, "--prod") {
		t.Error("--prod must be present for production deploy")
	}
}

// TestResolveAppDir finds an app in either <webRoot>/<app> or <webRoot>/apps/<app>.
func TestResolveAppDir(t *testing.T) {
	t.Parallel()

	// Case 1: app at direct child.
	dir1 := t.TempDir()
	appPath := filepath.Join(dir1, "org")
	if err := os.Mkdir(appPath, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveAppDir(dir1, "org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != appPath {
		t.Errorf("got %s, want %s", got, appPath)
	}

	// Case 2: app inside apps/ subdirectory.
	dir2 := t.TempDir()
	appsDir := filepath.Join(dir2, "apps")
	if err := os.MkdirAll(filepath.Join(appsDir, "nchat"), 0o755); err != nil {
		t.Fatal(err)
	}
	got2, err := resolveAppDir(dir2, "nchat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got2 != filepath.Join(appsDir, "nchat") {
		t.Errorf("got %s, want %s", got2, filepath.Join(appsDir, "nchat"))
	}
}

// TestResolveAppDir_NotFound returns an error when neither path exists.
func TestResolveAppDir_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := resolveAppDir(dir, "phantom")
	if err == nil {
		t.Fatal("expected error for missing app directory")
	}
}

// TestResolveWebRoot_FlagOverride uses --web-dir when explicitly set.
func TestResolveWebRoot_FlagOverride(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// resolveWebRoot with a valid path should return that path.
	got, err := resolveWebRoot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	abs, _ := filepath.Abs(dir)
	if got != abs {
		t.Errorf("got %s, want %s", got, abs)
	}
}

// TestResolveWebRoot_InvalidFlag returns an error for a missing path.
func TestResolveWebRoot_InvalidFlag(t *testing.T) {
	t.Parallel()
	_, err := resolveWebRoot("/this/path/does/not/exist/at/all")
	if err == nil {
		t.Fatal("expected error for non-existent --web-dir")
	}
}

// sliceHas reports whether s is in slice.
func sliceHas(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
