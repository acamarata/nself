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
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	apiKey := clawAPIKey()

	mux := http.NewServeMux()

	// Proxy handler for all /v1/* paths
	proxyHandler := func(w http.ResponseWriter, r *http.Request) {
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

		// Copy headers
		for key, vals := range r.Header {
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
