package license

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/errs"
)

// validKey returns a valid nself_pro_ key of exactly 32 characters total.
func validProKey() string {
	// "nself_pro_" = 10 chars; pad to 32 total
	return "nself_pro_" + strings.Repeat("a", 22)
}

// validOwnerKey returns a valid nself_owner_ key of 32+ characters.
func validOwnerKey() string {
	// "nself_owner_" = 12 chars; pad to 32 total
	return "nself_owner_" + strings.Repeat("b", 20)
}

// validEntKey returns a valid nself_ent_ key of 32+ characters.
func validEntKey() string {
	// "nself_ent_" = 10 chars; pad to 32 total
	return "nself_ent_" + strings.Repeat("c", 22)
}

// ---- validateFormat --------------------------------------------------------

func TestValidateFormat_ValidProKey(t *testing.T) {
	key := validProKey()
	if err := validateFormat(key); err != nil {
		t.Fatalf("expected nil error for valid pro key, got: %v", err)
	}
}

func TestValidateFormat_TooShort(t *testing.T) {
	// 31 characters total, valid prefix but too short
	key := "nself_pro_" + strings.Repeat("a", 21)
	if err := validateFormat(key); err != errs.ErrInvalidLicenseKey {
		t.Fatalf("expected ErrInvalidLicenseKey for short key, got: %v", err)
	}
}

func TestValidateFormat_InvalidPrefix(t *testing.T) {
	// Long enough but wrong prefix
	key := "invalid_xxx_" + strings.Repeat("a", 22)
	if err := validateFormat(key); err != errs.ErrInvalidLicenseKey {
		t.Fatalf("expected ErrInvalidLicenseKey for invalid prefix, got: %v", err)
	}
}

func TestValidateFormat_OwnerPrefix(t *testing.T) {
	key := validOwnerKey()
	if err := validateFormat(key); err != nil {
		t.Fatalf("expected nil error for nself_owner_ key, got: %v", err)
	}
}

func TestValidateFormat_EntPrefix(t *testing.T) {
	key := validEntKey()
	if err := validateFormat(key); err != nil {
		t.Fatalf("expected nil error for nself_ent_ key, got: %v", err)
	}
}

// ---- SetKey + GetKey lifecycle ---------------------------------------------

func TestSetKey_GetKey_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// Ensure env override is not active
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "")

	key := validProKey()
	if err := SetKey(key); err != nil {
		t.Fatalf("SetKey failed: %v", err)
	}

	got, err := GetKey()
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}
	if got != key {
		t.Fatalf("expected key %q, got %q", key, got)
	}
}

// ---- GetKey env override ---------------------------------------------------

func TestGetKey_EnvOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	envKey := "nself_pro_" + strings.Repeat("z", 22)
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", envKey)

	got, err := GetKey()
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}
	if got != envKey {
		t.Fatalf("expected env key %q, got %q", envKey, got)
	}
}

// ---- ClearKey --------------------------------------------------------------

func TestClearKey_RemovesKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "")

	key := validProKey()
	if err := SetKey(key); err != nil {
		t.Fatalf("SetKey failed: %v", err)
	}
	if err := ClearKey(); err != nil {
		t.Fatalf("ClearKey failed: %v", err)
	}

	got, err := GetKey()
	if err != nil {
		t.Fatalf("GetKey after ClearKey failed: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty key after ClearKey, got %q", got)
	}
}

// ---- ShowKey ---------------------------------------------------------------

func TestShowKey_ValidKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "")

	key := validProKey()
	if err := SetKey(key); err != nil {
		t.Fatalf("SetKey failed: %v", err)
	}

	masked, tier, err := ShowKey()
	if err != nil {
		t.Fatalf("ShowKey failed: %v", err)
	}
	if masked == "" {
		t.Fatal("expected non-empty masked key")
	}
	if tier != "Pro" {
		t.Fatalf("expected tier 'Pro', got %q", tier)
	}
	// Verify masking: first 10 chars should match, last 4 should match
	if !strings.HasPrefix(masked, key[:10]) {
		t.Fatalf("masked key should start with first 10 chars of original")
	}
	if !strings.HasSuffix(masked, key[len(key)-4:]) {
		t.Fatalf("masked key should end with last 4 chars of original")
	}
	// Middle should contain asterisks
	middle := masked[10 : len(masked)-4]
	if strings.ContainsAny(middle, "abcdefghijklmnopqrstuvwxyz0123456789") {
		t.Fatalf("middle of masked key should be asterisks, got %q", middle)
	}
}

func TestShowKey_NoKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "")

	masked, tier, err := ShowKey()
	if err != nil {
		t.Fatalf("ShowKey failed: %v", err)
	}
	if masked != "" {
		t.Fatalf("expected empty masked key when no key set, got %q", masked)
	}
	if tier != "" {
		t.Fatalf("expected empty tier when no key set, got %q", tier)
	}
}

func TestShowKey_OwnerTier(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "")

	key := validOwnerKey()
	if err := SetKey(key); err != nil {
		t.Fatalf("SetKey failed: %v", err)
	}

	_, tier, err := ShowKey()
	if err != nil {
		t.Fatalf("ShowKey failed: %v", err)
	}
	if tier != "Owner" {
		t.Fatalf("expected tier 'Owner', got %q", tier)
	}
}

// ---- MigrateLicenseFromV1 --------------------------------------------------

func TestMigrateLicenseFromV1_MigratesWhenV2Missing(t *testing.T) {
	home := t.TempDir()

	// Create v1 license file
	v1Dir := filepath.Join(home, LicenseDirV1)
	if err := os.MkdirAll(v1Dir, 0700); err != nil {
		t.Fatalf("setup: creating v1 dir: %v", err)
	}
	v1Path := filepath.Join(v1Dir, LicenseFile)
	content := []byte(`{"key":"nself_pro_testvalue"}`)
	if err := os.WriteFile(v1Path, content, 0600); err != nil {
		t.Fatalf("setup: writing v1 license: %v", err)
	}

	if err := MigrateLicenseFromV1(home); err != nil {
		t.Fatalf("MigrateLicenseFromV1 failed: %v", err)
	}

	v2Path := filepath.Join(home, LicenseDirV2, LicenseFile)
	data, err := os.ReadFile(v2Path)
	if err != nil {
		t.Fatalf("expected v2 license file to exist after migration: %v", err)
	}
	if string(data) != string(content) {
		t.Fatalf("v2 content mismatch: got %q, want %q", string(data), string(content))
	}
}

func TestMigrateLicenseFromV1_NoopWhenV2Exists(t *testing.T) {
	home := t.TempDir()

	// Create both v1 and v2 license files with different content
	v1Dir := filepath.Join(home, LicenseDirV1)
	if err := os.MkdirAll(v1Dir, 0700); err != nil {
		t.Fatalf("setup: creating v1 dir: %v", err)
	}
	v1Content := []byte(`{"key":"v1_key"}`)
	if err := os.WriteFile(filepath.Join(v1Dir, LicenseFile), v1Content, 0600); err != nil {
		t.Fatalf("setup: writing v1 license: %v", err)
	}

	v2Dir := filepath.Join(home, LicenseDirV2)
	if err := os.MkdirAll(v2Dir, 0700); err != nil {
		t.Fatalf("setup: creating v2 dir: %v", err)
	}
	v2Content := []byte(`{"key":"v2_key"}`)
	v2Path := filepath.Join(v2Dir, LicenseFile)
	if err := os.WriteFile(v2Path, v2Content, 0600); err != nil {
		t.Fatalf("setup: writing v2 license: %v", err)
	}

	if err := MigrateLicenseFromV1(home); err != nil {
		t.Fatalf("MigrateLicenseFromV1 failed: %v", err)
	}

	// v2 must not be overwritten
	data, err := os.ReadFile(v2Path)
	if err != nil {
		t.Fatalf("reading v2 after no-op migration: %v", err)
	}
	if string(data) != string(v2Content) {
		t.Fatalf("v2 was overwritten: got %q, want %q", string(data), string(v2Content))
	}
}

func TestMigrateLicenseFromV1_NoopWhenNeitherExists(t *testing.T) {
	home := t.TempDir()

	if err := MigrateLicenseFromV1(home); err != nil {
		t.Fatalf("MigrateLicenseFromV1 failed when neither file exists: %v", err)
	}

	// v2 should not have been created
	v2Path := filepath.Join(home, LicenseDirV2, LicenseFile)
	if _, err := os.Stat(v2Path); err == nil {
		t.Fatal("expected v2 license file to not exist when v1 was absent")
	}
}

// ---- maskKey (tested via ShowKey) ------------------------------------------

func TestMaskKey_ViaMaskKeyFunc(t *testing.T) {
	// Test maskKey directly since it's package-internal
	tests := []struct {
		name     string
		key      string
		wantPfx  string
		wantSfx  string
		wantStar bool
	}{
		{
			name:     "normal long key",
			key:      "nself_pro_" + strings.Repeat("x", 22),
			wantPfx:  "nself_pro_",
			wantSfx:  "xxxx",
			wantStar: true,
		},
		{
			name:     "exactly 14 chars (boundary: showStart+showEnd)",
			key:      strings.Repeat("a", 14),
			wantPfx:  "aaaa",
			wantSfx:  "",
			wantStar: true,
		},
		{
			name:     "very short key (4 chars or fewer)",
			key:      "ab",
			wantPfx:  "",
			wantSfx:  "",
			wantStar: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := maskKey(tc.key)
			if tc.wantPfx != "" && !strings.HasPrefix(got, tc.wantPfx) {
				t.Errorf("maskKey(%q) = %q, want prefix %q", tc.key, got, tc.wantPfx)
			}
			if tc.wantSfx != "" && !strings.HasSuffix(got, tc.wantSfx) {
				t.Errorf("maskKey(%q) = %q, want suffix %q", tc.key, got, tc.wantSfx)
			}
			if tc.wantStar && !strings.Contains(got, "*") {
				t.Errorf("maskKey(%q) = %q, expected asterisks in result", tc.key, got)
			}
			// Result length must equal original length
			if len(got) != len(tc.key) {
				t.Errorf("maskKey(%q) length %d != original length %d", tc.key, len(got), len(tc.key))
			}
		})
	}
}
