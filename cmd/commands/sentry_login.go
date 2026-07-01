package commands

// sentry_login.go — 'nself sentry login|logout|whoami' commands.
//
// Purpose: Store/remove the ɳSentry API key (~/.nself/sentry.json, 0600) and
//   show the authenticated account with tier + quota usage.
// Inputs:  --api-key / --api-url flags, NSELF_SENTRY_API_KEY env, stdin prompt.
// Outputs: credentials file; whoami table or --json.
// Constraints: key validated against GET /v1/me before saving; never echo the
//   full key back (mask to prefix + last 4).
// SPORT: CLI-CMD-SENTRY-004

import (
	"bufio"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/sentryapi"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var sentryLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate the CLI with ɳSentry (SaaS or self-hosted)",
	Long: `Store an ɳSentry API key for use by all 'nself sentry' cloud commands.

The key (nsk_*) is created in the ɳSentry dashboard at
https://sentry.nself.org/settings/api-keys, or seeded locally by the
sentry dev preset (see 'nself init --preset sentry').

The key is validated against the API, then stored at ~/.nself/sentry.json (0600).

Examples:
  nself sentry login --api-key nsk_abc123...
  nself sentry login --api-url http://localhost:3848 --api-key nsk_dev_local...
  NSELF_SENTRY_API_KEY=nsk_abc123 nself sentry login`,
	SilenceUsage: true,
	RunE:         runSentryLogin,
}

var sentryLogoutCmd = &cobra.Command{
	Use:          "logout",
	Short:        "Remove the stored ɳSentry API key",
	Long:         `Delete the local ɳSentry credentials file at ~/.nself/sentry.json.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := sentryapi.DeleteCredentials(); err != nil {
			return fmt.Errorf("logout: %w", err)
		}
		ui.Success("Logged out of ɳSentry (credentials removed).")
		return nil
	},
}

var sentryWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the authenticated ɳSentry account, tier, and quota usage",
	Long: `Show the account behind the configured API key: tenant, email, tier,
and per-dimension quota usage (monitors, status pages, error events, ...).

Examples:
  nself sentry whoami
  nself sentry whoami --json`,
	SilenceUsage: true,
	RunE:         runSentryWhoami,
}

func init() {
	addSentryCloudFlags(sentryLoginCmd)
	addSentryCloudFlags(sentryWhoamiCmd)
	sentryCmd.AddCommand(sentryLoginCmd, sentryLogoutCmd, sentryWhoamiCmd)
}

// runSentryLogin resolves the key (flag → env → prompt), validates it via
// GET /v1/me, and persists it.
func runSentryLogin(cmd *cobra.Command, _ []string) error {
	flagURL, _ := cmd.Flags().GetString("api-url")
	flagKey, _ := cmd.Flags().GetString("api-key")
	apiURL, apiKey := sentryapi.Resolve(flagURL, flagKey)

	if apiKey == "" {
		fmt.Fprint(cmd.OutOrStdout(), "Paste your ɳSentry API key (nsk_...): ")
		reader := bufio.NewReader(cmd.InOrStdin())
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return fmt.Errorf("reading API key: %w", err)
		}
		apiKey = strings.TrimSpace(line)
	}
	if err := sentryapi.ValidateKeyFormat(apiKey); err != nil {
		return err
	}

	client := sentryapi.New(apiURL, apiKey)
	acct, err := client.WhoAmI(cmd.Context())
	if err != nil {
		if errors.Is(err, sentryapi.ErrUnauthorized) {
			return fmt.Errorf("API key rejected by %s — check the key and try again", apiURL)
		}
		return fmt.Errorf("validating key against %s: %w", apiURL, err)
	}

	// Persist api_url only when it differs from the default, so a later
	// default change is picked up automatically.
	creds := &sentryapi.Credentials{APIKey: apiKey}
	if apiURL != sentryapi.DefaultAPIURL {
		creds.APIURL = apiURL
	}
	if err := sentryapi.WriteCredentials(creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	ui.Success(fmt.Sprintf("Logged in to ɳSentry as %s (tenant %s, tier %s) — key %s",
		acct.Email, acct.TenantID, dashIfEmpty(acct.Tier), maskSentryKey(apiKey)))
	return nil
}

// runSentryWhoami prints the account + quota table (or --json).
func runSentryWhoami(cmd *cobra.Command, _ []string) error {
	client := sentryClientFromCmd(cmd)
	acct, err := client.WhoAmI(cmd.Context())
	if err != nil {
		return err
	}

	if sentryJSONFlag(cmd) {
		return printSentryJSON(cmd, acct)
	}

	ui.CommandHeader("ɳSentry Account", acct.Email)
	fmt.Printf("\nTenant: %s\nTier:   %s\nAPI:    %s\n\n", acct.TenantID, dashIfEmpty(acct.Tier), client.BaseURL)

	if len(acct.Quotas) > 0 {
		// Stable ordering for the quota table.
		names := make([]string, 0, len(acct.Quotas))
		for name := range acct.Quotas {
			names = append(names, name)
		}
		sort.Strings(names)

		tbl := ui.NewTable("Quota", "Used", "Limit")
		for _, name := range names {
			q := acct.Quotas[name]
			tbl.AddRow(name, fmt.Sprintf("%d", q.Used), fmt.Sprintf("%d", q.Limit))
		}
		tbl.Render()
		fmt.Println()
	}
	return nil
}

// maskSentryKey shows only the prefix and last 4 characters of an API key.
func maskSentryKey(key string) string {
	if len(key) <= len(sentryapi.KeyPrefix)+4 {
		return sentryapi.KeyPrefix + "****"
	}
	return sentryapi.KeyPrefix + "****" + key[len(key)-4:]
}
