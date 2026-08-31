package access

// Tests drive HetznerMismatchWarnings against a local httptest.Server
// standing in for the Hetzner Cloud API (hetznerAPIBaseURL is redirected for
// the duration of each test) and a LocalFileTransport fixture standing in
// for the target host — no network call ever reaches the real Hetzner API
// or any real host, staging or production included.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// withHetznerServer starts a test server returning body for GET /ssh_keys,
// points hetznerAPIBaseURL at it for the duration of the test, and restores
// the original value on cleanup.
func withHetznerServer(t *testing.T, keys []hetznerSSHKey) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ssh_keys" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			SSHKeys []hetznerSSHKey `json:"ssh_keys"`
		}{SSHKeys: keys})
	}))
	t.Cleanup(srv.Close)

	oldURL := hetznerAPIBaseURL
	hetznerAPIBaseURL = srv.URL
	t.Cleanup(func() { hetznerAPIBaseURL = oldURL })
}

func TestHetznerMismatchWarnings_EmptyToken_NoOp(t *testing.T) {
	tp, _ := newFixture(t)
	warnings, err := HetznerMismatchWarnings(context.Background(), "", tp)
	if err != nil {
		t.Fatalf("HetznerMismatchWarnings: %v", err)
	}
	if warnings != nil {
		t.Errorf("warnings = %v, want nil for empty token", warnings)
	}
}

// TestHetznerMismatchWarnings_ProjectKeyMissingFromHost proves the
// true-positive case: a key registered at the Hetzner project level but
// absent from the host's authorized_keys produces exactly one warning
// naming that key.
func TestHetznerMismatchWarnings_ProjectKeyMissingFromHost(t *testing.T) {
	tp, _ := newFixture(t)
	ctx := context.Background()

	// alice is granted on the host; bob exists only at the project level.
	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("Grant: %v", err)
	}

	withHetznerServer(t, []hetznerSSHKey{
		{Name: "alice-project-key", PublicKey: aliceKeyLine},
		{Name: "bob-project-key", PublicKey: bobKeyLine},
	})

	warnings, err := HetznerMismatchWarnings(ctx, "test-token", tp)
	if err != nil {
		t.Fatalf("HetznerMismatchWarnings: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want exactly 1", warnings)
	}
	if got := warnings[0]; !containsAll(got, "bob-project-key", bobFP) {
		t.Errorf("warning = %q, want it to name bob-project-key and %s", got, bobFP)
	}
}

// TestHetznerMismatchWarnings_HostOnlyKey_NoWarning proves the
// true-negative case required by CR-B: a key present on the host but never
// registered at the Hetzner project level (the normal `nself access grant`
// case) must never produce a warning.
func TestHetznerMismatchWarnings_HostOnlyKey_NoWarning(t *testing.T) {
	tp, _ := newFixture(t)
	ctx := context.Background()

	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("Grant: %v", err)
	}

	// The Hetzner project has no keys registered at all — alice's host key
	// has no project-level counterpart, which is fine and not a mismatch.
	withHetznerServer(t, nil)

	warnings, err := HetznerMismatchWarnings(ctx, "test-token", tp)
	if err != nil {
		t.Fatalf("HetznerMismatchWarnings: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none (host-only key is not a mismatch)", warnings)
	}
}

// TestHetznerMismatchWarnings_AllMatch_NoWarning proves that when every
// project-level key is also present on the host (managed or foreign), no
// warnings fire.
func TestHetznerMismatchWarnings_AllMatch_NoWarning(t *testing.T) {
	tp, _ := newFixture(t)
	ctx := context.Background()

	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("Grant: %v", err)
	}

	withHetznerServer(t, []hetznerSSHKey{
		{Name: "alice-project-key", PublicKey: aliceKeyLine},
	})

	warnings, err := HetznerMismatchWarnings(ctx, "test-token", tp)
	if err != nil {
		t.Fatalf("HetznerMismatchWarnings: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none (project key matches host key)", warnings)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
