package claw

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/ports"
)

// TestProxyUpstreamURL verifies that ProxyUpstreamURL always points at AIGatewayPort.
func TestProxyUpstreamURL(t *testing.T) {
	wantPrefix := fmt.Sprintf("http://localhost:%d", ports.AIGatewayPort)

	cases := []struct {
		name     string
		path     string
		rawQuery string
		wantURL  string
	}{
		{
			name:    "v1 chat completions path, no query",
			path:    "/v1/chat/completions",
			wantURL: wantPrefix + "/v1/chat/completions",
		},
		{
			name:    "v1 models path, no query",
			path:    "/v1/models",
			wantURL: wantPrefix + "/v1/models",
		},
		{
			name:     "v1 chat completions with query",
			path:     "/v1/chat/completions",
			rawQuery: "stream=true",
			wantURL:  wantPrefix + "/v1/chat/completions?stream=true",
		},
		{
			name:    "root path, no query (base URL display)",
			path:    "",
			wantURL: wantPrefix,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ProxyUpstreamURL(tc.path, tc.rawQuery)
			if got != tc.wantURL {
				t.Errorf("ProxyUpstreamURL(%q, %q) = %q, want %q", tc.path, tc.rawQuery, got, tc.wantURL)
			}
		})
	}
}

// TestProxyUpstreamURL_UsesPort3761 explicitly verifies the proxy never targets
// any port other than 3761 (AIGatewayPort). This is the canonical contract check.
func TestProxyUpstreamURL_UsesPort3761(t *testing.T) {
	url := ProxyUpstreamURL("/v1/chat/completions", "")
	if !strings.Contains(url, ":3761") {
		t.Errorf("proxy upstream URL %q does not target port 3761 (AIGatewayPort)", url)
	}
}
