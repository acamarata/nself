package commands

import (
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
