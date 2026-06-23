package commands

import (
	"os"
	"testing"
)

// TestEncryptionCmd_Structure verifies subcommands are registered.
func TestEncryptionCmd_Structure(t *testing.T) {
	subs := encryptionCmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subs {
		names[cmd.Name()] = true
	}
	for _, name := range []string{"configure", "verify", "rotate", "status", "key-events"} {
		if !names[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// TestResolveByokBaseURL_Gate verifies the Enterprise gate: without NSELF_BYOK=true
// resolveByokBaseURL must return an error (BYOK is Enterprise-only).
func TestResolveByokBaseURL_Gate(t *testing.T) {
	// Ensure NSELF_BYOK is unset.
	orig := os.Getenv("NSELF_BYOK")
	os.Unsetenv("NSELF_BYOK")
	defer os.Setenv("NSELF_BYOK", orig)

	_, err := resolveByokBaseURL()
	if err == nil {
		t.Fatal("expected error when NSELF_BYOK is not set, got nil")
	}
}

// TestResolveByokBaseURL_EnterpriseEnabled verifies that with NSELF_BYOK=true
// the function resolves without error.
func TestResolveByokBaseURL_EnterpriseEnabled(t *testing.T) {
	origByok := os.Getenv("NSELF_BYOK")
	origURL := os.Getenv("BYOK_PLUGIN_URL")
	origAPIURL := os.Getenv("NSELF_API_URL")
	defer func() {
		os.Setenv("NSELF_BYOK", origByok)
		os.Setenv("BYOK_PLUGIN_URL", origURL)
		os.Setenv("NSELF_API_URL", origAPIURL)
	}()

	os.Setenv("NSELF_BYOK", "true")
	os.Unsetenv("BYOK_PLUGIN_URL")
	os.Unsetenv("NSELF_API_URL")

	url, err := resolveByokBaseURL()
	if err != nil {
		t.Fatalf("unexpected error with NSELF_BYOK=true: %v", err)
	}
	if url == "" {
		t.Error("expected non-empty URL")
	}
}

// TestResolveByokBaseURL_CustomURL verifies BYOK_PLUGIN_URL override.
func TestResolveByokBaseURL_CustomURL(t *testing.T) {
	origByok := os.Getenv("NSELF_BYOK")
	origURL := os.Getenv("BYOK_PLUGIN_URL")
	defer func() {
		os.Setenv("NSELF_BYOK", origByok)
		os.Setenv("BYOK_PLUGIN_URL", origURL)
	}()

	os.Setenv("NSELF_BYOK", "true")
	os.Setenv("BYOK_PLUGIN_URL", "https://byok.example.com")

	url, err := resolveByokBaseURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://byok.example.com" {
		t.Errorf("url = %q, want https://byok.example.com", url)
	}
}

// TestResolveKeyRef_RequiresProvider verifies --provider validation.
func TestResolveKeyRef_RequiresProvider(t *testing.T) {
	origProvider := encProviderFlag
	encProviderFlag = ""
	defer func() { encProviderFlag = origProvider }()

	_, _, err := resolveKeyRef()
	if err == nil {
		t.Fatal("expected error when --provider is missing")
	}
}

// TestResolveKeyRef_AWS verifies AWS provider requires --key-id.
func TestResolveKeyRef_AWS(t *testing.T) {
	origProvider := encProviderFlag
	origKeyID := encKeyIDFlag
	defer func() {
		encProviderFlag = origProvider
		encKeyIDFlag = origKeyID
	}()

	encProviderFlag = "aws"
	encKeyIDFlag = ""

	_, _, err := resolveKeyRef()
	if err == nil {
		t.Fatal("expected error when --key-id is missing for aws")
	}

	encKeyIDFlag = "arn:aws:kms:us-east-1:123:key/abc"
	provider, keyRef, err := resolveKeyRef()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != "aws" {
		t.Errorf("provider = %q, want aws", provider)
	}
	if keyRef != "arn:aws:kms:us-east-1:123:key/abc" {
		t.Errorf("keyRef = %q", keyRef)
	}
}

// TestResolveKeyRef_InvalidProvider verifies unsupported provider is rejected.
func TestResolveKeyRef_InvalidProvider(t *testing.T) {
	origProvider := encProviderFlag
	encProviderFlag = "azure"
	defer func() { encProviderFlag = origProvider }()

	_, _, err := resolveKeyRef()
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

// TestEncryptionRotateDryRun_Flag verifies --dry-run flag registration.
func TestEncryptionRotateDryRun_Flag(t *testing.T) {
	f := encryptionRotateCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Fatal("--dry-run flag not registered on rotate cmd")
	}
	if f.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want false", f.DefValue)
	}
}
