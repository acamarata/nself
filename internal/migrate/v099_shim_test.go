package migrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedV099Home creates a fixture v0.9.9 home directory inside tmp and
// returns its path. It plants license.key, channel, plugin-install.log,
// and an ssh keypair under ~/.nself/v099-state/.
func seedV099Home(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	legacy := filepath.Join(home, ".nself", "v099-state")
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatalf("seed: mkdir legacy: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(legacy, "ssh"), 0700); err != nil {
		t.Fatalf("seed: mkdir ssh: %v", err)
	}

	files := map[string]string{
		"license.key":        "nself_pro_legacy_key_abc123\n",
		"channel":            "canary\n",
		"plugin-install.log": "ai installed 2025-08-12\n",
		"ssh/id_rsa":         "PRIVATE-KEY-FAKE\n",
		"ssh/id_rsa.pub":     "PUBLIC-KEY-FAKE\n",
	}
	for rel, content := range files {
		full := filepath.Join(legacy, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatalf("seed: mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0600); err != nil {
			t.Fatalf("seed: write %s: %v", rel, err)
		}
	}
	return home
}

func TestDetectV099Home_AbsentReturnsClean(t *testing.T) {
	home := t.TempDir()
	got := DetectV099Home(home)
	if got.Present {
		t.Errorf("Present = true on empty home, want false")
	}
	if len(got.Found) != 0 {
		t.Errorf("Found = %d entries, want 0", len(got.Found))
	}
}

func TestDetectV099Home_ListsAllSeededEntries(t *testing.T) {
	home := seedV099Home(t)
	got := DetectV099Home(home)
	if !got.Present {
		t.Fatal("Present = false on seeded home, want true")
	}
	wantPaths := map[string]bool{
		"license.key":        true,
		"channel":            true,
		"plugin-install.log": true,
		"ssh":                true,
	}
	for _, e := range got.Found {
		if !wantPaths[e.LegacyPath] {
			t.Errorf("unexpected legacy entry %q", e.LegacyPath)
		}
		delete(wantPaths, e.LegacyPath)
	}
	if len(wantPaths) != 0 {
		t.Errorf("missing legacy entries: %v", wantPaths)
	}
}

func TestRunV099Migration_FullMigration(t *testing.T) {
	home := seedV099Home(t)
	res, err := RunV099Migration(home)
	if err != nil {
		t.Fatalf("RunV099Migration: %v", err)
	}
	if res.BackupDir == "" {
		t.Fatal("BackupDir empty")
	}

	// 1. license.json must be valid JSON with verified=false.
	data, err := os.ReadFile(filepath.Join(home, ".config", "nself", "license.json"))
	if err != nil {
		t.Fatalf("reading migrated license.json: %v", err)
	}
	var lic struct {
		Key      string `json:"key"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(data, &lic); err != nil {
		t.Fatalf("parsing license.json: %v", err)
	}
	if lic.Verified {
		t.Error("license verified=true after migration; want false (operator must re-verify)")
	}
	if !strings.Contains(lic.Key, "nself_pro_legacy_key") {
		t.Errorf("migrated license key = %q; expected legacy value preserved", lic.Key)
	}

	// 2. channel.json must be canary.
	chData, err := os.ReadFile(filepath.Join(home, ".config", "nself", "channel.json"))
	if err != nil {
		t.Fatalf("reading channel.json: %v", err)
	}
	if !strings.Contains(string(chData), "canary") {
		t.Errorf("channel.json = %q, want canary", string(chData))
	}

	// 3. ssh dir must be re-created at ~/.nself/ssh/.
	if _, err := os.Stat(filepath.Join(home, ".nself", "ssh", "id_rsa")); err != nil {
		t.Errorf("ssh/id_rsa not migrated: %v", err)
	}

	// 4. backup dir contains plugin-install.log.
	if _, err := os.Stat(filepath.Join(res.BackupDir, "plugin-install.log")); err != nil {
		t.Errorf("plugin-install.log not in backup: %v", err)
	}

	// 5. legacy dir was renamed aside (no longer detected).
	again := DetectV099Home(home)
	if again.Present {
		t.Error("legacy dir still present after migration; expected to be moved aside")
	}
}

func TestRunV099Migration_IsIdempotent(t *testing.T) {
	home := seedV099Home(t)
	if _, err := RunV099Migration(home); err != nil {
		t.Fatalf("first run: %v", err)
	}
	res, err := RunV099Migration(home)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(res.Migrated) != 0 || len(res.Backed) != 0 {
		t.Errorf("second run did work: migrated=%v backed=%v", res.Migrated, res.Backed)
	}
	if len(res.Warnings) == 0 {
		t.Error("second run produced no warning; expected 'no v0.9.9 home state found'")
	}
}

func TestRunV099Migration_LicenseFilePerm0600(t *testing.T) {
	home := seedV099Home(t)
	if _, err := RunV099Migration(home); err != nil {
		t.Fatalf("RunV099Migration: %v", err)
	}
	info, err := os.Stat(filepath.Join(home, ".config", "nself", "license.json"))
	if err != nil {
		t.Fatalf("stat license.json: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("license.json perms = %o, want 0600", info.Mode().Perm())
	}
}

func TestRunV099Migration_NoLegacyDirReturnsCleanly(t *testing.T) {
	home := t.TempDir()
	res, err := RunV099Migration(home)
	if err != nil {
		t.Fatalf("RunV099Migration on empty home: %v", err)
	}
	if len(res.Migrated) != 0 {
		t.Errorf("Migrated = %v on empty home, want empty", res.Migrated)
	}
	if len(res.Warnings) == 0 {
		t.Error("expected warning on empty home")
	}
}

func TestTrimWhitespace(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  hi  ", "hi"},
		{"\nfoo\n", "foo"},
		{"\t\rbar\r\n", "bar"},
		{"unchanged", "unchanged"},
		{"", ""},
		{"   ", ""},
	}
	for _, tc := range cases {
		if got := trimWhitespace(tc.in); got != tc.want {
			t.Errorf("trimWhitespace(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
