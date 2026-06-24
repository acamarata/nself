// Purpose: CheckStatus health-checks all three canonical AI services in parallel
// (nself-ai-cc:3760, nself-ai-gateway:3761, nself-ai-mcp:3762) and returns
// per-service health results. Exit code 0 only if all three are healthy.
//
// Inputs:  none (ports pulled from ports package constants)
// Outputs: []ServiceStatus; AllHealthy() helper for exit-code decision
// Constraints: checks run concurrently with 5s timeout each; never blocks on a single service
// SPORT: F02-COMMAND-INVENTORY.md (nself gateway status)
package gateway

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nself-org/cli/internal/ports"
)

// ServiceStatus holds the health result for one nSelf AI service.
type ServiceStatus struct {
	Name    string // human-readable service name
	Port    int    // TCP port
	Healthy bool   // true = responded 200 to /health
	Error   string // set when Healthy == false
}

// AllHealthy returns true if every entry in statuses has Healthy == true.
func AllHealthy(statuses []ServiceStatus) bool {
	for _, s := range statuses {
		if !s.Healthy {
			return false
		}
	}
	return true
}

// CheckStatus health-checks all three canonical AI services in parallel.
// Returns one ServiceStatus per service. The caller checks AllHealthy() to determine
// whether to exit 0 or 1.
func CheckStatus() []ServiceStatus {
	services := []struct {
		name string
		port int
	}{
		{"nself-ai-cc", ports.AICCPort},
		{"nself-ai-gateway", ports.AIGatewayPort},
		{"nself-ai-mcp", ports.AIMCPPort},
	}

	results := make([]ServiceStatus, len(services))
	var wg sync.WaitGroup
	httpClient := &http.Client{Timeout: 5 * time.Second}

	for i, svc := range services {
		wg.Add(1)
		go func(idx int, name string, port int) {
			defer wg.Done()
			results[idx] = healthCheck(httpClient, name, port)
		}(i, svc.name, svc.port)
	}

	wg.Wait()
	return results
}

// healthCheck GETs /health on the given port and returns a ServiceStatus.
func healthCheck(client *http.Client, name string, port int) ServiceStatus {
	url := fmt.Sprintf("http://localhost:%d/health", port)
	resp, err := client.Get(url)
	if err != nil {
		return ServiceStatus{
			Name:    name,
			Port:    port,
			Healthy: false,
			Error:   fmt.Sprintf("unreachable: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return ServiceStatus{Name: name, Port: port, Healthy: true}
	}
	return ServiceStatus{
		Name:    name,
		Port:    port,
		Healthy: false,
		Error:   fmt.Sprintf("HTTP %d", resp.StatusCode),
	}
}
