package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/security"
)

const (
	// DefaultRegistryURL is the primary registry endpoint (Cloudflare Worker).
	DefaultRegistryURL = "https://plugins.nself.org/registry.json"

	// FallbackRegistryURL is the GitHub raw fallback when the primary is down.
	FallbackRegistryURL = "https://raw.githubusercontent.com/nself-org/plugins/main/registry.json"

	// DefaultCacheTTL is the registry cache lifetime in seconds.
	DefaultCacheTTL = 300

	// registryCacheFile is the filename used inside the cache directory.
	registryCacheFile = "registry.json"

	// httpTimeout is the per-request timeout for registry fetches.
	httpTimeout = 15 * time.Second
)

// registryHTTPClient implements RegistryClient using HTTP with caching
// and a two-endpoint fallback chain.
type registryHTTPClient struct {
	primaryURL  string
	fallbackURL string
	cacheDir    string
	cacheTTL    time.Duration
	httpClient  *http.Client
}

// NewRegistryClient creates a RegistryClient that fetches from the given
// primary URL with automatic fallback and local file caching.
// If registryURL is empty, NSELF_PLUGIN_REGISTRY is checked first, then
// DefaultRegistryURL is used as the final fallback.
func newRegistryClient(registryURL, cacheDir string, cacheTTLSeconds int) RegistryClient {
	if registryURL == "" {
		if envURL := os.Getenv("NSELF_PLUGIN_REGISTRY"); envURL != "" {
			registryURL = envURL
		} else {
			registryURL = DefaultRegistryURL
		}
	}
	// S-011: Reject non-HTTPS registry URLs except localhost for dev.
	registryURL = enforceRegistryHTTPS(registryURL)
	if cacheTTLSeconds <= 0 {
		cacheTTLSeconds = DefaultCacheTTL
	}
	return &registryHTTPClient{
		primaryURL:  registryURL,
		fallbackURL: FallbackRegistryURL,
		cacheDir:    cacheDir,
		cacheTTL:    time.Duration(cacheTTLSeconds) * time.Second,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// Fetch retrieves the full plugin registry. It follows the fallback chain:
//  1. Try primaryURL
//  2. Try fallbackURL
//  3. Return stale cache (if any)
func (c *registryHTTPClient) Fetch(ctx context.Context) (*Registry, error) {
	// Check fresh cache first.
	if reg, err := c.readCache(); err == nil {
		return reg, nil
	}

	// Try primary endpoint.
	reg, err := c.fetchFromURL(ctx, c.primaryURL)
	if err == nil {
		_ = c.writeCache(reg)
		return reg, nil
	}
	primaryErr := err

	// Try fallback endpoint.
	reg, err = c.fetchFromURL(ctx, c.fallbackURL)
	if err == nil {
		_ = c.writeCache(reg)
		return reg, nil
	}

	// Both endpoints failed. Try stale cache as last resort.
	if reg, cacheErr := c.readStaleCache(); cacheErr == nil {
		return reg, nil
	}

	return nil, fmt.Errorf("registry fetch failed (primary: %v, fallback: %v)", primaryErr, err)
}

// GetPlugin looks up a single plugin by name.
func (c *registryHTTPClient) GetPlugin(ctx context.Context, name string) (*PluginManifest, error) {
	reg, err := c.Fetch(ctx)
	if err != nil {
		return nil, err
	}
	for i := range reg.Plugins {
		if strings.EqualFold(reg.Plugins[i].Name, name) {
			return &reg.Plugins[i], nil
		}
	}
	return nil, fmt.Errorf("plugin %q not found in registry", name)
}

// FetchRegistry is the standalone entry point for fetching a registry.
// It creates an ephemeral client and delegates to Fetch.
//
// Fallback chain:
//  1. registryURL (default: https://plugins.nself.org/registry.json)
//  2. GitHub raw fallback
//  3. Stale cache at cacheDir/registry.json
func FetchRegistry(ctx context.Context, registryURL string, cacheDir string) (*Registry, error) {
	if registryURL == "" {
		if envURL := os.Getenv("NSELF_PLUGIN_REGISTRY"); envURL != "" {
			registryURL = envURL
		}
	}
	if registryURL != "" {
		registryURL = enforceRegistryHTTPS(registryURL)
	}
	client := newRegistryClient(registryURL, cacheDir, DefaultCacheTTL)
	return client.Fetch(ctx)
}

// enforceRegistryHTTPS validates that a registry URL uses HTTPS.
// HTTP is allowed only for localhost/127.0.0.1 (dev use). Non-HTTPS
// URLs are replaced with the default secure registry URL.
func enforceRegistryHTTPS(rawURL string) string {
	if strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	// Allow plain HTTP for local development.
	if strings.HasPrefix(rawURL, "http://localhost") || strings.HasPrefix(rawURL, "http://127.0.0.1") {
		return rawURL
	}
	fmt.Fprintf(os.Stderr, "warning: rejecting non-HTTPS registry URL %q, using default\n", rawURL)
	return DefaultRegistryURL
}

// --- HTTP fetch ---

func (c *registryHTTPClient) fetchFromURL(ctx context.Context, url string) (*Registry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "nself-cli")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP GET %s: status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %s: %w", url, err)
	}

	return parseRegistryJSON(body)
}

// --- JSON parsing (handles both object and array formats) ---

// registryEnvelope is an intermediate type for unmarshaling the two
// registry formats. The Plugins field is kept as raw JSON so we can
// detect whether it is an object or an array.
type registryEnvelope struct {
	Version     string          `json:"version"`
	LastUpdated string          `json:"lastUpdated"`
	GeneratedAt string          `json:"generated_at"`
	Tier        string          `json:"tier"`
	Plugins     json.RawMessage `json:"plugins"`
}

// pluginImplementation holds the nested implementation block present in the
// Cloudflare Worker registry format (plugins.nself.org).
type pluginImplementation struct {
	Language       string `json:"language"`
	Runtime        string `json:"runtime"`
	DefaultPort    int    `json:"defaultPort"`
	EntryPoint     string `json:"entryPoint"`
	CLI            string `json:"cli"`
	PackageManager string `json:"packageManager"`
	Framework      string `json:"framework"`
}

// pluginEndpointEntry holds the object form of an API endpoint as returned
// by the Cloudflare Worker registry: {"method":"GET","path":"/v1/foo","description":"..."}.
type pluginEndpointEntry struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

// pluginEntry matches the fields present in the array-format (pro)
// registry as well as the object-format (free) registry.
// APIEndpoints is kept as json.RawMessage because the live registry returns
// it as an array of objects while older/local registries use an array of
// strings. We normalise both into []string during entryToManifest conversion.
type pluginEntry struct {
	Name           string               `json:"name"`
	Version        string               `json:"version"`
	Description    string               `json:"description"`
	Category       string               `json:"category"`
	Tier           string               `json:"tier"`
	License        string               `json:"license"`
	LicenseType    string               `json:"licenseType"`
	Repository     string               `json:"repository"`
	Checksum       string               `json:"checksum"`
	DownloadURL    string               `json:"download_url"`
	RequiresLicense bool                `json:"requires_license"`
	Tags            []string            `json:"tags"`
	Tables          []string            `json:"tables,omitempty"`
	Port            int                 `json:"port,omitempty"`
	Dependencies    []string            `json:"dependencies,omitempty"`
	// Implementation may appear as a nested object (Cloudflare Worker format)
	// or as flat fields (older registry format).
	Implementation *pluginImplementation `json:"implementation,omitempty"`
	// APIEndpoints is raw JSON because the registry format is not stable:
	// the live registry returns objects; older registries return strings.
	APIEndpoints json.RawMessage `json:"apiEndpoints,omitempty"`
	// Compat holds CLI and service version constraints.
	Compat *CompatBlock `json:"compat,omitempty"`
}

// parseAPIEndpoints converts the raw apiEndpoints JSON value from either the
// string-array format (["path1","path2"]) or the object-array format
// ([{"method":"GET","path":"/v1/foo"},...]) into a normalised []string.
// Unknown/null values yield nil without error.
func parseAPIEndpoints(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "null" || trimmed == "" {
		return nil
	}

	// Try string array first (legacy format).
	var strs []string
	if err := json.Unmarshal(raw, &strs); err == nil {
		return strs
	}

	// Fall back to object array (Cloudflare Worker / live registry format).
	var objs []pluginEndpointEntry
	if err := json.Unmarshal(raw, &objs); err != nil {
		return nil
	}
	out := make([]string, 0, len(objs))
	for _, ep := range objs {
		if ep.Path != "" {
			out = append(out, ep.Path)
		}
	}
	return out
}

func parseRegistryJSON(data []byte) (*Registry, error) {
	var env registryEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse registry JSON: %w", err)
	}

	if len(env.Plugins) == 0 {
		return &Registry{}, nil
	}

	// Detect format by the first non-whitespace byte.
	trimmed := strings.TrimSpace(string(env.Plugins))
	if len(trimmed) == 0 || trimmed == "null" {
		return &Registry{}, nil
	}

	var plugins []PluginManifest

	switch trimmed[0] {
	case '[':
		// Array format (pro registry).
		var entries []pluginEntry
		if err := json.Unmarshal(env.Plugins, &entries); err != nil {
			return nil, fmt.Errorf("parse registry array: %w", err)
		}
		plugins = make([]PluginManifest, 0, len(entries))
		for _, e := range entries {
			plugins = append(plugins, entryToManifest(e))
		}

	case '{':
		// Object format (free registry): {"plugin_name": {...}, ...}
		var entries map[string]pluginEntry
		if err := json.Unmarshal(env.Plugins, &entries); err != nil {
			return nil, fmt.Errorf("parse registry object: %w", err)
		}
		plugins = make([]PluginManifest, 0, len(entries))
		for key, e := range entries {
			// In object format the key is the plugin name. The inner
			// object may or may not repeat the name field.
			if e.Name == "" {
				e.Name = key
			}
			plugins = append(plugins, entryToManifest(e))
		}

	default:
		return nil, fmt.Errorf("unexpected registry plugins format (starts with %q)", string(trimmed[0]))
	}

	return &Registry{Plugins: plugins}, nil
}

func entryToManifest(e pluginEntry) PluginManifest {
	tier := e.Tier
	if tier == "" {
		// Derive tier from license info when not explicit.
		if e.LicenseType == "pro" || e.RequiresLicense {
			tier = "pro"
		} else {
			tier = "free"
		}
	}

	// Resolve port: prefer flat Port field, fall back to implementation.defaultPort.
	port := e.Port
	if port == 0 && e.Implementation != nil {
		port = e.Implementation.DefaultPort
	}

	// Resolve implementation fields from the nested block when flat fields are absent.
	language := ""
	runtime := ""
	if e.Implementation != nil {
		language = e.Implementation.Language
		runtime = e.Implementation.Runtime
	}

	return PluginManifest{
		Name:            e.Name,
		Version:         e.Version,
		Description:     e.Description,
		Category:        e.Category,
		License:         e.License,
		LicenseType:     e.LicenseType,
		Tier:            tier,
		Repository:      e.Repository,
		Checksum:        e.Checksum,
		Tags:            e.Tags,
		RequiresLicense: e.RequiresLicense,
		Tables:          e.Tables,
		Port:            port,
		Dependencies:    e.Dependencies,
		APIEndpoints:    parseAPIEndpoints(e.APIEndpoints),
		Language:        language,
		Runtime:         runtime,
		Compat:          e.Compat,
	}
}

// --- Cache ---

func (c *registryHTTPClient) cachePath() string {
	return filepath.Join(c.cacheDir, registryCacheFile)
}

// readCache returns the registry from cache only if the file exists
// and is younger than cacheTTL.
func (c *registryHTTPClient) readCache() (*Registry, error) {
	return c.readCacheWithMaxAge(c.cacheTTL)
}

// readStaleCache returns the registry from cache regardless of age.
func (c *registryHTTPClient) readStaleCache() (*Registry, error) {
	return c.readCacheWithMaxAge(0)
}

func (c *registryHTTPClient) readCacheWithMaxAge(maxAge time.Duration) (*Registry, error) {
	p := c.cachePath()
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}

	// If maxAge > 0, enforce freshness.
	if maxAge > 0 && time.Since(info.ModTime()) > maxAge {
		return nil, fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	return parseRegistryJSON(data)
}

func (c *registryHTTPClient) writeCache(reg *Registry) error {
	if c.cacheDir == "" {
		return nil
	}
	if err := os.MkdirAll(c.cacheDir, 0o755); err != nil {
		return err
	}

	// Re-serialize as a normalized JSON blob for the cache.
	data, err := json.Marshal(reg)
	if err != nil {
		return err
	}
	cachePath := c.cachePath()
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return err
	}
	return security.EnforceFilePermissions(cachePath, 0600)
}

// MarshalJSON implements json.Marshaler for Registry so the cache
// round-trips through the array format consistently.
func (r Registry) MarshalJSON() ([]byte, error) {
	type envelope struct {
		Version string        `json:"version"`
		Plugins []pluginEntry `json:"plugins"`
	}
	entries := make([]pluginEntry, 0, len(r.Plugins))
	for _, p := range r.Plugins {
		// Re-serialise APIEndpoints ([]string) as a JSON string array so the
		// cache file is always in the normalised string format, which
		// parseAPIEndpoints can read back without ambiguity.
		var rawEPs json.RawMessage
		if len(p.APIEndpoints) > 0 {
			b, err := json.Marshal(p.APIEndpoints)
			if err != nil {
				return nil, fmt.Errorf("marshaling api endpoints for %q: %w", p.Name, err)
			}
			rawEPs = b
		}
		entries = append(entries, pluginEntry{
			Name:            p.Name,
			Version:         p.Version,
			Description:     p.Description,
			Category:        p.Category,
			License:         p.License,
			LicenseType:     p.LicenseType,
			RequiresLicense: p.RequiresLicense,
			Tier:            p.Tier,
			Repository:      p.Repository,
			Checksum:        p.Checksum,
			Tags:            p.Tags,
			Tables:          p.Tables,
			Port:            p.Port,
			Dependencies:    p.Dependencies,
			APIEndpoints:    rawEPs,
			Compat:          p.Compat,
		})
	}
	return json.Marshal(envelope{
		Version: "1.0.0",
		Plugins: entries,
	})
}
