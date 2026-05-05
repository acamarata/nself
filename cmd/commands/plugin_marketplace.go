package commands

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

	"github.com/spf13/cobra"
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
	defer resp.Body.Close()

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

// --- parent command ---

var pluginMarketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Browse the nSelf plugin marketplace",
	Long: `Browse, search, and view details for plugins on the nSelf marketplace.

  nself plugin marketplace list
  nself plugin marketplace list --tier free
  nself plugin marketplace search ai
  nself plugin marketplace info claw`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// --- list subcommand ---

var pluginMarketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all plugins in the marketplace",
	Long: `Fetch and display all plugins available on the nSelf marketplace.

  nself plugin marketplace list
  nself plugin marketplace list --tier free
  nself plugin marketplace list --bundle nclaw
  nself plugin marketplace list --category ai
  nself plugin marketplace list --json`,
	RunE: runPluginMarketplaceList,
}

// --- search subcommand ---

var pluginMarketplaceSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search marketplace plugins by keyword",
	Long: `Search the nSelf marketplace by name, description, or tags.

  nself plugin marketplace search ai
  nself plugin marketplace search "media streaming" --tier pro
  nself plugin marketplace search claw --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginMarketplaceSearch,
}

// --- info subcommand ---

var pluginMarketplaceInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed marketplace information for a plugin",
	Long: `Display detailed marketplace information for a single plugin.

  nself plugin marketplace info ai
  nself plugin marketplace info claw --json
  nself plugin marketplace info voice --open`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginMarketplaceInfo,
}

func init() {
	// Flags on list.
	pluginMarketplaceListCmd.Flags().String("tier", "", "Filter by tier: free or pro")
	pluginMarketplaceListCmd.Flags().String("bundle", "", "Filter by bundle slug")
	pluginMarketplaceListCmd.Flags().String("category", "", "Filter by category name")
	pluginMarketplaceListCmd.Flags().Bool("json", false, "Output raw JSON")

	// Flags on search.
	pluginMarketplaceSearchCmd.Flags().String("tier", "", "Filter by tier: free or pro")
	pluginMarketplaceSearchCmd.Flags().String("bundle", "", "Filter by bundle slug")
	pluginMarketplaceSearchCmd.Flags().String("category", "", "Filter by category name")
	pluginMarketplaceSearchCmd.Flags().Bool("json", false, "Output raw JSON")

	// Flags on info.
	pluginMarketplaceInfoCmd.Flags().Bool("json", false, "Output raw JSON")
	pluginMarketplaceInfoCmd.Flags().Bool("open", false, "Open the plugin page in a browser")

	// Register subcommands under marketplace.
	pluginMarketplaceCmd.AddCommand(pluginMarketplaceListCmd)
	pluginMarketplaceCmd.AddCommand(pluginMarketplaceSearchCmd)
	pluginMarketplaceCmd.AddCommand(pluginMarketplaceInfoCmd)

	// Register marketplace under plugin.
	pluginCmd.AddCommand(pluginMarketplaceCmd)
}

// --- run functions ---

func runPluginMarketplaceList(cmd *cobra.Command, args []string) error {
	tier, _ := cmd.Flags().GetString("tier")
	bundle, _ := cmd.Flags().GetString("bundle")
	category, _ := cmd.Flags().GetString("category")
	asJSON, _ := cmd.Flags().GetBool("json")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseURL := resolveMarketplaceURL()
	params := buildFilterParams(tier, bundle, category, "")

	mr, err := fetchMarketplace(ctx, baseURL, params)
	if err != nil {
		return fmt.Errorf("listing marketplace plugins: %w", err)
	}

	plugins := applyClientFilters(mr.Plugins, tier, bundle, category)

	if len(plugins) == 0 {
		fmt.Fprintln(os.Stderr, "No plugins found matching the given filters.")
		return nil
	}

	if asJSON {
		return ui.PrintJSON(plugins)
	}

	renderMarketplaceTable(plugins)
	fmt.Fprintf(os.Stderr, "\n%d plugin(s) listed.\n", len(plugins))
	return nil
}

func runPluginMarketplaceSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	tier, _ := cmd.Flags().GetString("tier")
	bundle, _ := cmd.Flags().GetString("bundle")
	category, _ := cmd.Flags().GetString("category")
	asJSON, _ := cmd.Flags().GetBool("json")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseURL := resolveMarketplaceURL()
	params := buildFilterParams(tier, bundle, category, query)

	mr, err := fetchMarketplace(ctx, baseURL, params)
	if err != nil {
		return fmt.Errorf("searching marketplace: %w", err)
	}

	plugins := applyClientFilters(mr.Plugins, tier, bundle, category)

	if len(plugins) == 0 {
		fmt.Fprintf(os.Stderr, "No plugins found matching %q.\n", query)
		return nil
	}

	if asJSON {
		return ui.PrintJSON(plugins)
	}

	renderMarketplaceTable(plugins)
	fmt.Fprintf(os.Stderr, "\n%d plugin(s) matched %q.\n", len(plugins), query)
	return nil
}

func runPluginMarketplaceInfo(cmd *cobra.Command, args []string) error {
	name := args[0]
	asJSON, _ := cmd.Flags().GetBool("json")
	openBrowser, _ := cmd.Flags().GetBool("open")

	baseURL := resolveMarketplaceURL()
	pluginPageURL := baseURL + "/" + name

	if openBrowser {
		return openURL(pluginPageURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mr, err := fetchMarketplace(ctx, baseURL, url.Values{})
	if err != nil {
		return fmt.Errorf("fetching marketplace for plugin info: %w", err)
	}

	var found *marketplacePlugin
	for i := range mr.Plugins {
		if strings.EqualFold(mr.Plugins[i].Name, name) {
			found = &mr.Plugins[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("plugin %q not found in marketplace", name)
	}

	if asJSON {
		return ui.PrintJSON(found)
	}

	tbl := ui.NewTable("Field", "Value")
	tbl.AddRow("Name", found.Name)
	tbl.AddRow("Display Name", found.DisplayName)
	tbl.AddRow("Version", found.Version)
	tbl.AddRow("Description", found.Description)
	tbl.AddRow("Category", found.Category)
	tbl.AddRow("Tier", found.Tier)
	tbl.AddRow("Bundle", found.Bundle)
	if found.Author != "" {
		tbl.AddRow("Author", found.Author)
	}
	tbl.AddRow("Price", found.Price)
	if found.Rating > 0 {
		tbl.AddRow("Rating", fmt.Sprintf("%.1f (%d reviews)", found.Rating, found.ReviewCount))
	}
	if found.Downloads > 0 {
		tbl.AddRow("Downloads", fmt.Sprintf("%d", found.Downloads))
	}
	if len(found.Tags) > 0 {
		tbl.AddRow("Tags", strings.Join(found.Tags, ", "))
	}
	if len(found.Related) > 0 {
		tbl.AddRow("Related", strings.Join(found.Related, ", "))
	}
	if found.Homepage != "" {
		tbl.AddRow("Homepage", found.Homepage)
	}
	if found.Repository != "" {
		tbl.AddRow("Repository", found.Repository)
	}
	tbl.Render()

	fmt.Printf("\nMarketplace: %s\n", pluginPageURL)
	fmt.Printf("Install:     nself plugin install %s\n", found.Name)
	return nil
}
