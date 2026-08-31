package commands

// Purpose: the bundleListRow type and runBundleList, the RunE for "nself
// bundle list". Inputs are the cobra command/args; outputs are a printed
// table or JSON of canonical bundles.
// Constraints: split out of bundle.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/plugin"

	"github.com/spf13/cobra"
)

// bundleListRow is the JSON-serialisable form of one bundle in `nself bundle list --json`.
type bundleListRow struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Price       string   `json:"price"`
	PluginCount int      `json:"plugin_count"`
	Plugins     []string `json:"plugins"`
	Status      string   `json:"status"`
	HasActive   bool     `json:"has_active"`
}

func runBundleList(cmd *cobra.Command, _ []string) error {
	var showInstalled, asJSON bool
	if cmd != nil {
		showInstalled, _ = cmd.Flags().GetBool("installed")
		asJSON, _ = cmd.Flags().GetBool("json")
	}

	// Build the row set, optionally filtering to installed-only.
	var rows []bundleListRow
	pluginDir := resolvePluginDir()
	installedPlugins, _ := plugin.ListInstalled(pluginDir)
	installedSet := make(map[string]bool, len(installedPlugins))
	for _, p := range installedPlugins {
		installedSet[strings.ToLower(p.Name)] = true
	}

	for _, key := range bundleDisplayOrder {
		b, ok := canonicalBundles[key]
		if !ok {
			continue
		}

		status := "active"
		switch key {
		case "ntask":
			status = "free"
		case "nsentry", "nfamily", "clawde":
			status = "planned"
		}

		// Count how many bundle plugins are installed.
		activeCount := 0
		for _, p := range b.Plugins {
			if installedSet[strings.ToLower(p)] {
				activeCount++
			}
		}
		hasActive := activeCount > 0

		if showInstalled && !hasActive {
			continue
		}

		pluginList := b.Plugins
		if pluginList == nil {
			pluginList = []string{}
		}

		rows = append(rows, bundleListRow{
			Slug:        key,
			Name:        b.Name,
			Price:       b.Price,
			PluginCount: len(b.Plugins),
			Plugins:     pluginList,
			Status:      status,
			HasActive:   hasActive,
		})
	}

	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}

	fmt.Println("nSelf Plugin Bundles")
	fmt.Println(strings.Repeat("─", 60))

	for _, r := range rows {
		pluginCount := ""
		if r.PluginCount > 0 {
			pluginCount = fmt.Sprintf("  (%d plugins)", r.PluginCount)
		}

		b := canonicalBundles[r.Slug]
		desc := b.Description
		if desc == "" && len(b.Plugins) > 0 {
			preview := b.Plugins
			if len(preview) > 4 {
				preview = append(preview[:4:4], "...")
			}
			desc = strings.Join(preview, ", ")
		}

		fmt.Printf("  %-14s  %-26s  %s%s\n", r.Slug, r.Name, r.Price, pluginCount)
		if desc != "" {
			fmt.Printf("  %-14s  %s\n", "", desc)
		}
		fmt.Println()
	}

	fmt.Println("Run 'nself bundle info <name>' for full plugin membership.")
	fmt.Println("Buy at: https://nself.org/pricing")
	return nil
}

// ── bundle info ─────────────────────────────────────────────────────
