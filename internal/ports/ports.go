// Package ports provides canonical port constants for all nSelf services.
//
// Purpose: Single source of truth for TCP port assignments so CLI code never
// hardcodes raw port numbers. Import this package wherever a port is needed.
//
// Inputs:  none
// Outputs: typed port constants
// Constraints: port values are locked; any reassignment requires a SPORT F10 update
// SPORT: F10-PORT-REGISTRY.md
package ports

const (
	// AICCPort is the port for the nself-ai-cc PTY session relay service.
	AICCPort = 3760

	// AIGatewayPort is the port for the nself-ai-gateway provider key vault and routing service.
	AIGatewayPort = 3761

	// AIMCPPort is the port for the nself-ai-mcp MCP tool server.
	AIMCPPort = 3762

	// EvalGatePort is the port for the nself-eval-gate plugin HTTP API.
	// Canonical assignment per F10-PORT-REGISTRY.md and eval-harness-foundation-spec.md §3.
	EvalGatePort = 3770
)

// AIGatewayBaseURL returns the base URL for the nself-ai-gateway service.
func AIGatewayBaseURL() string {
	return "http://localhost:3761"
}

// AICCBaseURL returns the base URL for the nself-ai-cc PTY session relay service.
func AICCBaseURL() string {
	return "http://localhost:3760"
}

// AIMCPBaseURL returns the base URL for the nself-ai-mcp MCP tool server.
func AIMCPBaseURL() string {
	return "http://localhost:3762"
}

// EvalGateBaseURL returns the base URL for the nself-eval-gate plugin HTTP API.
func EvalGateBaseURL() string {
	return "http://localhost:3770"
}
