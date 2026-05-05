package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

// licenseMigrateCmd — SP-04.O11 T11: dual-mode license migration.
//
// Without --account-id: legacy check mode (explain what will change, show expiry).
// With --account-id <uuid>: link the legacy key to the given nSelf account.
//
// The server endpoint POST /license/migrate handles the actual DB work.
// This command is safe to run multiple times (server is idempotent).
var licenseMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate legacy license key to a nSelf account",
	Long: `Check or link a legacy nself_pro_ license key to a nSelf account.

The new model uses per-product keys (nself_claw_, nself_chat_, etc.) linked
to a nSelf account, instead of the old tier-based nself_pro_ keys.

Without --account-id: shows migration status for all configured keys.
With --account-id <uuid>: links the legacy key to that nSelf account.

Examples:
  nself license migrate                             # check status
  nself license migrate --account-id <uuid>         # link key to account
  nself license migrate --key <key> --account-id <uuid>  # link a specific key`,
	RunE: runLicenseMigrate,
}

var (
	migrateAccountID string
	migrateKeyFlag   string
)

type migrationInfo struct {
	Valid              bool     `json:"valid"`
	Tier               string   `json:"tier"`
	Product            string   `json:"product"`
	KeyType            string   `json:"key_type"`
	LegacyExpiresAt    string   `json:"legacy_expires_at"`
	GrandfatheredUntil string   `json:"grandfathered_until"`
	PluginsAllowed     []string `json:"plugins_allowed"`
	MigrationStatus    string   `json:"migration_status"`
}

type migrateResponse struct {
	Migrated        bool   `json:"migrated"`
	AlreadyMigrated bool   `json:"already_migrated"`
	AccountID       string `json:"account_id"`
	Tier            string `json:"tier"`
	Error           string `json:"error"`
}

type migrationStatusResponse struct {
	Migrated            bool   `json:"migrated"`
	MigrationStatus     string `json:"migration_status"`
	MigratedToAccountID string `json:"migrated_to_account_id"`
	GraceExpiresAt      string `json:"grace_expires_at"`
	DaysRemaining       int    `json:"days_remaining"`
	CutoverEnforced     bool   `json:"cutover_enforced"`
	Error               string `json:"error"`
}

func runLicenseMigrate(cmd *cobra.Command, args []string) error {
	pingURL := os.Getenv("NSELF_PING_API_URL")
	if pingURL == "" {
		pingURL = defaultPingURL
	}

	// If --account-id provided, perform the actual account-link migration.
	if migrateAccountID != "" {
		return runMigrateLink(cmd.Context(), pingURL)
	}

	// Otherwise, show migration status for all configured keys.
	return runMigrateCheck(cmd.Context(), pingURL)
}

// runMigrateCheck — status report for all legacy keys (no mutation).
func runMigrateCheck(ctx context.Context, pingURL string) error {
	keys := license.CollectLicenseKeys()
	if migrateKeyFlag != "" {
		keys = []string{migrateKeyFlag}
	}

	if len(keys) == 0 {
		fmt.Println("No license keys configured.")
		fmt.Printf("\nGet a product license at %s\n", pricingURL)
		return nil
	}

	hasLegacy := false
	for _, key := range keys {
		pp := license.DetectProduct(key)
		if pp != nil && (pp.Product == "pro" || pp.Product == "enterprise" || pp.Product == "max") {
			hasLegacy = true
			break
		}
	}

	if !hasLegacy {
		fmt.Println("No legacy keys found. All your keys use the new product model.")
		return nil
	}

	fmt.Println("Legacy License Migration Status")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Println()

	tbl := ui.NewTable("Key", "Tier", "Expiry", "Migration Status")

	for _, key := range keys {
		pp := license.DetectProduct(key)
		if pp == nil {
			continue
		}

		masked := license.MaskKey(key)

		if pp.Product != "pro" && pp.Product != "enterprise" && pp.Product != "max" {
			tbl.AddRow(masked, pp.DisplayName, "(new format)", "N/A")
			continue
		}

		// Check migration status from the server.
		status, err := fetchMigrationStatus(ctx, key, pingURL)
		if err != nil {
			tbl.AddRow(masked, pp.DisplayName, "?", fmt.Sprintf("Error: %v", err))
			continue
		}

		if status.Error != "" {
			tbl.AddRow(masked, pp.DisplayName, "?", fmt.Sprintf("Error: %s", status.Error))
			continue
		}

		migStatus := status.MigrationStatus
		if migStatus == "" {
			migStatus = "legacy"
		}

		daysStr := "-"
		if status.DaysRemaining > 0 {
			daysStr = fmt.Sprintf("%d days", status.DaysRemaining)
		}

		tbl.AddRow(masked, pp.DisplayName, daysStr, migStatus)
	}

	tbl.Render()

	fmt.Println()
	fmt.Println("To link a legacy key to your nSelf account, run:")
	fmt.Println("  nself license migrate --account-id <your-account-uuid>")
	fmt.Println()
	fmt.Printf("  View pricing and subscribe: %s\n", pricingURL)
	fmt.Println()

	return nil
}

// runMigrateLink — link the legacy key to a nSelf account via POST /license/migrate.
func runMigrateLink(ctx context.Context, pingURL string) error {
	// Determine which key to migrate.
	var keyToMigrate string

	if migrateKeyFlag != "" {
		keyToMigrate = migrateKeyFlag
	} else {
		keys := license.CollectLicenseKeys()
		for _, k := range keys {
			pp := license.DetectProduct(k)
			if pp != nil && (pp.Product == "pro" || pp.Product == "enterprise" || pp.Product == "max") {
				keyToMigrate = k
				break
			}
		}
		if keyToMigrate == "" && len(keys) > 0 {
			// If no legacy key found, use the first key (might be a legacy format without prefix).
			keyToMigrate = keys[0]
		}
	}

	if keyToMigrate == "" {
		return fmt.Errorf("no license key found to migrate; configure one with: nself license set <key>")
	}

	masked := license.MaskKey(keyToMigrate)
	fmt.Printf("Linking key %s to account %s ...\n", masked, migrateAccountID)

	resp, err := sendMigrateRequest(ctx, keyToMigrate, migrateAccountID, pingURL)
	if err != nil {
		return fmt.Errorf("migration request failed: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.AlreadyMigrated {
		fmt.Printf("Key %s is already linked to account %s (tier: %s).\n",
			masked, resp.AccountID, resp.Tier)
		return nil
	}

	if resp.Migrated {
		fmt.Printf("Migration complete.\n")
		fmt.Printf("  Key:       %s\n", masked)
		fmt.Printf("  Account:   %s\n", resp.AccountID)
		fmt.Printf("  Tier:      %s\n", resp.Tier)
		fmt.Println()
		fmt.Println("Your key is now linked to your nSelf account.")
		fmt.Println("Run 'nself license status' to verify.")
		return nil
	}

	return fmt.Errorf("migration returned unexpected response (migrated=false, no error)")
}

// sendMigrateRequest — POST /license/migrate on the ping_api.
func sendMigrateRequest(ctx context.Context, licenseKey, accountID, pingURL string) (*migrateResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	payload := map[string]string{
		"license_key": licenseKey,
		"account_id":  accountID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", pingURL+"/license/migrate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httptimeout.License.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result migrateResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid server response: %w", err)
	}

	// Surface HTTP-level errors.
	if resp.StatusCode >= 400 && result.Error == "" {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return &result, nil
}

// fetchMigrationStatus — GET /license/migration-status?key=<key>.
func fetchMigrationStatus(ctx context.Context, key string, pingURL string) (*migrationStatusResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/license/migration-status?key=%s", pingURL, key)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httptimeout.License.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result migrationStatusResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid server response: %w", err)
	}

	return &result, nil
}

// fetchMigrationInfo is kept for backward compatibility with any callers.
func fetchMigrationInfo(ctx context.Context, key string, pingURL string) (*migrationInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	bodyStr := fmt.Sprintf(`{"license_key":"%s"}`, key)
	req, err := http.NewRequestWithContext(ctx, "POST", pingURL+"/license/validate", strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httptimeout.License.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info migrationInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func productDisplayName(product string) string {
	names := map[string]string{
		"claw":   "ɳClaw ($0.99/mo)",
		"chat":   "ɳChat ($0.99/mo)",
		"media":  "nTV ($0.99/mo)",
		"family": "nFamily ($0.99/mo)",
		"clawde": "ClawDE ($0.99/mo)",
		"plus":   "ɳSelf+ ($3.99/mo)",
		"owner":  "Owner (all access)",
	}
	if n, ok := names[product]; ok {
		return n
	}
	return product
}

func init() {
	licenseMigrateCmd.Flags().StringVar(&migrateAccountID, "account-id", "", "nSelf account UUID to link the legacy key to")
	licenseMigrateCmd.Flags().StringVar(&migrateKeyFlag, "key", "", "specific license key to migrate (defaults to auto-detected legacy key)")
	licenseCmd.AddCommand(licenseMigrateCmd)
}
