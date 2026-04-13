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

var clawKeysCreateName string

var clawKeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	RunE:  runClawKeysCreate,
}

var clawKeysRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runClawKeysRevoke,
}

func init() {
	clawKeysCreateCmd.Flags().StringVar(&clawKeysCreateName, "name", "", "Name for the API key")
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
