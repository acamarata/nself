package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/bundle"
	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/plugin"

	"github.com/spf13/cobra"
)

// Bundle describes a canonical nSelf plugin bundle.
type Bundle struct {
	Name        string
	Price       string
	Description string
	Plugins     []string
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

// bundleListRow is the JSON-serialisable form of one bundle in `nself bundle list --json`.
type bundleListRow struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Price       string   `json:"price"`
	PluginCount int      `json:"plugin_count"`
	Plugins     []string `json:"plugins"`
	Status      string   `json:"status"`
	HasActive   bool     `json:"has_active"`
}

func runBundleList(cmd *cobra.Command, _ []string) error {
	var showInstalled, asJSON bool
	if cmd != nil {
		showInstalled, _ = cmd.Flags().GetBool("installed")
		asJSON, _ = cmd.Flags().GetBool("json")
	}

	// Build the row set, optionally filtering to installed-only.
	var rows []bundleListRow
	pluginDir := resolvePluginDir()
	installedPlugins, _ := plugin.ListInstalled(pluginDir)
	installedSet := make(map[string]bool, len(installedPlugins))
	for _, p := range installedPlugins {
		installedSet[strings.ToLower(p.Name)] = true
	}

	for _, key := range bundleDisplayOrder {
		b, ok := canonicalBundles[key]
		if !ok {
			continue
		}

		status := "active"
		if key == "ntask" {
			status = "free"
		} else if key == "nsentry" || key == "nfamily" || key == "clawde" {
			status = "planned"
		}

		// Count how many bundle plugins are installed.
		activeCount := 0
		for _, p := range b.Plugins {
			if installedSet[strings.ToLower(p)] {
				activeCount++
			}
		}
		hasActive := activeCount > 0

		if showInstalled && !hasActive {
			continue
		}

		pluginList := b.Plugins
		if pluginList == nil {
			pluginList = []string{}
		}

		rows = append(rows, bundleListRow{
			Slug:        key,
			Name:        b.Name,
			Price:       b.Price,
			PluginCount: len(b.Plugins),
			Plugins:     pluginList,
			Status:      status,
			HasActive:   hasActive,
		})
	}

	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}

	fmt.Println("nSelf Plugin Bundles")
	fmt.Println(strings.Repeat("─", 60))

	for _, r := range rows {
		pluginCount := ""
		if r.PluginCount > 0 {
			pluginCount = fmt.Sprintf("  (%d plugins)", r.PluginCount)
		}

		b := canonicalBundles[r.Slug]
		desc := b.Description
		if desc == "" && len(b.Plugins) > 0 {
			preview := b.Plugins
			if len(preview) > 4 {
				preview = append(preview[:4:4], "...")
			}
			desc = strings.Join(preview, ", ")
		}

		fmt.Printf("  %-14s  %-26s  %s%s\n", r.Slug, r.Name, r.Price, pluginCount)
		if desc != "" {
			fmt.Printf("  %-14s  %s\n", "", desc)
		}
		fmt.Println()
	}

	fmt.Println("Run 'nself bundle info <name>' for full plugin membership.")
	fmt.Println("Buy at: https://nself.org/pricing")
	return nil
}

// ── bundle info ─────────────────────────────────────────────────────

// bundleInfoJSON is the JSON-serialisable form of `nself bundle info --json`.
type bundleInfoJSON struct {
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	Price         string   `json:"price"`
	Description   string   `json:"description,omitempty"`
	Plugins       []string `json:"plugins"`
	PluginCount   int      `json:"plugin_count"`
	LicenseStatus string   `json:"license_status"`
	InstallHint   string   `json:"install_hint"`
}

var bundleInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show bundle details, plugins, and license status",
	Long: `Display the full plugin membership, pricing, and your current license status for a bundle.

Examples:
  nself bundle info nclaw
  nself bundle info nclaw --json
  nself bundle info nself-plus`,
	Args: cobra.ExactArgs(1),
	RunE: runBundleInfo,
}

func runBundleInfo(cmd *cobra.Command, args []string) error {
	key := strings.ToLower(strings.TrimSpace(args[0]))
	b, ok := canonicalBundles[key]
	if !ok {
		// Collect valid names for the error hint.
		var names []string
		for k := range canonicalBundles {
			names = append(names, k)
		}
		sort.Strings(names)
		return fmt.Errorf("bundle not found: %q\n\nRun 'nself bundle list' for available bundles.\nAvailable: %s", key, strings.Join(names, ", "))
	}

	asJSON := false
	if cmd != nil {
		asJSON, _ = cmd.Flags().GetBool("json")
	}

	licenseStatus := resolveBundleLicenseStatus(key)

	// Build install hint string.
	var installHint string
	switch key {
	case "ntask":
		installHint = "nself plugin install <plugin-name>  (no license required)"
	case "nself-plus":
		installHint = "nself license set <key>  |  Buy: https://nself.org/pricing"
	default:
		if len(b.Plugins) > 0 {
			installHint = fmt.Sprintf("nself license set <key> && nself plugin install %s  |  Buy: https://nself.org/pricing", b.Plugins[0])
		} else {
			installHint = "nself license set <key>  |  Buy: https://nself.org/pricing"
		}
	}

	if asJSON {
		plugins := b.Plugins
		if plugins == nil {
			plugins = []string{}
		}
		row := bundleInfoJSON{
			Slug:          key,
			Name:          b.Name,
			Price:         b.Price,
			Description:   b.Description,
			Plugins:       plugins,
			PluginCount:   len(plugins),
			LicenseStatus: licenseStatus,
			InstallHint:   installHint,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(row)
	}

	fmt.Printf("Bundle: %s (%s)\n", b.Name, key)
	fmt.Printf("Price:  %s\n", b.Price)
	if b.Description != "" {
		fmt.Printf("Note:   %s\n", b.Description)
	}

	// ── License status ─────────────────────────────────────────────
	fmt.Printf("License: %s\n", licenseStatus)

	// ── Plugin membership ──────────────────────────────────────────
	fmt.Println()
	if key == "nself-plus" {
		fmt.Println("Includes all 6 paid bundles (nclaw, nchat, nfamily, ntv, clawde, nsentry)")
		fmt.Println("+ all nSelf apps + support via chat.nself.org or the nChat app")
	} else if b.Plugins != nil {
		fmt.Printf("Plugins (%d):\n", len(b.Plugins))
		for _, p := range b.Plugins {
			installed := checkPluginInstalled(p)
			marker := " "
			if installed {
				marker = "✓"
			}
			fmt.Printf("  [%s] %s\n", marker, p)
		}
		fmt.Println()
		fmt.Println("  ✓ = installed on this machine")
	}

	// ── Install hint ───────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("Install: %s\n", installHint)

	return nil
}

// ── Helpers ─────────────────────────────────────────────────────────

// resolveBundleLicenseStatus returns a human-readable license status string
// for the given bundle key. Reads the local license cache — no network call.
func resolveBundleLicenseStatus(bundleKey string) string {
	if bundleKey == "ntask" {
		return "active (free)"
	}

	cache, err := license.ReadCache()
	if err != nil || cache == nil {
		return "not activated (run: nself license set <key>)"
	}

	tier := strings.ToLower(cache.Tier)

	// ɳSelf+ (owner / enterprise / plus tiers) covers all bundles.
	if tier == "plus" || tier == "enterprise" || tier == "owner" {
		return fmt.Sprintf("active via ɳSelf+ (%s tier)", cache.Tier)
	}

	// Check if the specific bundle is in PluginsAllowed.
	// The license server populates PluginsAllowed with all plugin names
	// included in the licensed bundle(s).
	b, ok := canonicalBundles[bundleKey]
	if ok && len(b.Plugins) > 0 {
		for _, allowed := range cache.PluginsAllowed {
			for _, bp := range b.Plugins {
				if strings.EqualFold(allowed, bp) {
					return fmt.Sprintf("active (%s tier)", cache.Tier)
				}
			}
		}
	}

	return fmt.Sprintf("not included in current license (%s tier — run: nself license set <key>)", cache.Tier)
}

// checkPluginInstalled returns true if the plugin appears to be installed locally.
func checkPluginInstalled(pluginName string) bool {
	if strings.HasPrefix(pluginName, "(") {
		return false // placeholder entry
	}
	pluginDir := resolvePluginDir()
	installed, err := plugin.ListInstalled(pluginDir)
	if err != nil {
		return false
	}
	for _, p := range installed {
		if strings.EqualFold(p.Name, pluginName) {
			return true
		}
	}
	return false
}

// ── bundle install ──────────────────────────────────────────────────

var bundleInstallCmd = &cobra.Command{
	Use:   "install <bundle>",
	Short: "Install every plugin in a bundle (license-gated, atomic rollback on failure)",
	Long: `Install all plugins in the named bundle in a single transaction.

License validation runs for every plugin BEFORE any filesystem change.
On any per-plugin install failure, every plugin installed in this invocation
is rolled back so the system is left in a clean state.

Examples:
  nself bundle install nsentry
  nself bundle install nclaw --channel beta
  nself bundle install nchat --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runBundleInstall,
}

var bundleRemoveCmd = &cobra.Command{
	Use:   "remove <bundle>",
	Short: "Remove every plugin in a bundle",
	Long: `Remove all plugins in the named bundle. Plugin data is dropped by default;
pass --keep-data to preserve schema/tables for later reinstall.

Examples:
  nself bundle remove nsentry
  nself bundle remove nclaw --keep-data
  nself bundle remove nchat --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runBundleRemove,
}

func runBundleInstall(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	strict, _ := cmd.Flags().GetBool("strict")
	channelStr, _ := cmd.Flags().GetString("channel")

	opts := bundle.InstallOpts{
		DryRun:  dryRun,
		Force:   force,
		Strict:  strict,
		Channel: bundle.Channel(channelStr),
	}
	_, err := bundle.Install(context.Background(), args[0], opts)
	return err
}

func runBundleRemove(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	keepData, _ := cmd.Flags().GetBool("keep-data")

	opts := bundle.RemoveOpts{
		DryRun:   dryRun,
		KeepData: keepData,
	}
	_, err := bundle.Remove(context.Background(), args[0], opts)
	return err
}

// ── Registration ─────────────────────────────────────────────────────

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
