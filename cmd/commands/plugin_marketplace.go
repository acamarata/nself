package commands

// Purpose: Shared marketplace API types and helpers for `nself plugin
// marketplace`, split from the cobra commands that use them (CLI-R12
// Batch B mechanical file-size split — commands now in
// plugin_marketplace_cmds.go). Fetches and filters plugin listings from
// the nSelf marketplace API.
// Inputs: NSELF_MARKETPLACE_URL env override, and tier/bundle/category/
// query filter values from the calling command.
// Outputs: a parsed marketplaceResponse, a filtered plugin slice, or a
// rendered table.
// Constraints: pure move, no behavior change.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ui"
)

// defaultMarketplaceURL is the base endpoint for marketplace API calls.
const defaultMarketplaceURL = "https://plugins.nself.org/marketplace"

// marketplacePlugin represents a single plugin entry from the marketplace API.
type marketplacePlugin struct {
	Name            string   `json:"name"`
	DisplayName     string   `json:"displayName"`
	Version         string   `json:"version"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	Tier            string   `json:"tier"`
	Bundle          string   `json:"bundle"`
	Author          string   `json:"author"`
	Icon            string   `json:"icon"`
	Tags            []string `json:"tags"`
	Downloads       int      `json:"downloads"`
	Rating          float64  `json:"rating"`
	ReviewCount     int      `json:"reviewCount"`
	LicenseRequired bool     `json:"licenseRequired"`
	Price           string   `json:"price"`
	Related         []string `json:"related"`
	Homepage        string   `json:"homepage"`
	Repository      string   `json:"repository"`
}

// marketplaceStats holds the aggregate counts from the marketplace API.
type marketplaceStats struct {
	Total     int    `json:"total"`
	Free      int    `json:"free"`
	Pro       int    `json:"pro"`
	UpdatedAt string `json:"updatedAt"`
}

// marketplaceResponse is the top-level shape of GET /marketplace.
type marketplaceResponse struct {
	Plugins    []marketplacePlugin `json:"plugins"`
	Bundles    []json.RawMessage   `json:"bundles"`
	Categories []json.RawMessage   `json:"categories"`
	Stats      marketplaceStats    `json:"stats"`
}

// resolveMarketplaceURL returns the marketplace base URL, respecting the
// NSELF_MARKETPLACE_URL environment variable override.
func resolveMarketplaceURL() string {
	if u := os.Getenv("NSELF_MARKETPLACE_URL"); u != "" {
		return u
	}
	return defaultMarketplaceURL
}

// fetchMarketplace calls the marketplace API and returns the parsed response.
// query params: tier, bundle, category, q are passed as URL query params.
func fetchMarketplace(ctx context.Context, baseURL string, params url.Values) (*marketplaceResponse, error) {
	endpoint := baseURL
	if encoded := params.Encode(); encoded != "" {
		endpoint = baseURL + "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating marketplace request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "nself-cli")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching marketplace: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marketplace API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading marketplace response: %w", err)
	}

	var mr marketplaceResponse
	if err := json.Unmarshal(body, &mr); err != nil {
		return nil, fmt.Errorf("parsing marketplace response: %w", err)
	}
	return &mr, nil
}

// buildFilterParams constructs url.Values from the common filter flags.
func buildFilterParams(tier, bundle, category, query string) url.Values {
	params := url.Values{}
	if tier != "" {
		params.Set("tier", tier)
	}
	if bundle != "" {
		params.Set("bundle", bundle)
	}
	if category != "" {
		params.Set("category", category)
	}
	if query != "" {
		params.Set("q", query)
	}
	return params
}

// applyClientFilters applies tier/bundle/category filters on the client side
// for cases where the server may not support them (or for defence in depth).
func applyClientFilters(plugins []marketplacePlugin, tier, bundle, category string) []marketplacePlugin {
	if tier == "" && bundle == "" && category == "" {
		return plugins
	}
	filtered := make([]marketplacePlugin, 0, len(plugins))
	for _, p := range plugins {
		if tier != "" && !strings.EqualFold(p.Tier, tier) {
			continue
		}
		if bundle != "" && !strings.EqualFold(p.Bundle, bundle) {
			continue
		}
		if category != "" && !strings.EqualFold(p.Category, category) {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

// renderMarketplaceTable prints a marketplace plugin list as a table.
func renderMarketplaceTable(plugins []marketplacePlugin) {
	tbl := ui.NewTable("Name", "Display Name", "Tier", "Category", "Bundle", "Rating", "Price")
	for _, p := range plugins {
		rating := ""
		if p.Rating > 0 {
			rating = fmt.Sprintf("%.1f (%d)", p.Rating, p.ReviewCount)
		}
		tbl.AddRow(p.Name, p.DisplayName, p.Tier, p.Category, p.Bundle, rating, p.Price)
	}
	tbl.Render()
}
