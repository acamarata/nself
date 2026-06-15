package doctor

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// CheckPayPalCSVParity verifies that PayPal environment variables have matching counts
// for PAYPAL_CLIENT_IDS and PAYPAL_CLIENT_SECRETS when multi-account mode is enabled.
// Purpose: Prevent silent mispairing of credentials in multi-account PayPal setups.
// Inputs: Environment variables (PAYPAL_CLIENT_IDS, PAYPAL_CLIENT_SECRETS).
// Outputs: CheckResult with status (ok/warn) and message naming the count mismatch if any.
// Constraints: Runs only when PAYPAL_CLIENT_IDS is set (indicating PayPal is installed).
func CheckPayPalCSVParity(ctx context.Context) CheckResult {
	ids := os.Getenv("PAYPAL_CLIENT_IDS")
	secrets := os.Getenv("PAYPAL_CLIENT_SECRETS")

	// If neither is set, PayPal multi-account is not configured — skip the check.
	if ids == "" && secrets == "" {
		return CheckResult{
			Section: "plugins",
			Name:    "PAYPAL-CSV-PARITY",
			Status:  "ok",
			Message: "PayPal multi-account not configured (both PAYPAL_CLIENT_IDS and PAYPAL_CLIENT_SECRETS empty)",
		}
	}

	idCount := countCSVEntries(ids)
	secretCount := countCSVEntries(secrets)

	// If counts mismatch, warn with the exact counts.
	if idCount != secretCount {
		return CheckResult{
			Section: "plugins",
			Name:    "PAYPAL-CSV-PARITY",
			Status:  "warn",
			Message: fmt.Sprintf(
				"PayPal CSV count mismatch: PAYPAL_CLIENT_IDS has %d entries but PAYPAL_CLIENT_SECRETS has %d entries — counts must match to avoid credential mispairing",
				idCount, secretCount,
			),
			FixCmd: fmt.Sprintf(
				"export PAYPAL_CLIENT_IDS='...' (update to %d entries) and PAYPAL_CLIENT_SECRETS='...' (ensure exactly %d entries match)",
				idCount, idCount,
			),
		}
	}

	// If counts match, all is good.
	return CheckResult{
		Section: "plugins",
		Name:    "PAYPAL-CSV-PARITY",
		Status:  "ok",
		Message: fmt.Sprintf("PayPal CSV parity OK: %d client IDs matched with %d secrets", idCount, secretCount),
	}
}

// countCSVEntries counts the number of non-empty, trimmed entries in a CSV string.
// Purpose: Count entries in comma-separated environment variables.
// Inputs: CSV string (e.g., "a,b,c" or "a, b , c").
// Outputs: Integer count of non-empty trimmed entries.
// Constraints: Matches the same logic as the PayPal plugin's splitCSV function.
func countCSVEntries(s string) int {
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ",")
	count := 0
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			count++
		}
	}
	return count
}
