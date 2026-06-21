package commands

import (
	"os"
	"strings"
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
			t.Errorf("encryption: missing subcommand %q", name)
		}
	}
}

// TestEncryptionConfigureCmd_Flags verifies required flags are registered.
func TestEncryptionConfigureCmd_Flags(t *testing.T) {
	required := []string{"provider", "key-id", "key-name", "key-path", "endpoint", "region", "credentials-ref"}
	for _, name := range required {
		if f := encryptionConfigureCmd.Flags().Lookup(name); f == nil {
			t.Errorf("configure: missing flag --%s", name)
		}
	}
}

// TestEncryptionRotateCmd_DryRunFlag verifies --dry-run is registered with default false.
func TestEncryptionRotateCmd_DryRunFlag(t *testing.T) {
	f := encryptionRotateCmd.Flags().Lookup("dry-run")
	if f == nil {
		t.Fatal("rotate: --dry-run flag not registered")
	}
	if f.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want false", f.DefValue)
	}
}

// TestResolveByokBaseURL_GateEnforced verifies that resolveByokBaseURL returns an
// error when NSELF_BYOK is not set to "true". This is the BYOK/Enterprise gate.
func TestResolveByokBaseURL_GateEnforced(t *testing.T) {
	// Save and clear env to ensure a clean state.
	orig := os.Getenv("NSELF_BYOK")
	defer func() {
		if orig == "" {
			os.Unsetenv("NSELF_BYOK")
		} else {
			os.Setenv("NSELF_BYOK", orig)
		}
	}()

	cases := []struct {
		name      string
		byokVal   string
		wantError bool
	}{
		{"unset", "", true},
		{"false", "false", true},
		{"0", "0", true},
		{"TRUE uppercase", "TRUE", true}, // gate is case-sensitive: requires lowercase "true"
		{"true lowercase", "true", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.byokVal == "" {
				os.Unsetenv("NSELF_BYOK")
			} else {
				os.Setenv("NSELF_BYOK", tc.byokVal)
			}
			// Also set BYOK_PLUGIN_URL so the URL resolution won't dial out.
			os.Setenv("BYOK_PLUGIN_URL", "http://localhost:9999")
			defer os.Unsetenv("BYOK_PLUGIN_URL")

			_, err := resolveByokBaseURL()
			if tc.wantError && err == nil {
				t.Errorf("NSELF_BYOK=%q: expected gate error, got nil", tc.byokVal)
			}
			if !tc.wantError && err != nil {
				t.Errorf("NSELF_BYOK=%q: unexpected error: %v", tc.byokVal, err)
			}
		})
	}
}

// TestResolveByokBaseURL_EnterpriseGateMessage verifies error text includes useful guidance.
func TestResolveByokBaseURL_EnterpriseGateMessage(t *testing.T) {
	os.Unsetenv("NSELF_BYOK")
	defer os.Unsetenv("NSELF_BYOK")

	_, err := resolveByokBaseURL()
	if err == nil {
		t.Fatal("expected gate error when NSELF_BYOK unset")
	}
	msg := err.Error()
	if !strings.Contains(msg, "NSELF_BYOK") {
		t.Errorf("error message should mention NSELF_BYOK, got: %s", msg)
	}
	if !strings.Contains(msg, "Enterprise") {
		t.Errorf("error message should mention Enterprise, got: %s", msg)
	}
}

// TestResolveByokBaseURL_URLPrecedence verifies BYOK_PLUGIN_URL > NSELF_API_URL > default.
func TestResolveByokBaseURL_URLPrecedence(t *testing.T) {
	os.Setenv("NSELF_BYOK", "true")
	defer os.Unsetenv("NSELF_BYOK")

	t.Run("BYOK_PLUGIN_URL wins", func(t *testing.T) {
		os.Setenv("BYOK_PLUGIN_URL", "http://byok.local:1234")
		defer os.Unsetenv("BYOK_PLUGIN_URL")
		u, err := resolveByokBaseURL()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u != "http://byok.local:1234" {
			t.Errorf("URL = %q, want http://byok.local:1234", u)
		}
	})

	t.Run("NSELF_API_URL fallback", func(t *testing.T) {
		os.Unsetenv("BYOK_PLUGIN_URL")
		os.Setenv("NSELF_API_URL", "http://api.local:8080")
		defer os.Unsetenv("NSELF_API_URL")
		u, err := resolveByokBaseURL()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u != "http://api.local:8080" {
			t.Errorf("URL = %q, want http://api.local:8080", u)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		os.Unsetenv("BYOK_PLUGIN_URL")
		os.Unsetenv("NSELF_API_URL")
		u, err := resolveByokBaseURL()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u != "http://localhost:3741" {
			t.Errorf("URL = %q, want http://localhost:3741", u)
		}
	})
}

// TestResolveKeyRef_Providers verifies key-ref extraction for all three providers.
func TestResolveKeyRef_Providers(t *testing.T) {
	orig := struct{ id, name, path string }{encKeyIDFlag, encKeyNameFlag, encKeyPathFlag}
	defer func() {
		encKeyIDFlag = orig.id
		encKeyNameFlag = orig.name
		encKeyPathFlag = orig.path
		encProviderFlag = ""
	}()

	cases := []struct {
		provider string
		setup    func()
		wantRef  string
		wantErr  bool
	}{
		{
			"aws",
			func() { encKeyIDFlag = "arn:aws:kms:us-east-1:123:key/abc" },
			"arn:aws:kms:us-east-1:123:key/abc",
			false,
		},
		{
			"gcp",
			func() { encKeyNameFlag = "projects/p/locations/global/keyRings/r/cryptoKeys/k" },
			"projects/p/locations/global/keyRings/r/cryptoKeys/k",
			false,
		},
		{
			"vault",
			func() { encKeyPathFlag = "transit/keys/tenant-abc" },
			"transit/keys/tenant-abc",
			false,
		},
		{"unsupported", func() {}, "", true},
	}

	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			encKeyIDFlag, encKeyNameFlag, encKeyPathFlag = "", "", ""
			encProviderFlag = tc.provider
			tc.setup()

			p, ref, err := resolveKeyRef()
			if tc.wantErr {
				if err == nil {
					t.Error("expected error for unsupported provider, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p != tc.provider {
				t.Errorf("provider = %q, want %q", p, tc.provider)
			}
			if ref != tc.wantRef {
				t.Errorf("ref = %q, want %q", ref, tc.wantRef)
			}
		})
	}
}
