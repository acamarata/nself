package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

var licenseMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Check and migrate legacy license keys to new product model",
	Long: `Check if your license key is a legacy nself_pro_ key and explain
what it maps to in the new per-product pricing model.

The new model uses product-specific keys (nself_claw_, nself_chat_, etc.)
instead of the old tier-based nself_pro_ keys. Legacy keys continue to work
during the grandfather period (1 year from migration date).

This command:
  1. Checks if you have a legacy nself_pro_ key
  2. Validates it against the server to see its current mapping
  3. Explains what product bundle it maps to
  4. Provides next steps for transitioning to a new key`,
	RunE: runLicenseMigrate,
}

type migrationInfo struct {
	Valid             bool   `json:"valid"`
	Tier              string `json:"tier"`
	Product           string `json:"product"`
	KeyType           string `json:"key_type"`
	LegacyExpiresAt   string `json:"legacy_expires_at"`
	GrandfatheredUntil string `json:"grandfathered_until"`
	PluginsAllowed    []string `json:"plugins_allowed"`
}

func runLicenseMigrate(cmd *cobra.Command, args []string) error {
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

	hasLegacy := false
	hasNew := false

	for _, key := range keys {
		pp := license.DetectProduct(key)
		if pp == nil {
			continue
		}

		if pp.Product == "pro" || pp.Product == "enterprise" || pp.Product == "max" {
			hasLegacy = true
		} else {
			hasNew = true
		}
	}

	if !hasLegacy {
		fmt.Println("No legacy keys found. All your keys use the new product model.")
		if hasNew {
			fmt.Println("\nYour keys are already up to date.")
		}
		return nil
	}

	fmt.Println("Legacy License Migration Check")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Println()

	tbl := ui.NewTable("Key", "Current Tier", "Maps To", "Expires", "Status")

	for _, key := range keys {
		pp := license.DetectProduct(key)
		if pp == nil {
			continue
		}

		masked := license.MaskKey(key)

		// Only process legacy keys
		if pp.Product != "pro" && pp.Product != "enterprise" && pp.Product != "max" {
			tbl.AddRow(masked, pp.DisplayName, "(already new format)", "-", "OK")
			continue
		}

		// Query server for migration info
		info, err := fetchMigrationInfo(cmd.Context(), key, pingURL)
		if err != nil {
			tbl.AddRow(masked, pp.DisplayName, "?", "?", fmt.Sprintf("Error: %v", err))
			continue
		}

		mapsTo := info.Product
		if mapsTo == "" {
			mapsTo = "claw (default)"
		}
		productDisplay := productDisplayName(mapsTo)

		expires := "-"
		if info.LegacyExpiresAt != "" {
			t, err := time.Parse(time.RFC3339, info.LegacyExpiresAt)
			if err == nil {
				daysLeft := int(time.Until(t).Hours() / 24)
				expires = fmt.Sprintf("%s (%d days)", t.Format("Jan 2, 2006"), daysLeft)
			}
		}

		status := "Active"
		if !info.Valid {
			status = "Expired"
		}

		tbl.AddRow(masked, pp.DisplayName, productDisplay, expires, status)
	}

	tbl.Render()

	fmt.Println()
	fmt.Println("What's happening:")
	fmt.Println("  We're moving from tier-based pricing to per-product bundles.")
	fmt.Println("  Your legacy key continues to work during the grandfather period.")
	fmt.Println()
	fmt.Println("After the grandfather period:")
	fmt.Println("  Subscribe to the specific product bundle you need.")
	fmt.Println("  Basic-tier keys default to the claw bundle.")
	fmt.Println("  Pro and higher tiers get full nSelf+ access during the grandfather period.")
	fmt.Println()
	fmt.Printf("  View pricing and subscribe: %s\n", pricingURL)
	fmt.Println()
	fmt.Println("To add a new product key alongside your legacy key:")
	fmt.Println("  nself license add <new-key>")
	fmt.Println()

	return nil
}

func fetchMigrationInfo(ctx context.Context, key string, pingURL string) (*migrationInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	body := fmt.Sprintf(`{"license_key":"%s"}`, key)
	req, err := http.NewRequestWithContext(ctx, "POST", pingURL+"/license/validate", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
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
		"claw":   "\u03B7Claw ($0.99/mo)",
		"chat":   "\u03B7Chat ($0.99/mo)",
		"media":  "nMedia ($0.99/mo)",
		"family": "nFamily ($0.99/mo)",
		"clawde": "ClawDE+ ($0.99/mo)",
		"plus":   "\u03B7Self+ ($4.99/mo)",
		"owner":  "Owner (all access)",
	}
	if n, ok := names[product]; ok {
		return n
	}
	return product
}

func init() {
	licenseCmd.AddCommand(licenseMigrateCmd)
}
