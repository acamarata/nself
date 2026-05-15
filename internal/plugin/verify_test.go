package plugin

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goldenBundle holds the reference inputs used by the golden-file tests.
// These MUST match the inputs used in plugin-sign.test.ts so both suites
// validate byte-identical canonical output.
const (
	goldenBundle     = "nclaw"
	goldenName       = "claw"
	goldenSHA256     = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	goldenSize int64 = 102400
	goldenVis        = "public"
	goldenVersion    = "1.2.3"
)

// TestCanonicalPluginObject_GoldenFile verifies that canonicalPluginObject
// produces byte-identical output to the shared golden fixture that the TypeScript
// implementation is also tested against.
func TestCanonicalPluginObject_GoldenFile(t *testing.T) {
	got, err := canonicalPluginObject(goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize)
	if err != nil {
		t.Fatalf("canonicalPluginObject returned error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "canonical_golden.json")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden fixture %s: %v", goldenPath, err)
	}

	// Trim any trailing newline the editor may have appended to the fixture file.
	want = []byte(strings.TrimRight(string(want), "\n"))

	if string(got) != string(want) {
		t.Errorf("canonical object mismatch:\n  got:  %s\n  want: %s", got, want)
	}
}

// TestCanonicalPluginObject_FieldOrder verifies the JSON key order is strictly
// alphabetical: bundle, name, sha256, size, visibility, version.
func TestCanonicalPluginObject_FieldOrder(t *testing.T) {
	got, err := canonicalPluginObject(goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize)
	if err != nil {
		t.Fatalf("canonicalPluginObject returned error: %v", err)
	}

	s := string(got)
	keys := []string{"\"bundle\"", "\"name\"", "\"sha256\"", "\"size\"", "\"visibility\"", "\"version\""}
	prev := -1
	for _, key := range keys {
		pos := strings.Index(s, key)
		if pos < 0 {
			t.Errorf("key %s not found in canonical output: %s", key, s)
			continue
		}
		if pos <= prev {
			t.Errorf("key %s appears before previous key (pos %d <= %d) in: %s", key, pos, prev, s)
		}
		prev = pos
	}
}

// TestCanonicalPluginObject_NoWhitespace verifies the canonical JSON has no
// whitespace characters (compact serialization).
func TestCanonicalPluginObject_NoWhitespace(t *testing.T) {
	got, err := canonicalPluginObject(goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize)
	if err != nil {
		t.Fatalf("canonicalPluginObject returned error: %v", err)
	}
	for _, c := range got {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			t.Errorf("canonical output contains whitespace (byte %d): %s", c, got)
			break
		}
	}
}

// TestCanonicalPluginObject_ValidationErrors verifies that empty/invalid inputs
// are rejected.
func TestCanonicalPluginObject_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		bundle     string
		pluginName string
		sha256     string
		visibility string
		version    string
		size       int64
		wantErr    bool
	}{
		{"valid", goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize, false},
		{"empty bundle", "", goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize, true},
		{"empty name", goldenBundle, "", goldenSHA256, goldenVis, goldenVersion, goldenSize, true},
		{"empty sha256", goldenBundle, goldenName, "", goldenVis, goldenVersion, goldenSize, true},
		{"empty version", goldenBundle, goldenName, goldenSHA256, goldenVis, "", goldenSize, true},
		{"empty visibility", goldenBundle, goldenName, goldenSHA256, "", goldenVersion, goldenSize, true},
		{"negative size", goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, -1, true},
		{"zero size ok", goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := canonicalPluginObject(tt.bundle, tt.pluginName, tt.sha256, tt.visibility, tt.version, tt.size)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestVerifyCanonicalSignature_RoundTrip signs the canonical object and verifies
// it using VerifyCanonicalSignature, confirming the full sign→verify cycle works.
func TestVerifyCanonicalSignature_RoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	pubHex := hex.EncodeToString(pub)
	privHex := hex.EncodeToString(priv)

	sigHex, err := SignCanonicalObject(goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize, privHex)
	if err != nil {
		t.Fatalf("SignCanonicalObject failed: %v", err)
	}

	if err := VerifyCanonicalSignature(goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize, pubHex, sigHex); err != nil {
		t.Errorf("VerifyCanonicalSignature rejected valid signature: %v", err)
	}
}

// TestVerifyCanonicalSignature_TamperedField verifies that changing any field
// after signing causes verification to fail.
func TestVerifyCanonicalSignature_TamperedField(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	pubHex := hex.EncodeToString(pub)
	privHex := hex.EncodeToString(priv)

	sigHex, err := SignCanonicalObject(goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize, privHex)
	if err != nil {
		t.Fatalf("SignCanonicalObject failed: %v", err)
	}

	tampers := []struct {
		name    string
		bundle  string
		pname   string
		sha256  string
		vis     string
		version string
		size    int64
	}{
		{"tampered_bundle", "TAMPERED", goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize},
		{"tampered_name", goldenBundle, "TAMPERED", goldenSHA256, goldenVis, goldenVersion, goldenSize},
		{"tampered_sha256", goldenBundle, goldenName, "aabbcc", goldenVis, goldenVersion, goldenSize},
		{"tampered_visibility", goldenBundle, goldenName, goldenSHA256, "private", goldenVersion, goldenSize},
		{"tampered_version", goldenBundle, goldenName, goldenSHA256, goldenVis, "9.9.9", goldenSize},
		{"tampered_size", goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, 999},
	}

	for _, tt := range tampers {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyCanonicalSignature(tt.bundle, tt.pname, tt.sha256, tt.vis, tt.version, tt.size, pubHex, sigHex)
			if err == nil {
				t.Errorf("expected verification failure for tampered field, got nil")
			}
		})
	}
}

// TestVerifyCanonicalSignature_WrongKey verifies that a valid signature from
// key A fails verification against key B.
func TestVerifyCanonicalSignature_WrongKey(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	wrongPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	privHex := hex.EncodeToString(priv)
	wrongPubHex := hex.EncodeToString(wrongPub)

	sigHex, err := SignCanonicalObject(goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize, privHex)
	if err != nil {
		t.Fatalf("SignCanonicalObject failed: %v", err)
	}

	err = VerifyCanonicalSignature(goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize, wrongPubHex, sigHex)
	if err == nil {
		t.Errorf("expected verification failure with wrong key, got nil")
	}
}

// TestVerifyCanonicalSignature_InvalidHex verifies that malformed hex inputs
// are rejected with an error rather than a panic.
func TestVerifyCanonicalSignature_InvalidHex(t *testing.T) {
	if err := VerifyCanonicalSignature(goldenBundle, goldenName, goldenSHA256, goldenVis, goldenVersion, goldenSize, "ZZZZ", "ZZZZ"); err == nil {
		t.Errorf("expected error for invalid public key hex, got nil")
	}
}
