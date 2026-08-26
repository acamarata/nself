// Package access implements the business logic behind `nself access grant /
// revoke / list` — managing SSH public keys in the authorized_keys file of an
// already-deployed nself host. It never touches private key material: every
// entry point here accepts or produces public keys and fingerprints only.
//
// Purpose: fill the gap where hcloud only injects SSH keys at server-creation
// time, leaving no CLI path to grant or revoke access on a running box
// without hand-editing authorized_keys over a raw ssh session.
// Inputs: a Transport (SSHTransport in production, LocalFileTransport in
// tests) plus a parsed PublicKey and request options.
// Outputs: GrantResult / RevokeResult / ListResult describing what changed,
// including the key fingerprint for verification.
// Constraints: idempotent grant, timestamped backup before every mutation,
// a lockout guard on revoke, and an audit line per mutation. See
// manager.go, transport.go, and audit.go for the pieces of that contract.
package access

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// knownKeyTypes are the SSH public key algorithm identifiers this package
// accepts in an authorized_keys line. FIDO2/U2F "sk-" types are deliberately
// excluded: their security-key application string can't be verified without
// real hardware, which is out of scope for static authorized_keys management.
var knownKeyTypes = map[string]bool{
	"ssh-ed25519":         true,
	"ssh-rsa":             true,
	"ecdsa-sha2-nistp256": true,
	"ecdsa-sha2-nistp384": true,
	"ecdsa-sha2-nistp521": true,
}

// PublicKey is a parsed SSH public key: its algorithm identifier and the
// base64-encoded key blob. A PublicKey never carries private key material.
type PublicKey struct {
	Type string
	Data string // base64, undecoded
}

// Fingerprint returns the key's fingerprint in the same format `ssh-keygen
// -lf` prints: "SHA256:<unpadded base64 of sha256(key blob)>".
func (k PublicKey) Fingerprint() (string, error) {
	raw, err := base64.StdEncoding.DecodeString(k.Data)
	if err != nil {
		return "", fmt.Errorf("decode key data: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]), nil
}

// Line renders "<type> <data>" with no comment. Callers append their own
// nself-managed tag comment (see entry.go).
func (k PublicKey) Line() string {
	return k.Type + " " + k.Data
}

// ParsePublicKey parses a single authorized_keys-style public key line
// ("<type> <base64> [comment...]"), ignoring any comment. It refuses input
// that looks like a private key without echoing that input anywhere, so a
// pasted private key is never reflected back to the terminal or a log line.
func ParsePublicKey(input string) (PublicKey, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return PublicKey{}, fmt.Errorf("empty key")
	}
	if strings.Contains(trimmed, "PRIVATE KEY") || strings.HasPrefix(trimmed, "-----BEGIN") {
		return PublicKey{}, fmt.Errorf("input looks like a private key, refusing to process it")
	}
	fields := strings.Fields(trimmed)
	if len(fields) < 2 {
		return PublicKey{}, fmt.Errorf(`not a valid public key: expected "<type> <base64> [comment]"`)
	}
	keyType, data := fields[0], fields[1]
	if !knownKeyTypes[keyType] {
		return PublicKey{}, fmt.Errorf("unrecognized key type %q", keyType)
	}
	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		return PublicKey{}, fmt.Errorf("key data is not valid base64: %w", err)
	}
	return PublicKey{Type: keyType, Data: data}, nil
}

// LoadPublicKeyArg resolves the `--key <pubkey|@file>` convention: a leading
// '@' means read the key from the given file path, otherwise arg is the key
// material itself.
func LoadPublicKeyArg(arg string) (PublicKey, error) {
	if strings.HasPrefix(arg, "@") {
		path := strings.TrimPrefix(arg, "@")
		data, err := os.ReadFile(path)
		if err != nil {
			return PublicKey{}, fmt.Errorf("read key file %s: %w", path, err)
		}
		return ParsePublicKey(string(data))
	}
	return ParsePublicKey(arg)
}
