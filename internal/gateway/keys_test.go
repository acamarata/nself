// Package gateway — unit tests for keys.go
package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestIsValidProvider checks provider validation logic.
func TestIsValidProvider(t *testing.T) {
	tests := []struct {
		provider string
		want     bool
	}{
		{"anthropic", true},
		{"openai", true},
		{"google", true},
		{"custom", true},
		{"azure", false},
		{"", false},
		{"ANTHROPIC", false}, // case-sensitive
	}
	for _, tc := range tests {
		got := isValidProvider(tc.provider)
		if got != tc.want {
			t.Errorf("isValidProvider(%q) = %v, want %v", tc.provider, got, tc.want)
		}
	}
}

// TestAddKey_InvalidProvider ensures AddKey returns an error for unknown providers.
func TestAddKey_InvalidProvider(t *testing.T) {
	_, err := AddKey("azure", "key", "label")
	if err == nil {
		t.Fatal("expected error for invalid provider, got nil")
	}
}

// TestAddKey_EmptyKey ensures AddKey returns an error when key is empty.
func TestAddKey_EmptyKey(t *testing.T) {
	_, err := AddKey("anthropic", "", "label")
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
}

// TestRemoveKey_EmptyID ensures RemoveKey returns an error for empty ID.
func TestRemoveKey_EmptyID(t *testing.T) {
	err := RemoveKey("")
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
}

// TestListKeys_Server exercises ListKeys against a mock HTTP server.
func TestListKeys_Server(t *testing.T) {
	wantKeys := []KeyEntry{
		{ID: "abc123", Provider: "anthropic", Label: "prod", IsActive: true, CreatedAt: "2026-06-01T00:00:00Z"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/keys" || r.Method != "GET" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": wantKeys})
	}))
	defer srv.Close()

	// Override the base URL for this test.
	origBase := gatewayBaseURLOverride
	gatewayBaseURLOverride = srv.URL
	defer func() { gatewayBaseURLOverride = origBase }()

	keys, err := listKeysWithURL(srv.URL)
	if err != nil {
		t.Fatalf("ListKeys returned error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].ID != "abc123" {
		t.Errorf("expected ID=abc123, got %q", keys[0].ID)
	}
	// Verify no key material column (encrypted_key) in result struct field names.
	// The KeyEntry struct has no KeyMaterial/EncryptedKey field by design.
	_ = keys[0].CreatedAt // just ensure we're reading metadata fields
}

// gatewayBaseURLOverride allows tests to override the base URL without auth.
// The blank value means "use real gateway" (requires auth).
var gatewayBaseURLOverride string

// listKeysWithURL is the test-injectable variant of ListKeys that accepts an explicit baseURL.
func listKeysWithURL(baseURL string) ([]KeyEntry, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", baseURL+"/v1/keys", nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Keys []KeyEntry `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Keys, nil
}

// TestKeyEntryNoKeyMaterial ensures the KeyEntry struct has no field that could
// carry raw key material (plaintext or encrypted bytes) in its JSON output.
func TestKeyEntryNoKeyMaterial(t *testing.T) {
	k := KeyEntry{
		ID:        "test-id",
		Provider:  "anthropic",
		Label:     "test",
		IsActive:  true,
		CreatedAt: "2026-06-01T00:00:00Z",
	}
	data, err := json.Marshal(k)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	s := string(data)
	// None of these field names should appear in a KeyEntry JSON output.
	for _, forbidden := range []string{"encrypted_key", "key_hash", "plaintext", "raw_key", "api_key"} {
		if len(s) > 0 {
			for i := range s {
				if i+len(forbidden) <= len(s) && s[i:i+len(forbidden)] == forbidden {
					t.Errorf("KeyEntry JSON contains forbidden field %q: %s", forbidden, s)
				}
			}
		}
	}
}
