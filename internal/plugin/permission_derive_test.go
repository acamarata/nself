package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestDeriveCanonicalPermissions covers the mapping's direction, which is the
// only property that matters for a fail-closed check: every mapping must widen
// or stay equal, never narrow.
func TestDeriveCanonicalPermissions(t *testing.T) {
	tests := []struct {
		name    string
		grouped map[string][]string
		want    []string
	}{
		{
			name:    "CRUD splits into read and write",
			grouped: map[string][]string{"database": {"create", "read", "update", "delete"}},
			want:    []string{"db:read", "db:write"},
		},
		{
			name:    "an unknown database verb widens to write, never read",
			grouped: map[string][]string{"database": {"upsert"}},
			want:    []string{"db:write"},
		},
		{
			name:    "a named host is still internet access",
			grouped: map[string][]string{"network": {"api.stripe.com"}},
			want:    []string{"network:internet"},
		},
		{
			name:    "a wildcard host is the same permission as a named one",
			grouped: map[string][]string{"network": {"*"}},
			want:    []string{"network:internet"},
		},
		{
			name:    "a plugin target uses the parameterized inter-plugin form",
			grouped: map[string][]string{"network": {"ai_plugin"}},
			want:    []string{"network:plugin:ai"},
		},
		{
			name:    "a filesystem scope becomes a write scope",
			grouped: map[string][]string{"filesystem": {"logs"}},
			want:    []string{"fs:write:logs"},
		},
		{
			name:    "a path scope keeps its shape within the parameter grammar",
			grouped: map[string][]string{"filesystem": {"data/vpn"}},
			want:    []string{"fs:write:data-vpn"},
		},
		{
			name:    "a declared filesystem read is still a write permission",
			grouped: map[string][]string{"filesystem": {"read"}},
			want:    []string{"fs:write:read"},
		},
		{
			name:    "duplicates collapse",
			grouped: map[string][]string{"database": {"create", "update"}, "network": {"a.com", "b.com"}},
			want:    []string{"db:write", "network:internet"},
		},
		{
			name:    "nothing in, nothing out",
			grouped: nil,
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveCanonicalPermissions(tt.grouped)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q (full: %v)", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}

// TestDeriveNeverNarrows states the safety property directly. A write operation
// must never come back as read-only, because that is the one direction that
// turns a fail-closed check into a silent grant.
func TestDeriveNeverNarrows(t *testing.T) {
	for _, op := range []string{"create", "update", "delete", "ddl", "insert", "truncate", "drop", "*", "somethingnew"} {
		got := DeriveCanonicalPermissions(map[string][]string{"database": {op}})
		if len(got) != 1 || got[0] != "db:write" {
			t.Errorf("database:%s derived %v, want [db:write] — a write must never reduce to a read", op, got)
		}
	}
}

// TestEveryShippedPermissionDerivesToAKnownOne is the test that decides whether
// this mapping can be turned on at all.
//
// Validation is fail-closed: a permission that is not on the allowlist blocks
// the install. So deriving canonical permissions for the 121 manifests that use
// the descriptive form is only safe if EVERY token in the real catalogue
// derives to something the allowlist recognises. Anything that does not would
// turn a plugin that installs today into one that refuses to.
//
// This reads the shipped manifests rather than a fixture list, so a new
// descriptive token appearing in a future plugin fails here rather than at a
// user's install.
func TestEveryShippedPermissionDerivesToAKnownOne(t *testing.T) {
	siblings := filepath.Dir(repoRootDir(t))

	var manifests []string
	for _, dir := range []string{
		filepath.Join(siblings, "plugins", "free"),
		filepath.Join(siblings, "plugins-pro", "paid"),
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(dir, e.Name(), "plugin.json")
			if _, err := os.Stat(p); err == nil {
				manifests = append(manifests, p)
			}
		}
	}
	if len(manifests) == 0 {
		t.Skip("no sibling plugin repos checked out")
	}

	checked, derived := 0, 0
	for _, p := range manifests {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var m PluginManifest
		if err := json.Unmarshal(data, &m); err != nil {
			t.Errorf("%s does not parse: %v", filepath.Base(filepath.Dir(p)), err)
			continue
		}
		if !m.Permissions.UsesLegacyVocabulary() {
			continue
		}
		checked++
		for _, perm := range DeriveCanonicalPermissions(m.Permissions.Grouped) {
			derived++
			if !isKnownPermission(perm) {
				t.Errorf("%s: %q derives to %q, which the allowlist rejects — "+
					"turning this on would break a plugin that installs today",
					filepath.Base(filepath.Dir(p)), m.Permissions.Grouped, perm)
			}
		}
	}
	t.Logf("checked %d manifests using the descriptive form, %d derived permissions, all recognised", checked, derived)
}

// repoRootDir walks up from the working directory to the module root.
func repoRootDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}
