package license

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCollectLicenseKeys_SingleEnvVar(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "nself_pro_abcdefghijklmnopqrstuvwxyz12345")
	for i := 1; i <= 10; i++ {
		t.Setenv(fmt.Sprintf("NSELF_LICENSE_KEY_%d", i), "")
	}

	keys := CollectLicenseKeys()
	if len(keys) < 1 {
		t.Fatal("expected at least 1 key from NSELF_PLUGIN_LICENSE_KEY")
	}
}

func TestCollectLicenseKeys_NumberedKeys(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "")
	key1 := "nself_claw_" + "a]bcdefghijklmnopqrstuvwxyz123456"
	key2 := "nself_chat_" + "abcdefghijklmnopqrstuvwxyz123456"
	t.Setenv("NSELF_LICENSE_KEY_1", key1)
	t.Setenv("NSELF_LICENSE_KEY_2", key2)
	for i := 3; i <= 10; i++ {
		t.Setenv(fmt.Sprintf("NSELF_LICENSE_KEY_%d", i), "")
	}

	keys := CollectLicenseKeys()
	if len(keys) < 2 {
		t.Fatalf("expected at least 2 keys, got %d", len(keys))
	}
}

func TestCollectLicenseKeys_Deduplication(t *testing.T) {
	key := "nself_pro_abcdefghijklmnopqrstuvwxyz123456"
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", key)
	t.Setenv("NSELF_LICENSE_KEY_1", key) // duplicate
	for i := 2; i <= 10; i++ {
		t.Setenv(fmt.Sprintf("NSELF_LICENSE_KEY_%d", i), "")
	}

	keys := CollectLicenseKeys()
	count := 0
	for _, k := range keys {
		if k == key {
			count++
		}
	}
	if count > 1 {
		t.Fatalf("expected deduplication, got %d copies", count)
	}
}

func TestCollectLicenseKeys_EmptySkipped(t *testing.T) {
	t.Setenv("NSELF_PLUGIN_LICENSE_KEY", "")
	for i := 1; i <= 10; i++ {
		t.Setenv(fmt.Sprintf("NSELF_LICENSE_KEY_%d", i), "")
	}

	// Temporarily override home to avoid reading real key files.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpDir)
	}

	keys := CollectLicenseKeys()
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys with all empty env vars, got %d", len(keys))
	}
}

func TestValidateKeyFormat_ValidPrefixes(t *testing.T) {
	prefixes := []string{
		"nself_claw_", "nself_chat_", "nself_media_", "nself_family_",
		"nself_clawde_", "nself_plus_", "nself_pro_", "nself_owner_",
	}
	for _, prefix := range prefixes {
		key := prefix + "abcdefghijklmnopqrstuvwxyz123456"
		if err := ValidateKeyFormat(key); err != nil {
			t.Errorf("expected valid for prefix %q, got: %v", prefix, err)
		}
	}
}

func TestValidateKeyFormat_InvalidPrefix(t *testing.T) {
	if err := ValidateKeyFormat("invalid_prefix_abcdefghijklmnopqrstuvwxyz"); err == nil {
		t.Error("expected error for invalid prefix")
	}
}

func TestValidateKeyFormat_TooShort(t *testing.T) {
	if err := ValidateKeyFormat("nself_pro_short"); err == nil {
		t.Error("expected error for short key")
	}
}

func TestDetectProduct(t *testing.T) {
	tests := []struct {
		key     string
		product string
	}{
		{"nself_claw_abcdefghijklmnopqrstuvwxyz1234", "claw"},
		{"nself_chat_abcdefghijklmnopqrstuvwxyz1234", "chat"},
		{"nself_plus_abcdefghijklmnopqrstuvwxyz1234", "plus"},
		{"nself_owner_abcdefghijklmnopqrstuvwxyz123", "owner"},
	}
	for _, tt := range tests {
		pp := DetectProduct(tt.key)
		if pp == nil {
			t.Errorf("expected product for key prefix, got nil")
			continue
		}
		if pp.Product != tt.product {
			t.Errorf("expected product %q, got %q", tt.product, pp.Product)
		}
	}
}

func TestAddAndRemoveKeys(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpDir)
	}

	key1 := "nself_claw_abcdefghijklmnopqrstuvwxyz1234"
	key2 := "nself_chat_abcdefghijklmnopqrstuvwxyz1234"

	// Add first key.
	if err := AddKey(key1); err != nil {
		t.Fatalf("AddKey(key1): %v", err)
	}

	// Add second key.
	if err := AddKey(key2); err != nil {
		t.Fatalf("AddKey(key2): %v", err)
	}

	stored := GetAllStoredKeys()
	if len(stored) != 2 {
		t.Fatalf("expected 2 stored keys, got %d", len(stored))
	}

	// Remove by product name.
	n, err := RemoveKey("claw")
	if err != nil {
		t.Fatalf("RemoveKey(claw): %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 removed, got %d", n)
	}

	remaining := GetAllStoredKeys()
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining key, got %d", len(remaining))
	}
	if remaining[0] != key2 {
		t.Fatalf("expected chat key remaining, got %s", remaining[0])
	}
}

func TestSetKeyReplaceAll(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpDir)
	}

	key1 := "nself_claw_abcdefghijklmnopqrstuvwxyz1234"
	key2 := "nself_chat_abcdefghijklmnopqrstuvwxyz1234"
	key3 := "nself_pro_abcdefghijklmnopqrstuvwxyz12345"

	_ = AddKey(key1)
	_ = AddKey(key2)

	replaced, err := SetKeyReplaceAll(key3)
	if err != nil {
		t.Fatalf("SetKeyReplaceAll: %v", err)
	}
	if replaced != 2 {
		t.Fatalf("expected 2 replaced, got %d", replaced)
	}

	stored := GetAllStoredKeys()
	if len(stored) != 1 {
		t.Fatalf("expected 1 key after replace, got %d", len(stored))
	}
	if stored[0] != key3 {
		t.Fatalf("expected pro key, got %s", stored[0])
	}
}

func TestGapRenumbering(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmpDir)
	}

	key1 := "nself_claw_abcdefghijklmnopqrstuvwxyz1234"
	key2 := "nself_chat_abcdefghijklmnopqrstuvwxyz1234"
	key3 := "nself_pro_abcdefghijklmnopqrstuvwxyz12345"

	_ = AddKey(key1)
	_ = AddKey(key2)
	_ = AddKey(key3)

	// Remove middle key.
	_, _ = RemoveKey("chat")

	stored := GetAllStoredKeys()
	if len(stored) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(stored))
	}

	// Verify files are contiguous (key and key.2, no gaps).
	dir, _ := licenseDirPath()
	if _, err := os.Stat(filepath.Join(dir, "key")); err != nil {
		t.Error("expected primary key file to exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "key.2")); err != nil {
		t.Error("expected key.2 file to exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "key.3")); !os.IsNotExist(err) {
		t.Error("expected key.3 to not exist after renumbering")
	}
}
