package commands

// Purpose: nself gateway keys subcommand group — list/add/remove AI provider keys via nself-ai-gateway.
// Inputs: provider name, key label, key ID, masked key material read from stdin.
// Outputs: formatted key tables; never displays key material.
// Constraints: split out of gateway_cmd.go as a pure move (CLI-R12); no behavior change. Keys remain write-only with masked input on add.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/nself-org/cli/internal/gateway"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// --- nself gateway keys ---

var gatewayKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage AI provider keys",
	Long: `Manage API keys stored in nself-ai-gateway.

Keys are AES-256-GCM encrypted at rest. Key material is write-only:
once added it is never returned in list or status output.

Subcommands:
  list              List all keys (no key material)
  add               Add a new provider key
  remove <id>       Remove a key by ID`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var gatewayKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered provider keys",
	Long: `List provider keys registered in nself-ai-gateway.

Key material is never shown. The output table contains:
  id, provider, label, is_active, created_at

Exit codes:
  0  Success
  2  Authentication error
  3  Connection error`,
	RunE: runGatewayKeysList,
}

func runGatewayKeysList(cmd *cobra.Command, args []string) error {
	token, err := gatewayToken()
	if err != nil {
		return err
	}

	keys, err := gateway.ListKeys(cmd.Context(), token)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Println("No keys registered.")
		fmt.Println("Hint: add one with `nself gateway keys add --provider anthropic --label my-key`")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPROVIDER\tLABEL\tACTIVE\tCREATED")
	for _, k := range keys {
		active := "yes"
		if !k.IsActive {
			active = "no"
		}
		created := k.CreatedAt.Format("2006-01-02")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", k.ID, k.Provider, k.Label, active, created)
	}
	return w.Flush()
}

var (
	keysAddProvider string
	keysAddLabel    string
	keysAddKey      string
)

var gatewayKeysAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a provider API key",
	Long: `Add a new AI provider API key to nself-ai-gateway.

The key is AES-256-GCM encrypted before storage. If --key is not provided,
you will be prompted to enter it (masked input).

Supported providers: anthropic, openai, google, custom

Exit codes:
  0  Key added
  1  Invalid input or server error
  2  Authentication error
  3  Connection error`,
	RunE: runGatewayKeysAdd,
}

func init() {
	gatewayKeysAddCmd.Flags().StringVar(&keysAddProvider, "provider", "", "Provider name (anthropic|openai|google|custom)")
	gatewayKeysAddCmd.Flags().StringVar(&keysAddLabel, "label", "", "Human-readable label for the key")
	gatewayKeysAddCmd.Flags().StringVar(&keysAddKey, "key", "", "API key (omit to enter interactively, masked)")
}

func runGatewayKeysAdd(cmd *cobra.Command, args []string) error {
	if keysAddProvider == "" {
		return fmt.Errorf("provider required\n\nHint: --provider anthropic|openai|google|custom\nExit: 1")
	}

	keyMaterial := keysAddKey
	if keyMaterial == "" {
		// Prompt with masked input.
		fmt.Printf("Enter %s API key (input hidden): ", keysAddProvider)
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			// Fallback to plain bufio if not a terminal.
			fmt.Printf("Enter %s API key: ", keysAddProvider)
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				keyMaterial = strings.TrimSpace(scanner.Text())
			}
		} else {
			keyMaterial = strings.TrimSpace(string(raw))
		}
	}

	if keyMaterial == "" {
		return fmt.Errorf("key material required\n\nHint: provide --key or enter it when prompted\nExit: 1")
	}

	token, err := gatewayToken()
	if err != nil {
		return err
	}

	id, err := gateway.AddKey(cmd.Context(), token, keysAddProvider, keysAddLabel, keyMaterial)
	if err != nil {
		return err
	}
	fmt.Printf("Key added. ID: %s\n", id)
	return nil
}

var gatewayKeysRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a provider key by ID",
	Args:  cobra.ExactArgs(1),
	Long: `Remove a key from nself-ai-gateway by its ID.

Use 'nself gateway keys list' to find key IDs.

Exit codes:
  0  Key removed
  1  Key not found or server error
  2  Authentication error
  3  Connection error`,
	RunE: runGatewayKeysRemove,
}

func runGatewayKeysRemove(cmd *cobra.Command, args []string) error {
	token, err := gatewayToken()
	if err != nil {
		return err
	}
	if err := gateway.RemoveKey(cmd.Context(), token, args[0]); err != nil {
		return err
	}
	fmt.Printf("Key %s removed.\n", args[0])
	return nil
}
