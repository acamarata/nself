package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/auth"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

// ── Parent command ──────────────────────────────────────────────────────────

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage your nSelf account, sessions, licenses, team, and devices",
	Long: `Manage your nSelf account.

With no subcommand, shows the current account status (same as 'nself account status').

Subcommands:
  login      Log in via device-code OAuth flow
  logout     Revoke current session and clear local credentials
  status     Show current account summary (default)
  team       List team members; --invite, --remove, --role
  licenses   List active licenses; --activate <id>
  devices    List registered devices; --revoke <id>
  transfer   Move a license to another account`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default: show status.
		return runAccountStatus(cmd, args)
	},
}

func init() {
	accountCmd.AddCommand(accountLoginCmd)
	accountCmd.AddCommand(accountLogoutCmd)
	accountCmd.AddCommand(accountStatusCmd)
	accountCmd.AddCommand(accountTeamCmd)
	accountCmd.AddCommand(accountLicensesCmd)
	accountCmd.AddCommand(accountDevicesCmd)
	accountCmd.AddCommand(accountTransferCmd)
	RootCmd.AddCommand(accountCmd)
}

// ── account login ────────────────────────────────────────────────────────────

var accountLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your nSelf account",
	Long: `Log in to your nSelf account using the device authorization flow.

Opens your browser to authorize this CLI session. Confirm the short code
shown here matches the one on screen, then click Authorize.

After authorization, your session is stored at ~/.nself/auth.json (0600).`,
	SilenceUsage: true,
	RunE:         runAccountLogin,
}

func init() {
	accountLoginCmd.Flags().Bool("reauth", false, "Force new auth even if already logged in")
	accountLoginCmd.Flags().Bool("no-browser", false, "Print URL and code; do not open browser")
	accountLoginCmd.Flags().Int("timeout", 300, "Device-code poll timeout in seconds")
}

func runAccountLogin(cmd *cobra.Command, _ []string) error {
	reauth, _ := cmd.Flags().GetBool("reauth")
	noBrowser, _ := cmd.Flags().GetBool("no-browser")
	timeoutSec, _ := cmd.Flags().GetInt("timeout")

	// NSELF_NO_BROWSER env var overrides flag.
	if os.Getenv("NSELF_NO_BROWSER") == "1" {
		noBrowser = true
	}

	// Guard: already logged in.
	if !reauth && auth.IsLoggedIn() {
		af, err := auth.ReadAuthFile()
		if err == nil && af != nil {
			ui.Warn(fmt.Sprintf("Already logged in as %s. Use --reauth to force.", af.Email))
			return nil
		}
	}

	// Step 1: request device code.
	ui.Info("Contacting nSelf auth server...")
	dcResp, err := auth.DeviceAuthorize(cmdCtx(cmd))
	if err != nil {
		return fmt.Errorf("cannot reach nself.org — check your connection: %w", err)
	}

	// Step 2: show user code and verification URL.
	verificationURL := dcResp.VerificationURL
	if verificationURL == "" {
		verificationURL = fmt.Sprintf("%s?code=%s", auth.CLIAuthBaseURL(), dcResp.UserCode)
	}

	fmt.Printf("\n%s Your login code: %s\n", ui.C(ui.Bold, "»"), ui.C(ui.BrightYellow, dcResp.UserCode))
	fmt.Printf("%s Open: %s\n\n", ui.C(ui.Bold, "»"), ui.C(ui.Cyan, verificationURL))

	if !noBrowser {
		openBrowser(verificationURL)
		fmt.Println("Browser opened. Waiting for authorization...")
	} else {
		fmt.Println("Open the URL above in your browser, then wait here for confirmation.")
	}

	// Step 3: poll for the token.
	baseCtx := cmdCtx(cmd)
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	timeout := time.Duration(timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(baseCtx, timeout+time.Minute)
	defer cancel()

	var lastDot time.Time
	result, err := auth.PollDeviceCode(ctx, dcResp.DeviceCode, func(elapsed time.Duration) {
		if time.Since(lastDot) >= 10*time.Second {
			fmt.Print(".")
			lastDot = time.Now()
		}
	})
	fmt.Println()

	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	// Step 4: persist credentials.
	af := &auth.AuthFile{
		AccessToken:  result.Token.AccessToken,
		SessionToken: result.Token.SessionToken,
		Email:        result.Token.Email,
		Tier:         result.Token.Tier,
		DisplayName:  result.Token.DisplayName,
		Bundles:      result.Token.Bundles,
		ExpiresAt:    result.Token.ExpiresAt,
	}
	if err := auth.WriteAuthFile(af); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	tier := result.Token.Tier
	if tier == "" {
		tier = "free"
	}
	ui.Success(fmt.Sprintf("Logged in as %s (%s).", result.Token.Email, tier))

	if len(result.Token.Bundles) > 0 {
		fmt.Printf("Bundles: %v\n", result.Token.Bundles)
	}

	return nil
}

// ── account logout ───────────────────────────────────────────────────────────

var accountLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of your nSelf account",
	Long: `Log out of your nSelf account.

Revokes the current session on the auth server and deletes the local
credentials file at ~/.nself/auth.json.

Use --all to revoke every active session (useful when a device is lost).`,
	SilenceUsage: true,
	RunE:         runAccountLogout,
}

func init() {
	accountLogoutCmd.Flags().Bool("all", false, "Revoke all active sessions, not just the current one")
}

func runAccountLogout(cmd *cobra.Command, _ []string) error {
	all, _ := cmd.Flags().GetBool("all")

	af, err := auth.ReadAuthFile()
	if err != nil {
		if err == auth.ErrNotLoggedIn {
			fmt.Println("Not logged in.")
			return nil
		}
		return fmt.Errorf("logout: %w", err)
	}

	// Revoke on server (best-effort — still delete local file on failure).
	if revokeErr := auth.RevokeSession(cmdCtx(cmd), af.AccessToken, all); revokeErr != nil {
		ui.Warn(fmt.Sprintf("Could not revoke session on server: %v", revokeErr))
		ui.Warn("Deleting local credentials anyway.")
	}

	if err := auth.DeleteAuthFile(); err != nil {
		return fmt.Errorf("deleting credentials: %w", err)
	}

	if all {
		ui.Success("Logged out. All sessions revoked.")
	} else {
		ui.Success("Logged out. Session revoked.")
	}
	return nil
}

// ── account status ───────────────────────────────────────────────────────────

var accountStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current account summary",
	Long: `Show current account summary.

Displays email, tier, license count, and registered devices.
Use --json for raw JSON output.`,
	SilenceUsage: true,
	RunE:         runAccountStatus,
}

func init() {
	accountStatusCmd.Flags().Bool("json", false, "Output raw JSON")
}

func runAccountStatus(cmd *cobra.Command, _ []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")

	af, err := auth.ReadAuthFile()
	if err != nil {
		if err == auth.ErrNotLoggedIn {
			fmt.Println("not logged in — run: nself account login")
			return nil
		}
		return fmt.Errorf("account status: %w", err)
	}

	info, infoErr := auth.GetSession(cmdCtx(cmd), af.AccessToken)
	if infoErr != nil {
		// Fall back to cached data.
		if jsonOut {
			return printCachedAccountJSON(af)
		}
		ui.Warn("Could not reach auth server — showing cached account info.")
		printCachedAccount(af)
		return nil
	}

	if !info.Authenticated {
		fmt.Println("session expired — run: nself account login")
		return nil
	}

	acc := info.Account
	tier := acc.Tier
	if tier == "" {
		tier = "free"
	}

	if jsonOut {
		out := map[string]interface{}{
			"email":          acc.Email,
			"display_name":   acc.DisplayName,
			"tier":           tier,
			"email_verified": acc.EmailVerified,
			"mfa_enabled":    acc.MFAEnabled,
			"bundles":        af.Bundles,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("Account:   %s\n", acc.Email)
	if acc.DisplayName != "" {
		fmt.Printf("Name:      %s\n", acc.DisplayName)
	}
	fmt.Printf("Tier:      %s\n", tier)

	verified := "no"
	if acc.EmailVerified {
		verified = "yes"
	}
	fmt.Printf("Verified:  %s\n", verified)

	mfa := "disabled"
	if acc.MFAEnabled {
		mfa = "enabled"
	}
	fmt.Printf("MFA:       %s\n", mfa)

	if len(af.Bundles) > 0 {
		fmt.Printf("Bundles:   %s\n", strings.Join(af.Bundles, ", "))
	}
	if af.ExpiresAt != "" {
		fmt.Printf("Expires:   %s\n", af.ExpiresAt)
	}

	return nil
}

// printCachedAccount prints account info from the local auth file (no network).
func printCachedAccount(af *auth.AuthFile) {
	tier := af.Tier
	if tier == "" {
		tier = "free"
	}
	fmt.Printf("Account:   %s (cached)\n", af.Email)
	if af.DisplayName != "" {
		fmt.Printf("Name:      %s\n", af.DisplayName)
	}
	fmt.Printf("Tier:      %s\n", tier)
	if len(af.Bundles) > 0 {
		fmt.Printf("Bundles:   %s\n", strings.Join(af.Bundles, ", "))
	}
}

func printCachedAccountJSON(af *auth.AuthFile) error {
	tier := af.Tier
	if tier == "" {
		tier = "free"
	}
	out := map[string]interface{}{
		"email":        af.Email,
		"display_name": af.DisplayName,
		"tier":         tier,
		"bundles":      af.Bundles,
		"cached":       true,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// ── account show (legacy alias — same as status) ─────────────────────────────

// accountShowCmd remains for backward compat. It delegates to runAccountStatus.
var accountShowCmd = &cobra.Command{
	Use:          "show",
	Short:        "Show your account details (alias for status)",
	SilenceUsage: true,
	Hidden:       true,
	RunE:         runAccountStatus,
}

// ── account team ─────────────────────────────────────────────────────────────

var accountTeamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage team members",
	Long: `Manage team members on your account.

With no flags, lists all current members.

Flags:
  --invite <email>          Send a team invite
  --remove <email>          Remove a member (prompts for confirmation)
  --role <email> <role>     Set a member role (owner|admin|member)
  --json                    Output raw JSON`,
	SilenceUsage: true,
	RunE:         runAccountTeam,
}

func init() {
	accountTeamCmd.Flags().String("invite", "", "Email address to invite")
	accountTeamCmd.Flags().String("remove", "", "Email address to remove")
	accountTeamCmd.Flags().StringSlice("role", nil, "Set role: <email> <role>")
	accountTeamCmd.Flags().Bool("json", false, "Output raw JSON")
}

func runAccountTeam(cmd *cobra.Command, _ []string) error {
	invite, _ := cmd.Flags().GetString("invite")
	remove, _ := cmd.Flags().GetString("remove")
	roleArgs, _ := cmd.Flags().GetStringSlice("role")
	jsonOut, _ := cmd.Flags().GetBool("json")

	af, err := requireLogin()
	if err != nil {
		return err
	}

	// --invite
	if invite != "" {
		if err := auth.InviteTeamMember(cmdCtx(cmd), af.AccessToken, invite); err != nil {
			return handleAuthError(err)
		}
		ui.Success(fmt.Sprintf("Invite sent to %s.", invite))
		return nil
	}

	// --remove
	if remove != "" {
		if !promptConfirm(fmt.Sprintf("Remove %s from your team? [y/N]: ", remove)) {
			fmt.Println("Canceled.")
			return nil
		}
		if err := auth.RemoveTeamMember(cmdCtx(cmd), af.AccessToken, remove); err != nil {
			return handleAuthError(err)
		}
		ui.Success(fmt.Sprintf("Removed %s from team.", remove))
		return nil
	}

	// --role <email> <role>
	if len(roleArgs) >= 2 {
		email := roleArgs[0]
		role := roleArgs[1]
		if err := auth.SetTeamMemberRole(cmdCtx(cmd), af.AccessToken, email, role); err != nil {
			return handleAuthError(err)
		}
		ui.Success(fmt.Sprintf("Role for %s set to %s.", email, role))
		return nil
	}

	// Default: list members.
	members, err := auth.GetTeamMembers(cmdCtx(cmd), af.AccessToken)
	if err != nil {
		return handleAuthError(err)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(members)
	}

	if len(members) == 0 {
		fmt.Println("No team members. Invite someone with --invite <email>.")
		return nil
	}

	tbl := ui.NewTable("NAME", "EMAIL", "ROLE", "JOINED")
	for _, m := range members {
		joined := m.JoinedAt
		if len(joined) >= 10 {
			joined = joined[:10]
		}
		tbl.AddRow(m.Name, m.Email, m.Role, joined)
	}
	tbl.Render()
	return nil
}

// ── account licenses ─────────────────────────────────────────────────────────

var accountLicensesCmd = &cobra.Command{
	Use:   "licenses",
	Short: "List and manage account licenses",
	Long: `List and manage licenses linked to your account.

With no flags, displays all licenses in a table.

Flags:
  --activate <id>   Activate a license key on this device
  --json            Output raw JSON`,
	SilenceUsage: true,
	RunE:         runAccountLicenses,
}

func init() {
	accountLicensesCmd.Flags().String("activate", "", "License ID to activate on this device")
	accountLicensesCmd.Flags().Bool("json", false, "Output raw JSON")
}

func runAccountLicenses(cmd *cobra.Command, _ []string) error {
	activate, _ := cmd.Flags().GetString("activate")
	jsonOut, _ := cmd.Flags().GetBool("json")

	af, err := requireLogin()
	if err != nil {
		return err
	}

	// --activate
	if activate != "" {
		if !promptConfirm(fmt.Sprintf("Activate license %s on this device? [y/N]: ", activate)) {
			fmt.Println("Canceled.")
			return nil
		}
		if err := auth.ActivateLicense(cmdCtx(cmd), af.AccessToken, activate); err != nil {
			return handleAuthError(err)
		}
		ui.Success(fmt.Sprintf("License %s activated.", activate))
		return nil
	}

	// Default: list licenses.
	licenses, err := auth.GetLicenses(cmdCtx(cmd), af.AccessToken)
	if err != nil {
		return handleAuthError(err)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(licenses)
	}

	if len(licenses) == 0 {
		fmt.Println("No licenses linked to your account.")
		fmt.Printf("Activate a license at https://nself.org/account/licenses or via 'nself license set <key>'.\n")
		return nil
	}

	tbl := ui.NewTable("KEY PREFIX", "TIER", "STATUS", "EXPIRES")
	for _, lic := range licenses {
		prefix := lic.ID
		if len(prefix) > 12 {
			prefix = prefix[:12] + "..."
		}
		status := lic.Tier
		if !lic.IsActive {
			status = lic.Tier + " (inactive)"
		}
		expires := lic.ExpiresAt
		if expires == "" {
			expires = "never"
		}
		tbl.AddRow(prefix, lic.Tier, status, expires)
	}
	tbl.Render()
	return nil
}

// ── account devices ───────────────────────────────────────────────────────────

var accountDevicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List and revoke registered devices",
	Long: `List all devices registered to your account.

Flags:
  --revoke <device-id>   Revoke a device session
  --force                Required to revoke the current device
  --json                 Output raw JSON`,
	SilenceUsage: true,
	RunE:         runAccountDevices,
}

func init() {
	accountDevicesCmd.Flags().String("revoke", "", "Device ID to revoke")
	accountDevicesCmd.Flags().Bool("force", false, "Required to revoke the current device")
	accountDevicesCmd.Flags().Bool("json", false, "Output raw JSON")
}

func runAccountDevices(cmd *cobra.Command, _ []string) error {
	revoke, _ := cmd.Flags().GetString("revoke")
	force, _ := cmd.Flags().GetBool("force")
	jsonOut, _ := cmd.Flags().GetBool("json")

	af, err := requireLogin()
	if err != nil {
		return err
	}

	// --revoke
	if revoke != "" {
		// Check if this is the current device without --force.
		devices, listErr := auth.GetDevices(cmdCtx(cmd), af.AccessToken)
		if listErr == nil {
			for _, d := range devices {
				if d.ID == revoke && d.IsCurrent && !force {
					return fmt.Errorf("cannot revoke the current device without --force")
				}
			}
		}

		if !promptConfirm(fmt.Sprintf("Revoke device %s? [y/N]: ", revoke)) {
			fmt.Println("Canceled.")
			return nil
		}
		if err := auth.RevokeDevice(cmdCtx(cmd), af.AccessToken, revoke); err != nil {
			return handleAuthError(err)
		}
		ui.Success(fmt.Sprintf("Device %s revoked.", revoke))
		return nil
	}

	// Default: list devices.
	devices, err := auth.GetDevices(cmdCtx(cmd), af.AccessToken)
	if err != nil {
		return handleAuthError(err)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(devices)
	}

	if len(devices) == 0 {
		fmt.Println("No registered devices found.")
		return nil
	}

	tbl := ui.NewTable("DEVICE ID", "NAME", "OS", "LAST ACTIVE")
	for _, d := range devices {
		lastActive := d.LastActive
		if len(lastActive) >= 10 {
			lastActive = lastActive[:10]
		}
		name := d.Name
		if d.IsCurrent {
			name = name + " (current)"
		}
		idShort := d.ID
		if len(idShort) > 8 {
			idShort = idShort[:8]
		}
		tbl.AddRow(idShort, name, d.OS, lastActive)
	}
	tbl.Render()
	return nil
}

// ── account transfer ─────────────────────────────────────────────────────────

var accountTransferCmd = &cobra.Command{
	Use:   "transfer <key-id> <email>",
	Short: "Transfer a license to another account",
	Long: `Transfer a license from your account to another email address.

The recipient will receive an email to accept the transfer.`,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(2),
	RunE:         runAccountTransfer,
}

func runAccountTransfer(cmd *cobra.Command, args []string) error {
	keyID := args[0]
	toEmail := args[1]

	af, err := requireLogin()
	if err != nil {
		return err
	}

	prefix := keyID
	if len(prefix) > 12 {
		prefix = prefix[:12] + "..."
	}
	msg := fmt.Sprintf("Transfer license %s to %s? This removes it from your account. [y/N]: ", prefix, toEmail)
	if !promptConfirm(msg) {
		fmt.Println("Canceled.")
		return nil
	}

	if err := auth.TransferLicense(cmdCtx(cmd), af.AccessToken, keyID, toEmail); err != nil {
		return handleAuthError(err)
	}

	ui.Success(fmt.Sprintf("License transferred. %s will receive an email to accept.", toEmail))
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// cmdCtx returns cmd.Context() or context.Background() if the command was
// invoked outside the normal cobra Execute() path (e.g. unit tests).
func cmdCtx(cmd *cobra.Command) context.Context {
	if c := cmd.Context(); c != nil {
		return c
	}
	return context.Background()
}

// requireLogin reads the auth file and returns it, or prints a friendly error.
func requireLogin() (*auth.AuthFile, error) {
	af, err := auth.ReadAuthFile()
	if err != nil {
		if err == auth.ErrNotLoggedIn {
			return nil, fmt.Errorf("not logged in — run: nself account login")
		}
		return nil, fmt.Errorf("reading credentials: %w", err)
	}
	return af, nil
}

// handleAuthError converts auth API errors to user-friendly messages.
func handleAuthError(err error) error {
	if err == nil {
		return nil
	}
	if apiErr, ok := err.(*auth.AuthAPIError); ok {
		switch apiErr.Status {
		case 401, 403:
			return fmt.Errorf("permission denied — your tier does not include this feature")
		case 429:
			return fmt.Errorf("rate limited — try again in a moment")
		default:
			if apiErr.Status >= 500 {
				return fmt.Errorf("nself.org returned an error — try again later")
			}
		}
	}
	msg := err.Error()
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host") {
		return fmt.Errorf("cannot reach nself.org — check your connection")
	}
	return err
}

// promptConfirm prints msg and reads y/Y from stdin. Returns false on any other input.
func promptConfirm(msg string) bool {
	fmt.Print(msg)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
