package build

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadComposeManifest_MergesUserOverride is the regression guard for the
// bug the ntask clean-fork self-host drill found on 2026-08-24: `docker compose`
// only auto-merges docker-compose.override.yml when invoked with NO -f, and the
// CLI always passes -f, so the override was inert on every project. Containers
// came up without it and nothing reported the fact.
func TestReadComposeManifest_MergesUserOverride(t *testing.T) {
	t.Run("no manifest, override present", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}\n")
		writeFile(t, filepath.Join(dir, "docker-compose.override.yml"), "services: {}\n")

		got, err := ReadComposeManifest(dir)
		if err != nil {
			t.Fatalf("ReadComposeManifest: %v", err)
		}
		want := []string{"docker-compose.yml", "docker-compose.override.yml"}
		if !equalStrings(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("no manifest, no override", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}\n")

		got, err := ReadComposeManifest(dir)
		if err != nil {
			t.Fatalf("ReadComposeManifest: %v", err)
		}
		if !equalStrings(got, []string{"docker-compose.yml"}) {
			t.Fatalf("got %v, want just the base file", got)
		}
	})

	t.Run("override goes last so it beats plugin fragments", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}\n")
		writeFile(t, filepath.Join(dir, "plugin-a.yml"), "services: {}\n")
		writeFile(t, filepath.Join(dir, "docker-compose.override.yml"), "services: {}\n")
		writeFile(t, filepath.Join(dir, composeManifestFile), "docker-compose.yml\nplugin-a.yml\n")

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(cwd) })
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		got, err := ReadComposeManifest(dir)
		if err != nil {
			t.Fatalf("ReadComposeManifest: %v", err)
		}
		if len(got) == 0 || got[len(got)-1] != "docker-compose.override.yml" {
			t.Fatalf("override must be last so Compose lets it win; got %v", got)
		}
	})

	t.Run("override already in manifest is not duplicated", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}\n")
		writeFile(t, filepath.Join(dir, "docker-compose.override.yml"), "services: {}\n")
		writeFile(t, filepath.Join(dir, composeManifestFile), "docker-compose.yml\ndocker-compose.override.yml\n")

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(cwd) })
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}

		got, err := ReadComposeManifest(dir)
		if err != nil {
			t.Fatalf("ReadComposeManifest: %v", err)
		}
		count := 0
		for _, p := range got {
			if p == "docker-compose.override.yml" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("override listed %d times, want 1: %v", count, got)
		}
	})
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
