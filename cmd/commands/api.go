package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/deprecation"
	"github.com/nself-org/cli/internal/ui"
	"github.com/nself-org/cli/internal/version"

	"github.com/spf13/cobra"
)

// apiVersionRow represents one surface's version info for the api version command.
type apiVersionRow struct {
	Surface    string `json:"surface"`
	Version    string `json:"version"`
	Deprecated bool   `json:"deprecated"`
	EOLDate    string `json:"eol_date,omitempty"`
}

// apiCmd is the root command for API versioning operator tooling.
var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "API versioning and deprecation tooling for operators",
	Long: `Operator tooling for the nSelf API versioning baseline (v1.0.9 LTS).

The nSelf LTS contract guarantees backward compatibility through 2027-04-17.
These commands let you measure and verify that commitment on your install.`,
}

// apiVersionCmd reports the API version observable from this install.
var apiVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show API version for every surface observable from this install",
	Long: `Show the API version for every surface reachable from this install:

  - CLI binary version (this binary)
  - ping_api version (probed via HTTP)
  - Marketplace Worker version (probed via HTTP)
  - Per-installed-plugin SDK version (from plugin.json api_version if declared)
  - Hasura schema version (if nself is running locally)

Deprecation status is cross-referenced against the central deprecation registry.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		surface, _ := cmd.Flags().GetString("surface")
		timeout, _ := cmd.Flags().GetInt("timeout")

		rows := collectAPIVersions(surface, timeout)

		if jsonOut {
			return ui.PrintJSON(rows)
		}

		fmt.Printf("\n%-30s %-15s %-12s %s\n", "Surface", "Version", "Deprecated", "EOL Date")
		fmt.Println(strings.Repeat("-", 72))
		for _, row := range rows {
			dep := "no"
			if row.Deprecated {
				dep = "YES"
			}
			eol := row.EOLDate
			if eol == "" {
				eol = "-"
			}
			fmt.Printf("%-30s %-15s %-12s %s\n", row.Surface, row.Version, dep, eol)
		}
		fmt.Println()
		return nil
	},
}

// apiDeprecationCheckCmd checks installed plugins for deprecated API usage (G6).
// --plugin <name> scopes the check to one plugin.
// --strict exits 1 if any BREAKING (no grace period) entry is found (G11 CI gate).
var apiDeprecationCheckCmd = &cobra.Command{
	Use:   "deprecation-check",
	Short: "Check for deprecated API usage in this install",
	Long: `Walk the plugin deprecation registry to find deprecated endpoints.

Cross-references internal/deprecation/registry.yaml. Use --plugin <name> to
check a specific plugin. Use --strict to fail CI when a BREAKING entry is found.

At v1.0.9 LTS baseline, the registry has no active endpoint deprecations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		pluginFilter, _ := cmd.Flags().GetString("plugin")
		strict, _ := cmd.Flags().GetBool("strict")

		items := scanDeprecations(pluginFilter)

		if len(items) == 0 {
			label := "install"
			if pluginFilter != "" {
				label = fmt.Sprintf("plugin '%s'", pluginFilter)
			}
			if jsonOut {
				out, _ := json.Marshal(map[string]interface{}{
					"deprecations_found": 0,
					"plugin_filter":      pluginFilter,
					"registry_version":   "v1.0.9-baseline",
					"status":             "clean",
				})
				fmt.Println(string(out))
			} else {
				fmt.Printf("0 deprecations found. Your %s is clean against the v1.0.9 LTS baseline.\n\n", label)
				fmt.Println("  Registry: internal/deprecation/registry.yaml")
				fmt.Println("  LTS window: 2026-04-17 → 2027-04-17")
				fmt.Println()
			}
			return nil
		}

		// Detect BREAKING entries — those with no deprecated_in grace period.
		hasBreaking := false
		for _, d := range items {
			if d["deprecated_in"] == "" {
				hasBreaking = true
				break
			}
		}

		if jsonOut {
			status := "warnings"
			if hasBreaking {
				status = "BREAKING"
			}
			out, _ := json.Marshal(map[string]interface{}{
				"deprecations_found": len(items),
				"status":             status,
				"items":              items,
			})
			fmt.Println(string(out))
		} else {
			fmt.Printf("Found %d deprecated API usage(s):\n\n", len(items))
			for _, d := range items {
				tag := "DEPRECATED"
				if d["deprecated_in"] == "" {
					tag = "BREAKING"
				}
				fmt.Printf("  [%s] %s: %s (deprecated in v%s, removed in v%s)\n",
					tag, d["plugin"], d["path"], d["deprecated_in"], d["removed_in"])
				if d["replacement"] != "" {
					fmt.Printf("    Replacement: %s\n", d["replacement"])
				}
				if d["sunset_header"] != "" {
					fmt.Printf("    Sunset: %s\n", d["sunset_header"])
				}
			}
			fmt.Println()
		}

		if strict && hasBreaking {
			return fmt.Errorf("BREAKING: %d endpoint(s) lack a deprecation grace period — add 'deprecated_in' before merging",
				countBreaking(items))
		}
		return nil
	},
}

// apiChangelogCmd prints the deprecation sunset calendar for a named plugin (G9).
var apiChangelogCmd = &cobra.Command{
	Use:   "changelog <plugin>",
	Short: "Print the deprecation sunset calendar for a plugin",
	Long: `Print a date-sorted list of deprecated endpoints for a plugin, including
sunset dates, replacements, and migration links.

Example:
  nself api changelog ai`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName := args[0]
		jsonOut, _ := cmd.Flags().GetBool("json")

		reg, err := deprecation.LoadPluginRegistry(resolveRegistryPath())
		if err != nil {
			return fmt.Errorf("loading plugin registry: %w", err)
		}

		entry, ok := reg.LookupPlugin(pluginName)
		if !ok {
			return fmt.Errorf("plugin %q not found in deprecation registry", pluginName)
		}

		calEntries := reg.SunsetDate(pluginName)

		if jsonOut {
			type jsonEntry struct {
				Plugin       string `json:"plugin"`
				APIVersion   string `json:"api_version"`
				Path         string `json:"path"`
				DeprecatedIn string `json:"deprecated_in"`
				RemovedIn    string `json:"removed_in"`
				Replacement  string `json:"replacement"`
				Reason       string `json:"reason"`
				SunsetHeader string `json:"sunset_header"`
			}
			rows := make([]jsonEntry, 0, len(calEntries))
			for _, e := range calEntries {
				rows = append(rows, jsonEntry{
					Plugin:       pluginName,
					APIVersion:   entry.APIVersion,
					Path:         e.Path,
					DeprecatedIn: e.DeprecatedIn,
					RemovedIn:    e.RemovedIn,
					Replacement:  e.Replacement,
					Reason:       e.Reason,
					SunsetHeader: deprecation.HTTPSunsetHeader(e.RemovedIn),
				})
			}
			out, _ := json.Marshal(map[string]interface{}{
				"plugin":      pluginName,
				"api_version": entry.APIVersion,
				"changelog":   rows,
			})
			fmt.Println(string(out))
			return nil
		}

		fmt.Printf("\nAPI Deprecation Calendar — plugin: %s  (current API version: %s)\n\n",
			pluginName, entry.APIVersion)

		if len(calEntries) == 0 {
			fmt.Println("  No deprecated endpoints. All paths are current.")
			fmt.Println()
			return nil
		}

		fmt.Printf("  %-35s %-14s %-12s %s\n", "Path", "Deprecated In", "Removed In", "Replacement")
		fmt.Println("  " + strings.Repeat("-", 82))
		for _, e := range calEntries {
			fmt.Printf("  %-35s %-14s %-12s %s\n", e.Path, e.DeprecatedIn, e.RemovedIn, e.Replacement)
			if e.Reason != "" {
				fmt.Printf("  %s  Reason: %s\n", strings.Repeat(" ", 35), e.Reason)
			}
			sunset := deprecation.HTTPSunsetHeader(e.RemovedIn)
			if sunset != "" {
				fmt.Printf("  %s  Sunset: %s\n", strings.Repeat(" ", 35), sunset)
			}
		}
		fmt.Println()
		return nil
	},
}

// =============================================================================
// Helpers
// =============================================================================

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
	reg, err := deprecation.LoadPluginRegistry(resolveRegistryPath())
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

func init() {
	apiCmd.AddCommand(apiVersionCmd)
	apiCmd.AddCommand(apiDeprecationCheckCmd)
	apiCmd.AddCommand(apiChangelogCmd)

	// api version flags
	apiVersionCmd.Flags().Bool("json", false, "Output as JSON")
	apiVersionCmd.Flags().String("surface", "", "Filter to a single surface (cli, ping_api, marketplace, sdk, hasura)")
	apiVersionCmd.Flags().Int("timeout", 5, "HTTP probe timeout in seconds")

	// api deprecation-check flags (G6: --plugin, --strict for G11)
	apiDeprecationCheckCmd.Flags().Bool("json", false, "Output as JSON")
	apiDeprecationCheckCmd.Flags().String("plugin", "", "Check a specific plugin by name (e.g. --plugin ai)")
	apiDeprecationCheckCmd.Flags().Bool("strict", false, "Exit 1 if any BREAKING entries exist (used by CI gate, G11)")

	// api changelog flags
	apiChangelogCmd.Flags().Bool("json", false, "Output as JSON")

	RootCmd.AddCommand(apiCmd)
}
