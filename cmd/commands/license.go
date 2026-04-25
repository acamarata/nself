package commands

import (
	"context"
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
			if err := license.ValidateKeyFormat(key); err != nil {
				return fmt.Errorf("invalid key %q: %w", license.MaskKey(key), err)
			}

			// Validate against server.
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			valid, err := plugin.ValidateLicenseRemote(ctx, key, pingURL)
			cancel()
			if err != nil {
				return fmt.Errorf("validating key: %w", err)
			}
			if !valid {
				return fmt.Errorf("key %s is invalid or expired", license.MaskKey(key))
			}

			if err := license.AddKey(key); err != nil {
				return fmt.Errorf("adding key: %w", err)
			}

			pp := license.DetectProduct(key)
			if pp != nil {
				fmt.Printf("%s license activated.\n", pp.DisplayName)
			} else {
				fmt.Printf("License key %s added.\n", license.MaskKey(key))
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

var licenseShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display saved key (masked) and tier",
	RunE: func(cmd *cobra.Command, args []string) error {
		masked, tier, err := license.ShowKey()
		if err != nil {
			return fmt.Errorf("reading license key: %w", err)
		}
		if masked == "" {
			fmt.Println("No license key configured.")
			return nil
		}
		fmt.Printf("Key:  %s\n", masked)
		fmt.Printf("Tier: %s\n", tier)
		return nil
	},
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
	licenseCmd.AddCommand(licenseSetCmd)
	licenseCmd.AddCommand(licenseAddCmd)
	licenseCmd.AddCommand(licenseRemoveCmd)
	licenseCmd.AddCommand(licenseStatusCmd)
	licenseCmd.AddCommand(licenseListCmd)
	licenseCmd.AddCommand(licenseShowCmd)
	licenseCmd.AddCommand(licenseValidateCmd)
	licenseCmd.AddCommand(licenseClearCmd)
	licenseCmd.AddCommand(licenseUpgradeCmd)
	RootCmd.AddCommand(licenseCmd)
}
