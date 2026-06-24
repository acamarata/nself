// Package claw provides client helpers for nSelf AI services wired to
// the canonical port assignments defined in internal/ports.
//
// Purpose: Upstream URL construction for the nself claw proxy command.
//   The proxy always forwards to nself-ai-gateway (AIGatewayPort) per
//   gateway-unification-spec.md §5, §11 (no env-var override in dev).
// Inputs:  request path + query from the local proxy server
// Outputs: fully-qualified upstream URL targeting localhost:3761
// Constraints: port is read from ports.AIGatewayPort; never hardcoded here;
//   no env-var override — dev and prod use the same port per §11.
// SPORT: F10-PORT-REGISTRY.md · F02-COMMAND-INVENTORY.md (nself claw proxy)
package claw

import (
	"fmt"

	"github.com/nself-org/cli/internal/ports"
)

// ProxyUpstreamURL constructs the upstream URL for a given request path and
// raw query, pointing at nself-ai-gateway on AIGatewayPort.
//
// The proxy routes /v1/* paths through the gateway's OpenAI-compatible API
// surface. This function is the single call site for that URL construction
// so the port source is never duplicated.
func ProxyUpstreamURL(path, rawQuery string) string {
	base := fmt.Sprintf("http://localhost:%d", ports.AIGatewayPort)
	u := base + path
	if rawQuery != "" {
		u += "?" + rawQuery
	}
	return u
}
