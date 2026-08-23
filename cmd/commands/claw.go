package commands

import (
	"time"

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
	clawCmd.AddCommand(clawMigrateCmd)
	clawCmd.AddCommand(clawSessionCmd)
	RootCmd.AddCommand(clawCmd)
}
