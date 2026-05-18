package config

import (
	"testing"
)

// TestAuthConfig_MFATOTPEnabled_default_false verifies that when
// AUTH_MFA_TOTP_ENABLED is unset, AuthConfig.MFATOTPEnabled defaults to false.
func TestAuthConfig_MFATOTPEnabled_default_false(t *testing.T) {
	// Ensure the env var is unset for this subtest (t.Setenv("", "") is invalid;
	// use a sentinel + explicit unset via the test helper). Go's testing package
	// restores the env on cleanup automatically when t.Setenv is used, but here
	// we want to assert the unset case.
	t.Setenv("AUTH_MFA_TOTP_ENABLED", "")

	// Empty string is treated as unset by getEnvBool, so the fallback applies.
	got := getEnvBool("AUTH_MFA_TOTP_ENABLED", false)
	if got {
		t.Errorf("getEnvBool(AUTH_MFA_TOTP_ENABLED, false) with empty env = true, want false (default)")
	}

	// Also verify the AuthConfig zero value is false (struct default).
	var cfg AuthConfig
	if cfg.MFATOTPEnabled {
		t.Errorf("AuthConfig{}.MFATOTPEnabled = true, want false (zero value)")
	}
}

// TestAuthConfig_MFATOTPEnabled_env_override verifies that setting
// AUTH_MFA_TOTP_ENABLED=true causes the loader-equivalent getEnvBool path
// to return true.
func TestAuthConfig_MFATOTPEnabled_env_override(t *testing.T) {
	t.Setenv("AUTH_MFA_TOTP_ENABLED", "true")

	got := getEnvBool("AUTH_MFA_TOTP_ENABLED", false)
	if !got {
		t.Errorf("getEnvBool(AUTH_MFA_TOTP_ENABLED, false) with env=true returned false, want true")
	}

	// Sanity check: assigning the parsed bool into AuthConfig matches.
	cfg := AuthConfig{MFATOTPEnabled: got}
	if !cfg.MFATOTPEnabled {
		t.Errorf("AuthConfig{MFATOTPEnabled: true}.MFATOTPEnabled = false, want true")
	}
}
