package commands

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/httptimeout"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/spf13/cobra"
)

// pairAlphabet excludes 0, O, 1, I, L to avoid visual confusion.
const pairAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// pairCodeLength is the number of characters in a pairing code.
const pairCodeLength = 6

// pairTimeout is how long to wait for a client to complete pairing.
const pairTimeout = 10 * time.Minute

// pairCloudURL is the public pairing relay endpoint.
const pairCloudURL = "https://pair.nself.org"

var clawCmd = &cobra.Command{
	Use:   "claw",
	Short: "Manage nClaw AI assistant",
	Long: `Commands for interacting with and managing your nClaw AI assistant.

Subcommands:
  prompt    Send a single prompt and get a response
  chat      Start an interactive chat session
  config    View or modify CLI configuration
  pair      Generate a pairing code for nClaw clients
  unlock    Temporarily unlock the web UI for setup
  topics    List or search topics
  memories  List or search memories
  keys      Manage API keys
  status    Show server status and health
  proxy     Start a local OpenAI-compatible proxy
  mcp       Start an MCP server for AI tools
  export    Export all data (JSON or CSV)
  ingest    Ingest a document or URL into the knowledge base
  migrate   Apply pending claw schema migrations`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var clawPairQR bool
var clawPairDirect bool
var clawUnlockMinutes int

var clawPairCmd = &cobra.Command{
	Use:   "pair",
	Short: "Generate a pairing code for nClaw clients",
	Long: `Generate a 6-character pairing code that nClaw clients use to connect.

The code is displayed on screen (and optionally as a QR code) for the user
to enter in their nClaw app. The command waits up to 10 minutes for a client
to pair, then the code expires.

Use --direct to skip cloud registration and generate a local-only code.
Use --qr to display a scannable QR code in the terminal.`,
	RunE: runClawPair,
}

var clawUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Temporarily unlock the web UI for first-time account setup",
	Long: `Unlock the nClaw web UI so you can create your first account.

The unlock window is time-limited (default 10 minutes) and single-use:
once an account is created, the unlock is consumed. Only requests from
localhost (the server itself) can trigger an unlock.

Examples:
  nself claw unlock              # unlock for 10 minutes
  nself claw unlock --minutes 5  # unlock for 5 minutes`,
	RunE: runClawUnlock,
}

func init() {
	clawPairCmd.Flags().BoolVar(&clawPairQR, "qr", false, "Display QR code in terminal")
	clawPairCmd.Flags().BoolVar(&clawPairDirect, "direct", false, "Skip cloud relay, local pairing only")
	clawUnlockCmd.Flags().IntVar(&clawUnlockMinutes, "minutes", 10, "How many minutes the unlock lasts")
	clawCmd.AddCommand(clawPairCmd)
	clawCmd.AddCommand(clawUnlockCmd)
	clawCmd.AddCommand(clawPromptCmd)
	clawCmd.AddCommand(clawChatCmd)
	clawCmd.AddCommand(clawConfigCmd)
	clawCmd.AddCommand(clawTopicsCmd)
	clawCmd.AddCommand(clawMemoriesCmd)
	clawCmd.AddCommand(clawKeysCmd)
	clawCmd.AddCommand(clawStatusCmd)
	clawCmd.AddCommand(clawProxyCmd)
	clawCmd.AddCommand(clawMCPCmd)
	clawCmd.AddCommand(clawExportCmd)
	clawCmd.AddCommand(clawIngestCmd)
	clawCmd.AddCommand(clawMigrateCmd)
	RootCmd.AddCommand(clawCmd)
}

// generatePairCode produces a cryptographically random 6-char code from the safe alphabet.
func generatePairCode() (string, error) {
	var buf strings.Builder
	alphabetLen := big.NewInt(int64(len(pairAlphabet)))
	for i := 0; i < pairCodeLength; i++ {
		idx, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", fmt.Errorf("generating pairing code: %w", err)
		}
		buf.WriteByte(pairAlphabet[idx.Int64()])
	}
	return buf.String(), nil
}

// getServerURL reads the external URL from the nself project config.
// Falls back to NSELF_EXTERNAL_URL env var, then constructs from BASE_DOMAIN.
func getServerURL() string {
	if u := os.Getenv("NSELF_EXTERNAL_URL"); u != "" {
		return u
	}

	// Try loading project config from current directory.
	cfg, err := config.Load(".")
	if err == nil && cfg.BaseDomain != "" {
		scheme := "https"
		if cfg.Env == "dev" {
			scheme = "http"
		}
		return scheme + "://" + cfg.BaseDomain
	}

	return "http://localhost"
}

// registerPairCloud registers the pairing code with the cloud relay.
func registerPairCloud(ctx context.Context, code, serverURL string) error {
	payload, err := json.Marshal(map[string]string{
		"code":       code,
		"server_url": serverURL,
	})
	if err != nil {
		return fmt.Errorf("marshal pair request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", pairCloudURL+"/pair/register", strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("create pair request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return fmt.Errorf("pair registration failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pair registration returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// registerPairLocal stores the pairing code via the claw plugin's HTTP API.
func registerPairLocal(ctx context.Context, code, serverURL string) error {
	clawURL := os.Getenv("PLUGIN_CLAW_INTERNAL_URL")
	if clawURL == "" {
		clawURL = "http://claw:3720"
	}

	payload, err := json.Marshal(map[string]string{
		"code":       code,
		"server_url": serverURL,
		"expires_at": time.Now().Add(pairTimeout).UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal local pair request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", clawURL+"/internal/pair/register", strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("create local pair request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Nself-Internal", "true")

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return fmt.Errorf("local pair registration failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("local pair registration returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// pollPairStatus checks whether a client has completed pairing.
func pollPairStatus(ctx context.Context, code string) (bool, error) {
	clawURL := os.Getenv("PLUGIN_CLAW_INTERNAL_URL")
	if clawURL == "" {
		clawURL = "http://claw:3720"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", clawURL+"/internal/pair/status/"+code, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("X-Nself-Internal", "true")

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result struct {
			Paired bool `json:"paired"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			return result.Paired, nil
		}
	}
	return false, nil
}

func runClawPair(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	code, err := generatePairCode()
	if err != nil {
		return err
	}

	serverURL := getServerURL()

	// Register the code with the claw plugin (local store).
	if err := registerPairLocal(ctx, code, serverURL); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not register with local claw plugin: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure the claw plugin is running (nself plugin install claw && nself start)")
	}

	// Register with cloud relay unless --direct.
	if !clawPairDirect {
		if err := registerPairCloud(ctx, code, serverURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cloud registration failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "Clients can still pair using the server URL directly.")
		}
	}

	// Display the pairing info.
	pairURL := serverURL + "/login?code=" + code
	fmt.Println()
	fmt.Println("  nClaw Pairing Code")
	fmt.Println("  ------------------")
	fmt.Printf("  Code:   %s\n", code)
	fmt.Printf("  Server: %s\n", serverURL)
	fmt.Println()

	// Always print QR code so mobile users can scan without extra flags.
	// --qr flag is kept for backward compatibility.
	printQRCode(pairURL)

	fmt.Printf("Waiting up to %s for a client to pair...\n", pairTimeout)
	fmt.Println("Press Ctrl+C to cancel.")

	// Poll for pairing completion.
	deadline := time.Now().Add(pairTimeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				fmt.Println("\nPairing code expired. Run 'nself claw pair' to generate a new one.")
				return nil
			}
			paired, err := pollPairStatus(ctx, code)
			if err != nil {
				continue // transient error, keep trying
			}
			if paired {
				fmt.Println("\nClient paired successfully!")
				return nil
			}
		}
	}
}

func runClawUnlock(cmd *cobra.Command, args []string) error {
	if clawUnlockMinutes < 1 || clawUnlockMinutes > 60 {
		return fmt.Errorf("--minutes must be between 1 and 60")
	}

	clawURL := os.Getenv("PLUGIN_CLAW_INTERNAL_URL")
	if clawURL == "" {
		clawURL = "http://claw:3720"
	}

	payload, err := json.Marshal(map[string]int{"duration_minutes": clawUnlockMinutes})
	if err != nil {
		return fmt.Errorf("marshal unlock request: %w", err)
	}

	ctx := cmd.Context()
	req, err := http.NewRequestWithContext(ctx, "POST", clawURL+"/claw/auth/unlock", strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("create unlock request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return fmt.Errorf("unlock request failed: %w\nMake sure the claw plugin is running (nself plugin install claw && nself start)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("unlock rejected: this command must be run on the server (localhost only)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		msg := body["error"]
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("unlock failed: %s", msg)
	}

	var result struct {
		ExpiresAt string `json:"expires_at"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	expiresDisplay := result.ExpiresAt
	if t, err := time.Parse(time.RFC3339, result.ExpiresAt); err == nil {
		expiresDisplay = t.Local().Format("15:04:05")
	}

	serverURL := getServerURL()
	fmt.Println()
	fmt.Printf("  nClaw Web UI unlocked for %d minutes.\n", clawUnlockMinutes)
	fmt.Println()
	fmt.Printf("  Visit: %s/claw/ui\n", serverURL)
	fmt.Println()
	fmt.Println("  You will be able to:")
	fmt.Println("    - Create an account (set display name + password)")
	fmt.Println("    - Register a passkey (WebAuthn)")
	fmt.Println()
	fmt.Printf("  The unlock expires at %s or after first use.\n", expiresDisplay)
	fmt.Println()

	return nil
}

// printQRCode renders a QR code to the terminal using Unicode block characters.
func printQRCode(data string) {
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		fmt.Println("  Scan or visit:")
		fmt.Printf("  %s\n", data)
		fmt.Println()
		return
	}
	fmt.Println(qr.ToSmallString(false))
	fmt.Printf("  %s\n", data)
	fmt.Println()
}
