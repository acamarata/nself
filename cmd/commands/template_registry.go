package commands

// Purpose: Talks to the nself.org template registry API — resolving the
// registry base URL (env override or default), fetching a filtered
// template list, fetching a single template's manifest by slug, and
// rendering a template list as a terminal table. Split out of
// template.go (CLI-R12) to separate the HTTP/rendering primitives from the
// cobra command wiring (template.go) and the list/info/publish/update
// command handlers (template_list_info.go, template_publish_update.go)
// that call them.
// Inputs: a context.Context, the registry base URL, optional url.Values
// filters, and (for single-fetch) a template slug.
// Outputs: []templateEntry / *templateEntry results, or a printed table.
// Constraints: pure move — no behavior changes.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/nself-org/cli/internal/ui"
)

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
