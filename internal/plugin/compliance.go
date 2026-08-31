package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// StandardEndpoint describes one of the standard plugin interface endpoints.
type StandardEndpoint struct {
	Path     string
	Required bool // if false, missing endpoint is a warning not an error
}

// standardEndpoints defines the plugin interface contract.
var standardEndpoints = []StandardEndpoint{
	{Path: "/health", Required: true},
	{Path: "/version", Required: false},
	{Path: "/capabilities", Required: false},
}

// ComplianceResult holds the outcome of a plugin compliance check.
type ComplianceResult struct {
	PluginName string
	Compliant  bool
	Endpoints  []EndpointCheck
}

// EndpointCheck records whether a specific endpoint responded.
type EndpointCheck struct {
	Path      string
	Available bool
	Status    int
}

// CheckCompliance verifies that a running plugin implements the standard
// interface endpoints (/health, /version, /capabilities). Non-compliant
// plugins log a warning but are not stopped.
func CheckCompliance(ctx context.Context, name string, port int) *ComplianceResult {
	result := &ComplianceResult{
		PluginName: name,
		Compliant:  true,
	}

	client := &http.Client{Timeout: 3 * time.Second}

	for _, ep := range standardEndpoints {
		check := EndpointCheck{Path: ep.Path}
		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, ep.Path)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			check.Available = false
			if ep.Required {
				result.Compliant = false
			}
			result.Endpoints = append(result.Endpoints, check)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			check.Available = false
			if ep.Required {
				result.Compliant = false
			}
		} else {
			check.Status = resp.StatusCode
			check.Available = resp.StatusCode >= 200 && resp.StatusCode < 300
			_ = resp.Body.Close()
			if !check.Available && ep.Required {
				result.Compliant = false
			}
		}

		result.Endpoints = append(result.Endpoints, check)
	}

	if !result.Compliant {
		slog.Warn("plugin does not implement the standard interface contract", "plugin", name)
		for _, ep := range result.Endpoints {
			if !ep.Available {
				slog.Warn("plugin missing required endpoint", "plugin", name, "endpoint", ep.Path)
			}
		}
	}

	return result
}

// CheckComplianceJSON returns the compliance result as JSON bytes.
func CheckComplianceJSON(ctx context.Context, name string, port int) ([]byte, error) {
	result := CheckCompliance(ctx, name, port)
	return json.MarshalIndent(result, "", "  ")
}
