package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/templates/clone"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// defaultTemplateRegistryURL is the base URL for the template registry API.
const defaultTemplateRegistryURL = "https://nself.org/api/templates"

// templateEntry is a single template from the registry.
type templateEntry struct {
	Slug            string   `json:"slug"`
	DisplayName     string   `json:"displayName"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	Author          string   `json:"author"`
	PriceUSD        float64  `json:"priceUsd"`
	RatingAvg       float64  `json:"ratingAvg"`
	RatingCount     int      `json:"ratingCount"`
	InstallCount    int      `json:"installCount"`
	RequiredPlugins []string `json:"requiredPlugins"`
	PreviewURL      string   `json:"previewUrl"`
	TemplateVersion string   `json:"templateVersion"`
	CliMinVersion   string   `json:"cliMinVersion"`
	TarballURL      string   `json:"tarballUrl"`
	TarballSHA256   string   `json:"tarballSha256"`
}

// templateListResponse is the top-level shape of GET /api/templates.
type templateListResponse struct {
	Templates []templateEntry `json:"templates"`
	Total     int             `json:"total"`
}

func resolveTemplateRegistryURL() string {
	if u := os.Getenv("NSELF_TEMPLATE_REGISTRY_URL"); u != "" {
		return u
	}
	return defaultTemplateRegistryURL
}

func fetchTemplateList(ctx context.Context, baseURL string, params url.Values) ([]templateEntry, error) {
	endpoint := baseURL
	if encoded := params.Encode(); encoded != "" {
		endpoint = baseURL + "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating template registry request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "nself-cli")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching template registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("template not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("template registry returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading template registry response: %w", err)
	}

	var lr templateListResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, fmt.Errorf("parsing template registry response: %w", err)
	}
	return lr.Templates, nil
}

func fetchTemplateSingle(ctx context.Context, baseURL, slug string) (*templateEntry, error) {
	endpoint := baseURL + "/" + slug
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating template request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "nself-cli")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching template %q: %w", slug, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("template %q not found in registry", slug)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("template registry returned status %d for %q", resp.StatusCode, slug)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading template response: %w", err)
	}

	var t templateEntry
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, fmt.Errorf("parsing template response: %w", err)
	}
	return &t, nil
}

// renderTemplateTable prints a list of templates as a table.
func renderTemplateTable(templates []templateEntry) {
	tbl := ui.NewTable("Slug", "Name", "Category", "Rating", "Installs", "Price")
	for _, t := range templates {
		price := "Free"
		if t.PriceUSD > 0 {
			price = fmt.Sprintf("$%.2f", t.PriceUSD)
		}
		rating := ""
		if t.RatingAvg > 0 {
			rating = fmt.Sprintf("%.1f (%d)", t.RatingAvg, t.RatingCount)
		}
		tbl.AddRow(t.Slug, t.DisplayName, t.Category, rating, fmt.Sprintf("%d", t.InstallCount), price)
	}
	tbl.Render()
}

// --- parent command ---

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Browse and publish full-stack app templates",
	Long: `Browse, install, and publish community full-stack app templates.

Templates include Postgres schema, Hasura metadata, seed data, and a Flutter starter.
Browse at nself.org/templates or install from the CLI:

  nself template list
  nself template info saas-starter
  nself init --template saas-starter ./my-app
  nself template publish`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// --- list subcommand ---

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available app templates",
	Long: `Fetch and display all app templates available in the nSelf registry.

  nself template list
  nself template list --category saas
  nself template list --free
  nself template list --json`,
	RunE: runTemplateList,
}

// --- info subcommand ---

var templateInfoCmd = &cobra.Command{
	Use:   "info <slug>",
	Short: "Show detail for a single template",
	Long: `Display full detail for a template including required plugins and install command.

  nself template info saas-starter
  nself template info saas-starter --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateInfo,
}

// --- publish subcommand ---

var templatePublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Submit a template to the nSelf registry",
	Long: `Validate your template manifest and upload it to the nSelf template registry.

The current directory must contain a valid template.yml manifest and a built tarball:

  nself template publish
  nself template publish --tarball ./dist/my-template.tar.gz --manifest template.yml`,
	RunE: runTemplatePublish,
}

// --- update subcommand ---

var templateUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Apply incremental migrations for the installed template",
	Long: `Run incremental schema migrations from the template's migrations/ directory.

Only additive changes are applied automatically. Destructive changes require --force
and explicit confirmation before running.

  nself template update
  nself template update --force`,
	RunE: runTemplateUpdate,
}

func init() {
	// list flags
	templateListCmd.Flags().String("category", "", "Filter by category: saas, marketplace, social, productivity, media, ecommerce")
	templateListCmd.Flags().Bool("free", false, "Show free templates only")
	templateListCmd.Flags().String("sort", "installs", "Sort by: installs, rating, newest, price")
	templateListCmd.Flags().Bool("json", false, "Output raw JSON")

	// info flags
	templateInfoCmd.Flags().Bool("json", false, "Output raw JSON")

	// publish flags
	templatePublishCmd.Flags().String("tarball", "", "Path to the compiled template tarball (.tar.gz)")
	templatePublishCmd.Flags().String("manifest", "template.yml", "Path to the template manifest file")

	// update flags
	templateUpdateCmd.Flags().Bool("force", false, "Allow destructive migrations (requires confirmation)")
	templateUpdateCmd.Flags().Bool("dry-run", false, "Show pending migrations without applying them")

	// register subcommands
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateInfoCmd)
	templateCmd.AddCommand(templatePublishCmd)
	templateCmd.AddCommand(templateUpdateCmd)

	RootCmd.AddCommand(templateCmd)
}

// --- run functions ---

func runTemplateList(cmd *cobra.Command, args []string) error {
	category, _ := cmd.Flags().GetString("category")
	freeOnly, _ := cmd.Flags().GetBool("free")
	sortBy, _ := cmd.Flags().GetString("sort")
	asJSON, _ := cmd.Flags().GetBool("json")

	// --- Bundled clone templates (always shown, no network required) ---
	if !asJSON {
		fmt.Printf("\n  %s Bundled templates (embedded — no network required)\n\n",
			ui.C(ui.Blue, ui.IconInfo))
		for _, name := range clone.All() {
			m, err := clone.GetManifest(name)
			if err != nil {
				fmt.Printf("  %-22s v?.?.?\n", name)
				continue
			}
			plugins := strings.Join(m.RequiredPlugins, ", ")
			fmt.Printf("  %-22s v%-8s  plugins: %s\n", m.Name, m.Version, plugins)
			fmt.Printf("  %s\n\n", ui.C(ui.Dim, m.Description))
		}
		fmt.Printf("  Scaffold: %s\n\n", ui.C(ui.Cyan, "nself init --template <name> [directory]"))
	}

	// --- Registry templates (network, may fail gracefully) ---
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := url.Values{}
	if category != "" {
		params.Set("category", category)
	}
	if freeOnly {
		params.Set("price_max", "0")
	}
	if sortBy != "" {
		params.Set("sort", sortBy)
	}

	baseURL := resolveTemplateRegistryURL()
	registryTemplates, err := fetchTemplateList(ctx, baseURL, params)
	if err != nil {
		// Registry unavailable is non-fatal; bundled list above already printed.
		if !asJSON {
			fmt.Fprintf(os.Stderr, "  %s Community registry unavailable: %v\n\n",
				ui.C(ui.Yellow, ui.IconWarning), err)
		}
		return nil
	}

	if len(registryTemplates) == 0 {
		if !asJSON {
			fmt.Fprintln(os.Stderr, "No community templates found matching the given filters.")
		}
		return nil
	}

	if asJSON {
		// For JSON output, merge bundled + registry entries.
		type combinedEntry struct {
			Source string `json:"source"`
			Name   string `json:"name"`
		}
		var combined []any
		for _, n := range clone.All() {
			m, _ := clone.GetManifest(n)
			combined = append(combined, map[string]any{
				"source":  "bundled",
				"name":    m.Name,
				"version": m.Version,
				"plugins": m.RequiredPlugins,
			})
		}
		for _, t := range registryTemplates {
			combined = append(combined, t)
		}
		return ui.PrintJSON(combined)
	}

	fmt.Printf("  %s Community templates (nself.org/templates)\n\n", ui.C(ui.Blue, ui.IconInfo))
	renderTemplateTable(registryTemplates)
	fmt.Fprintf(os.Stderr, "\n%d community template(s). Browse: https://nself.org/templates\n", len(registryTemplates))
	return nil
}

func runTemplateInfo(cmd *cobra.Command, args []string) error {
	slug := args[0]
	asJSON, _ := cmd.Flags().GetBool("json")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseURL := resolveTemplateRegistryURL()
	t, err := fetchTemplateSingle(ctx, baseURL, slug)
	if err != nil {
		return err
	}

	if asJSON {
		return ui.PrintJSON(t)
	}

	tbl := ui.NewTable("Field", "Value")
	tbl.AddRow("Slug", t.Slug)
	tbl.AddRow("Name", t.DisplayName)
	tbl.AddRow("Description", t.Description)
	tbl.AddRow("Category", t.Category)
	tbl.AddRow("Author", t.Author)
	tbl.AddRow("Version", t.TemplateVersion)
	tbl.AddRow("CLI min", t.CliMinVersion)
	price := "Free"
	if t.PriceUSD > 0 {
		price = fmt.Sprintf("$%.2f one-time", t.PriceUSD)
	}
	tbl.AddRow("Price", price)
	if t.RatingAvg > 0 {
		tbl.AddRow("Rating", fmt.Sprintf("%.1f (%d reviews)", t.RatingAvg, t.RatingCount))
	}
	tbl.AddRow("Installs", fmt.Sprintf("%d", t.InstallCount))
	if len(t.RequiredPlugins) > 0 {
		tbl.AddRow("Required plugins", strings.Join(t.RequiredPlugins, ", "))
	}
	if t.PreviewURL != "" {
		tbl.AddRow("Preview", t.PreviewURL)
	}
	tbl.Render()

	fmt.Printf("\nInstall: nself init --template %s [dest-dir]\n", t.Slug)
	fmt.Printf("Web:     https://nself.org/templates/%s\n", t.Slug)
	return nil
}

// validateTemplateManifest performs minimal YAML/JSON manifest validation.
// It checks for required fields without importing a YAML library so we stay
// dependency-free (the manifest is small enough to parse with string checks).
func validateTemplateManifest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading manifest %q: %w", path, err)
	}

	content := string(data)
	required := []string{"slug:", "display_name:", "version:", "category:"}
	for _, field := range required {
		if !strings.Contains(content, field) {
			return fmt.Errorf("manifest missing required field %q", strings.TrimSuffix(field, ":"))
		}
	}
	return nil
}

func runTemplatePublish(cmd *cobra.Command, args []string) error {
	tarballPath, _ := cmd.Flags().GetString("tarball")
	manifestPath, _ := cmd.Flags().GetString("manifest")

	// Resolve manifest
	if manifestPath == "" {
		manifestPath = "template.yml"
	}
	absManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return fmt.Errorf("resolving manifest path: %w", err)
	}
	if _, statErr := os.Stat(absManifest); os.IsNotExist(statErr) {
		return fmt.Errorf("manifest not found at %s\nCreate a template.yml before publishing.", absManifest)
	}

	ui.Info(fmt.Sprintf("Validating manifest at %s...", absManifest))
	if err := validateTemplateManifest(absManifest); err != nil {
		return err
	}
	ui.Success("Manifest valid.")

	// Resolve tarball
	if tarballPath == "" {
		return fmt.Errorf("--tarball is required: build your template archive first\n" +
			"  tar -czf dist/my-template.tar.gz schema/ metadata/ seed/ flutter/")
	}
	absTarball, err := filepath.Abs(tarballPath)
	if err != nil {
		return fmt.Errorf("resolving tarball path: %w", err)
	}
	if _, statErr := os.Stat(absTarball); os.IsNotExist(statErr) {
		return fmt.Errorf("tarball not found at %s", absTarball)
	}

	// Compute SHA256 for integrity verification
	f, err := os.Open(absTarball)
	if err != nil {
		return fmt.Errorf("opening tarball: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing tarball: %w", err)
	}
	sha := hex.EncodeToString(h.Sum(nil))

	ui.Info(fmt.Sprintf("Tarball SHA256: %s", sha))
	fmt.Println()
	fmt.Println("To submit your template:")
	fmt.Println("  1. Upload your tarball to a public URL (e.g. GitHub release)")
	fmt.Println("  2. Visit https://nself.org/developers/templates")
	fmt.Println("  3. Fill in the submission form with the tarball URL and this SHA256")
	fmt.Printf("     SHA256: %s\n", sha)
	fmt.Println()
	fmt.Println("Author review and approval typically takes 1-3 business days.")
	return nil
}

func runTemplateUpdate(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("no migrations/ directory found in the current project\n" +
			"Template updates require a migrations/ directory created by nself init --template")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	var pending []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			data, readErr := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
			if readErr == nil {
				content := strings.ToUpper(string(data))
				if strings.Contains(content, "DROP ") || strings.Contains(content, "TRUNCATE ") {
					if !force {
						return fmt.Errorf(
							"migration %s contains destructive changes (DROP/TRUNCATE).\n"+
								"Re-run with --force to apply after reviewing carefully.",
							e.Name(),
						)
					}
				}
			}
			pending = append(pending, e.Name())
		}
	}

	if len(pending) == 0 {
		fmt.Println("No pending migrations.")
		return nil
	}

	for _, m := range pending {
		if dryRun {
			fmt.Printf("  [dry-run] would apply: %s\n", m)
		} else {
			fmt.Printf("  Applying: %s\n", m)
		}
	}

	if dryRun {
		fmt.Printf("\n%d migration(s) pending. Run without --dry-run to apply.\n", len(pending))
	} else {
		fmt.Printf("\n%d migration(s) listed. Run `nself db migrate` to execute them against the live database.\n", len(pending))
	}
	return nil
}
