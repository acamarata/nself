// P4 T07: tests for ai_pool.go — verifies the 30-key pool cap constant and
// client-side enforcement in runPoolAdd.
package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestMaxPoolKeysConstant verifies the cap is set to 30 (F-PLUGIN:ai-pool-keys).
func TestMaxPoolKeysConstant(t *testing.T) {
	if maxPoolKeys != 30 {
		t.Errorf("maxPoolKeys = %d, want 30 (F-PLUGIN:ai-pool-keys requires 30)", maxPoolKeys)
	}
}

// mockAIPluginServer starts a minimal AI-plugin HTTP server for pool tests.
// statusTotalKeys controls what /ai/pool/status returns for total_keys.
func mockAIPluginServer(t *testing.T, statusTotalKeys int) (*httptest.Server, func()) {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/ai/pool/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"total_keys":     statusTotalKeys,
			"active_keys":    statusTotalKeys,
			"exhausted_keys": 0,
			"daily_capacity": statusTotalKeys * 1500,
			"daily_used":     0,
			"keys":           []any{},
		})
	})

	mux.HandleFunc("/ai/pool/oauth/start", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"auth_url": "https://accounts.google.com/o/oauth2/auth?mock=1",
		})
	})

	srv := httptest.NewServer(mux)
	return srv, srv.Close
}

// newPoolTestCmd returns a cobra.Command with context.Background() set.
func newPoolTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

// TestPoolAddCapEnforced verifies that runPoolAdd returns an error when the pool is full (30 keys).
func TestPoolAddCapEnforced(t *testing.T) {
	srv, cleanup := mockAIPluginServer(t, maxPoolKeys) // exactly at cap
	defer cleanup()

	origURL := os.Getenv("PLUGIN_AI_INTERNAL_URL")
	os.Setenv("PLUGIN_AI_INTERNAL_URL", srv.URL)
	defer func() { os.Setenv("PLUGIN_AI_INTERNAL_URL", origURL) }()

	err := runPoolAdd(newPoolTestCmd(), nil)
	if err == nil {
		t.Fatal("expected a capacity error, got nil")
	}
	if !strings.Contains(err.Error(), "capacity") {
		t.Errorf("error should mention capacity, got: %s", err.Error())
	}
}

// TestPoolAddBelowCap verifies that runPoolAdd does not return a capacity error
// when the pool has room (maxPoolKeys-1 keys).
func TestPoolAddBelowCap(t *testing.T) {
	srv, cleanup := mockAIPluginServer(t, maxPoolKeys-1) // one slot free
	defer cleanup()

	origURL := os.Getenv("PLUGIN_AI_INTERNAL_URL")
	os.Setenv("PLUGIN_AI_INTERNAL_URL", srv.URL)
	defer func() { os.Setenv("PLUGIN_AI_INTERNAL_URL", origURL) }()

	// openBrowser is best-effort; we only verify no capacity error is produced.
	err := runPoolAdd(newPoolTestCmd(), nil)
	if err != nil && strings.Contains(err.Error(), "capacity") {
		t.Errorf("should not hit capacity with %d/%d keys, got: %s",
			maxPoolKeys-1, maxPoolKeys, err.Error())
	}
}
