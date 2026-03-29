package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"nself/internal/license"
	"nself/internal/plugin"

	"github.com/spf13/cobra"
)

// defaultPingURL is the production license validation endpoint.
const defaultPingURL = "https://ping.nself.org"

// pricingURL is the page opened by the upgrade subcommand.
const pricingURL = "https://nself.org/pricing"

var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Manage Pro membership license key",
	Long:  "Set, show, validate, clear, or upgrade your nSelf Pro license key.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var licenseSetCmd = &cobra.Command{
	Use:   "set <key>",
	Short: "Save Pro license key",
	Long:  "Save a Pro license key to ~/.nself/license/key (permissions 0600).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		if err := license.SetKey(key); err != nil {
			return fmt.Errorf("setting license key: %w", err)
		}
		fmt.Println("License key saved successfully.")
		return nil
	},
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
		key, err := license.GetKey()
		if err != nil {
			return fmt.Errorf("reading license key: %w", err)
		}
		if key == "" {
			return fmt.Errorf("no license key configured — run 'nself license set <key>' first")
		}

		pingURL := os.Getenv("NSELF_PING_API_URL")
		if pingURL == "" {
			pingURL = defaultPingURL
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		valid, err := plugin.ValidateLicenseRemote(ctx, key, pingURL)
		if err != nil {
			return fmt.Errorf("license validation failed: %w", err)
		}
		if valid {
			fmt.Println("License key is valid.")
		} else {
			fmt.Fprintln(cmd.ErrOrStderr(), "License key is invalid or expired.")
		}
		return nil
	},
}

var licenseClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove saved license key",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := license.ClearKey(); err != nil {
			return fmt.Errorf("clearing license key: %w", err)
		}
		fmt.Println("License key removed.")
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

func init() {
	licenseCmd.AddCommand(licenseSetCmd)
	licenseCmd.AddCommand(licenseShowCmd)
	licenseCmd.AddCommand(licenseValidateCmd)
	licenseCmd.AddCommand(licenseClearCmd)
	licenseCmd.AddCommand(licenseUpgradeCmd)
	RootCmd.AddCommand(licenseCmd)
}
