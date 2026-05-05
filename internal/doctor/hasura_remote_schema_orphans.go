// S77-T08: doctor --deep check for orphaned Hasura remote schemas.
//
// A remote schema is "orphaned" when it is registered in Hasura metadata but the
// plugin that provides it is no longer installed.  This can happen after
// `nself plugin uninstall <name>` if Hasura metadata was not cleaned up, or after a
// manual plugin removal.  Orphaned remote schemas cause Hasura to emit schema-load
// errors on startup and break the GraphQL API surface.
//
// Detection strategy:
//  1. Read NSELF_HASURA_GRAPHQL_URL (or derive from HASURA_GRAPHQL_URL) to locate
//     the running Hasura instance.
//  2. Call the Hasura Metadata API to export remote_schemas (POST /v1/metadata with
//     {"type":"export_metadata","args":{}}).  Requires HASURA_GRAPHQL_ADMIN_SECRET.
//  3. For each remote schema entry, derive the expected plugin name from the
//     remote schema's URL: internal plugin URLs follow the pattern
//     http://host:PORT  where PORT is registered in PLUGIN_<NAME>_INTERNAL_URL.
//  4. Cross-reference against loaded plugins: env var NSELF_PLUGINS_LOADED (comma-
//     separated list set by `nself build`) and per-plugin PLUGIN_<NAME>_INTERNAL_URL.
//  5. Emit WARN for each orphan found; PASS if none; SKIP if Hasura is unreachable.
//
// The check is read-only — it never modifies Hasura metadata.
// S77-T08
package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
)

// CheckOrphanRemoteSchemas implements S77-T08.
// It queries Hasura metadata and compares registered remote schemas against the
// currently installed nself plugins.  Any remote schema whose providing plugin is
// absent is flagged as an orphan.
func CheckOrphanRemoteSchemas(ctx context.Context) CheckResult {
	const (
		checkID   = "HASURA-ORPHAN-RS-01"
		section   = "hasura"
		checkName = checkID + ": orphaned remote schemas"
	)

	hasuraURL := resolveHasuraURL()
	if hasuraURL == "" {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "skip",
			Message: "HASURA_GRAPHQL_URL not set — check skipped",
		}
	}

	adminSecret := os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET")
	if adminSecret == "" {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "skip",
			Message: "HASURA_GRAPHQL_ADMIN_SECRET not set — check skipped",
		}
	}

	remoteSchemas, err := exportRemoteSchemas(ctx, hasuraURL, adminSecret)
	if err != nil {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "warn",
			Message: fmt.Sprintf("cannot export Hasura metadata: %s", err.Error()),
		}
	}

	if len(remoteSchemas) == 0 {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "pass",
			Message: "no remote schemas registered",
		}
	}

	loadedPlugins := resolveLoadedPlugins()
	orphans := findOrphanedRemoteSchemas(remoteSchemas, loadedPlugins)

	if len(orphans) == 0 {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "pass",
			Message: fmt.Sprintf("all %d remote schema(s) have a matching installed plugin", len(remoteSchemas)),
		}
	}

	return CheckResult{
		Section: section,
		Name:    checkName,
		Status:  "warn",
		Message: fmt.Sprintf(
			"%d orphaned remote schema(s) detected: %s — run fix to remove from Hasura metadata",
			len(orphans),
			strings.Join(orphans, ", "),
		),
		FixCmd: buildOrphanFixCmd(orphans),
	}
}

// hasuraMetadataExport is the minimal shape we need from the metadata export response.
type hasuraMetadataExport struct {
	RemoteSchemas []hasuraRemoteSchema `json:"remote_schemas"`
}

type hasuraRemoteSchema struct {
	Name       string `json:"name"`
	Definition struct {
		URL string `json:"url"`
	} `json:"definition"`
}

// exportRemoteSchemas calls the Hasura Metadata API and returns the list of remote schemas.
func exportRemoteSchemas(ctx context.Context, hasuraURL, adminSecret string) ([]hasuraRemoteSchema, error) {
	payload := `{"type":"export_metadata","args":{}}`
	metadataEndpoint := strings.TrimRight(hasuraURL, "/") + "/v1/metadata"

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, metadataEndpoint,
		bytes.NewBufferString(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Admin-Secret", adminSecret)

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return nil, fmt.Errorf("metadata API unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("metadata API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var meta hasuraMetadataExport
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("decode metadata response: %w", err)
	}
	return meta.RemoteSchemas, nil
}

// resolveHasuraURL returns the Hasura GraphQL endpoint from env, or empty string.
func resolveHasuraURL() string {
	if u := os.Getenv("NSELF_HASURA_GRAPHQL_URL"); u != "" {
		return u
	}
	return os.Getenv("HASURA_GRAPHQL_URL")
}

// resolveLoadedPlugins returns the set of plugin names currently installed.
// Sources (in priority order):
//  1. NSELF_PLUGINS_LOADED — comma-separated list set by `nself build`
//  2. Per-plugin sentinel vars: NSELF_<NAME>_LOADED=1
//  3. Per-plugin URL vars: PLUGIN_<NAME>_INTERNAL_URL non-empty
func resolveLoadedPlugins() map[string]bool {
	loaded := make(map[string]bool)

	// Source 1: NSELF_PLUGINS_LOADED=ai,moderation,stripe,...
	if list := strings.TrimSpace(os.Getenv("NSELF_PLUGINS_LOADED")); list != "" {
		for _, name := range strings.Split(list, ",") {
			name = strings.TrimSpace(strings.ToLower(name))
			if name != "" {
				loaded[name] = true
			}
		}
	}

	// Source 2 + 3: scan env for NSELF_<NAME>_LOADED and PLUGIN_<NAME>_INTERNAL_URL.
	for _, kv := range os.Environ() {
		k, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		// NSELF_<NAME>_LOADED=1
		if strings.HasPrefix(k, "NSELF_") && strings.HasSuffix(k, "_LOADED") {
			name := strings.ToLower(k[len("NSELF_") : len(k)-len("_LOADED")])
			name = strings.ReplaceAll(name, "_", "-")
			if name != "" {
				loaded[name] = true
			}
		}
		// PLUGIN_<NAME>_INTERNAL_URL=http://...
		if strings.HasPrefix(k, "PLUGIN_") && strings.HasSuffix(k, "_INTERNAL_URL") {
			name := strings.ToLower(k[len("PLUGIN_") : len(k)-len("_INTERNAL_URL")])
			name = strings.ReplaceAll(name, "_", "-")
			if name != "" {
				loaded[name] = true
			}
		}
	}

	return loaded
}

// findOrphanedRemoteSchemas returns the names of remote schemas whose plugin is absent.
// Matching strategy:
//   - Remote schema names follow the convention "<plugin>-schema" or plain "<plugin>".
//   - We strip the "-schema" suffix and compare against the loaded set.
//   - If the remote schema URL contains an internal plugin URL env var value, it is
//     considered loaded regardless of name convention.
func findOrphanedRemoteSchemas(schemas []hasuraRemoteSchema, loaded map[string]bool) []string {
	// Build URL→pluginName map for URL-based matching.
	urlToPlugin := make(map[string]string)
	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || v == "" {
			continue
		}
		if strings.HasPrefix(k, "PLUGIN_") && strings.HasSuffix(k, "_INTERNAL_URL") {
			name := strings.ToLower(k[len("PLUGIN_") : len(k)-len("_INTERNAL_URL")])
			name = strings.ReplaceAll(name, "_", "-")
			urlToPlugin[strings.TrimRight(v, "/")] = name
		}
	}

	var orphans []string
	for _, rs := range schemas {
		if isRemoteSchemaLoaded(rs, loaded, urlToPlugin) {
			continue
		}
		orphans = append(orphans, rs.Name)
	}
	return orphans
}

// isRemoteSchemaLoaded returns true if the remote schema has a corresponding installed plugin.
func isRemoteSchemaLoaded(rs hasuraRemoteSchema, loaded map[string]bool, urlToPlugin map[string]string) bool {
	// URL-based check: if the schema URL starts with a known plugin's INTERNAL_URL base, it's loaded.
	// Remote schema URLs often include a path suffix (e.g. /graphql) beyond the base INTERNAL_URL.
	rsURL := strings.TrimRight(rs.Definition.URL, "/")
	for baseURL, pluginName := range urlToPlugin {
		if rsURL == baseURL || strings.HasPrefix(rsURL, baseURL+"/") {
			return loaded[pluginName]
		}
	}

	// Name-based check: strip "-schema" suffix, normalise to kebab-case.
	name := strings.ToLower(rs.Name)
	name = strings.TrimSuffix(name, "-schema")
	name = strings.ReplaceAll(name, "_", "-")

	return loaded[name]
}

// buildOrphanFixCmd returns a suggested `nself` command to remove orphaned remote schemas.
func buildOrphanFixCmd(orphans []string) string {
	if len(orphans) == 0 {
		return ""
	}
	// Each orphan has its own untrack call; show the first one as the canonical example.
	return fmt.Sprintf("nself hasura remote-schema untrack %s", strings.Join(orphans, " "))
}
