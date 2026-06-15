package doctor

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestCheckPayPalCSVParity_Mismatch verifies that the check warns on mismatched counts.
func TestCheckPayPalCSVParity_Mismatch(t *testing.T) {
	// Save original environment.
	originalIDs := os.Getenv("PAYPAL_CLIENT_IDS")
	originalSecrets := os.Getenv("PAYPAL_CLIENT_SECRETS")
	defer func() {
		os.Setenv("PAYPAL_CLIENT_IDS", originalIDs)
		os.Setenv("PAYPAL_CLIENT_SECRETS", originalSecrets)
	}()

	// Test: 3 IDs, 2 secrets.
	os.Setenv("PAYPAL_CLIENT_IDS", "id1,id2,id3")
	os.Setenv("PAYPAL_CLIENT_SECRETS", "sec1,sec2")

	result := CheckPayPalCSVParity(context.Background())

	if result.Status != "warn" {
		t.Errorf("CheckPayPalCSVParity with mismatch should return warn, got %q", result.Status)
	}
	if result.Name != "PAYPAL-CSV-PARITY" {
		t.Errorf("CheckPayPalCSVParity name mismatch, got %q", result.Name)
	}
	if !strings.Contains(result.Message, "3") || !strings.Contains(result.Message, "2") {
		t.Errorf("Message should contain both counts (3 and 2), got: %s", result.Message)
	}
}

// TestCheckPayPalCSVParity_Match verifies that the check passes on matched counts.
func TestCheckPayPalCSVParity_Match(t *testing.T) {
	// Save original environment.
	originalIDs := os.Getenv("PAYPAL_CLIENT_IDS")
	originalSecrets := os.Getenv("PAYPAL_CLIENT_SECRETS")
	defer func() {
		os.Setenv("PAYPAL_CLIENT_IDS", originalIDs)
		os.Setenv("PAYPAL_CLIENT_SECRETS", originalSecrets)
	}()

	// Test: 2 IDs, 2 secrets.
	os.Setenv("PAYPAL_CLIENT_IDS", "id1,id2")
	os.Setenv("PAYPAL_CLIENT_SECRETS", "sec1,sec2")

	result := CheckPayPalCSVParity(context.Background())

	if result.Status != "ok" {
		t.Errorf("CheckPayPalCSVParity with matching counts should return ok, got %q", result.Status)
	}
	if result.Name != "PAYPAL-CSV-PARITY" {
		t.Errorf("CheckPayPalCSVParity name mismatch, got %q", result.Name)
	}
}

// TestCheckPayPalCSVParity_NotConfigured verifies that the check passes when neither var is set.
func TestCheckPayPalCSVParity_NotConfigured(t *testing.T) {
	// Save original environment.
	originalIDs := os.Getenv("PAYPAL_CLIENT_IDS")
	originalSecrets := os.Getenv("PAYPAL_CLIENT_SECRETS")
	defer func() {
		os.Setenv("PAYPAL_CLIENT_IDS", originalIDs)
		os.Setenv("PAYPAL_CLIENT_SECRETS", originalSecrets)
	}()

	// Test: both empty.
	os.Setenv("PAYPAL_CLIENT_IDS", "")
	os.Setenv("PAYPAL_CLIENT_SECRETS", "")

	result := CheckPayPalCSVParity(context.Background())

	if result.Status != "ok" {
		t.Errorf("CheckPayPalCSVParity with no config should return ok, got %q", result.Status)
	}
}

// TestCountCSVEntries verifies the CSV counting logic.
func TestCountCSVEntries(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"a,b,c", 3},
		{"  a , b , c  ", 3},
		{"single", 1},
		{"a,,b", 2},
		{"", 0},
		{"  ,  ,  ", 0},
	}
	for _, tc := range cases {
		got := countCSVEntries(tc.input)
		if got != tc.want {
			t.Errorf("countCSVEntries(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
