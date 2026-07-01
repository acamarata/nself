package config

// Purpose: Regression tests for JWT-ALGO-01 (gap #3) and the JWT key
//          persistence fallbacks (gap #4).
//
// Inputs:  Config structs with varying Hasura.JWTType/JWTKey and simulated
//          .env cascade values via t.Setenv.
// Outputs: none (t.Error/t.Fatal on assertion failure).
//
// Constraints: Must not depend on filesystem state — pure struct + env var
//              manipulation, matching loader_test.go's style.
//
// SPORT: cli/internal/config — gap #3/#4 fix (env-schema parity ticket).

import (
	"encoding/json"
	"strings"
	"testing"
)

// ── Gap #3: JWT algorithm default must be safely matched (HS256/HS256) ──────

// TestApplyDefaults_JWTType_DefaultsToHS256 verifies that a fresh config with
// no JWTType set gets HS256, not RS256. RS256 previously defaulted here while
// JWTKey was auto-generated as a plain random string (not an RSA keypair),
// which Hasura could never actually verify — breaking auth on every fresh
// install. HS256 is the only default under which the auto-generated random
// JWTKey is valid key material for the chosen algorithm.
func TestApplyDefaults_JWTType_DefaultsToHS256(t *testing.T) {
	cfg := &Config{BaseDomain: "example.com"}
	cfg, err := ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	if cfg.Hasura.JWTType != "HS256" {
		t.Errorf("Hasura.JWTType = %q, want %q", cfg.Hasura.JWTType, "HS256")
	}
	if cfg.Hasura.JWTKey == "" {
		t.Error("Hasura.JWTKey should be auto-generated when unset")
	}
}

// TestApplyDefaults_JWTType_ExplicitRS256RequiresKey verifies that a user
// explicitly requesting RS256 without supplying HASURA_JWT_KEY/AUTH_JWT_KEY
// gets a clear error instead of a silently-broken auto-generated "RSA" key
// that is actually just a random string.
func TestApplyDefaults_JWTType_ExplicitRS256RequiresKey(t *testing.T) {
	cfg := &Config{
		BaseDomain: "example.com",
		Hasura:     HasuraConfig{JWTType: "RS256"},
	}
	_, err := ApplyDefaults(cfg)
	if err == nil {
		t.Fatal("expected error when RS256 is requested without key material, got nil")
	}
	if !strings.Contains(err.Error(), "RS256") {
		t.Errorf("expected error to mention RS256, got: %v", err)
	}
}

// TestApplyDefaults_JWTType_ExplicitRS256WithKey_Succeeds verifies that a
// user who supplies both RS256 and key material is honored as-is (no forced
// downgrade to HS256).
func TestApplyDefaults_JWTType_ExplicitRS256WithKey_Succeeds(t *testing.T) {
	cfg := &Config{
		BaseDomain: "example.com",
		Hasura: HasuraConfig{
			JWTType: "RS256",
			JWTKey:  "-----BEGIN PUBLIC KEY-----\nMIIB...\n-----END PUBLIC KEY-----",
		},
	}
	cfg, err := ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	if cfg.Hasura.JWTType != "RS256" {
		t.Errorf("Hasura.JWTType = %q, want %q (explicit user choice must be honored)", cfg.Hasura.JWTType, "RS256")
	}
}

// TestApplyDefaults_JWTType_ExplicitHS256Preserved verifies an explicit
// HS256 setting passes through unchanged (no warning-driven mutation).
func TestApplyDefaults_JWTType_ExplicitHS256Preserved(t *testing.T) {
	cfg := &Config{
		BaseDomain: "example.com",
		Hasura:     HasuraConfig{JWTType: "HS256"},
	}
	cfg, err := ApplyDefaults(cfg)
	if err != nil {
		t.Fatalf("ApplyDefaults() error: %v", err)
	}
	if cfg.Hasura.JWTType != "HS256" {
		t.Errorf("Hasura.JWTType = %q, want %q", cfg.Hasura.JWTType, "HS256")
	}
}

// TestBuildJWTSecret_MatchesAuthAndHasura verifies the JWT-ALGO-01 invariant:
// the JSON blob BuildJWTSecret produces for Hasura's JWT_SECRET always encodes
// the SAME type+key that buildAuthService would send to auth as
// AUTH_JWT_TYPE/AUTH_JWT_SECRET (both read from cfg.Hasura.JWTType/JWTKey).
func TestBuildJWTSecret_MatchesAuthAndHasura(t *testing.T) {
	t.Setenv("HASURA_GRAPHQL_JWT_SECRET", "")
	cfg := &Config{
		Hasura: HasuraConfig{JWTKey: "sharedsecretkey1234567890", JWTType: "HS256"},
	}
	secretJSON, err := BuildJWTSecret(cfg)
	if err != nil {
		t.Fatalf("BuildJWTSecret() error: %v", err)
	}
	var obj struct {
		Type string `json:"type"`
		Key  string `json:"key"`
	}
	if err := json.Unmarshal([]byte(secretJSON), &obj); err != nil {
		t.Fatalf("BuildJWTSecret() did not return valid JSON: %v", err)
	}
	if obj.Type != cfg.Hasura.JWTType {
		t.Errorf("Hasura JWT secret type = %q, want %q (must match AUTH_JWT_TYPE source)", obj.Type, cfg.Hasura.JWTType)
	}
	if obj.Key != cfg.Hasura.JWTKey {
		t.Errorf("Hasura JWT secret key = %q, want %q (must match AUTH_JWT_SECRET source)", obj.Key, cfg.Hasura.JWTKey)
	}
}

// ── Gap #4: JWT key persistence across rebuilds (incl. --force) ────────────

// TestParseEnvToConfig_AuthJWTSecretAlias verifies that when a user (or app
// .env, e.g. ntask) declares AUTH_JWT_SECRET but not HASURA_JWT_KEY, the
// loader still populates cfg.Hasura.JWTKey from it — so ApplyDefaults does
// not regenerate a brand new key on every build.
func TestParseEnvToConfig_AuthJWTSecretAlias(t *testing.T) {
	t.Setenv("HASURA_JWT_KEY", "")
	t.Setenv("AUTH_JWT_SECRET", "user-supplied-signing-key-1234567890")
	t.Setenv("HASURA_GRAPHQL_JWT_SECRET", "")

	cfg := parseEnvToConfig()
	if cfg.Hasura.JWTKey != "user-supplied-signing-key-1234567890" {
		t.Errorf("Hasura.JWTKey = %q, want the AUTH_JWT_SECRET value", cfg.Hasura.JWTKey)
	}
}

// TestParseEnvToConfig_HasuraJWTKeyWinsOverAuthJWTSecret verifies
// HASURA_JWT_KEY (the more specific/canonical var) takes priority over the
// AUTH_JWT_SECRET alias when both are set.
func TestParseEnvToConfig_HasuraJWTKeyWinsOverAuthJWTSecret(t *testing.T) {
	t.Setenv("HASURA_JWT_KEY", "canonical-key")
	t.Setenv("AUTH_JWT_SECRET", "alias-key")
	t.Setenv("HASURA_GRAPHQL_JWT_SECRET", "")

	cfg := parseEnvToConfig()
	if cfg.Hasura.JWTKey != "canonical-key" {
		t.Errorf("Hasura.JWTKey = %q, want %q (HASURA_JWT_KEY must win)", cfg.Hasura.JWTKey, "canonical-key")
	}
}

// TestParseEnvToConfig_PersistedJWTSecretJSONSurvivesRebuild verifies that
// when only the full HASURA_GRAPHQL_JWT_SECRET JSON blob is present on disk
// (as persistGeneratedSecrets writes to .env.secrets), the loader extracts
// JWTKey/JWTType from it — this is the exact --force regression: without
// this fallback, ApplyDefaults would see an empty JWTKey and regenerate a
// new one even though a perfectly good key was already persisted.
func TestParseEnvToConfig_PersistedJWTSecretJSONSurvivesRebuild(t *testing.T) {
	t.Setenv("HASURA_JWT_KEY", "")
	t.Setenv("AUTH_JWT_SECRET", "")
	t.Setenv("HASURA_GRAPHQL_JWT_SECRET", `{"type":"HS256","key":"persisted-key-from-env-secrets"}`)

	cfg := parseEnvToConfig()
	if cfg.Hasura.JWTKey != "persisted-key-from-env-secrets" {
		t.Errorf("Hasura.JWTKey = %q, want the key extracted from HASURA_GRAPHQL_JWT_SECRET JSON", cfg.Hasura.JWTKey)
	}
	if cfg.Hasura.JWTType != "HS256" {
		t.Errorf("Hasura.JWTType = %q, want %q extracted from the persisted JSON", cfg.Hasura.JWTType, "HS256")
	}
}

// TestParseHasuraJWTSecretJSON_InvalidInputReturnsNotOK verifies the helper
// degrades gracefully (ok=false) on empty or malformed input rather than
// panicking or returning a misleading zero-value key.
func TestParseHasuraJWTSecretJSON_InvalidInputReturnsNotOK(t *testing.T) {
	cases := []string{"", "not json", `{"type":"HS256"}`, `{"key":""}`}
	for _, raw := range cases {
		if _, _, ok := parseHasuraJWTSecretJSON(raw); ok {
			t.Errorf("parseHasuraJWTSecretJSON(%q) ok = true, want false", raw)
		}
	}
}
