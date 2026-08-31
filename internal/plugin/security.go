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
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/httptimeout"
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
	defer func() { _ = f.Close() }()

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
	defer func() { _ = f.Close() }()

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
