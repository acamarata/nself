package plugin

// token_validator.go — Q01 Phase B-1: inbound JWT validation for plugin-to-plugin calls.
//
// ValidateIncomingJWT verifies that a JWT presented by a calling plugin:
//  1. Has a valid EdDSA signature from ping_api (verified against the cached public key).
//  2. Has not expired (exp claim).
//  3. Has the correct audience (aud == "plugin:<expectedAudience>").
//  4. Has not been replayed (jti is unique within the Redis replay cache).
//
// The ping_api public key is fetched on first call and cached in memory for the
// process lifetime. Key rotation is supported via the kid header — new keys are
// fetched when an unknown kid is encountered.
//
// Environment variables:
//   PLUGIN_TOKEN_ISSUER    — base URL for ping_api (default: https://ping.nself.org)
//   PLUGIN_REPLAY_CACHE_TTL — Redis TTL in seconds for jti replay (default: 300)

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// maxClockSkew is the maximum tolerated difference between the JWT iat and the
// local clock. This is not the same as the replay window; the exp claim is
// authoritative for expiry.
const maxClockSkew = 30 * time.Second

// jwtHeader is the decoded JWT header.
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

// jwtClaims is the subset of JWT claims used for validation.
type jwtClaims struct {
	Sub string `json:"sub"` // "plugin:<name>"
	Aud string `json:"aud"` // "plugin:<name>"
	Iss string `json:"iss"` // "ping.nself.org"
	Jti string `json:"jti"` // UUID v4 — replay cache key
	Iat int64  `json:"iat"` // issued-at (Unix seconds)
	Exp int64  `json:"exp"` // expiry (Unix seconds)
}

// publicKeyCache holds fetched ping_api public keys keyed by kid.
var (
	publicKeyCache   = map[string]ed25519.PublicKey{}
	publicKeyCacheMu sync.RWMutex
)

// ReplayStore is the interface for jti replay caching. A Redis-backed
// implementation is provided by the shared auth package. A no-op shim is used
// when Redis is unavailable (degrades replay protection, logs a warning).
type ReplayStore interface {
	// Seen returns true and records the jti if it has not been seen before.
	// The entry expires after ttl. Returns (false, nil) if jti is new.
	Seen(ctx context.Context, jti string, ttl time.Duration) (bool, error)
}

// ValidateIncomingJWT validates a JWT presented in an inter-plugin HTTP call.
// expectedAudience is the short plugin name (e.g., "mux"), NOT the full aud
// claim (which has the "plugin:" prefix).
//
// Returns nil on success, a descriptive error on any validation failure.
func ValidateIncomingJWT(ctx context.Context, tokenStr, expectedAudience string, store ReplayStore) error {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return fmt.Errorf("plugin JWT: malformed token (expected 3 parts, got %d)", len(parts))
	}

	// Decode header.
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("plugin JWT: invalid header encoding: %w", err)
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return fmt.Errorf("plugin JWT: invalid header JSON: %w", err)
	}
	if header.Alg != "EdDSA" {
		return fmt.Errorf("plugin JWT: unexpected algorithm %q (want EdDSA)", header.Alg)
	}

	// Decode payload.
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("plugin JWT: invalid payload encoding: %w", err)
	}
	var claims jwtClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return fmt.Errorf("plugin JWT: invalid payload JSON: %w", err)
	}

	// Decode signature.
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("plugin JWT: invalid signature encoding: %w", err)
	}

	// Fetch public key (cache by kid).
	pubKey, err := getPublicKey(ctx, header.Kid)
	if err != nil {
		return fmt.Errorf("plugin JWT: fetching public key (kid=%q): %w", header.Kid, err)
	}

	// Verify Ed25519 signature over <header>.<payload>.
	signingInput := parts[0] + "." + parts[1]
	if !ed25519.Verify(pubKey, []byte(signingInput), sig) {
		return fmt.Errorf("plugin JWT: signature verification failed")
	}

	// Validate expiry.
	now := time.Now()
	if now.Unix() > claims.Exp {
		return fmt.Errorf("plugin JWT: token expired at %s", time.Unix(claims.Exp, 0).Format(time.RFC3339))
	}

	// Validate issuer.
	if claims.Iss != "ping.nself.org" {
		return fmt.Errorf("plugin JWT: unexpected issuer %q", claims.Iss)
	}

	// Validate audience: must be "plugin:<expectedAudience>".
	wantAud := "plugin:" + expectedAudience
	if claims.Aud != wantAud {
		return fmt.Errorf("plugin JWT: audience mismatch: got %q, want %q", claims.Aud, wantAud)
	}

	// Replay check via jti.
	if claims.Jti == "" {
		return fmt.Errorf("plugin JWT: missing jti claim")
	}
	if store != nil {
		replayTTL := replayCacheTTL()
		seen, err := store.Seen(ctx, claims.Jti, replayTTL)
		if err != nil {
			return fmt.Errorf("plugin JWT: replay store error: %w", err)
		}
		if seen {
			return fmt.Errorf("plugin JWT: token replay detected (jti=%s)", claims.Jti)
		}
	}

	return nil
}

// getPublicKey returns the ping_api Ed25519 public key for the given kid,
// fetching from the public endpoint if not cached.
func getPublicKey(ctx context.Context, kid string) (ed25519.PublicKey, error) {
	publicKeyCacheMu.RLock()
	key, ok := publicKeyCache[kid]
	publicKeyCacheMu.RUnlock()
	if ok {
		return key, nil
	}

	// Fetch from ping_api.
	issuer := os.Getenv("PLUGIN_TOKEN_ISSUER")
	if issuer == "" {
		issuer = defaultTokenIssuer
	}
	url := issuer + "/plugin/identity/" + kid + "/public-key"

	httpCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating public-key request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching public key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("public-key endpoint returned HTTP %d for kid=%q", resp.StatusCode, kid)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return nil, fmt.Errorf("reading public key response: %w", err)
	}

	// Expect base64-encoded 32-byte public key.
	keyBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(body)))
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key has unexpected length %d (want %d)", len(keyBytes), ed25519.PublicKeySize)
	}

	pubKey := ed25519.PublicKey(keyBytes)

	publicKeyCacheMu.Lock()
	publicKeyCache[kid] = pubKey
	publicKeyCacheMu.Unlock()

	return pubKey, nil
}

// replayCacheTTL reads PLUGIN_REPLAY_CACHE_TTL and returns the duration.
// Defaults to 300 seconds if the env var is absent or invalid.
func replayCacheTTL() time.Duration {
	s := os.Getenv("PLUGIN_REPLAY_CACHE_TTL")
	if s == "" {
		return defaultTokenTTL * time.Second
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return defaultTokenTTL * time.Second
	}
	return time.Duration(n) * time.Second
}
