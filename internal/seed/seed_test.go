package seed

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSeedDir_ReturnsSeedPath verifies SeedDir returns db/seeds under projectDir.
func TestSeedDir_ReturnsSeedPath(t *testing.T) {
	got := SeedDir("/my/project")
	want := filepath.Join("/my/project", "db", "seeds")
	if got != want {
		t.Errorf("SeedDir: got %q, want %q", got, want)
	}
}

// TestFixtureDir_ReturnsFixturePath verifies the fixture directory path.
func TestFixtureDir_ReturnsFixturePath(t *testing.T) {
	got := FixtureDir("/my/project", "demo")
	want := filepath.Join("/my/project", "db", "seeds", "fixtures", "demo")
	if got != want {
		t.Errorf("FixtureDir: got %q, want %q", got, want)
	}
}

// TestListSeeds_EmptyWhenMissing verifies ListSeeds returns nil when the
// seeds directory does not exist (not an error).
func TestListSeeds_EmptyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	seeds, err := ListSeeds(dir)
	if err != nil {
		t.Fatalf("ListSeeds on missing dir: %v", err)
	}
	if len(seeds) != 0 {
		t.Errorf("expected 0 seeds for missing dir, got %d", len(seeds))
	}
}

// TestListSeeds_FindsSQLFiles verifies ListSeeds discovers SQL files.
func TestListSeeds_FindsSQLFiles(t *testing.T) {
	dir := t.TempDir()
	devDir := filepath.Join(dir, "db", "seeds", "dev")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// Create test SQL files.
	for _, name := range []string{"01-users.sql", "02-items.sql"} {
		if err := os.WriteFile(filepath.Join(devDir, name), []byte("-- test\n"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	seeds, err := ListSeeds(dir)
	if err != nil {
		t.Fatalf("ListSeeds: %v", err)
	}
	if len(seeds) < 2 {
		t.Errorf("expected ≥2 seeds, got %d", len(seeds))
	}
}

// TestResolveDependencyOrder_EmptyInput verifies empty input returns empty output.
func TestResolveDependencyOrder_EmptyInput(t *testing.T) {
	result, err := ResolveDependencyOrder(nil)
	if err != nil {
		t.Fatalf("ResolveDependencyOrder(nil): %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}

// TestResolveDependencyOrder_NoDeps verifies no-dependency seeds are returned
// in some valid order.
func TestResolveDependencyOrder_NoDeps(t *testing.T) {
	seeds := []SeedFile{
		{Name: "users", Path: "users.sql"},
		{Name: "items", Path: "items.sql"},
		{Name: "orders", Path: "orders.sql"},
	}
	result, err := ResolveDependencyOrder(seeds)
	if err != nil {
		t.Fatalf("ResolveDependencyOrder: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 seeds, got %d", len(result))
	}
}

// TestResolveDependencyOrder_WithDeps verifies dependent seeds come after their
// dependencies.
func TestResolveDependencyOrder_WithDeps(t *testing.T) {
	seeds := []SeedFile{
		{Name: "orders", Path: "orders.sql", DependsOn: []string{"users", "items"}},
		{Name: "items", Path: "items.sql"},
		{Name: "users", Path: "users.sql"},
	}
	result, err := ResolveDependencyOrder(seeds)
	if err != nil {
		t.Fatalf("ResolveDependencyOrder: %v", err)
	}
	// Build position map.
	pos := make(map[string]int, len(result))
	for i, s := range result {
		pos[s.Name] = i
	}
	if pos["orders"] < pos["users"] {
		t.Error("orders must come after users (dependency order violated)")
	}
	if pos["orders"] < pos["items"] {
		t.Error("orders must come after items (dependency order violated)")
	}
}

// TestResolveDependencyOrder_CircularDetected verifies that a circular
// dependency returns an error.
func TestResolveDependencyOrder_CircularDetected(t *testing.T) {
	seeds := []SeedFile{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"a"}},
	}
	_, err := ResolveDependencyOrder(seeds)
	if err == nil {
		t.Error("expected error for circular dependency, got nil")
	}
}

// TestSeedHeaders_Constants verifies header constants are non-empty and distinct.
func TestSeedHeaders_Constants(t *testing.T) {
	headers := []string{HeaderIdempotent, HeaderDestructive, HeaderDependsOn}
	seen := make(map[string]bool)
	for _, h := range headers {
		if h == "" {
			t.Errorf("seed header constant is empty")
		}
		if seen[h] {
			t.Errorf("duplicate seed header: %q", h)
		}
		seen[h] = true
	}
}

// TestBuildEnvMatrix_EmptyOnMissing verifies BuildEnvMatrix returns empty map
// when no seeds directory exists.
func TestBuildEnvMatrix_EmptyOnMissing(t *testing.T) {
	dir := t.TempDir()
	matrix, err := BuildEnvMatrix(dir)
	if err != nil {
		t.Fatalf("BuildEnvMatrix on missing dir: %v", err)
	}
	if len(matrix) != 0 {
		t.Errorf("expected empty matrix, got %+v", matrix)
	}
}
