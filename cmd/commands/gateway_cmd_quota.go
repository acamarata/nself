package commands

// Purpose: nself gateway quota and routes subcommands — usage/quota reporting and route listing via nself-ai-gateway.
// Inputs: optional provider filter for quota.
// Outputs: formatted quota and route tables.
// Constraints: split out of gateway_cmd.go as a pure move (CLI-R12); no behavior change.

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nself-org/cli/internal/gateway"
	"github.com/spf13/cobra"
)

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
