package plugin

// identity_register.go — Q01 Phase B-1: public key registration and revocation.
//
// RegisterIdentity posts the plugin's Ed25519 public key to ping_api at
// install time. The public key is used by other plugins (via the shared
// Verifier) to verify inbound inter-plugin JWTs.
//
// RevokeIdentity deletes the registration from ping_api at uninstall time.
// The private key is removed from disk after successful revocation.
//
// Bootstrap auth: registration uses the PLUGIN_INTERNAL_SECRET header (a
// shared bootstrap credential). Token-based auth is not used here because
// the plugin does not yet have a JWT signing identity at registration time.

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const registrationTimeout = 15 * time.Second

// identityRegisterRequest is the body for POST /plugin/identity/register.
type identityRegisterRequest struct {
	PluginName string `json:"plugin_name"`
	// PublicKey is base64url-encoded Ed25519 public key (32 bytes).
	PublicKey string `json:"public_key"`
	// Action is "register" or "rotate".
	Action string `json:"action"`
}

// RegisterIdentity posts the plugin's public key to ping_api.
// pubKey is the raw 32-byte Ed25519 public key returned by GenerateEd25519Keypair.
//
// If the plugin has previously registered (e.g., re-install without uninstall),
// this is treated as a "register" action; ping_api will update the record if the
// plugin_name already exists (idempotent).
func RegisterIdentity(ctx context.Context, pluginName string, pubKey ed25519.PublicKey) error {
	secret := os.Getenv("PLUGIN_INTERNAL_SECRET")
	if secret == "" {
		return fmt.Errorf("PLUGIN_INTERNAL_SECRET not set — cannot register plugin identity with ping_api")
	}

	reqBody := identityRegisterRequest{
		PluginName: pluginName,
		PublicKey:  base64.RawURLEncoding.EncodeToString(pubKey),
		Action:     "register",
	}
	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshalling identity registration request: %w", err)
	}

	issuer := os.Getenv("PLUGIN_TOKEN_ISSUER")
	if issuer == "" {
		issuer = defaultTokenIssuer
	}
	url := issuer + "/plugin/identity/register"

	httpCtx, cancel := context.WithTimeout(ctx, registrationTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("creating registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Plugin-Internal-Secret", secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting identity registration to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("identity registration returned HTTP %d for plugin %q", resp.StatusCode, pluginName)
	}

	return nil
}

// RevokeIdentity calls DELETE /plugin/identity/<name> on ping_api to mark the
// plugin identity as revoked. Call this before removing the private key from disk
// so that in-flight tokens issued to this plugin are rejected promptly.
//
// Auth: the owner license key from NSELF_PLUGIN_LICENSE_KEY is used as the
// authorization credential for the delete endpoint.
func RevokeIdentity(ctx context.Context, pluginName string) error {
	issuer := os.Getenv("PLUGIN_TOKEN_ISSUER")
	if issuer == "" {
		issuer = defaultTokenIssuer
	}
	url := issuer + "/plugin/identity/" + pluginName

	httpCtx, cancel := context.WithTimeout(ctx, registrationTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating revocation request: %w", err)
	}

	licenseKey := os.Getenv("NSELF_PLUGIN_LICENSE_KEY")
	if licenseKey != "" {
		req.Header.Set("Authorization", "Bearer "+licenseKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("revoking identity for %q: %w", pluginName, err)
	}
	defer resp.Body.Close()

	// 404 means already gone — treat as success.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("identity revocation returned HTTP %d for plugin %q", resp.StatusCode, pluginName)
	}

	return nil
}
