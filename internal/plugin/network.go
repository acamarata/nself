package plugin

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

// ValidateNetworkAccess does a HEAD request to the registry URL.
// Returns an error with a clear message if the registry is unreachable.
// Call before any operation that requires network access to the plugin registry.
func ValidateNetworkAccess(ctx context.Context, registryURL string) error {
	if registryURL == "" {
		if envURL := os.Getenv("NSELF_PLUGIN_REGISTRY"); envURL != "" {
			registryURL = envURL
		} else {
			registryURL = DefaultRegistryURL
		}
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, registryURL, nil)
	if err != nil {
		return fmt.Errorf("cannot reach plugin registry %s: %w", registryURL, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach plugin registry %s: %w\nCheck your network connection or set NSELF_PLUGIN_REGISTRY to a local mirror.", registryURL, err)
	}
	_ = resp.Body.Close()
	return nil
}
