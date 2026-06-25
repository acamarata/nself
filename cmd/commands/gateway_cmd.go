package commands

// Purpose: nself gateway command group — status, keys, quota, routes.
//   Wires to nself-ai-gateway (port 3761) for AI provider key management,
//   quota reporting, route listing, and service health checks.
// Inputs: Provider name, key labels, key IDs, optional filters.
// Outputs: Formatted tables; never displays key material.
// Constraints: Keys are write-only; masked input on add.
// SPORT: F02 — 6 new nself gateway commands.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/nself-org/cli/internal/auth"
	"github.com/nself-org/cli/internal/gateway"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// gatewayToken returns the nself JWT for gateway requests.
func gatewayToken() (string, error) {
	af, err := auth.ReadAuthFile()
	if err != nil {
		return "", fmt.Errorf("not logged in\n\nHint: run `nself login` first\nExit: 2")
	}
	return af.AccessToken, nil
}

// --- nself gateway ---

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Manage the nSelf AI gateway (nself-ai-gateway, port 3761)",
	Long: `Commands for managing the canonical nSelf AI provider gateway.

The gateway handles provider key encryption, request routing, and quota
enforcement for all AI features (ɳClaw, ClawDE, ɳSelf+).

Subcommands:
  status     Health-check all three AI services (3760, 3761, 3762)
  keys       Manage provider API keys (list / add / remove)
  quota      Show quota usage by provider and model
  routes     List registered routing rules`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// --- nself gateway status ---

var gatewayStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Health-check all three AI services",
	Long: `Check health of nself-ai-cc (3760), nself-ai-gateway (3761), and nself-ai-mcp (3762).

Exit codes:
  0  All three services healthy
  1  One or more services down`,
	RunE: runGatewayStatus,
}

func runGatewayStatus(cmd *cobra.Command, args []string) error {
	services, allHealthy := gateway.StatusAll(cmd.Context())

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tPORT\tSTATUS\tMESSAGE")
	for _, s := range services {
		status := "✓"
		if !s.Healthy {
			status = "✗"
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", s.Name, s.Port, status, s.Message)
	}
	w.Flush()

	if !allHealthy {
		return fmt.Errorf("one or more AI services are down\n\nHint: run `nself plugin status nself-ai-gateway` to diagnose\nExit: 1")
	}
	return nil
}

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

// --- nself gateway quota ---

var (
	quotaProvider string
	quotaModel    string
)

var gatewayQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Show AI request quota usage",
	Long: `Show quota usage from nself-ai-gateway, grouped by provider and model.

Use --provider or --model to filter results.

Exit codes:
  0  Success
  2  Authentication error
  3  Connection error`,
	RunE: runGatewayQuota,
}

func init() {
	gatewayQuotaCmd.Flags().StringVar(&quotaProvider, "provider", "", "Filter by provider")
	gatewayQuotaCmd.Flags().StringVar(&quotaModel, "model", "", "Filter by model")
}

func runGatewayQuota(cmd *cobra.Command, args []string) error {
	token, err := gatewayToken()
	if err != nil {
		return err
	}

	rows, err := gateway.GetQuota(cmd.Context(), token, quotaProvider, quotaModel)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		fmt.Println("No quota data found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tMODEL\tUSED\tLIMIT\tRESETS")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n", r.Provider, r.Model, r.Used, r.Limit, r.ResetAt)
	}
	return w.Flush()
}

// --- nself gateway routes ---

var gatewayRoutesCmd = &cobra.Command{
	Use:   "routes",
	Short: "List gateway routing rules",
	Long: `List routing rules registered in nself-ai-gateway.

Routes map providers and models to upstream endpoints.

Exit codes:
  0  Success
  2  Authentication error
  3  Connection error`,
	RunE: runGatewayRoutes,
}

func runGatewayRoutes(cmd *cobra.Command, args []string) error {
	token, err := gatewayToken()
	if err != nil {
		return err
	}
	resp, err := gateway.ListRoutes(cmd.Context(), token)
	if err != nil {
		return err
	}
	if len(resp) == 0 {
		fmt.Println("No routes configured.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPROVIDER\tMODEL\tTARGET\tACTIVE")
	for _, r := range resp {
		active := "yes"
		if !r.Active {
			active = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.ID, r.Provider, r.Model, r.Target, active)
	}
	return w.Flush()
}

func init() {
	gatewayKeysCmd.AddCommand(gatewayKeysListCmd)
	gatewayKeysCmd.AddCommand(gatewayKeysAddCmd)
	gatewayKeysCmd.AddCommand(gatewayKeysRemoveCmd)

	gatewayCmd.AddCommand(gatewayStatusCmd)
	gatewayCmd.AddCommand(gatewayKeysCmd)
	gatewayCmd.AddCommand(gatewayQuotaCmd)
	gatewayCmd.AddCommand(gatewayRoutesCmd)

	RootCmd.AddCommand(gatewayCmd)
}
