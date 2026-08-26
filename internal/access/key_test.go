package access

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Real ed25519 test key pairs (private keys never committed; only the public
// half is used below). Fingerprints were cross-checked against
// `ssh-keygen -lf` at generation time.
const (
	aliceKeyLine = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMHXHuK8L4SFSmmpHWBnzPFAcJGYHjABCulfo5ZbKvum alice@laptop"
	aliceFP      = "SHA256:xSHvs9nsoAaHR9Qv9VEm9Y6DcfkbzATUCdhmauT0wPE"

	bobKeyLine = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDmqUPnm+iR5W527+R5Ywlz6xa4buFbxjsuRNndFuV0x bob@laptop"
	bobFP      = "SHA256:LKOm0DCMdjvdbE/MMP/x5Z76+9erJDhRdDYPydH3yL0"
)

func TestParsePublicKey_ValidKey(t *testing.T) {
	k, err := ParsePublicKey(aliceKeyLine)
	if err != nil {
		t.Fatalf("ParsePublicKey: %v", err)
	}
	if k.Type != "ssh-ed25519" {
		t.Errorf("Type = %q, want ssh-ed25519", k.Type)
	}
	fp, err := k.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	if fp != aliceFP {
		t.Errorf("Fingerprint = %q, want %q", fp, aliceFP)
	}
}

func TestParsePublicKey_SecondKeyDifferentFingerprint(t *testing.T) {
	k, err := ParsePublicKey(bobKeyLine)
	if err != nil {
		t.Fatalf("ParsePublicKey: %v", err)
	}
	fp, _ := k.Fingerprint()
	if fp != bobFP {
		t.Errorf("Fingerprint = %q, want %q", fp, bobFP)
	}
	if fp == aliceFP {
		t.Error("bob's fingerprint must not equal alice's")
	}
}

func TestParsePublicKey_RejectsPrivateKeyLookingInput(t *testing.T) {
	_, err := ParsePublicKey("-----BEGIN OPENSSH PRIVATE KEY-----\nsomefakecontent\n-----END OPENSSH PRIVATE KEY-----")
	if err == nil {
		t.Fatal("expected error for private-key-looking input")
	}
	if strings.Contains(err.Error(), "fakecontent") {
		t.Error("error message must not echo the raw input")
	}
}

func TestParsePublicKey_RejectsEmpty(t *testing.T) {
	if _, err := ParsePublicKey(""); err == nil {
		t.Fatal("expected error for empty input")
	}
	if _, err := ParsePublicKey("   "); err == nil {
		t.Fatal("expected error for whitespace-only input")
	}
}

func TestParsePublicKey_RejectsUnknownType(t *testing.T) {
	_, err := ParsePublicKey("ssh-made-up AAAAB3NzaC1yc2EAAAADAQABAAABAQ== comment")
	if err == nil {
		t.Fatal("expected error for unrecognized key type")
	}
}

func TestParsePublicKey_RejectsBadBase64(t *testing.T) {
	_, err := ParsePublicKey("ssh-ed25519 not-valid-base64!!! comment")
	if err == nil {
		t.Fatal("expected error for invalid base64 key data")
	}
}

func TestParsePublicKey_RejectsSingleField(t *testing.T) {
	if _, err := ParsePublicKey("ssh-ed25519"); err == nil {
		t.Fatal("expected error for a single-field line")
	}
}

func TestParsePublicKey_IgnoresComment(t *testing.T) {
	k, err := ParsePublicKey("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMHXHuK8L4SFSmmpHWBnzPFAcJGYHjABCulfo5ZbKvum this comment is discarded")
	if err != nil {
		t.Fatalf("ParsePublicKey: %v", err)
	}
	fp, _ := k.Fingerprint()
	if fp != aliceFP {
		t.Errorf("Fingerprint = %q, want %q", fp, aliceFP)
	}
}

func TestLoadPublicKeyArg_InlineKey(t *testing.T) {
	k, err := LoadPublicKeyArg(aliceKeyLine)
	if err != nil {
		t.Fatalf("LoadPublicKeyArg: %v", err)
	}
	fp, _ := k.Fingerprint()
	if fp != aliceFP {
		t.Errorf("Fingerprint = %q, want %q", fp, aliceFP)
	}
}

func TestLoadPublicKeyArg_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alice.pub")
	if err := os.WriteFile(path, []byte(aliceKeyLine+"\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	k, err := LoadPublicKeyArg("@" + path)
	if err != nil {
		t.Fatalf("LoadPublicKeyArg: %v", err)
	}
	fp, _ := k.Fingerprint()
	if fp != aliceFP {
		t.Errorf("Fingerprint = %q, want %q", fp, aliceFP)
	}
}

func TestLoadPublicKeyArg_MissingFile(t *testing.T) {
	_, err := LoadPublicKeyArg("@/nonexistent/path/to/key.pub")
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
}
