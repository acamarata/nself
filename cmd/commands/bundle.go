package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Bundle describes a canonical nSelf plugin bundle.
type Bundle struct {
	Name        string
	Price       string
	Description string
	Plugins     []string
}

// bundleSlugAliases maps informal/marketing slugs to their canonical bundle
// slug for user-facing lookups (info). Kept in sync with
// internal/bundle.bundleAliases; aliases never appear in listings.
var bundleSlugAliases = map[string]string{
	"sentry": "nsentry",
}

// canonicalBundles is the authoritative bundle membership map (mirrors F06-BUNDLE-INVENTORY.md).
// Keys are the system names used as CLI arguments.
var canonicalBundles = map[string]Bundle{
	"nclaw": {
		Name:  "ɳClaw",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"ai", "claw", "claw-web", "mux", "voice", "browser",
			"google", "notify", "cron", "claw-budget", "claw-news",
			"mcp", "knowledge-base",
		},
	},
	"nchat": {
		Name:  "ɳChat",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"chat", "livekit", "recording", "moderation", "bots", "realtime", "auth",
			"support",
		},
	},
	"nfamily": {
		Name:  "ɳFamily",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"social", "photos", "activity-feed", "moderation", "realtime", "cms", "chat",
			"geolocation", "calendar",
		},
	},
	"ntv": {
		Name:  "ɳTV",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"media-processing", "streaming", "epg", "tmdb", "podcast", "recording",
			"game-metadata", "file-processing", "subtitle-manager", "vpn", "stream-gateway",
		},
	},
	"clawde": {
		Name:  "ClawDE",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"claw", "ai", "realtime", "auth", "notify", "cms",
			"mcp", "knowledge-base",
		},
	},
	"nsentry": {
		Name:  "ɳSentry",
		Price: "$0.99/mo or $9.99/yr",
		Plugins: []string{
			"nself-uptime-monitor", "nself-status-page", "nself-incident-mgmt",
			"nself-alert-router", "nself-slo-tracker", "nself-synthetic-monitor",
			"nself-rum", "nself-errors", "nself-cron-monitor", "nself-oncall",
			"nself-crash", "nself-anomaly", "nself-audit",
		},
	},
	"nself-plus": {
		Name:        "ɳSelf+",
		Price:       "$3.99/mo or $39.99/yr",
		Description: "All 6 paid bundles + all apps + support",
		Plugins:     nil, // meta-bundle — no individual plugin list
	},
	"ntask": {
		Name:        "ɳTask",
		Price:       "FREE",
		Description: "Free bundle — uses free plugins only",
		Plugins:     []string{"(free plugins only — see: nself plugin list)"},
	},
}

// bundleDisplayOrder sets the print order for bundle list.
var bundleDisplayOrder = []string{
	"nclaw", "nchat", "nfamily", "ntv", "clawde", "nsentry", "ntask", "nself-plus",
}

// ── Parent command ──────────────────────────────────────────────────

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Manage and inspect nSelf plugin bundles",
	Long: `Bundle information and management.

Subcommands:
  list         List all available bundles with pricing
  info <name>  Show bundle details, plugins, and license status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return fmt.Errorf("unknown bundle subcommand %q; run 'nself bundle --help' for the list", args[0])
	},
}

// ── bundle list ─────────────────────────────────────────────────────

var bundleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available nSelf plugin bundles",
	Long: `Display every bundle — name, price, and plugin count.

Paid bundles require a license key set via: nself license set <key>
Buy or manage subscriptions at: https://nself.org/pricing`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBundleList(cmd, args)
	},
}

func init() {
	// list flags
	bundleListCmd.Flags().Bool("installed", false, "Show only bundles with at least one active plugin")
	bundleListCmd.Flags().Bool("json", false, "Output machine-readable JSON array")

	// info flags
	bundleInfoCmd.Flags().Bool("json", false, "Output machine-readable JSON object")

	// install flags
	bundleInstallCmd.Flags().Bool("dry-run", false, "Print the planned actions without installing")
	bundleInstallCmd.Flags().Bool("force", false, "Re-install even if already installed; skips same-version check (repair/upgrade path). License is still validated.")
	bundleInstallCmd.Flags().Bool("strict", false, "Fail if any plugin in the bundle is missing from the registry")
	bundleInstallCmd.Flags().String("channel", "stable", "Release channel: stable | beta | canary")

	// remove flags
	bundleRemoveCmd.Flags().Bool("dry-run", false, "Print the planned actions without removing")
	bundleRemoveCmd.Flags().Bool("keep-data", false, "Preserve plugin data (schema/tables) on remove")

	bundleCmd.AddCommand(bundleListCmd)
	bundleCmd.AddCommand(bundleInfoCmd)
	bundleCmd.AddCommand(bundleInstallCmd)
	bundleCmd.AddCommand(bundleRemoveCmd)
	RootCmd.AddCommand(bundleCmd)
}
