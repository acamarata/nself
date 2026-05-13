package lifecycle_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/plugin/lifecycle"
)

func TestCheckExpiry_ActiveToDoormant(t *testing.T) {
	now := time.Now()
	s := &lifecycle.Store{
		Version: 1,
		Records: map[string]*lifecycle.Record{
			"ai": {
				Name:          "ai",
				State:         lifecycle.StateActive,
				LicenseExpiry: now.Add(-1 * time.Hour), // already expired
				GracePeriod:   lifecycle.DefaultGracePeriod,
			},
		},
	}
	dormant, autoRemove := s.CheckExpiry(now)
	if len(dormant) != 1 || dormant[0] != "ai" {
		t.Fatalf("expected [ai] dormant, got %v", dormant)
	}
	if len(autoRemove) != 0 {
		t.Fatalf("expected no autoRemove, got %v", autoRemove)
	}
	if s.Records["ai"].State != lifecycle.StateDormant {
		t.Errorf("expected state=dormant, got %v", s.Records["ai"].State)
	}
}

func TestCheckExpiry_DormantToExpired(t *testing.T) {
	dormantSince := time.Now().Add(-8 * 24 * time.Hour) // 8 days ago, grace=7d → expired
	s := &lifecycle.Store{
		Version: 1,
		Records: map[string]*lifecycle.Record{
			"claw": {
				Name:         "claw",
				State:        lifecycle.StateDormant,
				DormantSince: &dormantSince,
				GracePeriod:  lifecycle.DefaultGracePeriod,
			},
		},
	}
	dormant, autoRemove := s.CheckExpiry(time.Now())
	if len(autoRemove) != 1 || autoRemove[0] != "claw" {
		t.Fatalf("expected [claw] autoRemove, got %v", autoRemove)
	}
	if len(dormant) != 0 {
		t.Fatalf("expected no dormant, got %v", dormant)
	}
	if s.Records["claw"].State != lifecycle.StateExpired {
		t.Errorf("expected state=expired, got %v", s.Records["claw"].State)
	}
}

func TestMarkRenewal_ClearsDormant(t *testing.T) {
	dormantSince := time.Now().Add(-2 * 24 * time.Hour)
	s := &lifecycle.Store{
		Version: 1,
		Records: map[string]*lifecycle.Record{
			"mux": {
				Name:         "mux",
				State:        lifecycle.StateDormant,
				DormantSince: &dormantSince,
				GracePeriod:  lifecycle.DefaultGracePeriod,
			},
		},
	}
	newExpiry := time.Now().Add(30 * 24 * time.Hour)
	s.MarkRenewal("mux", newExpiry)

	rec := s.Records["mux"]
	if rec.State != lifecycle.StateActive {
		t.Errorf("expected active after renewal, got %v", rec.State)
	}
	if rec.DormantSince != nil {
		t.Errorf("expected DormantSince cleared, got %v", rec.DormantSince)
	}
}

func TestDormantBanner(t *testing.T) {
	dormantSince := time.Now().Add(-1 * 24 * time.Hour) // 1 day ago
	rec := &lifecycle.Record{
		Name:         "ai",
		State:        lifecycle.StateDormant,
		DormantSince: &dormantSince,
		GracePeriod:  lifecycle.DefaultGracePeriod,
	}
	banner := lifecycle.DormantBanner(rec, time.Now())
	if banner == "" {
		t.Fatal("expected non-empty banner for dormant plugin")
	}
	// Should mention the plugin name and days remaining
	if len(banner) < 10 {
		t.Errorf("banner suspiciously short: %q", banner)
	}
}

func TestDormantBanner_NilRecord(t *testing.T) {
	if got := lifecycle.DormantBanner(nil, time.Now()); got != "" {
		t.Errorf("expected empty banner for nil record, got %q", got)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	// Override store path via env (the real store uses $HOME; we patch it via a
	// symlink under $HOME/.config/nself pointing at the temp dir).
	// Simpler: write directly to a temp file and verify json round-trip.
	_ = dir

	now := time.Now().Round(time.Second) // json marshaling truncates sub-second
	s := &lifecycle.Store{
		Version: 1,
		Records: map[string]*lifecycle.Record{
			"voice": {
				Name:          "voice",
				State:         lifecycle.StateActive,
				LicenseExpiry: now.Add(30 * 24 * time.Hour),
				GracePeriod:   lifecycle.DefaultGracePeriod,
			},
		},
	}

	// Write to temp dir by overriding HOME for this test.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	if err := os.MkdirAll(filepath.Join(tmpHome, ".config", "nself"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := lifecycle.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rec, ok := loaded.Records["voice"]
	if !ok {
		t.Fatal("record 'voice' missing after round-trip")
	}
	if rec.State != lifecycle.StateActive {
		t.Errorf("state mismatch: got %v", rec.State)
	}
}
