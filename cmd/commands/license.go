package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// defaultPingURL is the production license validation endpoint.
const defaultPingURL = "https://ping.nself.org"

// pricingURL is the page opened by the upgrade subcommand.
const pricingURL = "https://nself.org/pricing"

var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Manage license keys for nSelf product bundles",
	Long: `Manage license keys for nSelf product bundles (ɳClaw, ɳChat, nTV, etc.).

Supports multiple keys for different product bundles. Keys can also be set via
environment variables: NSELF_PLUGIN_LICENSE_KEY (legacy) or NSELF_LICENSE_KEY_1
through NSELF_LICENSE_KEY_10.

Subcommands:
  set      Replace all keys with a single key (backward compatible)
  add      Add one or more keys without replacing existing ones
  remove   Remove a key by value or product name
  status   Show all configured keys, products, and plugin coverage
  list     Alias for status
  show     Display saved key (masked) and tier
  validate Validate key against ping.nself.org
  clear    Remove all saved license keys
  upgrade  Open pricing page in browser`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var licenseSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Replace all keys with a single key",
	Long: `Save a license key, replacing any existing keys.

If multiple keys are currently configured, they will all be replaced with the
new key. A warning is printed when replacing multiple keys. Use 'nself license add'
to add a key without replacing existing ones.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		replaced, err := license.SetKeyReplaceAll(key)
		if err != nil {
			return fmt.Errorf("setting license key: %w", err)
		}
		if replaced > 1 {
			fmt.Fprintf(os.Stderr, "Warning: Replacing %d existing keys with single key.\n", replaced)
		}
		pp := license.DetectProduct(key)
		if pp != nil {
			fmt.Printf("%s license key saved.\n", pp.DisplayName)
		} else {
			fmt.Println("License key saved.")
		}
		return nil
	},
}

var licenseAddCmd = &cobra.Command{
	Use:   "add <key> [key...]",
	Short: "Add one or more license keys",
	Long: `Add license keys without replacing existing ones.

Each key is validated for format and verified against ping.nself.org before
being saved. Supports up to 10 keys total.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pingURL := os.Getenv("NSELF_PING_API_URL")
		if pingURL == "" {
			pingURL = defaultPingURL
		}

		for _, key := range args {
			masked := license.MaskKey(key)

			if err := license.ValidateKeyFormat(key); err != nil {
				return fmt.Errorf("invalid key %q: %w", masked, err)
			}

			// Validate against server with spinner feedback.
			sp := ui.NewSpinner(fmt.Sprintf("Validating key %s ...", masked))
			sp.Start()
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			valid, err := plugin.ValidateLicenseRemote(ctx, key, pingURL)
			cancel()
			if err != nil {
				sp.Fail(fmt.Sprintf("Validation error for %s: %v", masked, err))
				return fmt.Errorf("validating key: %w", err)
			}
			if !valid {
				sp.Fail(fmt.Sprintf("Key %s is invalid or expired", masked))
				return fmt.Errorf("key %s is invalid or expired", masked)
			}
			sp.Success(fmt.Sprintf("Key %s validated", masked))

			if err := license.AddKey(key); err != nil {
				return fmt.Errorf("adding key: %w", err)
			}

			pp := license.DetectProduct(key)
			if pp != nil {
				ui.Success(fmt.Sprintf("%s license activated.", pp.DisplayName))
				fmt.Printf("  %s Install plugins: %s\n",
					ui.C(ui.Blue, ui.IconArrow),
					ui.C(ui.Cyan, "nself plugin list --pro"),
				)
			} else {
				ui.Success(fmt.Sprintf("License key %s added.", masked))
			}
		}
		return nil
	},
}

var licenseRemoveCmd = &cobra.Command{
	Use:   "remove <key-or-product>",
	Short: "Remove a license key by value or product name",
	Long: `Remove a license key by key prefix or product name.

Examples:
  nself license remove nself_claw_xxx   Remove by key prefix
  nself license remove claw             Remove all claw keys
  nself license remove chat             Remove all chat keys`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		removed, err := license.RemoveKey(query)
		if err != nil {
			return err
		}
		if removed == 1 {
			fmt.Println("License key removed.")
		} else {
			fmt.Printf("%d license keys removed.\n", removed)
		}
		return nil
	},
}

var licenseStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show all configured licenses and plugin coverage",
	RunE:  runLicenseStatus,
}

var licenseListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all configured licenses (alias for status)",
	RunE:  runLicenseStatus,
}

func runLicenseStatus(cmd *cobra.Command, args []string) error {
	keys := license.CollectLicenseKeys()
	if len(keys) == 0 {
		fmt.Println("No license keys configured.")
		fmt.Printf("\nGet a product license at %s\n", pricingURL)
		return nil
	}

	pingURL := os.Getenv("NSELF_PING_API_URL")
	if pingURL == "" {
		pingURL = defaultPingURL
	}

	tbl := ui.NewTable("License", "Product", "Status", "Expires", "Plugins")

	var allProducts []string
	allPlugins := make(map[string]bool)
	hasPlus := false

	for _, key := range keys {
		masked := license.MaskKey(key)
		pp := license.DetectProduct(key)
		productName := "Unknown"
		if pp != nil {
			productName = pp.DisplayName
			if pp.Product == "plus" || pp.Product == "owner" {
				hasPlus = true
			}
		}

		// Try to validate and get entitlements.
		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		valid, lvr, err := plugin.ValidateLicenseRemoteWithDetails(ctx, key, pingURL)
		cancel()

		status := "Unknown"
		expires := "-"
		plugins := "-"

		if err != nil {
			status = "Error"
		} else if !valid {
			status = "Invalid"
		} else {
			status = "Active"
			if lvr != nil {
				if lvr.Tier != "" {
					productName = lvr.Tier
				}
				if lvr.Expires != "" {
					expires = lvr.Expires
				}
				if len(lvr.Plugins) > 0 {
					for _, p := range lvr.Plugins {
						allPlugins[p] = true
					}
					if len(lvr.Plugins) > 5 {
						plugins = strings.Join(lvr.Plugins[:5], ", ") + fmt.Sprintf(" (+%d more)", len(lvr.Plugins)-5)
					} else {
						plugins = strings.Join(lvr.Plugins, ", ")
					}
				}
			}
		}

		allProducts = append(allProducts, productName)
		tbl.AddRow(masked, productName, status, expires, plugins)
	}

	tbl.Render()

	// Summary.
	if len(allProducts) > 0 {
		fmt.Printf("\nProducts: %s\n", strings.Join(uniqueStrings(allProducts), ", "))
	}
	if len(allPlugins) > 0 {
		var pluginList []string
		for p := range allPlugins {
			pluginList = append(pluginList, p)
		}
		fmt.Printf("Plugins available: %s\n", strings.Join(pluginList, ", "))
	}

	if !hasPlus {
		fmt.Printf("\nUpgrade to ɳSelf+ ($3.99/mo or $39.99/yr) for all plugins: %s\n", pricingURL)
	}

	return nil
}

// licenseShowJSON is the structured output for --json mode of `nself license show`.
type licenseShowJSON struct {
	KeyPrefix    string   `json:"key_prefix"`
	Tier         string   `json:"tier"`
	Bundles      []string `json:"bundles_unlocked"`
	PluginsCount int      `json:"plugins_available_count"`
	DaysToExpire *int     `json:"days_to_expire,omitempty"`
	CacheExpiry  string   `json:"cache_expiry,omitempty"`
}

var licenseShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display license tier, bundles, plugins, and cache expiry",
	Long: `Display the current license configuration.

Shows tier, bundles unlocked, plugins available, days until the server-side
license expires, and cache TTL status. The raw key is never shown — only the
last 4 characters are printed as a prefix hint.

Use --json for machine-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonMode, _ := cmd.Flags().GetBool("json")

		keys := license.CollectLicenseKeys()
		if len(keys) == 0 {
			if jsonMode {
				return json.NewEncoder(os.Stdout).Encode(map[string]string{"error": "no license key configured"})
			}
			fmt.Println("No license key configured.")
			fmt.Printf("Get a license at %s\n", pricingURL)
			return nil
		}

		// Use first key for display (status shows all keys).
		key := keys[0]
		masked := license.MaskKey(key)
		tier := license.DetectTierFromKey(key)
		pp := license.DetectProduct(key)

		// Last 4 chars of key as prefix hint (per S132-T04 spec).
		keyHint := key
		if len(key) > 4 {
			keyHint = "..." + key[len(key)-4:]
		}

		// Read cache for TTL/expiry info.
		cache, _ := license.ReadCache()
		var daysToExpire *int
		cacheExpiry := ""
		if cache != nil {
			fetchedAt := cache.FetchedAt
			ttlExpiry := license.CacheTTLExpiry(cache.Tier, timeFromUnix(fetchedAt))
			remaining := ttlExpiry.Sub(time.Now())
			if remaining > 0 {
				days := int(remaining.Hours() / 24)
				daysToExpire = &days
				cacheExpiry = ttlExpiry.Format("2006-01-02")
			} else {
				zero := 0
				daysToExpire = &zero
				cacheExpiry = "expired"
			}
		}

		// Bundles from product prefix.
		var bundles []string
		if pp != nil {
			bundles = []string{pp.DisplayName}
		}

		// Plugins from cache.
		var pluginsCount int
		if cache != nil {
			pluginsCount = len(cache.PluginsAllowed)
		}

		if jsonMode {
			out := licenseShowJSON{
				KeyPrefix:    keyHint,
				Tier:         tier,
				Bundles:      bundles,
				PluginsCount: pluginsCount,
				DaysToExpire: daysToExpire,
				CacheExpiry:  cacheExpiry,
			}
			return json.NewEncoder(os.Stdout).Encode(out)
		}

		fmt.Printf("Key:              %s\n", masked)
		fmt.Printf("Key hint:         %s\n", keyHint)
		fmt.Printf("Tier:             %s\n", tier)
		if len(bundles) > 0 {
			fmt.Printf("Bundles unlocked: %s\n", strings.Join(bundles, ", "))
		}
		if pluginsCount > 0 {
			fmt.Printf("Plugins available: %d\n", pluginsCount)
		}
		if daysToExpire != nil {
			if *daysToExpire == 0 || cacheExpiry == "expired" {
				fmt.Println("Cache expiry:     expired — run 'nself license revalidate' after connecting")
			} else {
				fmt.Printf("Cache expires:    %s (%d day(s))\n", cacheExpiry, *daysToExpire)
			}
		} else {
			fmt.Println("Cache expiry:     no cache — run 'nself license validate' to populate")
		}

		// Pre-expiry warning (S132-T01 / T02).
		if cache != nil {
			if msg, _ := license.PreExpiryWarning(cache); msg != "" {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", msg)
			}
		}

		return nil
	},
}

// timeFromUnix converts a Unix timestamp (int64) to time.Time.
func timeFromUnix(ts int64) time.Time {
	return time.Unix(ts, 0)
}

var licenseValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate key against ping.nself.org",
	RunE: func(cmd *cobra.Command, args []string) error {
		keys := license.CollectLicenseKeys()
		if len(keys) == 0 {
			return fmt.Errorf("no license key configured — run 'nself license add <key>' first")
		}

		pingURL := os.Getenv("NSELF_PING_API_URL")
		if pingURL == "" {
			pingURL = defaultPingURL
		}

		allValid := true
		for _, key := range keys {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			valid, err := plugin.ValidateLicenseRemote(ctx, key, pingURL)
			cancel()

			masked := license.MaskKey(key)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s: validation failed: %v\n", masked, err)
				allValid = false
			} else if valid {
				fmt.Printf("%s: valid\n", masked)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s: invalid or expired\n", masked)
				allValid = false
			}
		}

		if !allValid {
			return fmt.Errorf("one or more keys failed validation")
		}
		return nil
	},
}

var licenseClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all saved license keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := license.ClearKey(); err != nil {
			return fmt.Errorf("clearing license key: %w", err)
		}
		fmt.Println("All license keys removed.")
		return nil
	},
}

var licenseUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Open pricing page in browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		var openCmd string
		switch runtime.GOOS {
		case "darwin":
			openCmd = "open"
		case "linux":
			openCmd = "xdg-open"
		default:
			return fmt.Errorf("unsupported platform %s — visit %s manually", runtime.GOOS, pricingURL)
		}

		if err := exec.Command(openCmd, pricingURL).Start(); err != nil {
			return fmt.Errorf("opening browser: %w", err)
		}
		fmt.Printf("Opening %s\n", pricingURL)
		return nil
	},
}

// licenseHealthJSON is the structured output for `nself license health --json`.
type licenseHealthJSON struct {
	FormatOK       bool   `json:"format_ok"`
	ServerReachable bool  `json:"server_reachable"`
	ServerValid    bool   `json:"server_valid"`
	CacheIntegrity bool   `json:"cache_integrity"`
	HealthScore    int    `json:"health_score"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
}

var licenseHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Validate format, ping server, and report cache integrity",
	Long: `Run a combined license health check:

  1. Format validation — key prefix and length.
  2. Server ping — reach ping.nself.org and validate the key.
  3. Cache integrity — verify Ed25519 signature on cached entry.

Exits 0 when all checks pass, 1 when any check fails.
Use --json for machine-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonMode, _ := cmd.Flags().GetBool("json")

		keys := license.CollectLicenseKeys()
		if len(keys) == 0 {
			if jsonMode {
				out := licenseHealthJSON{Status: "no_key", Message: "no license key configured"}
				return json.NewEncoder(os.Stdout).Encode(out)
			}
			fmt.Println("No license key configured.")
			return fmt.Errorf("no license key — run 'nself license add <key>'")
		}

		key := keys[0]

		// 1. Format check.
		formatOK := license.ValidateKeyFormat(key) == nil

		// 2. Server ping.
		pingURL := os.Getenv("NSELF_PING_API_URL")
		if pingURL == "" {
			pingURL = defaultPingURL
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		serverValid, pingErr := plugin.ValidateLicenseRemote(ctx, key, pingURL)
		cancel()
		serverReachable := pingErr == nil

		// 3. Cache integrity.
		cacheOK := false
		if cache, err := license.ReadCache(); err == nil && cache != nil {
			cacheOK = cache.VerifySignature()
		}

		// Health score: 0-3 checks passing.
		score := 0
		if formatOK {
			score++
		}
		if serverReachable && serverValid {
			score++
		}
		if cacheOK {
			score++
		}

		status := "unhealthy"
		switch score {
		case 3:
			status = "healthy"
		case 2:
			status = "degraded"
		}

		var msgParts []string
		if !formatOK {
			msgParts = append(msgParts, "key format invalid")
		}
		if !serverReachable {
			msgParts = append(msgParts, "server unreachable")
		} else if !serverValid {
			msgParts = append(msgParts, "key invalid or expired on server")
		}
		if !cacheOK {
			msgParts = append(msgParts, "cache signature invalid or missing")
		}
		message := strings.Join(msgParts, "; ")

		if jsonMode {
			out := licenseHealthJSON{
				FormatOK:        formatOK,
				ServerReachable: serverReachable,
				ServerValid:     serverValid,
				CacheIntegrity:  cacheOK,
				HealthScore:     score,
				Status:          status,
				Message:         message,
			}
			return json.NewEncoder(os.Stdout).Encode(out)
		}

		check := func(ok bool) string {
			if ok {
				return "✓"
			}
			return "✗"
		}
		fmt.Printf("Format valid:      %s\n", check(formatOK))
		fmt.Printf("Server reachable:  %s\n", check(serverReachable))
		fmt.Printf("Server valid:      %s\n", check(serverReachable && serverValid))
		fmt.Printf("Cache integrity:   %s\n", check(cacheOK))
		fmt.Printf("Health score:      %d/3\n", score)
		fmt.Printf("Status:            %s\n", status)
		if message != "" {
			fmt.Printf("Issues:            %s\n", message)
		}

		if score < 3 {
			return fmt.Errorf("license health check failed (score %d/3)", score)
		}
		return nil
	},
}

var licenseRevalidateCmd = &cobra.Command{
	Use:   "revalidate",
	Short: "Force a fresh validation against ping.nself.org and update the cache",
	Long: `Revalidate the current license key against ping.nself.org and refresh
the local cache. Useful after cache expiry or when reconnecting after offline use.

Requires network connectivity.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		keys := license.CollectLicenseKeys()
		if len(keys) == 0 {
			return fmt.Errorf("no license key configured — run 'nself license add <key>' first")
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		result, err := license.RefreshCache(ctx, keys[0])
		if err != nil {
			return fmt.Errorf("revalidation failed: %w", err)
		}
		if !result.Valid {
			return fmt.Errorf("license is invalid: %s", result.Message)
		}

		fmt.Printf("License revalidated. Tier: %s\n", result.Tier)
		if !result.ExpiresAt.IsZero() {
			fmt.Printf("Expires: %s\n", result.ExpiresAt.Format("2006-01-02"))
		}
		fmt.Println("Cache updated.")
		return nil
	},
}

var licenseRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Mark local license as revoked and wipe the stored key",
	Long: `Revoke the local license:

  - Marks the local cache as revoked.
  - Wipes the stored license key from disk.
  - Plugins using this license go DORMANT (not uninstalled) on the next build.

Use 'nself license restore --key <new-key>' to reactivate.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		confirm, _ := cmd.Flags().GetBool("yes")
		if !confirm {
			fmt.Println("This will revoke the local license and wipe the stored key.")
			fmt.Println("Plugins will go DORMANT on the next build.")
			fmt.Println("Run with --yes to confirm.")
			return nil
		}

		if err := license.RevokeLicense(); err != nil {
			return fmt.Errorf("revocation failed: %w", err)
		}

		fmt.Println("License revoked. Stored key removed.")
		fmt.Println("Plugins will show DORMANT status after next 'nself build'.")
		fmt.Println("To reactivate: nself license restore --key <new-key>")
		return nil
	},
}

var licenseRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Reactivate dormant plugins with a new license key",
	Long: `Restore a previously revoked license:

  1. Validates the new key format.
  2. Validates the key against ping.nself.org.
  3. Clears the revoked marker.
  4. Stores the new key so plugins re-activate on the next build.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		newKey, _ := cmd.Flags().GetString("key")
		newKey = strings.TrimSpace(newKey)
		if newKey == "" {
			return fmt.Errorf("--key <new-key> is required")
		}

		// Format validation.
		if err := license.ValidateKeyFormat(newKey); err != nil {
			return fmt.Errorf("invalid key format: %w", err)
		}

		// Remote validation.
		pingURL := os.Getenv("NSELF_PING_API_URL")
		if pingURL == "" {
			pingURL = defaultPingURL
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		valid, err := plugin.ValidateLicenseRemote(ctx, newKey, pingURL)
		cancel()
		if err != nil {
			return fmt.Errorf("key validation failed: %w", err)
		}
		if !valid {
			return fmt.Errorf("key is invalid or expired on the server")
		}

		// Restore.
		if err := license.RestoreWithKey(newKey); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}

		pp := license.DetectProduct(newKey)
		if pp != nil {
			fmt.Printf("%s license restored.\n", pp.DisplayName)
		} else {
			fmt.Println("License restored.")
		}
		fmt.Println("Dormant plugins will re-activate on the next 'nself build'.")
		return nil
	},
}

// uniqueStrings returns a deduplicated copy of the input slice preserving order.
func uniqueStrings(in []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func init() {
	// Flags for new commands.
	licenseShowCmd.Flags().Bool("json", false, "Output as JSON")
	licenseHealthCmd.Flags().Bool("json", false, "Output as JSON")
	licenseRevokeCmd.Flags().Bool("yes", false, "Confirm revocation without interactive prompt")
	licenseRestoreCmd.Flags().String("key", "", "New license key to restore with")

	licenseCmd.AddCommand(licenseSetCmd)
	licenseCmd.AddCommand(licenseAddCmd)
	licenseCmd.AddCommand(licenseRemoveCmd)
	licenseCmd.AddCommand(licenseStatusCmd)
	licenseCmd.AddCommand(licenseListCmd)
	licenseCmd.AddCommand(licenseShowCmd)
	licenseCmd.AddCommand(licenseValidateCmd)
	licenseCmd.AddCommand(licenseRevalidateCmd)
	licenseCmd.AddCommand(licenseHealthCmd)
	licenseCmd.AddCommand(licenseRevokeCmd)
	licenseCmd.AddCommand(licenseRestoreCmd)
	licenseCmd.AddCommand(licenseClearCmd)
	licenseCmd.AddCommand(licenseUpgradeCmd)
	RootCmd.AddCommand(licenseCmd)
}
