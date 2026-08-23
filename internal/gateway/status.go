// Package gateway provides client utilities for nself-ai-gateway (port 3761).
//
// Purpose: Wrap HTTP calls to nself-ai-gateway, nself-ai-cc, and nself-ai-mcp
//
//	for use in CLI gateway commands.
//
// Inputs: Port constants from internal/ports; JWT from internal/auth.
// Outputs: Typed structs for keys, quota, routes, service health.
// Constraints: Key material never returned in any output type.
// SPORT: F02 — nself gateway command group.
package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nself-org/cli/internal/ports"
)

// ServiceHealth reports the health of one AI service.
type ServiceHealth struct {
	Name    string
	Port    int
	Healthy bool
	Message string
}

// StatusAll checks all three canonical AI services in parallel.
// Returns a slice of three ServiceHealth results.
// allHealthy is true only when all three are healthy.
func StatusAll(ctx context.Context) ([]ServiceHealth, bool) {
	services := []ServiceHealth{
		{Name: "nself-ai-cc", Port: ports.AICCPort},
		{Name: "nself-ai-gateway", Port: ports.AIGatewayPort},
		{Name: "nself-ai-mcp", Port: ports.AIMCPPort},
	}

	var wg sync.WaitGroup
	client := &http.Client{Timeout: 5 * time.Second}

	for i := range services {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			url := fmt.Sprintf("http://localhost:%d/health", services[i].Port)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				services[i].Healthy = false
				services[i].Message = fmt.Sprintf("request error: %v", err)
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				services[i].Healthy = false
				services[i].Message = fmt.Sprintf("unreachable: %v", err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				services[i].Healthy = true
				services[i].Message = "ok"
			} else {
				services[i].Healthy = false
				services[i].Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()

	allHealthy := true
	for _, s := range services {
		if !s.Healthy {
			allHealthy = false
		}
	}
	return services, allHealthy
}
