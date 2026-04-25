package federation

import (
	"fmt"
	"regexp"
	"strings"
)

// subgraphNamePattern matches valid Apollo Federation subgraph names:
// lowercase letter followed by lowercase letters, digits, or underscores.
var subgraphNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// hasuraSubgraphName is the fixed name used for the Hasura core subgraph.
// It must not collide with any plugin subgraph name.
const hasuraSubgraphName = "hasura"

// hasuraSubgraphURL is the template URL for the Hasura subgraph. It is
// interpolated at build time using the project's API domain.
const hasuraSubgraphURL = "http://localhost:8080/v1/graphql"

// ManifestGraphQLBlock is the subset of the plugin manifest that carries
// federation registration. It mirrors the YAML fields that plugin authors
// write in their plugin.yaml under the graphql: key.
//
// This type lives here (not in plugin/) to avoid a circular import:
// federation → plugin is not allowed because plugin/ is a lower-level package.
// The caller (build/) reads manifests with plugin.PluginManifest and then
// calls federation.FromManifestBlocks with the extracted GraphQL fields.
type ManifestGraphQLBlock struct {
	// PluginName is the plugin.yaml name field (used in error messages).
	PluginName string
	// PluginPort is the plugin's declared port (substituted for ${PORT}).
	PluginPort int
	// GraphQL is the graphql: block from the manifest.
	GraphQL SubgraphConfig
}

// FromManifestBlocks converts a slice of plugin manifest GraphQL blocks into
// a deduplicated, validated list of SubgraphEntry values ready for supergraph
// composition. Hasura is prepended as the first entry.
//
// Rules:
//   - Plugins with graphql.enabled == false are silently excluded.
//   - Each enabled plugin must have a non-empty SubgraphName and SubgraphURL.
//   - SubgraphName must match [a-z][a-z0-9_]* and must not be "hasura".
//   - Duplicate SubgraphName values cause an error (fail-closed).
//   - ${PORT} in SubgraphURL is substituted with the plugin's declared port.
//
// hasuraURL is the Hasura GraphQL endpoint used as the first subgraph.
func FromManifestBlocks(blocks []ManifestGraphQLBlock, hasuraURL string) ([]SubgraphEntry, error) {
	if hasuraURL == "" {
		hasuraURL = hasuraSubgraphURL
	}

	entries := []SubgraphEntry{
		{Name: hasuraSubgraphName, URL: hasuraURL},
	}
	seen := map[string]string{hasuraSubgraphName: "core"}

	for _, b := range blocks {
		if !b.GraphQL.Enabled {
			continue
		}
		if err := validateSubgraphBlock(b); err != nil {
			return nil, fmt.Errorf("plugin %q graphql block: %w", b.PluginName, err)
		}
		name := b.GraphQL.SubgraphName
		if prior, exists := seen[name]; exists {
			return nil, fmt.Errorf("plugin %q: subgraph name %q already registered by %q", b.PluginName, name, prior)
		}
		seen[name] = b.PluginName

		url := interpolatePort(b.GraphQL.SubgraphURL, b.PluginPort)
		entries = append(entries, SubgraphEntry{
			Name:       name,
			URL:        url,
			SchemaPath: b.GraphQL.SchemaPath,
		})
	}

	return entries, nil
}

// validateSubgraphBlock performs field-level validation on an enabled plugin
// graphql block. Returns a descriptive error for any violation.
func validateSubgraphBlock(b ManifestGraphQLBlock) error {
	if b.GraphQL.SubgraphName == "" {
		return fmt.Errorf("subgraph_name is required when graphql.enabled is true")
	}
	if b.GraphQL.SubgraphName == hasuraSubgraphName {
		return fmt.Errorf("subgraph_name %q is reserved for the Hasura core subgraph", hasuraSubgraphName)
	}
	if !subgraphNamePattern.MatchString(b.GraphQL.SubgraphName) {
		return fmt.Errorf("subgraph_name %q must match [a-z][a-z0-9_]*", b.GraphQL.SubgraphName)
	}
	if b.GraphQL.SubgraphURL == "" {
		return fmt.Errorf("subgraph_url is required when graphql.enabled is true")
	}
	return nil
}

// interpolatePort replaces the literal string "${PORT}" in url with the
// plugin's declared port number. If port is zero or url contains no
// placeholder, url is returned unchanged.
func interpolatePort(url string, port int) string {
	if port == 0 || !strings.Contains(url, "${PORT}") {
		return url
	}
	return strings.ReplaceAll(url, "${PORT}", fmt.Sprintf("%d", port))
}
