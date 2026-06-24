// Package commands — nself gateway command group.
//
// Purpose: Cobra commands for managing the nself-ai-gateway service:
//
//	nself gateway status         — health-check all three AI services
//	nself gateway keys list      — list stored provider keys (no key material)
//	nself gateway keys add       — add a provider key (masked input)
//	nself gateway keys remove    — remove a provider key by ID
//	nself gateway quota          — show daily usage by provider/model
//	nself gateway routes         — list active routing rules
//
// Constraints: key material is NEVER displayed; masked input on keys add
// SPORT: F02-COMMAND-INVENTORY.md (6 gateway command rows)
package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/nself-org/cli/internal/gateway"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// ─── Root: nself gateway ────────────────────────────────────────────────────

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Manage the nSelf AI gateway service",
	Long: `Commands for the nself-ai-gateway provider key vault, quota, and routing.

Subcommands:
  status   Health-check all three canonical AI services (3760, 3761, 3762)
  keys     List, add, or remove provider API keys
  quota    Show daily token and request usage by provider/model
  routes   List active routing rules`,
	RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

// ─── nself gateway status ───────────────────────────────────────────────────

var gatewayStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Health-check all three nSelf AI services",
	Long: `Health-check nself-ai-cc (3760), nself-ai-gateway (3761), and nself-ai-mcp (3762)
in parallel. Exits 0 only when all three services report healthy.

Example:
  nself gateway status`,
	RunE: runGatewayStatus,
}

func runGatewayStatus(cmd *cobra.Command, _ []string) error {
	statuses := gateway.CheckStatus()

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tPORT\tSTATUS")
	fmt.Fprintln(w, "-------\t----\t------")
	for _, s := range statuses {
		status := "ok"
		if !s.Healthy {
			status = "unreachable"
			if s.Error != "" {
				status = s.Error
			}
		}
		fmt.Fprintf(w, "%s\t%d\t%s\n", s.Name, s.Port, status)
	}
	w.Flush()

	healthy := 0
	for _, s := range statuses {
		if s.Healthy {
			healthy++
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d/%d services healthy\n", healthy, len(statuses))

	if !gateway.AllHealthy(statuses) {
		fmt.Fprintf(cmd.ErrOrStderr(), "Hint: run `nself start` to bring all services online\n")
		return fmt.Errorf("one or more AI services are not running")
	}
	return nil
}

// ─── nself gateway keys ─────────────────────────────────────────────────────

var gatewayKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage provider API keys in the nSelf AI gateway",
	Long: `List, add, or remove provider API keys stored in the nself-ai-gateway key vault.

Subcommands:
  list     List all stored keys (no key material is displayed)
  add      Add a provider key (masked input)
  remove   Remove a key by ID`,
	RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

var gatewayKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored provider keys (no key material shown)",
	Long: `List all provider keys stored in the nself-ai-gateway vault.
Key material (actual API keys) is NEVER displayed — only metadata.

Columns: ID, PROVIDER, LABEL, ACTIVE, CREATED

Example:
  nself gateway keys list`,
	RunE: runGatewayKeysList,
}

func runGatewayKeysList(cmd *cobra.Command, _ []string) error {
	keys, err := gateway.ListKeys()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No provider keys stored.")
		fmt.Fprintln(cmd.OutOrStdout(), "Hint: run `nself gateway keys add --provider anthropic --key sk-...` to add one")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPROVIDER\tLABEL\tACTIVE\tCREATED")
	fmt.Fprintln(w, "--\t--------\t-----\t------\t-------")
	for _, k := range keys {
		created := k.CreatedAt
		if len(created) > 19 {
			created = created[:19]
		}
		active := "yes"
		if !k.IsActive {
			active = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", k.ID, k.Provider, k.Label, active, created)
	}
	w.Flush()
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d key(s)\n", len(keys))
	return nil
}

var (
	gatewayKeysAddProvider string
	gatewayKeysAddKey      string
	gatewayKeysAddLabel    string
)

var gatewayKeysAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a provider API key to the nSelf AI gateway vault",
	Long: `Add a provider API key. If --key is not provided, you will be prompted
to enter the key with masked input (characters not echoed to the terminal).

Supported providers: anthropic, openai, google, custom

Example:
  nself gateway keys add --provider anthropic
  nself gateway keys add --provider openai --label "production" --key sk-...`,
	RunE: runGatewayKeysAdd,
}

func runGatewayKeysAdd(cmd *cobra.Command, _ []string) error {
	keyVal := gatewayKeysAddKey

	// If --key not supplied, prompt with masked input.
	if keyVal == "" {
		fmt.Fprintf(os.Stderr, "Enter API key for provider %q (input hidden): ", gatewayKeysAddProvider)
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // newline after masked input
		if err != nil {
			return fmt.Errorf("reading key from stdin: %w", err)
		}
		keyVal = strings.TrimSpace(string(raw))
	}

	result, err := gateway.AddKey(gatewayKeysAddProvider, keyVal, gatewayKeysAddLabel)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Key added:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  ID:       %s\n", result.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "  Provider: %s\n", result.Provider)
	fmt.Fprintf(cmd.OutOrStdout(), "  Label:    %s\n", result.Label)
	fmt.Fprintf(cmd.OutOrStdout(), "  Active:   %v\n", result.IsActive)
	return nil
}

var gatewayKeysRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a provider key by ID",
	Args:  cobra.ExactArgs(1),
	Long: `Remove a provider API key from the nself-ai-gateway vault by its ID.

Example:
  nself gateway keys remove 550e8400-e29b-41d4-a716-446655440000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := gateway.RemoveKey(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Key %s removed\n", args[0])
		return nil
	},
}

// ─── nself gateway quota ────────────────────────────────────────────────────

var (
	gatewayQuotaProvider string
	gatewayQuotaModel    string
)

var gatewayQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Show daily token and request usage by provider/model",
	Long: `Display today's quota usage from the nself-ai-gateway service.
Optionally filter by --provider and/or --model.

Columns: PROVIDER, MODEL, DATE, TOKENS_IN, TOKENS_OUT, REQUESTS, LIMIT

Example:
  nself gateway quota
  nself gateway quota --provider anthropic
  nself gateway quota --provider openai --model gpt-4o`,
	RunE: runGatewayQuota,
}

func runGatewayQuota(cmd *cobra.Command, _ []string) error {
	rows, err := gateway.GetQuota(gatewayQuotaProvider, gatewayQuotaModel)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No quota usage recorded for today.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tMODEL\tDATE\tTOKENS_IN\tTOKENS_OUT\tREQUESTS\tLIMIT")
	fmt.Fprintln(w, "--------\t-----\t----\t---------\t----------\t--------\t-----")
	for _, r := range rows {
		limit := "unlimited"
		if r.DailyLimit != nil {
			limit = fmt.Sprintf("%d", *r.DailyLimit)
		}
		date := r.Date
		if len(date) > 10 {
			date = date[:10]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%s\n",
			r.Provider, r.Model, date,
			r.TokensIn, r.TokensOut, r.RequestCount, limit)
	}
	w.Flush()
	return nil
}

// ─── nself gateway routes ───────────────────────────────────────────────────

var gatewayRoutesCmd = &cobra.Command{
	Use:   "routes",
	Short: "List active routing rules in the nSelf AI gateway",
	Long: `List the active provider routing rules from the nself-ai-gateway service.
Routes define which provider/model is selected for each request pattern.

Example:
  nself gateway routes`,
	RunE: runGatewayRoutes,
}

func runGatewayRoutes(cmd *cobra.Command, _ []string) error {
	routes, err := gateway.ListRoutes()
	if err != nil {
		return err
	}

	if len(routes) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No routing rules configured.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPROVIDER\tMODEL\tPRIORITY\tENABLED")
	fmt.Fprintln(w, "----\t--------\t-----\t--------\t-------")
	for _, r := range routes {
		enabled := "yes"
		if !r.Enabled {
			enabled = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", r.Name, r.Provider, r.Model, r.Priority, enabled)
	}
	w.Flush()
	return nil
}

// ─── init ───────────────────────────────────────────────────────────────────

func init() {
	// keys add flags
	gatewayKeysAddCmd.Flags().StringVar(&gatewayKeysAddProvider, "provider", "", "Provider: anthropic|openai|google|custom (required)")
	gatewayKeysAddCmd.Flags().StringVar(&gatewayKeysAddKey, "key", "", "API key value (leave empty for masked prompt)")
	gatewayKeysAddCmd.Flags().StringVar(&gatewayKeysAddLabel, "label", "", "Optional label for this key")
	_ = gatewayKeysAddCmd.MarkFlagRequired("provider")

	// quota flags
	gatewayQuotaCmd.Flags().StringVar(&gatewayQuotaProvider, "provider", "", "Filter by provider (e.g. anthropic)")
	gatewayQuotaCmd.Flags().StringVar(&gatewayQuotaModel, "model", "", "Filter by model (e.g. claude-sonnet-4-6)")

	// keys subcommands
	gatewayKeysCmd.AddCommand(gatewayKeysListCmd)
	gatewayKeysCmd.AddCommand(gatewayKeysAddCmd)
	gatewayKeysCmd.AddCommand(gatewayKeysRemoveCmd)

	// gateway subcommands
	gatewayCmd.AddCommand(gatewayStatusCmd)
	gatewayCmd.AddCommand(gatewayKeysCmd)
	gatewayCmd.AddCommand(gatewayQuotaCmd)
	gatewayCmd.AddCommand(gatewayRoutesCmd)

	// register with root
	RootCmd.AddCommand(gatewayCmd)
}
