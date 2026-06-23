// Purpose: Tests for claw pair — error message when backend unreachable (not panic),
// code generation, and flag registration.
//
// Acceptance: claw pair: clear error message when backend unreachable (not panic).
// SPORT: F02-COMMAND-INVENTORY row: claw pair

package commands

import (
	"context"
	"strings"
	"testing"
)

func TestGeneratePairCode_Length(t *testing.T) {
	code, err := generatePairCode()
	if err != nil {
		t.Fatalf("generatePairCode() error: %v", err)
	}
	if len(code) != pairCodeLength {
		t.Errorf("pair code length = %d, want %d", len(code), pairCodeLength)
	}
}

func TestGeneratePairCode_Alphabet(t *testing.T) {
	for i := 0; i < 50; i++ {
		code, err := generatePairCode()
		if err != nil {
			t.Fatalf("generatePairCode() iteration %d error: %v", i, err)
		}
		for _, ch := range code {
			if !strings.ContainsRune(pairAlphabet, ch) {
				t.Errorf("pair code %q contains character %q not in pairAlphabet", code, ch)
			}
		}
	}
}

func TestGeneratePairCode_Uniqueness(t *testing.T) {
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := generatePairCode()
		if err != nil {
			t.Fatalf("generatePairCode() iteration %d error: %v", i, err)
		}
		codes[code] = true
	}
	// With 31^6 = ~887 million possibilities, 100 draws should be unique.
	if len(codes) < 95 {
		t.Errorf("pair codes are not unique enough: only %d distinct out of 100", len(codes))
	}
}

// TestRegisterPairLocal_BackendUnreachable confirms that when the claw plugin
// is not running, registerPairLocal returns a descriptive error rather than panicking.
func TestRegisterPairLocal_BackendUnreachable(t *testing.T) {
	// Point PLUGIN_CLAW_INTERNAL_URL at a port that is not listening.
	t.Setenv("PLUGIN_CLAW_INTERNAL_URL", "http://127.0.0.1:19099")

	err := registerPairLocal(context.Background(), "ABCDEF", "http://localhost")
	if err == nil {
		t.Fatal("expected error when backend is unreachable, got nil")
	}
	// Must not contain "panic" and must be a descriptive network error.
	if strings.Contains(strings.ToLower(err.Error()), "panic") {
		t.Errorf("registerPairLocal returned panic-like error: %v", err)
	}
}

// TestRegisterPairCloud_BackendUnreachable mirrors the local test for cloud relay.
func TestRegisterPairCloud_BackendUnreachable(t *testing.T) {
	// Override the cloud URL constant via the env approach is not possible —
	// pairCloudURL is a const. We test the error path by calling with a context
	// that is already cancelled, which simulates network failure immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled immediately

	err := registerPairCloud(ctx, "ABCDEF", "http://localhost")
	if err == nil {
		t.Fatal("expected error when context is cancelled, got nil")
	}
}

// TestClawPairFlags verifies expected flags are registered.
func TestClawPairFlags(t *testing.T) {
	wantFlags := []string{"qr", "direct"}
	for _, name := range wantFlags {
		if clawPairCmd.Flags().Lookup(name) == nil {
			t.Errorf("claw pair: expected flag --%s to be registered", name)
		}
	}
}

// TestClawUnlockFlags verifies expected flags are registered.
func TestClawUnlockFlags(t *testing.T) {
	if clawUnlockCmd.Flags().Lookup("minutes") == nil {
		t.Error("claw unlock: expected flag --minutes to be registered")
	}
}

// TestClawUnlock_InvalidMinutes confirms exit-2-class validation.
func TestClawUnlock_InvalidMinutes(t *testing.T) {
	orig := clawUnlockMinutes
	defer func() { clawUnlockMinutes = orig }()

	for _, bad := range []int{0, -1, 61, 100} {
		clawUnlockMinutes = bad
		if err := runClawUnlock(nil, nil); err == nil {
			t.Errorf("expected error for --minutes %d, got nil", bad)
		}
	}
}
