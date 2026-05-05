package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/nself-org/cli/internal/auth"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your nSelf account",
	Long: `Log in to your nSelf account using the device authorization flow.

Opens your browser to authorize this CLI session. The browser shows a
short code — confirm it matches the one printed here, then click Authorize.

After authorization, your session is stored at ~/.nself/auth.json (0600).
Run 'nself account show' to verify your account details.`,
	SilenceUsage: true,
	RunE:         runLogin,
}

func init() {
	loginCmd.Flags().Bool("no-browser", false, "Print the login URL instead of opening a browser")
	loginCmd.Flags().Bool("force", false, "Log in even if already logged in (replaces existing session)")
	RootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, _ []string) error {
	noBrowser, _ := cmd.Flags().GetBool("no-browser")
	force, _ := cmd.Flags().GetBool("force")

	// Guard: already logged in.
	if !force && auth.IsLoggedIn() {
		af, err := auth.ReadAuthFile()
		if err == nil && af != nil {
			fmt.Printf("Already logged in as %s (%s).\n", af.Email, af.Tier)
			fmt.Println("Run 'nself login --force' to re-authenticate.")
			return nil
		}
	}

	// Step 1: request device code.
	ui.Info("Contacting nSelf auth server...")
	dcResp, err := auth.DeviceAuthorize(cmdCtx(cmd))
	if err != nil {
		return fmt.Errorf("login: %w", err)
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
	baseCtx := cmd.Context()
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, 6*time.Minute)
	defer cancel()

	var lastDot time.Time
	result, err := auth.PollDeviceCode(ctx, dcResp.DeviceCode, func(elapsed time.Duration) {
		// Print a progress dot every 10 seconds so the user knows we're alive.
		if time.Since(lastDot) >= 10*time.Second {
			fmt.Print(".")
			lastDot = time.Now()
		}
	})
	fmt.Println() // newline after dots.

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

	// Step 5: success output.
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
