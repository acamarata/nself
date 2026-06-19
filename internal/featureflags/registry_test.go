package featureflags

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestEmbeddedRegistry verifies that the embedded registry.yaml parses
// cleanly under Load() — guards against accidental YAML breakage.
func TestEmbeddedRegistry(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	all := r.All()
	if len(all) == 0 {
		t.Fatalf("Load returned zero flags; registry.yaml must declare at least one v1.1.0 flag")
	}
	// Spot-check the v1.1.0 flags called out by S12.T15 spec.
	required := []string{
		"nsentry-enabled",
		"nfamily-coppa-strict",
		"cloud-multi-tenant-strict",
	}
	for _, key := range required {
		if _, ok := r.Get(key); !ok {
			t.Errorf("required v1.1.0 flag %q missing from registry", key)
		}
	}
}

const validYAML = `version: 1
flags:
  - key: alpha
    type: release
    surface: cloud
    description: Test alpha flag.
    default: false
    introduced: "1.1.0"
  - key: beta
    type: ops
    surface: backend
    description: Test beta flag.
    default: true
    introduced: "1.1.0"
    auto_kill: "2026-12-31"
`

func TestLoadFromBytes_Valid(t *testing.T) {
	r, err := loadFromBytes([]byte(validYAML))
	if err != nil {
		t.Fatalf("loadFromBytes: %v", err)
	}
	if got := len(r.All()); got != 2 {
		t.Fatalf("got %d flags, want 2", got)
	}
	a, ok := r.Get("alpha")
	if !ok || a.Type != FlagTypeRelease || a.Surface != "cloud" || a.Default != false {
		t.Errorf("alpha: %+v", a)
	}
	b, ok := r.Get("beta")
	if !ok || b.AutoKill != "2026-12-31" {
		t.Errorf("beta: %+v", b)
	}
}

func TestLoadFromBytes_RejectsBadType(t *testing.T) {
	bad := `version: 1
flags:
  - key: oops
    type: bogus
    surface: cloud
    description: x
    default: false
    introduced: "1.1.0"
`
	if _, err := loadFromBytes([]byte(bad)); err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestLoadFromBytes_RejectsDuplicateKey(t *testing.T) {
	dup := `version: 1
flags:
  - key: x
    type: release
    surface: cloud
    description: a
    default: false
    introduced: "1.1.0"
  - key: x
    type: ops
    surface: backend
    description: b
    default: true
    introduced: "1.1.0"
`
	if _, err := loadFromBytes([]byte(dup)); err == nil {
		t.Fatal("expected error for duplicate key, got nil")
	}
}

func TestLoadFromBytes_RejectsBadVersion(t *testing.T) {
	bad := `version: 2
flags:
  - key: x
    type: release
    surface: cloud
    description: a
    default: false
    introduced: "1.1.0"
`
	if _, err := loadFromBytes([]byte(bad)); err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
}

func TestStateRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// Loading from a fresh dir yields an empty state, not an error.
	s, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState empty: %v", err)
	}
	if len(s.Overrides) != 0 {
		t.Fatalf("expected empty overrides, got %v", s.Overrides)
	}

	// Set an override and save.
	r, err := loadFromBytes([]byte(validYAML))
	if err != nil {
		t.Fatalf("loadFromBytes: %v", err)
	}
	if err := r.SetOverride(s, "alpha", true); err != nil {
		t.Fatalf("SetOverride alpha: %v", err)
	}
	if err := SaveState(dir, s); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// File exists with 0600 perms (Unix only — Windows always returns 0666).
	if runtime.GOOS != "windows" {
		st, err := os.Stat(StatePath(dir))
		if err != nil {
			t.Fatalf("Stat state: %v", err)
		}
		if perm := st.Mode().Perm(); perm != 0o600 {
			t.Errorf("state file perm = %o, want 0600", perm)
		}
	}

	// Reload yields the override.
	s2, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState reload: %v", err)
	}
	if v, ok := s2.Overrides["alpha"]; !ok || !v {
		t.Errorf("override not persisted: %v", s2.Overrides)
	}
}

func TestResolve(t *testing.T) {
	r, err := loadFromBytes([]byte(validYAML))
	if err != nil {
		t.Fatalf("loadFromBytes: %v", err)
	}
	// Empty state — every flag at its registry default.
	statuses := r.Resolve(nil)
	if len(statuses) != 2 {
		t.Fatalf("Resolve len = %d, want 2", len(statuses))
	}
	for _, s := range statuses {
		if s.Source != "default" {
			t.Errorf("flag %s: source=%s, want default", s.Flag.Key, s.Source)
		}
		if s.Enabled != s.Flag.Default {
			t.Errorf("flag %s: enabled=%v, want %v", s.Flag.Key, s.Enabled, s.Flag.Default)
		}
	}

	// Override flip.
	st := &State{Overrides: map[string]bool{"alpha": true}}
	if got, ok := r.ResolveOne(st, "alpha"); !ok || !got.Enabled || got.Source != "override" {
		t.Errorf("ResolveOne alpha override: ok=%v %+v", ok, got)
	}
	if got, ok := r.ResolveOne(st, "beta"); !ok || !got.Enabled || got.Source != "default" {
		// beta's default is true; with no override, Source must remain "default".
		t.Errorf("ResolveOne beta default: ok=%v %+v", ok, got)
	}
	if _, ok := r.ResolveOne(st, "nonexistent"); ok {
		t.Error("ResolveOne nonexistent: ok=true, want false")
	}
}

func TestSetOverride_RejectsUnknownKey(t *testing.T) {
	r, _ := loadFromBytes([]byte(validYAML))
	s := &State{Overrides: map[string]bool{}}
	if err := r.SetOverride(s, "ghost", true); err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

func TestClearOverride(t *testing.T) {
	r, _ := loadFromBytes([]byte(validYAML))
	s := &State{Overrides: map[string]bool{"alpha": true}}
	if err := r.ClearOverride(s, "alpha"); err != nil {
		t.Fatalf("ClearOverride: %v", err)
	}
	if _, present := s.Overrides["alpha"]; present {
		t.Errorf("override not cleared: %v", s.Overrides)
	}
	// Clearing an unknown flag is an error.
	if err := r.ClearOverride(s, "ghost"); err == nil {
		t.Fatal("expected error for unknown flag, got nil")
	}
}

// TestLoadState_RejectsMalformed verifies that a corrupt features.json
// surfaces as a load error rather than silently producing an empty state.
func TestLoadState_RejectsMalformed(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".nself"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(StatePath(dir), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadState(dir); err == nil {
		t.Fatal("expected parse error for malformed state, got nil")
	}
}

// TestSaveState_AtomicJSON verifies the JSON shape is round-trippable
// against the public State type (catches schema drift).
func TestSaveState_AtomicJSON(t *testing.T) {
	dir := t.TempDir()
	s := &State{Overrides: map[string]bool{"x": true, "y": false}}
	if err := SaveState(dir, s); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	raw, err := os.ReadFile(StatePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	var back State
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back.Overrides["x"] != true || back.Overrides["y"] != false {
		t.Errorf("round-trip mismatch: %v", back.Overrides)
	}
	if back.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set on save")
	}
}
