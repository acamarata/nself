package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var clawKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API keys",
	Long: `List, create, and revoke nClaw API keys.

Without subcommands, lists all API keys.

Examples:
  nself claw keys                       # list keys
  nself claw keys create --name "test"  # create a new key
  nself claw keys revoke <id>           # revoke a key`,
	RunE: runClawKeysList,
}

var (
	clawKeysCreateName      string
	clawKeysCreateBootstrap bool
	clawKeysCreateOwner     string
	clawKeysCreateTier      string
	clawKeysCreateMachineID string
)

var clawKeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	Long: `Create a new nClaw API key.

Without --bootstrap, requires a running claw backend and writes the key to
stdout in human form.

With --bootstrap, runs in headless mode for CI / scripts:
  - Skips all interactive prompts
  - Requires --owner-email, --tier, --machine-id
  - Writes the raw key to stdout (one line, no trailing prompt)
  - Exits 1 with error on stderr on failure

Examples:
  nself claw keys create --name "test"
  nself claw keys create --bootstrap --name ci --owner-email ci@example.com \
      --tier owner --machine-id $(hostname)`,
	RunE: runClawKeysCreate,
}

var clawKeysRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runClawKeysRevoke,
}

func init() {
	clawKeysCreateCmd.Flags().StringVar(&clawKeysCreateName, "name", "", "Name for the API key")
	clawKeysCreateCmd.Flags().BoolVar(&clawKeysCreateBootstrap, "bootstrap", false, "Headless mode: emit key to stdout, no prompts (for CI/scripts)")
	clawKeysCreateCmd.Flags().StringVar(&clawKeysCreateOwner, "owner-email", "", "Owner email (required with --bootstrap)")
	clawKeysCreateCmd.Flags().StringVar(&clawKeysCreateTier, "tier", "", "Key tier (required with --bootstrap): owner|plus|claw|chat|media|family|pro|enterprise")
	clawKeysCreateCmd.Flags().StringVar(&clawKeysCreateMachineID, "machine-id", "", "Machine identifier (required with --bootstrap)")
	clawKeysCreateCmd.MarkFlagRequired("name")
	clawKeysCmd.AddCommand(clawKeysCreateCmd)
	clawKeysCmd.AddCommand(clawKeysRevokeCmd)
}

type clawAPIKeyEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	KeyPrefix  string `json:"key_prefix"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
}

func runClawKeysList(cmd *cobra.Command, args []string) error {
	client, baseURL, err := clawClient()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	req, err := http.NewRequestWithContext(cmd.Context(), "GET", baseURL+"/claw/v1/api-keys", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Keys []clawAPIKeyEntry `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Keys) == 0 {
		fmt.Println("No API keys found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tKEY PREFIX\tCREATED\tLAST USED")
	fmt.Fprintln(w, "--\t----\t----------\t-------\t---------")
	for _, k := range result.Keys {
		lastUsed := k.LastUsedAt
		if lastUsed == "" {
			lastUsed = "never"
		} else if len(lastUsed) > 19 {
			lastUsed = lastUsed[:19]
		}
		created := k.CreatedAt
		if len(created) > 19 {
			created = created[:19]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", k.ID, k.Name, k.KeyPrefix, created, lastUsed)
	}
	w.Flush()

	fmt.Printf("\n%d key(s)\n", len(result.Keys))
	return nil
}

func runClawKeysCreate(cmd *cobra.Command, args []string) error {
	if clawKeysCreateBootstrap {
		return runClawKeysCreateBootstrap(cmd)
	}

	client, baseURL, err := clawClient()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	body, _ := json.Marshal(map[string]string{"name": clawKeysCreateName})
	req, err := http.NewRequestWithContext(cmd.Context(), "POST", baseURL+"/claw/v1/api-keys", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Key string `json:"key"`
		ID  string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	fmt.Println()
	fmt.Printf("  API Key created: %s\n", clawKeysCreateName)
	fmt.Printf("  ID:  %s\n", result.ID)
	fmt.Printf("  Key: %s\n", result.Key)
	fmt.Println()
	fmt.Println("  Save this key now. It will not be shown again.")
	fmt.Println()

	return nil
}

// validBootstrapTiers is the set of tiers accepted by --bootstrap. Mirrors the
// server-side enum; kept small and explicit so a typo on the CI command line
// fails fast with exit code 2 rather than reaching the server.
var validBootstrapTiers = map[string]bool{
	"owner":      true,
	"plus":       true,
	"claw":       true,
	"chat":       true,
	"media":      true,
	"family":     true,
	"pro":        true,
	"enterprise": true,
}

// bootstrapExitCode signals invalid arguments specifically — distinct from the
// generic non-zero exit on RunE error. CI scripts can branch on it.
const bootstrapExitCode = 2

// runClawKeysCreateBootstrap is the headless code path: no prompts, no
// human-readable banner, no clawClient() pre-auth requirement. Validates the
// three required flags up front and emits only the raw key on stdout. On
// validation failure: writes a one-line message to stderr and exits with
// status 2 so a CI runner can detect "you forgot a flag" vs "the request
// failed."
func runClawKeysCreateBootstrap(cmd *cobra.Command) error {
	owner := strings.TrimSpace(clawKeysCreateOwner)
	tier := strings.TrimSpace(clawKeysCreateTier)
	machineID := strings.TrimSpace(clawKeysCreateMachineID)

	var missing []string
	if owner == "" {
		missing = append(missing, "--owner-email")
	}
	if tier == "" {
		missing = append(missing, "--tier")
	}
	if machineID == "" {
		missing = append(missing, "--machine-id")
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "claw_keys: --bootstrap requires %s\n", strings.Join(missing, ", "))
		os.Exit(bootstrapExitCode)
	}
	if !validBootstrapTiers[tier] {
		fmt.Fprintf(os.Stderr, "claw_keys: invalid --tier %q (allowed: owner|plus|claw|chat|media|family|pro|enterprise)\n", tier)
		os.Exit(bootstrapExitCode)
	}
	if !strings.Contains(owner, "@") {
		fmt.Fprintf(os.Stderr, "claw_keys: --owner-email must be a valid email address\n")
		os.Exit(bootstrapExitCode)
	}

	baseURL := clawServerURL()
	if baseURL == "" {
		return fmt.Errorf("claw_keys: no server URL configured. Set NSELF_CLAW_SERVER or run 'nself claw config set server <url>'")
	}

	body, _ := json.Marshal(map[string]string{
		"name":        clawKeysCreateName,
		"owner_email": owner,
		"tier":        tier,
		"machine_id":  machineID,
	})
	req, err := http.NewRequestWithContext(cmd.Context(), "POST", baseURL+"/claw/v1/api-keys/bootstrap", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating bootstrap request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("bootstrap request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Key string `json:"key"`
		ID  string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	// Headless contract: ONE line on stdout, just the key. Captureable in CI
	// as KEY=$(nself claw keys create --bootstrap ...).
	fmt.Println(result.Key)
	return nil
}

func runClawKeysRevoke(cmd *cobra.Command, args []string) error {
	keyID := args[0]

	fmt.Printf("Revoke API key %s? This cannot be undone. [y/N] ", keyID)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	client, baseURL, err := clawClient()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	req, err := http.NewRequestWithContext(cmd.Context(), "DELETE", baseURL+"/claw/v1/api-keys/"+keyID, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("API key %s revoked.\n", keyID)
	return nil
}
