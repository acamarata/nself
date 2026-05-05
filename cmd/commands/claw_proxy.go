package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// parseAllowedOrigins reads a CSV list of origins from the CLAW_PROXY_ALLOWED_ORIGINS
// env var and returns the normalized set. An empty env var means no CORS headers
// are added (browsers will block cross-origin requests).
func parseAllowedOrigins(env string) map[string]bool {
	set := make(map[string]bool)
	for _, o := range strings.Split(env, ",") {
		o = strings.TrimSpace(strings.ToLower(o))
		if o == "" || o == "*" {
			continue
		}
		set[o] = true
	}
	return set
}

// isLocalhostOrigin returns true for http(s)://localhost[:port] or 127.0.0.1.
// The claw proxy always binds to 127.0.0.1 and is used by local-dev clients,
// so localhost origins are allowed by default.
func isLocalhostOrigin(origin string) bool {
	lo := strings.ToLower(origin)
	if idx := strings.Index(lo, "://"); idx != -1 {
		lo = lo[idx+3:]
	}
	if idx := strings.LastIndexByte(lo, ':'); idx != -1 {
		lo = lo[:idx]
	}
	return lo == "localhost" || lo == "127.0.0.1"
}

var clawProxyCmd = &cobra.Command{
	Use:   "proxy [port]",
	Short: "Start a local OpenAI-compatible proxy",
	Long: `Start a local HTTP server that proxies requests to the nClaw server.

Any OpenAI-compatible client can connect by setting:
  OPENAI_BASE_URL=http://localhost:8899/v1

Supported endpoints:
  /v1/chat/completions   Chat completions (streaming supported)
  /v1/models             List available models
  /v1/embeddings         Generate embeddings

Default port: 8899. Pass a port number as argument to override.

Examples:
  nself claw proxy        # start on port 8899
  nself claw proxy 9000   # start on port 9000`,
	RunE: runClawProxy,
}

func runClawProxy(cmd *cobra.Command, args []string) error {
	port := 8899
	if len(args) > 0 {
		p, err := strconv.Atoi(args[0])
		if err != nil || p < 1 || p > 65535 {
			return fmt.Errorf("invalid port: %s", args[0])
		}
		port = p
	}

	_, baseURL, err := clawClient()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	apiKey := clawAPIKey()

	mux := http.NewServeMux()

	// allowedOrigins is populated from CLAW_PROXY_ALLOWED_ORIGINS (csv).
	// localhost/127.0.0.1 is always allowed since the proxy binds to 127.0.0.1.
	allowedOrigins := parseAllowedOrigins(os.Getenv("CLAW_PROXY_ALLOWED_ORIGINS"))

	// setCORS echoes the request origin only if it is in the allowlist or is
	// a localhost origin. Wildcards are never used. Vary: Origin is set so
	// intermediary caches do not serve the response to a different origin.
	setCORS := func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Set("Vary", "Origin")
		if origin == "" {
			return
		}
		lo := strings.ToLower(origin)
		if !allowedOrigins[lo] && !isLocalhostOrigin(origin) {
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	}

	// Proxy handler for all /v1/* paths
	proxyHandler := func(w http.ResponseWriter, r *http.Request) {
		setCORS(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Build upstream URL
		upstreamURL := baseURL + "/claw" + r.URL.Path
		if r.URL.RawQuery != "" {
			upstreamURL += "?" + r.URL.RawQuery
		}

		// Create upstream request
		upReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
			return
		}

		// Copy headers, stripping hop-by-hop headers
		hopByHop := map[string]bool{
			"Connection":        true,
			"Keep-Alive":        true,
			"Transfer-Encoding": true,
			"Te":                true,
			"Trailer":           true,
			"Upgrade":           true,
		}
		for key, vals := range r.Header {
			if hopByHop[http.CanonicalHeaderKey(key)] {
				continue
			}
			for _, val := range vals {
				upReq.Header.Add(key, val)
			}
		}

		// Set auth from config (override any client-provided auth)
		upReq.Header.Set("Authorization", "Bearer "+apiKey)

		// Use a client without timeout for streaming
		proxyClient := &http.Client{}
		upResp, err := proxyClient.Do(upReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("upstream error: %v", err), http.StatusBadGateway)
			return
		}
		defer upResp.Body.Close()

		// Copy response headers
		for key, vals := range upResp.Header {
			for _, val := range vals {
				w.Header().Add(key, val)
			}
		}
		w.WriteHeader(upResp.StatusCode)

		// Stream the response through
		if f, ok := w.(http.Flusher); ok {
			buf := make([]byte, 4096)
			for {
				n, err := upResp.Body.Read(buf)
				if n > 0 {
					w.Write(buf[:n])
					f.Flush()
				}
				if err != nil {
					break
				}
			}
		} else {
			io.Copy(w, upResp.Body)
		}
	}

	mux.HandleFunc("/v1/chat/completions", proxyHandler)
	mux.HandleFunc("/v1/models", proxyHandler)
	mux.HandleFunc("/v1/embeddings", proxyHandler)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","proxy":true}`))
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Println("nClaw OpenAI Proxy")
	fmt.Println("------------------")
	fmt.Printf("  Listening: http://%s\n", addr)
	fmt.Printf("  Upstream:  %s\n", baseURL)
	fmt.Println()
	fmt.Println("  Usage:")
	fmt.Printf("    export OPENAI_BASE_URL=http://localhost:%d/v1\n", port)
	fmt.Println("    # Then use any OpenAI-compatible client")
	fmt.Println()
	fmt.Println("  Endpoints:")
	fmt.Printf("    POST http://localhost:%d/v1/chat/completions\n", port)
	fmt.Printf("    GET  http://localhost:%d/v1/models\n", port)
	fmt.Printf("    POST http://localhost:%d/v1/embeddings\n", port)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop.")

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Handle shutdown on context cancellation
	go func() {
		<-cmd.Context().Done()
		server.Close()
	}()

	err = server.ListenAndServe()
	if err != nil && !strings.Contains(err.Error(), "Server closed") {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
