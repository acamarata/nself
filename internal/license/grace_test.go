// Package license — grace_test.go tests the grace state machine.
// S88b.T01 coverage extension: all GraceState branches + edge cases.
package license

import (
	"testing"
	"time"
)

// makeCacheEntry returns a CacheEntry with the given fetched-at offset from now.
// expiryOffset is how far in the future the entry expires (use negative for expired).
func makeCacheEntry(keyHash string, fetchedAgoHours float64, expiryOffsetHours float64) *CacheEntry {
	now := time.Now()
	return &CacheEntry{
		KeyHash:        keyHash,
		Tier:           "pro",
		PluginsAllowed: []string{"ai", "claw", "mux"},
		FetchedAt:      now.Add(-time.Duration(fetchedAgoHours * float64(time.Hour))).Unix(),
		ExpiresAt:      now.Add(time.Duration(expiryOffsetHours * float64(time.Hour))).Unix(),
	}
}

// TestDetermineGraceState_NilEntry verifies that nil cache returns GraceExpired.
func TestDetermineGraceState_NilEntry(t *testing.T) {
	result := DetermineGraceState(nil)
	if result.State != GraceExpired {
		t.Errorf("nil entry: state = %q, want %q", result.State, GraceExpired)
	}
	if result.CanProceed {
		t.Error("nil entry: CanProceed should be false")
	}
	if result.WriteAllowed {
		t.Error("nil entry: WriteAllowed should be false")
	}
}

// TestDetermineGraceState_Valid verifies that a freshly-fetched entry (< 24h ago) returns GraceValid.
func TestDetermineGraceState_Valid(t *testing.T) {
	entry := makeCacheEntry("nself_pro_testkey", 1, 720) // fetched 1h ago, expires 720h from now
	result := DetermineGraceState(entry)
	if result.State != GraceValid {
		t.Errorf("fresh entry (1h ago): state = %q, want %q", result.State, GraceValid)
	}
	if !result.CanProceed {
		t.Error("fresh entry: CanProceed should be true")
	}
	if !result.WriteAllowed {
		t.Error("fresh entry: WriteAllowed should be true")
	}
}

// TestDetermineGraceState_SoftGrace verifies that a 24-7d old entry returns GraceSoft with writes still allowed.
func TestDetermineGraceState_SoftGrace(t *testing.T) {
	// 25h ago — just past the 24h soft threshold
	entry := makeCacheEntry("nself_pro_testkey", 25, 720)
	result := DetermineGraceState(entry)
	if result.State != GraceSoft {
		t.Errorf("25h old entry: state = %q, want %q", result.State, GraceSoft)
	}
	if !result.CanProceed {
		t.Error("grace_soft: CanProceed should be true (allow with warning)")
	}
	if !result.WriteAllowed {
		t.Error("grace_soft: WriteAllowed should be true")
	}
}

// TestDetermineGraceState_SoftGraceBoundary verifies the exact 24h boundary.
func TestDetermineGraceState_SoftGraceBoundary(t *testing.T) {
	// Exactly 24h ago is just at the soft boundary — should be soft or valid depending on impl.
	entry24h := makeCacheEntry("nself_pro_testkey", 24.01, 720)
	result := DetermineGraceState(entry24h)
	// 24h+ means we are in soft grace territory.
	if result.State != GraceSoft && result.State != GraceValid {
		t.Errorf("24h boundary entry: state = %q, want %q or %q", result.State, GraceSoft, GraceValid)
	}
}

// TestDetermineGraceState_HardGrace verifies that an 8d old entry returns GraceHard with writes blocked.
func TestDetermineGraceState_HardGrace(t *testing.T) {
	entry := makeCacheEntry("nself_pro_testkey", 8*24, 720) // 8 days ago
	result := DetermineGraceState(entry)
	if result.State != GraceHard {
		t.Errorf("8d old entry: state = %q, want %q", result.State, GraceHard)
	}
	if !result.CanProceed {
		t.Error("grace_hard: CanProceed should be true (read-only still allowed)")
	}
	if result.WriteAllowed {
		t.Error("grace_hard: WriteAllowed should be false (read-only degradation)")
	}
}

// TestDetermineGraceState_Expired verifies that an expired license returns GraceExpired.
func TestDetermineGraceState_Expired(t *testing.T) {
	entry := makeCacheEntry("nself_pro_testkey", 1, -1) // expires_at is 1h in the past
	result := DetermineGraceState(entry)
	if result.State != GraceExpired {
		t.Errorf("expired entry: state = %q, want %q", result.State, GraceExpired)
	}
	if result.CanProceed {
		t.Error("expired entry: CanProceed should be false")
	}
	if result.WriteAllowed {
		t.Error("expired entry: WriteAllowed should be false")
	}
}

// TestDetermineGraceState_HardGraceBoundary verifies the exact 7d boundary.
func TestDetermineGraceState_HardGraceBoundary(t *testing.T) {
	// 7d + 1h — just past the hard threshold
	entry := makeCacheEntry("nself_pro_testkey", 7*24+1, 720)
	result := DetermineGraceState(entry)
	if result.State != GraceHard {
		t.Errorf("7d+1h entry: state = %q, want %q", result.State, GraceHard)
	}
}

// TestDetermineGraceState_GraceMessageNotEmpty verifies that each state includes a non-empty message.
func TestDetermineGraceState_GraceMessageNotEmpty(t *testing.T) {
	cases := []struct {
		name  string
		entry *CacheEntry
	}{
		{"nil", nil},
		{"valid", makeCacheEntry("k", 1, 720)},
		{"soft", makeCacheEntry("k", 25, 720)},
		{"hard", makeCacheEntry("k", 8*24, 720)},
		{"expired", makeCacheEntry("k", 1, -1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := DetermineGraceState(tc.entry)
			if result.State == GraceValid {
				return // valid state may have empty message — that is fine
			}
			if result.Message == "" {
				t.Errorf("state %q: message should not be empty (user needs to know what to do)", result.State)
			}
		})
	}
}

// TestDetermineGraceState_Tier propagates from cache entry.
func TestDetermineGraceState_Tier(t *testing.T) {
	entry := makeCacheEntry("k", 1, 720)
	entry.Tier = "enterprise"
	result := DetermineGraceState(entry)
	if result.Tier != "enterprise" {
		t.Errorf("tier propagation: got %q, want enterprise", result.Tier)
	}
}

// TestDetermineGraceState_CacheAgeAccuracy verifies CacheAge matches the entry age.
func TestDetermineGraceState_CacheAgeAccuracy(t *testing.T) {
	entry := makeCacheEntry("k", 2, 720) // 2h ago
	result := DetermineGraceState(entry)
	// CacheAge should be approximately 2h (allow 10s tolerance).
	wantAge := 2 * time.Hour
	if result.CacheAge < wantAge-10*time.Second || result.CacheAge > wantAge+10*time.Second {
		t.Errorf("CacheAge = %v, want ~%v (±10s)", result.CacheAge, wantAge)
	}
}
