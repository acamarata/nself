package commands

import (
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/auth"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

// ── Parent command ──────────────────────────────────────────────────────────

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage your nSelf account",
	Long: `Manage your nSelf account.

Subcommands:
  show            Show account details (email, tier, bundles)
  licenses list   List active licenses linked to your account`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return fmt.Errorf("unknown account subcommand %q; run 'nself account --help'", args[0])
	},
}

func init() {
	accountCmd.AddCommand(accountShowCmd)
	accountCmd.AddCommand(accountLicensesCmd)
	RootCmd.AddCommand(accountCmd)
}

// ── account show ────────────────────────────────────────────────────────────

var accountShowCmd = &cobra.Command{
	Use:          "show",
	Short:        "Show your account details",
	SilenceUsage: true,
	RunE:         runAccountShow,
}

func runAccountShow(cmd *cobra.Command, _ []string) error {
	af, err := auth.ReadAuthFile()
	if err != nil {
		if err == auth.ErrNotLoggedIn {
			ui.Error("Not logged in. Run 'nself login' to authenticate.")
			return nil
		}
		return fmt.Errorf("account show: %w", err)
	}

	// Fetch live account info from server.
	info, err := auth.GetSession(af.AccessToken)
	if err != nil {
		// Fall back to cached data if server is unreachable.
		ui.Warn("Could not reach auth server — showing cached account info.")
		printCachedAccount(af)
		return nil
	}

	if !info.Authenticated {
		ui.Error("Session expired. Run 'nself login' to re-authenticate.")
		return nil
	}

	acc := info.Account
	tier := acc.Tier
	if tier == "" {
		tier = "free"
	}

	fmt.Printf("Email:         %s\n", acc.Email)
	if acc.DisplayName != "" {
		fmt.Printf("Name:          %s\n", acc.DisplayName)
	}
	fmt.Printf("Tier:          %s\n", tier)

	verified := "no"
	if acc.EmailVerified {
		verified = "yes"
	}
	fmt.Printf("Email verified: %s\n", verified)

	mfa := "disabled"
	if acc.MFAEnabled {
		mfa = "enabled"
	}
	fmt.Printf("MFA:           %s\n", mfa)

	if len(af.Bundles) > 0 {
		fmt.Printf("Bundles:       %s\n", strings.Join(af.Bundles, ", "))
	}

	if af.ExpiresAt != "" {
		fmt.Printf("Token expires: %s\n", af.ExpiresAt)
	}

	return nil
}

// printCachedAccount prints account info from the local auth file (no network).
func printCachedAccount(af *auth.AuthFile) {
	tier := af.Tier
	if tier == "" {
		tier = "free"
	}
	fmt.Printf("Email:   %s\n", af.Email)
	if af.DisplayName != "" {
		fmt.Printf("Name:    %s\n", af.DisplayName)
	}
	fmt.Printf("Tier:    %s\n", tier)
	if len(af.Bundles) > 0 {
		fmt.Printf("Bundles: %s\n", strings.Join(af.Bundles, ", "))
	}
	fmt.Printf("(cached — run 'nself account show' with network access for live data)\n")
}

// ── account licenses ─────────────────────────────────────────────────────────

var accountLicensesCmd = &cobra.Command{
	Use:          "licenses",
	Short:        "Manage account licenses",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return fmt.Errorf("unknown account licenses subcommand %q; run 'nself account licenses --help'", args[0])
	},
}

func init() {
	accountLicensesCmd.AddCommand(accountLicensesListCmd)
}

var accountLicensesListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List active licenses linked to your account",
	SilenceUsage: true,
	RunE:         runAccountLicensesList,
}

func runAccountLicensesList(cmd *cobra.Command, _ []string) error {
	af, err := auth.ReadAuthFile()
	if err != nil {
		if err == auth.ErrNotLoggedIn {
			ui.Error("Not logged in. Run 'nself login' to authenticate.")
			return nil
		}
		return fmt.Errorf("account licenses list: %w", err)
	}

	licenses, err := auth.GetLicenses(af.AccessToken)
	if err != nil {
		return fmt.Errorf("fetching licenses: %w", err)
	}

	if len(licenses) == 0 {
		fmt.Println("No licenses linked to your account.")
		fmt.Printf("Activate a license key at %s or via 'nself license set <key>'.\n", "https://nself.org/account/licenses")
		return nil
	}

	tbl := ui.NewTable("ID", "Product", "Tier", "Bundles", "Seats", "Expires")
	for _, lic := range licenses {
		seats := fmt.Sprintf("%d/%d", lic.SeatsUsed, lic.SeatsIncluded)
		bundles := strings.Join(lic.Bundles, ", ")
		if bundles == "" {
			bundles = "-"
		}
		expires := lic.ExpiresAt
		if expires == "" {
			expires = "never"
		}
		status := lic.Tier
		if !lic.IsActive {
			status = lic.Tier + " (inactive)"
		}
		tbl.AddRow(lic.ID[:8]+"...", lic.Product, status, bundles, seats, expires)
	}
	tbl.Render()

	return nil
}
