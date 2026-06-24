package ports

import (
	"fmt"
	"strings"
	"testing"
)

// TestAIPortConstants verifies the canonical AI service port values.
// These must match gateway-unification-spec.md §1 and SPORT F10-PORT-REGISTRY.md.
func TestAIPortConstants(t *testing.T) {
	cases := []struct {
		name     string
		port     int
		wantPort int
	}{
		{"AICCPort (nself-ai-cc PTY relay)", AICCPort, 3760},
		{"AIGatewayPort (nself-ai-gateway key vault + routing)", AIGatewayPort, 3761},
		{"AIMCPPort (nself-ai-mcp MCP tool server)", AIMCPPort, 3762},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.port != tc.wantPort {
				t.Errorf("%s = %d, want %d", tc.name, tc.port, tc.wantPort)
			}
		})
	}
}

// TestAIBaseURLs verifies that the helper URL functions embed the correct ports.
func TestAIBaseURLs(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantSub string
	}{
		{"AIGatewayBaseURL", AIGatewayBaseURL(), fmt.Sprintf(":%d", AIGatewayPort)},
		{"AICCBaseURL", AICCBaseURL(), fmt.Sprintf(":%d", AICCPort)},
		{"AIMCPBaseURL", AIMCPBaseURL(), fmt.Sprintf(":%d", AIMCPPort)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.url, tc.wantSub) {
				t.Errorf("%s = %q, want it to contain %q", tc.name, tc.url, tc.wantSub)
			}
			if !strings.HasPrefix(tc.url, "http://localhost:") {
				t.Errorf("%s = %q, want prefix http://localhost:", tc.name, tc.url)
			}
		})
	}
}

// TestAIPortsAreDistinct ensures no two AI service ports collide.
func TestAIPortsAreDistinct(t *testing.T) {
	ports := []struct{ name string; port int }{
		{"AICCPort", AICCPort},
		{"AIGatewayPort", AIGatewayPort},
		{"AIMCPPort", AIMCPPort},
	}
	seen := make(map[int]string)
	for _, p := range ports {
		if prev, ok := seen[p.port]; ok {
			t.Errorf("port collision: %s and %s both use port %d", prev, p.name, p.port)
		}
		seen[p.port] = p.name
	}
}
