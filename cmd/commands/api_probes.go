package commands

// Purpose: the version/header probing helpers used by "nself api version"
// and the deprecation scanner (collectAPIVersions, probeHTTPVersion,
// probeHTTPHeader, probeInstalledPluginSDKVersions, probeLocalHasura,
// scanDeprecations, countBreaking). Inputs are an http.Client and a target
// URL/field; outputs are version strings or scan results.
// Constraints: split out of api.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/deprecation"
	"github.com/nself-org/cli/internal/version"
)

// collectAPIVersions probes all reachable API surfaces and returns version rows.
func collectAPIVersions(filterSurface string, timeoutSec int) []apiVersionRow {
	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}

	var rows []apiVersionRow

	if filterSurface == "" || strings.EqualFold(filterSurface, "cli") {
		rows = append(rows, apiVersionRow{
			Surface: "cli",
			Version: version.GetVersion(),
		})
	}

	if filterSurface == "" || strings.EqualFold(filterSurface, "ping_api") {
		pingVersion := probeHTTPVersion(client, "https://ping.nself.org/version", "latestCliVersion")
		if pingVersion == "" {
			pingVersion = probeHTTPVersion(client, "http://localhost:8001/version", "latestCliVersion")
		}
		if pingVersion == "" {
			pingVersion = "unreachable"
		}
		rows = append(rows, apiVersionRow{Surface: "ping_api", Version: pingVersion})
	}

	if filterSurface == "" || strings.EqualFold(filterSurface, "marketplace") {
		marketVersion := probeHTTPHeader(client, "https://plugins.nself.org/health", "X-API-Version")
		if marketVersion == "" {
			marketVersion = "v1"
		}
		rows = append(rows, apiVersionRow{Surface: "marketplace", Version: marketVersion})
	}

	if filterSurface == "" || strings.EqualFold(filterSurface, "sdk") {
		rows = append(rows, probeInstalledPluginSDKVersions()...)
	}

	if filterSurface == "" || strings.EqualFold(filterSurface, "hasura") {
		rows = append(rows, apiVersionRow{Surface: "hasura", Version: probeLocalHasura(client)})
	}

	return rows
}

// probeHTTPVersion fetches a URL and extracts a string field from the JSON body.
func probeHTTPVersion(client *http.Client, url, field string) string {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("X-API-Version", "1")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}
	if val, ok := data[field]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// probeHTTPHeader fetches a URL and returns the value of a response header.
func probeHTTPHeader(client *http.Client, url, header string) string {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("X-API-Version", "1")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	return resp.Header.Get(header)
}

// probeInstalledPluginSDKVersions reads installed plugin manifests for api_version fields.
func probeInstalledPluginSDKVersions() []apiVersionRow {
	cfg, err := config.Load(".")
	if err != nil {
		return nil
	}
	pluginsDir := cfg.PluginSystem.Dir
	if pluginsDir == "" {
		return nil
	}
	// Future: walk pluginsDir, read plugin.json for each installed plugin,
	// extract api_version field if present. At v1.0.9 most plugins don't declare it yet.
	_ = pluginsDir
	return nil
}

// probeLocalHasura attempts to determine if Hasura is running locally.
func probeLocalHasura(client *http.Client) string {
	resp, err := client.Get("http://localhost:8080/healthz")
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return "running (version via `nself status --json`)"
	}
	return "unreachable"
}

// scanDeprecations loads the plugin registry and returns deprecated endpoint entries.
// pluginFilter scopes results to a single plugin name when non-empty.
func scanDeprecations(pluginFilter string) []map[string]string {
	reg, err := deprecation.LoadEmbeddedPluginRegistry()
	if err != nil {
		return nil
	}

	var results []map[string]string
	for _, p := range reg.AllPlugins() {
		if pluginFilter != "" && p.Name != pluginFilter {
			continue
		}
		for _, ep := range p.DeprecatedEndpoints {
			results = append(results, map[string]string{
				"plugin":        p.Name,
				"api_version":   p.APIVersion,
				"path":          ep.Path,
				"deprecated_in": ep.DeprecatedIn,
				"removed_in":    ep.RemovedIn,
				"replacement":   ep.Replacement,
				"reason":        ep.Reason,
				"sunset_header": deprecation.HTTPSunsetHeader(ep.RemovedIn),
			})
		}
	}
	return results
}

// countBreaking returns the number of entries without a deprecated_in grace period.
func countBreaking(items []map[string]string) int {
	n := 0
	for _, d := range items {
		if d["deprecated_in"] == "" {
			n++
		}
	}
	return n
}

// =============================================================================
// Registration
// =============================================================================
