package license

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A revoked licence must not pass the fail-open cache path. Before this was
// wired, bundleEntitledFromCache validated the key hash and the tier and
// returned true without ever consulting the revocation list, so revoking a
// licence had no effect on any client that could not reach the server.
func TestBundleEntitledFromCache_RefusesRevokedKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // os.UserHomeDir reads USERPROFILE on Windows
	t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, ".cache"))

	const key = "test-license-key"
	hash := HashKey(key)

	if err := WriteCache(&CacheEntry{
		KeyHash:   hash,
		Tier:      "pro",
		FetchedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}); err != nil {
		t.Fatalf("seeding licence cache: %v", err)
	}

	// Not revoked yet: the cached entitlement decides, whatever it decides.
	// We only assert that it does not refuse *for revocation* reasons.
	if _, err := bundleEntitledFromCache(key, "task"); err != nil {
		if got := err.Error(); strings.Contains(got, "revoked") {
			t.Fatalf("unrevoked licence refused as revoked: %v", err)
		}
	}

	// Now publish a revocation against this key hash.
	if err := WriteRevocationCache(&RevocationCache{
		FetchedAt: time.Now().Unix(),
		List: RevocationList{
			Revoked: []RevocationEntry{{Type: "key_hash", ID: hash, Reason: "test"}},
		},
	}); err != nil {
		t.Fatalf("seeding revocation cache: %v", err)
	}

	ok, err := bundleEntitledFromCache(key, "task")
	if ok {
		t.Fatal("revoked licence was entitled via the fail-open path")
	}
	if err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("expected a revocation refusal, got: %v", err)
	}
}
