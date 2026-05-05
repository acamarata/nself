package license

import (
	"testing"
	"time"
)

func TestCacheTTLForTier(t *testing.T) {
	tests := []struct {
		tier string
		want time.Duration
	}{
		{"", TTLFree},
		{"free", TTLFree},
		{"Free", TTLFree},
		{"Pro", TTLPro},
		{"pro", TTLPro},
		{"ɳSelf Pro", TTLPro},
		{"plus", TTLPlus},
		{"ɳSelf+", TTLPlus},
		{"max", TTLPlus},
		{"Business+", TTLPlus},
		{"enterprise", TTLPlus},
		{"Owner", TTLPlus},
		{"owner", TTLPlus},
		{"unknown_bundle", TTLPro}, // defaults to Pro
	}

	for _, tc := range tests {
		got := CacheTTLForTier(tc.tier)
		if got != tc.want {
			t.Errorf("CacheTTLForTier(%q) = %v, want %v", tc.tier, got, tc.want)
		}
	}
}

func TestPreExpiryWarning_NoWarning(t *testing.T) {
	// Cache fetched 1 hour ago for a Pro tier (7d TTL) — no warning yet.
	entry := &CacheEntry{
		Tier:      "Pro",
		FetchedAt: time.Now().Add(-1 * time.Hour).Unix(),
	}
	msg, isPost := PreExpiryWarning(entry)
	if msg != "" || isPost {
		t.Errorf("expected no warning, got msg=%q isPost=%v", msg, isPost)
	}
}

func TestPreExpiryWarning_PreExpiry(t *testing.T) {
	// Cache fetched 6 days ago for a Pro tier (7d TTL) — within warning window.
	entry := &CacheEntry{
		Tier:      "Pro",
		FetchedAt: time.Now().Add(-6 * 24 * time.Hour).Unix(),
	}
	msg, isPost := PreExpiryWarning(entry)
	if msg == "" {
		t.Error("expected pre-expiry warning message")
	}
	if isPost {
		t.Error("expected isPost=false for pre-expiry")
	}
	t.Logf("pre-expiry message: %s", msg)
}

func TestPreExpiryWarning_PostExpiry(t *testing.T) {
	// Cache fetched 8 days ago for a Pro tier (7d TTL) — expired.
	entry := &CacheEntry{
		Tier:      "Pro",
		FetchedAt: time.Now().Add(-8 * 24 * time.Hour).Unix(),
	}
	msg, isPost := PreExpiryWarning(entry)
	if msg == "" {
		t.Error("expected post-expiry message")
	}
	if !isPost {
		t.Error("expected isPost=true for expired cache")
	}
}

func TestPreExpiryWarning_FreeNoWarning(t *testing.T) {
	// Free tier never warns.
	entry := &CacheEntry{
		Tier:      "free",
		FetchedAt: time.Now().Add(-100 * 24 * time.Hour).Unix(),
	}
	msg, isPost := PreExpiryWarning(entry)
	if msg != "" || isPost {
		t.Errorf("free tier should never warn, got msg=%q isPost=%v", msg, isPost)
	}
}

func TestPreExpiryWarning_NilEntry(t *testing.T) {
	msg, isPost := PreExpiryWarning(nil)
	if msg != "" || isPost {
		t.Errorf("nil entry should return no warning, got msg=%q isPost=%v", msg, isPost)
	}
}

func TestItoa(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{{0, "0"}, {1, "1"}, {7, "7"}, {30, "30"}, {365, "365"}}
	for _, c := range cases {
		if got := itoa(c.n); got != c.want {
			t.Errorf("itoa(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}
