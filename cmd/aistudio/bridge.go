// Package aistudio implements the `nself ai-studio` command subtree.
// SP-12.B36: Google AI Studio bridge via Cloudflare Tunnel + short-lived JWT + read-only enforcement.
package aistudio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// maskToken returns a masked token showing only the first 8 characters followed by ****.
// If the token is 8 characters or shorter, returns ****.
// If NSELF_DEBUG=1, returns the full token.
func maskToken(t string) string {
	if os.Getenv("NSELF_DEBUG") == "1" {
		return t
	}
	if len(t) <= 8 {
		return "****"
	}
	return t[:8] + "****"
}

// BridgeSession holds runtime state for one active tunnel session.
type BridgeSession struct {
	TunnelURL  string    `json:"tunnel_url"`
	StartedAt  time.Time `json:"started_at"`
	RequestsN  int64     `json:"requests_n"`
	SchemaHash string    `json:"schema_hash,omitempty"`
	ProjectID  string    `json:"project_id,omitempty"`
}

// -----------------------------------------------------------------------------
// Command tree: nself ai-studio
// -----------------------------------------------------------------------------

// AiStudioCmd is the top-level `nself ai-studio` command.
var AiStudioCmd = &cobra.Command{
	Use:   "ai-studio",
	Short: "Google AI Studio integration for local nSelf instances",
	Long: `Integrate Google AI Studio with your local nSelf instance via a
secure Cloudflare Tunnel. Gemini can query your local Postgres schema
without any cloud deployment.`,
	RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

// bridgeFlagValues holds parsed CLI flags for the bridge subcommand.
var bridgeFlagValues struct {
	port        int
	noContext   bool
	dryRun      bool
	idleTimeout int
	ipAllowlist string
	region      string
}

// BridgeCmd is the `nself ai-studio bridge` subcommand.
var BridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "Start a secure AI Studio bridge via Cloudflare Tunnel",
	Long: `Start a local proxy that exposes your nSelf instance to Google AI Studio
via an ephemeral Cloudflare Tunnel (trycloudflare.com — no account required).

The bridge:
  - Issues a short-lived session token (30-minute idle TTL) scoped to one project
  - Enforces schema READ-ONLY (no mutations, DDL, or DML through the tunnel)
  - Injects your Postgres schema as X-Nself-Schema-Context on every response
  - Auto-closes after 30 minutes of inactivity (--idle-timeout to override)
  - Optionally restricts access to specific IP CIDRs via --ip-allowlist

Usage in AI Studio:
  1. Run:  nself ai-studio bridge
  2. Copy the printed tunnel URL
  3. In AI Studio: Custom connector, paste the URL + Authorization header

Examples:
  nself ai-studio bridge
  nself ai-studio bridge --no-context
  nself ai-studio bridge --dry-run
  nself ai-studio bridge --ip-allowlist 192.168.1.0/24
  nself ai-studio bridge --idle-timeout 60`,
	RunE: runBridge,
}

func init() {
	BridgeCmd.Flags().IntVar(&bridgeFlagValues.port, "port", 8890, "Local proxy port")
	BridgeCmd.Flags().BoolVar(&bridgeFlagValues.noContext, "no-context", false, "Disable X-Nself-Schema-Context header injection")
	BridgeCmd.Flags().BoolVar(&bridgeFlagValues.dryRun, "dry-run", false, "Print tunnel info without starting proxy")
	BridgeCmd.Flags().IntVar(&bridgeFlagValues.idleTimeout, "idle-timeout", 30, "Auto-close tunnel after N minutes of inactivity")
	BridgeCmd.Flags().StringVar(&bridgeFlagValues.ipAllowlist, "ip-allowlist", "", "Comma-separated CIDRs to restrict tunnel access (default: unrestricted)")
	BridgeCmd.Flags().StringVar(&bridgeFlagValues.region, "region", "auto", "Cloudflare tunnel region")

	AiStudioCmd.AddCommand(BridgeCmd)
}

func runBridge(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Parse IP allowlist.
	var ipAllowlist []netip.Prefix
	if bridgeFlagValues.ipAllowlist != "" {
		var err error
		ipAllowlist, err = parseIPAllowlist(bridgeFlagValues.ipAllowlist)
		if err != nil {
			return fmt.Errorf("--ip-allowlist: %w", err)
		}
	}

	localAddr := fmt.Sprintf("http://localhost:%d", bridgeFlagValues.port)
	idleTimeout := time.Duration(bridgeFlagValues.idleTimeout) * time.Minute

	// Dry run: only describe what would happen.
	if bridgeFlagValues.dryRun {
		fmt.Printf("Dry run: would start Cloudflare Tunnel → %s\n", localAddr)
		fmt.Printf("Idle timeout: %v | Schema context: %v\n", idleTimeout, !bridgeFlagValues.noContext)
		fmt.Println("Tunnel URL would be: https://<random>.trycloudflare.com")
		return nil
	}

	// Ensure cloudflared binary is available (auto-download if absent).
	cfBin, err := ensureCloudflared(ctx)
	if err != nil {
		return fmt.Errorf("cloudflared setup: %w", err)
	}

	// Generate session auth token.
	hasuraURL := hasuraURLFromEnv()
	adminSecret := hasuraAdminSecretFromEnv()
	authToken := os.Getenv("NSELF_AISTUDIO_AUTH_TOKEN")
	if authToken == "" {
		authToken, err = generateToken()
		if err != nil {
			return fmt.Errorf("generate session token: %w", err)
		}
	}

	cfg := proxyConfig{
		ListenPort:   bridgeFlagValues.port,
		HasuraURL:    hasuraURL,
		AdminSecret:  adminSecret,
		AuthToken:    authToken,
		InjectSchema: !bridgeFlagValues.noContext,
		IPAllowlist:  ipAllowlist,
		IdleTimeout:  idleTimeout,
	}

	// Build and start proxy.
	ps, err := newProxyServer(ctx, cfg)
	if err != nil {
		return fmt.Errorf("build proxy: %w", err)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", bridgeFlagValues.port),
		Handler: ps.mux,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", srv.Addr, err)
	}
	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "proxy error: %v\n", serveErr)
		}
	}()

	// Start Cloudflare Tunnel (blocks until URL is received, max 30s).
	tunnelURL, cfCmd, err := startTunnel(ctx, cfBin, localAddr)
	if err != nil {
		shutdownProxy(srv)
		return fmt.Errorf("start tunnel: %w", err)
	}

	_ = &BridgeSession{
		TunnelURL: tunnelURL,
		StartedAt: time.Now(),
		ProjectID: os.Getenv("PROJECT_NAME"),
	}

	fmt.Printf("\nAI Studio bridge ready: %s\n", tunnelURL)
	fmt.Printf("Auth token:             %s\n", maskToken(authToken))
	fmt.Printf("Schema context:         %v\n", !bridgeFlagValues.noContext)
	fmt.Printf("Idle timeout:           %v\n", idleTimeout)
	if bridgeFlagValues.ipAllowlist != "" {
		fmt.Printf("IP allowlist:           %s\n", bridgeFlagValues.ipAllowlist)
	}
	fmt.Printf("(Full token shown only with NSELF_DEBUG=1)\n")
	fmt.Printf("\nIn AI Studio → Custom connector → URL: %s/v1/graphql\n", tunnelURL)
	fmt.Printf("Authorization header:   Bearer %s\n", maskToken(authToken))
	fmt.Printf("\nPress Ctrl-C to stop.\n\n")

	// Idle timeout watcher.
	idleTicker := time.NewTicker(30 * time.Second)
	defer idleTicker.Stop()

	// Signal handler for clean shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-sigCh:
			fmt.Println("\nShutting down bridge...")
			shutdownAll(srv, cfCmd)
			return nil

		case <-idleTicker.C:
			idle := time.Since(ps.lastActivity)
			if idle >= idleTimeout {
				fmt.Printf("\nBridge idle for %v (threshold: %v). Closing.\n", idle.Round(time.Second), idleTimeout)
				shutdownAll(srv, cfCmd)
				return nil
			}

		case <-ctx.Done():
			shutdownAll(srv, cfCmd)
			return nil
		}
	}
}

// shutdownProxy gracefully stops the HTTP server.
func shutdownProxy(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// shutdownAll stops both the HTTP server and the cloudflared process.
func shutdownAll(srv *http.Server, cfCmd *exec.Cmd) {
	shutdownProxy(srv)
	if cfCmd != nil && cfCmd.Process != nil {
		cfCmd.Process.Kill()
	}
}

// readBodyPreserving reads r.Body and restores it so downstream handlers can re-read it.
func readBodyPreserving(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, err
}
