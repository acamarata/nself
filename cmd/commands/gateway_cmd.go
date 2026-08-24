package commands

// Purpose: nself gateway command group — status, keys, quota, routes.
//   Wires to nself-ai-gateway (port 3761) for AI provider key management,
//   quota reporting, route listing, and service health checks.
// Inputs: Provider name, key labels, key IDs, optional filters.
// Outputs: Formatted tables; never displays key material.
// Constraints: Keys are write-only; masked input on add.
// SPORT: F02 — 6 new nself gateway commands.

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nself-org/cli/internal/auth"
	"github.com/nself-org/cli/internal/gateway"
	"github.com/spf13/cobra"
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
