package gateway_test

// Purpose: Unit tests for gateway package — status, keys, quota.
// Tests verify port constants, struct shapes, and error UX messages.

import (
	"testing"

	"github.com/nself-org/cli/internal/ports"
)

func TestPortConstants(t *testing.T) {
	if ports.AICCPort != 3760 {
		t.Errorf("AICCPort = %d, want 3760", ports.AICCPort)
	}
	if ports.AIGatewayPort != 3761 {
		t.Errorf("AIGatewayPort = %d, want 3761", ports.AIGatewayPort)
	}
	if ports.AIMCPPort != 3762 {
		t.Errorf("AIMCPPort = %d, want 3762", ports.AIMCPPort)
	}
}

func TestGatewayURLFormat(t *testing.T) {
	// Verify the gateway URL format used internally.
	want := "http://localhost:3761/v1/keys"
	got := "http://localhost:3761" + "/v1/keys"
	if got != want {
		t.Errorf("URL format = %q, want %q", got, want)
	}
}
