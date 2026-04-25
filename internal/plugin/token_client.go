package plugin

// token_client.go — Q01 Phase B-1: per-plugin JWT token request client.
//
// RequestToken performs an Ed25519 challenge-response against the ping_api
// /plugin/token endpoint and returns a short-lived (5-min) JWT scoped to the
// specified target plugin.
//
// Flow:
//  1. Load the plugin's private key from disk.
//  2. POST to /plugin/token with plugin identity + signature.
//  3. Return the JWT string for use in outbound inter-plugin HTTP calls.
//
// Environment variables:
//   PLUGIN_TOKEN_ISSUER     — base URL for ping_api (default: https://ping.nself.org)
//   PLUGIN_TOKEN_TTL_SECONDS — requested TTL (default: 300, server may override)

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	defaultTokenIssuer    = "https://ping.nself.org"
	defaultTokenTTL       = 300
	tokenRequestTimeout   = 10 * time.Second
)

// tokenRequest is the JSON body sent to POST /plugin/token.
type tokenRequest struct {
	// SourcePlugin is the plugin requesting the token.
	SourcePlugin string `json:"source_plugin"`
	// TargetPlugin is the intended audience (aud claim in the issued JWT).
	TargetPlugin string `json:"target_plugin"`
	// Nonce is a random 32-byte value base64url-encoded to prevent replay of
	// the request itself (distinct from the JWT jti replay cache).
	Nonce string `json:"nonce"`
	// Timestamp is the Unix second at which the request was signed.
	Timestamp int64 `json:"timestamp"`
	// Signature is the Ed25519 signature over the canonical request body:
	//   <source_plugin>:<target_plugin>:<nonce>:<timestamp>
	Signature string `json:"signature"`
}

// tokenResponse is the JSON body returned by POST /plugin/token.
type tokenResponse struct {
	Token string `json:"token"`
	// ExpiresAt is informational; the JWT exp claim is authoritative.
	ExpiresAt int64 `json:"expires_at"`
}

// RequestToken authenticates sourcePlugin with ping_api via Ed25519
// challenge-response and returns a 5-minute JWT scoped to targetPlugin.
// The private key for sourcePlugin must already exist on disk (generated
// during plugin install).
func RequestToken(ctx context.Context, pluginDataDir, sourcePlugin, targetPlugin string) (string, error) {
	privKey, err := LoadPrivateKey(pluginDataDir, sourcePlugin)
	if err != nil {
		return "", fmt.Errorf("loading identity key for %q: %w", sourcePlugin, err)
	}

	nonceBytes := make([]byte, 32)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("generating request nonce: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	ts := time.Now().Unix()

	// Canonical message: <source>:<target>:<nonce>:<timestamp>
	msg := fmt.Sprintf("%s:%s:%s:%d", sourcePlugin, targetPlugin, nonce, ts)
	sig := ed25519.Sign(privKey, []byte(msg))

	reqBody := tokenRequest{
		SourcePlugin: sourcePlugin,
		TargetPlugin: targetPlugin,
		Nonce:        nonce,
		Timestamp:    ts,
		Signature:    base64.RawURLEncoding.EncodeToString(sig),
	}

	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshalling token request: %w", err)
	}

	issuer := os.Getenv("PLUGIN_TOKEN_ISSUER")
	if issuer == "" {
		issuer = defaultTokenIssuer
	}
	url := issuer + "/plugin/token"

	httpCtx, cancel := context.WithTimeout(ctx, tokenRequestTimeout)
	defer cancel()

	resp, err := http.NewRequestWithContext(httpCtx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	resp.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	httpResp, err := client.Do(resp)
	if err != nil {
		return "", fmt.Errorf("requesting plugin token from %s: %w", url, err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned HTTP %d for %s→%s", httpResp.StatusCode, sourcePlugin, targetPlugin)
	}

	var tr tokenResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}
	if tr.Token == "" {
		return "", fmt.Errorf("token endpoint returned empty token for %s→%s", sourcePlugin, targetPlugin)
	}

	return tr.Token, nil
}
