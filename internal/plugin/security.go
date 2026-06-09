package plugin

// Purpose: Plugin security checks — checksum/signature verification, Ed25519
//          CRL author revocation, license validation, EOL blocking, and the
//          anonymous install-event telemetry POST.
// Inputs:  archive file paths, hex-encoded keys/sigs, context with timeout.
// Outputs: error on any policy violation; nil on pass or when non-fatal
//          (network offline, non-stable status, or author not found in CRL).
// Constraints: Stable plugins MUST have checksum + signature (errs.ErrPlugin*).
//              License check tries all keys; first valid entitlement match wins.
//              CRL fetch errors are non-fatal — warns to stderr, never blocks.
//              postInstallEvent always runs in a goroutine; errors are silent.
// SPORT: security/verification pipeline; callers: installLocked in installer.go

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/httptimeout"
	"github.com/nself-org/cli/internal/license"
)

// postInstallEvent POSTs an anonymous install count event to the registry worker.
// It is always called in a goroutine and swallows all errors silently.
// The instanceId is a SHA-256 hex hash of the machineID — opaque, no PII.
func postInstallEvent(pluginName string) {
	mid := machineID() // 16-char hex
	// SHA-256 of the machineID to produce the required 64-char hex instanceId
	h := sha256.Sum256([]byte(mid))
	instanceID := hex.EncodeToString(h[:])

	body := `{"instanceId":"` + instanceID + `"}`
	url := "https://plugins.nself.org/plugins/" + pluginName + "/install-event"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return // silent
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httptimeout.Plugin.Do(req)
	if err != nil {
		return // silent
	}
	_ = resp.Body.Close()
}

// verifyChecksum computes the SHA256 hash of the file at filePath and compares
// it to expectedHash (hex-encoded). Returns an error if the hashes differ.
//
// publishStatus is the plugin's lifecycle status from the registry
// ("stable", "beta", "alpha", "experimental", etc.). When publishStatus is
// "stable" and expectedHash is empty, the function returns
// errs.ErrPluginMissingChecksum — install is refused. For non-stable plugins
// an empty expectedHash is permitted (a warning is emitted to stderr).
func verifyChecksum(filePath string, expectedHash string, publishStatus string) error {
	if expectedHash == "" {
		if publishStatus == "stable" {
			return fmt.Errorf("plugin %q is missing required checksum for stable publishStatus — install refused: %w",
				filePath, errs.ErrPluginMissingChecksum)
		}
		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("computing checksum: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expectedHash) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actual)
	}
	return nil
}

// verifyPluginSignature verifies that the Ed25519 signature stored in the
// plugin's registry manifest matches the SHA-256 hash of the downloaded
// tarball. The public key is pinned in the registry (never fetched at verify
// time, preventing TOCTOU attacks).
//
// publishStatus is the plugin's lifecycle status from the registry
// ("stable", "beta", "alpha", "experimental", etc.). When publishStatus is
// "stable" and either authorPublicKeyHex or signatureHex is empty, the
// function returns errs.ErrPluginUnsigned — install is refused. For
// non-stable plugins an empty key or signature skips verification with a
// warning (development workflow).
func verifyPluginSignature(archivePath, authorPublicKeyHex, signatureHex, publishStatus string) error {
	if authorPublicKeyHex == "" || signatureHex == "" {
		if publishStatus == "stable" {
			return fmt.Errorf("plugin is missing required signature for stable publishStatus — install refused: %w",
				errs.ErrPluginUnsigned)
		}
		return nil
	}

	pkBytes, err := hex.DecodeString(authorPublicKeyHex)
	if err != nil {
		return fmt.Errorf("decoding author public key: %w", err)
	}
	if len(pkBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("author public key has wrong length: expected %d bytes, got %d", ed25519.PublicKeySize, len(pkBytes))
	}
	pubKey := ed25519.PublicKey(pkBytes)

	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return fmt.Errorf("decoding plugin signature: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("plugin signature has wrong length: expected %d bytes, got %d", ed25519.SignatureSize, len(sigBytes))
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive for signature verification: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing archive for signature verification: %w", err)
	}
	digest := h.Sum(nil)

	if !ed25519.Verify(pubKey, digest, sigBytes) {
		return fmt.Errorf("plugin signature verification failed: tarball does not match registry signature (possible tampering)")
	}
	return nil
}

// checkLicense validates that a license key exists and is acceptable.
// It checks all configured keys (multi-key support) against the local
// entitlement cache first, then falls through to remote validation.
func checkLicense(ctx context.Context, name string) error {
	keys := license.CollectLicenseKeys()

	if len(keys) == 0 {
		key := os.Getenv("NSELF_PLUGIN_LICENSE_KEY")
		if key == "" {
			keyPath := licenseKeyPath()
			data, err := os.ReadFile(keyPath)
			if err != nil {
				return fmt.Errorf("plugin %q requires a license key. Run 'nself license add <key>' or visit %s",
					name, "nself.org/pricing")
			}
			key = strings.TrimSpace(string(data))
		}
		if key != "" {
			keys = []string{key}
		}
	}

	if len(keys) == 0 {
		return fmt.Errorf("plugin %q requires a license key. Run 'nself license add <key>' or visit %s",
			name, "nself.org/pricing")
	}

	cacheDir := licenseCacheDir()

	if allowed, found := checkEntitlements(cacheDir, name); found {
		if allowed {
			return nil
		}
	}

	var lastErr error
	for _, key := range keys {
		if err := validateLicenseFormat(key); err != nil {
			continue
		}

		if valid, found := checkLicenseCache(key, cacheDir); found {
			if valid {
				return nil
			}
			continue
		}

		valid, lvr, err := validateLicenseRemoteWithEntitlements(ctx, key, pingAPIURL())
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				lastErr = fmt.Errorf("plugin %q: %w", name, errs.ErrLicenseExpired)
				continue
			}
			if offlineValid, offlineFound := checkLicenseCacheOffline(key, cacheDir); offlineFound && offlineValid {
				return nil
			}
			lastErr = fmt.Errorf("plugin %q: %w", name, errs.ErrLicenseNetworkUnavailable)
			continue
		}
		_ = CacheLicense(key, valid, cacheDir)

		if lvr != nil && len(lvr.Plugins) > 0 {
			_ = cacheEntitlements(cacheDir, lvr.Tier, lvr.Plugins)
			for _, p := range lvr.Plugins {
				if p == name {
					return nil
				}
			}
		}

		if valid {
			lastErr = fmt.Errorf("plugin %q: %w", name, errs.ErrLicenseTierTooLow)
			continue
		}
		lastErr = fmt.Errorf("plugin %q: license key is not valid", name)
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("plugin %q requires a license key. Get one at %s", name, "nself.org/pricing")
}

// CheckEOLBlock fetches the registry entry for name and returns an error if
// the plugin's status is "eol" and allowEOL is false. (S58-T03)
// If the registry cannot be reached or the plugin is not found, this function
// returns nil (install will fail downstream with a more specific error).
func CheckEOLBlock(ctx context.Context, name string, allowEOL bool) error {
	cacheDir := defaultCacheDir()
	reg, err := FetchRegistry(ctx, "", cacheDir)
	if err != nil {
		return nil
	}
	manifest, found := findPlugin(reg, name)
	if !found {
		return nil
	}
	if manifest.PublishStatus == "eol" && !allowEOL {
		return fmt.Errorf(
			"plugin %q has reached end-of-life and cannot be installed.\n"+
				"Use --allow-eol to override (not recommended).\n"+
				"Run 'nself plugin info %s' for details.",
			name, name,
		)
	}
	return nil
}

// checkAuthorRevocation fetches the author CRL from plugins.nself.org and
// returns an error if the plugin author appears in the revocation list. (S58-T09)
// Short-circuits on any network / parse error — install proceeds and the
// full install step will catch other issues.
func checkAuthorRevocation(ctx context.Context, author string) error {
	if author == "" {
		return nil
	}

	url := "https://plugins.nself.org/.well-known/revoked-authors.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "nself-cli")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not fetch author revocation list (offline?): %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	type revokedEntry struct {
		AuthorKey string `json:"authorKey"`
		RevokedAt string `json:"revokedAt"`
		Reason    string `json:"reason,omitempty"`
	}
	type crlResponse struct {
		RevokedAuthors []revokedEntry `json:"revokedAuthors"`
	}

	var crl crlResponse
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(body, &crl); err != nil {
		return nil
	}

	for _, entry := range crl.RevokedAuthors {
		if strings.EqualFold(entry.AuthorKey, author) {
			msg := fmt.Sprintf(
				"plugin author %q has been revoked and cannot be installed (revoked: %s).\n"+
					"Remove any plugins from this author and contact support@nself.org.\n"+
					"See https://nself.org/security/revocations for details.",
				author, entry.RevokedAt,
			)
			if entry.Reason != "" {
				msg += "\nReason: " + entry.Reason
			}
			return fmt.Errorf("%s", msg)
		}
	}
	return nil
}
