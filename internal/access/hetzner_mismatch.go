package access

// Purpose: the Hetzner-project-vs-running-server key mismatch check —
// issue #238's remaining sub-item. `nself access grant/revoke/list` manage
// one host's authorized_keys directly; this file answers a different
// question: is there a key registered at the Hetzner Cloud project level
// that operators believe (incorrectly) already grants access to this
// already-running server? hcloud only injects project-level keys at
// server-creation time, so a project key with no matching entry on the
// host is silently inert — this is issue #238's "actual footgun".
// Inputs: a Hetzner Cloud API token and a Transport for the target host.
// Outputs: one warning string per project-level key absent from the host.
// Constraints: compares fingerprints computed locally by PublicKey.
// Fingerprint (SHA256) on both sides — never Hetzner's own "fingerprint"
// field, which is legacy MD5 colon-hex and would never match. Best-effort
// only: an empty token or any network/API failure returns an error the
// caller is expected to log and otherwise ignore, never one that fails the
// grant/revoke this check is attached to.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// hetznerAPIBaseURL and hetznerHTTPClient are var indirections so tests can
// point the mismatch check at a local httptest.Server instead of the real
// Hetzner Cloud API, mirroring the injectable explicit-timeout http.Client
// pattern used by internal/license/validate.go (never http.DefaultClient).
var (
	hetznerAPIBaseURL = "https://api.hetzner.cloud/v1"
	hetznerHTTPClient = &http.Client{Timeout: 10 * time.Second}
)

// hetznerSSHKey is the subset of a Hetzner Cloud API ssh_keys entry this
// check needs. Its own "fingerprint" field is deliberately unused: Hetzner
// reports the legacy MD5 colon-hex format, not the SHA256 format
// PublicKey.Fingerprint produces, so a direct string comparison would never
// match. Instead PublicKey is re-parsed from public_key and fingerprinted
// locally, comparing apples to apples against the host side.
type hetznerSSHKey struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// HetznerMismatchWarnings compares the SSH keys registered at the Hetzner
// Cloud project level (identified by token) against the keys actually
// present in t's authorized_keys, and returns one warning string per
// project-level key absent from the host.
//
// The reverse case is deliberately NOT a mismatch and never produces a
// warning: a key present in authorized_keys but never registered at the
// Hetzner project level (e.g. anything granted via `nself access grant`
// itself) is a normal, legitimate state.
//
// An empty token, or any network/API failure, returns (nil, err) so callers
// can treat this as a best-effort check — see access_grant.go's caller,
// which logs at most and never fails the grant/revoke this check is
// attached to.
func HetznerMismatchWarnings(ctx context.Context, token string, t Transport) ([]string, error) {
	if token == "" {
		return nil, nil
	}

	hostFPs, err := hostFingerprints(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("read host keys for mismatch check: %w", err)
	}

	projectKeys, err := fetchHetznerSSHKeys(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("fetch Hetzner project SSH keys: %w", err)
	}

	var warnings []string
	for _, pk := range projectKeys {
		key, err := ParsePublicKey(pk.PublicKey)
		if err != nil {
			continue // not a key format we understand; skip rather than false-positive
		}
		fp, err := key.Fingerprint()
		if err != nil || hostFPs[fp] {
			continue
		}
		warnings = append(warnings, fmt.Sprintf(
			"Hetzner project key %q (%s) is registered at the project level but NOT present in %s's authorized_keys — it will not grant access to this already-running server",
			pk.Name, fp, t.Describe()))
	}
	return warnings, nil
}

// hostFingerprints returns the fingerprint of every key line in t's
// authorized_keys — managed AND foreign — since the mismatch check must
// compare against everything that can actually authenticate a login, not
// just nself-managed entries.
func hostFingerprints(ctx context.Context, t Transport) (map[string]bool, error) {
	content, err := t.Read(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, line := range strings.Split(strings.TrimRight(string(content), "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		key, err := ParsePublicKey(fields[0] + " " + fields[1])
		if err != nil {
			continue
		}
		fp, err := key.Fingerprint()
		if err != nil {
			continue
		}
		out[fp] = true
	}
	return out, nil
}

// fetchHetznerSSHKeys calls GET /ssh_keys against the Hetzner Cloud API.
func fetchHetznerSSHKeys(ctx context.Context, token string) ([]hetznerSSHKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hetznerAPIBaseURL+"/ssh_keys", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := hetznerHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hetzner API returned %d", resp.StatusCode)
	}

	var parsed struct {
		SSHKeys []hetznerSSHKey `json:"ssh_keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return parsed.SSHKeys, nil
}
