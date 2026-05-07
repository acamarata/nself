package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nself-org/cli/internal/plugin"
	"github.com/spf13/cobra"
)

// hetznerMonthlyPrices maps common Hetzner server type names to monthly cost in USD.
// Prices are approximate and converted from EUR at ~1.09 USD/EUR (as of 2026).
// Source: hetzner.com/cloud — CX-series (AMD shared) pricing.
var hetznerMonthlyPrices = map[string]float64{
	// Shared vCPU (x86 Intel/AMD)
	"cx11": 4.15,
	"cx21": 6.54,
	"cx22": 5.45,
	"cx31": 11.43,
	"cx32": 8.72,
	"cx41": 19.62,
	"cx42": 15.27,
	"cx51": 36.40,
	"cx52": 27.27,
	// Shared vCPU (Arm64)
	"cax11": 4.15,
	"cax21": 7.09,
	"cax31": 13.09,
	"cax41": 25.20,
	// Dedicated vCPU (x86)
	"ccx13": 11.43,
	"ccx23": 20.73,
	"ccx33": 41.45,
	"ccx43": 82.91,
	"ccx53": 152.73,
	// nSelf reference deployments (canonical from PPI)
	"cx23": 5.83,
}

// bundleMonthlyCosts maps bundle identifiers to their monthly price.
var bundleMonthlyCosts = map[string]float64{
	"claw":   0.99,
	"chat":   0.99,
	"media":  0.99,
	"tv":     0.99,
	"ntv":    0.99,
	"family": 0.99,
	"clawde": 0.99,
	"sentry": 0.99,
}

var costsFormat string

var costsCmd = &cobra.Command{
	Use:   "costs",
	Short: "Show estimated per-install operational costs",
	Long: `Show an itemized breakdown of estimated monthly costs for your nself install.

Includes:
  - Hetzner VPS cost (based on server type detected from env or flag)
  - Stripe transaction fee structure
  - Plugin license costs for installed paid plugins
  - Operational misc (Cloudflare, Vercel, etc.)

Examples:
  nself costs                          # auto-detect server type from env
  nself costs --server-type cx23       # override server type
  nself costs --format json`,
	RunE: runCosts,
}

var costsServerType string

func init() {
	costsCmd.Flags().StringVar(&costsServerType, "server-type", "", "Hetzner server type (e.g. cx23, cax11). Auto-detected from HETZNER_SERVER_TYPE env var if unset.")
	costsCmd.Flags().StringVarP(&costsFormat, "format", "f", "table", "Output format: table|json")
	RootCmd.AddCommand(costsCmd)
}

type costLine struct {
	Category string
	Item     string
	Monthly  float64
	Notes    string
}

func runCosts(cmd *cobra.Command, _ []string) error {
	var lines []costLine
	var total float64

	// ── 1. Hetzner VPS ──────────────────────────────────────────────
	serverType := costsServerType
	if serverType == "" {
		serverType = os.Getenv("HETZNER_SERVER_TYPE")
	}
	if serverType == "" {
		serverType = "cx23" // nSelf reference default
	}
	vpsMonthly, ok := hetznerMonthlyPrices[serverType]
	if !ok {
		vpsMonthly = 0
		lines = append(lines, costLine{
			Category: "Infrastructure",
			Item:     fmt.Sprintf("Hetzner VPS (%s)", serverType),
			Monthly:  0,
			Notes:    "unknown type — check hetzner.com/cloud",
		})
	} else {
		lines = append(lines, costLine{
			Category: "Infrastructure",
			Item:     fmt.Sprintf("Hetzner VPS (%s)", serverType),
			Monthly:  vpsMonthly,
			Notes:    "shared AMD vCPU + 20 GB SSD + 20 TB traffic",
		})
		total += vpsMonthly
	}

	// ── 2. Cloudflare (free tier for most installs) ──────────────────
	lines = append(lines, costLine{
		Category: "DNS / CDN",
		Item:     "Cloudflare",
		Monthly:  0,
		Notes:    "free tier (DNS + DDoS + basic CDN)",
	})

	// ── 3. Vercel (free tier for hobby/self-hosted) ──────────────────
	lines = append(lines, costLine{
		Category: "Frontend hosting",
		Item:     "Vercel",
		Monthly:  0,
		Notes:    "free hobby tier; nself.org uses Pro ($20/mo — not included here)",
	})

	// ── 4. Stripe fees ───────────────────────────────────────────────
	lines = append(lines, costLine{
		Category: "Payment processing",
		Item:     "Stripe (per transaction)",
		Monthly:  0,
		Notes:    "2.9% + $0.30 per card txn; 0.5% + $0.15 ACH; no monthly fee",
	})

	// Example: 100 transactions at $0.99 = ~$3.20 in fees
	lines = append(lines, costLine{
		Category: "Payment processing",
		Item:     "Stripe — example (100 × $0.99 txns)",
		Monthly:  100*0.029*0.99 + 100*0.30,
		Notes:    "example only — actual depends on volume",
	})
	// Don't add example to total (it's illustrative)

	// ── 5. Installed paid plugin licenses ────────────────────────────
	pluginDir := resolvePluginDir()
	installed, err := plugin.ListInstalled(pluginDir)
	if err == nil && len(installed) > 0 {
		seenBundles := map[string]bool{}
		for _, p := range installed {
			if p.Tier == "free" || p.Tier == "" {
				continue
			}
			// Map plugin tier to bundle cost key — tier matches bundle name.
			tierKey := p.Tier
			if _, known := bundleMonthlyCosts[tierKey]; !known {
				// Treat unknown paid tiers as $0.99 (standard bundle price).
				bundleMonthlyCosts[tierKey] = 0.99
			}
			if seenBundles[tierKey] {
				continue // only count each bundle once
			}
			seenBundles[tierKey] = true
			cost := bundleMonthlyCosts[tierKey]
			lines = append(lines, costLine{
				Category: "Plugin licenses",
				Item:     fmt.Sprintf("nself plugin bundle: %s", tierKey),
				Monthly:  cost,
				Notes:    "via nself license set — paid to nself.org",
			})
			total += cost
		}
	} else if err != nil {
		lines = append(lines, costLine{
			Category: "Plugin licenses",
			Item:     "Installed plugins",
			Monthly:  0,
			Notes:    fmt.Sprintf("could not read plugin dir (%v)", err),
		})
	}

	// ── 6. Total ─────────────────────────────────────────────────────
	lines = append(lines, costLine{
		Category: "TOTAL",
		Item:     "(base: VPS + paid bundles)",
		Monthly:  total,
		Notes:    "excludes Stripe fees (volume-dependent)",
	})

	// ── Output ───────────────────────────────────────────────────────
	switch costsFormat {
	case "json":
		return printCostsJSON(cmd, lines)
	default:
		return printCostsTable(cmd, lines)
	}
}

func printCostsTable(cmd *cobra.Command, lines []costLine) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CATEGORY\tITEM\tMONTHLY (USD)\tNOTES")
	fmt.Fprintln(w, "--------\t----\t-------------\t-----")
	for _, l := range lines {
		cost := "-"
		if l.Monthly > 0 {
			cost = fmt.Sprintf("$%.2f", l.Monthly)
		} else if l.Category == "TOTAL" {
			cost = fmt.Sprintf("$%.2f", l.Monthly)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", l.Category, l.Item, cost, l.Notes)
	}
	return w.Flush()
}

func printCostsJSON(cmd *cobra.Command, lines []costLine) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "[")
	for i, l := range lines {
		comma := ","
		if i == len(lines)-1 {
			comma = ""
		}
		fmt.Fprintf(out, "  {\"category\":%q,\"item\":%q,\"monthly_usd\":%.2f,\"notes\":%q}%s\n",
			l.Category, l.Item, l.Monthly, l.Notes, comma)
	}
	fmt.Fprintln(out, "]")
	return nil
}
