// Package federation implements GraphQL Federation support for multi-service
// nSelf deployments. Federation is opt-in via NSELF_FEDERATION=true or
// nself config set federation=true. Single-service deployments continue to
// use Hasura as the sole GraphQL engine (no Apollo Router injected).
//
// Architecture:
//
//	Client
//	  → Apollo Router (CS_7, port 4000)   — supergraph gateway
//	    ├─ Hasura subgraph
//	    ├─ ai subgraph (if installed)
//	    ├─ claw subgraph (if installed)
//	    └─ … other plugin subgraphs
//
// Build pipeline (federation mode):
//  1. Registry.FromManifests reads installed plugins, collects graphql blocks.
//  2. compose.BuildSupergraphYAML writes rover supergraph compose config.
//  3. External: rover supergraph compose → supergraph.graphql.
//  4. router.ComposeService returns the Apollo Router docker-compose block
//     (CS_7 slot).
//  5. nginx template routes /graphql to port 4000 instead of 8080.
package federation

// SubgraphConfig describes a single plugin's GraphQL subgraph registration,
// parsed from the graphql block in plugin.yaml.
type SubgraphConfig struct {
	// Enabled must be true for the plugin to participate in federation.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// SubgraphName is the unique lowercase name used in the supergraph schema.
	// Must be a valid GraphQL SDL name: [a-z][a-z0-9_]*.
	SubgraphName string `yaml:"subgraph_name" json:"subgraph_name"`
	// SubgraphURL is the HTTP endpoint that serves the subgraph SDL and
	// resolves federation queries. May contain ${PORT} which is interpolated
	// from the plugin's Port field at build time.
	SubgraphURL string `yaml:"subgraph_url" json:"subgraph_url"`
	// SchemaPath is a repo-relative path to the plugin's static SDL file.
	// Used by rover supergraph compose. Either SchemaPath or SubgraphURL
	// must be non-empty for rover to compose the schema.
	SchemaPath string `yaml:"schema_path,omitempty" json:"schema_path,omitempty"`
	// Entities lists the Apollo Federation entity types this subgraph owns.
	// Each entry has a Type name and the Key field used for cross-subgraph
	// entity resolution.
	Entities []EntityKey `yaml:"entities,omitempty" json:"entities,omitempty"`
}

// EntityKey describes a single Apollo Federation entity type owned by a subgraph.
type EntityKey struct {
	// Type is the GraphQL type name (e.g. "User").
	Type string `yaml:"type" json:"type"`
	// Key is the field used as the @key directive value (e.g. "id").
	Key string `yaml:"key" json:"key"`
}

// SubgraphEntry is the resolved form of a plugin subgraph — name, URL, and
// schema path are fully interpolated and validated. Used as input to
// compose.BuildSupergraphYAML.
type SubgraphEntry struct {
	// Name is the unique subgraph identifier in the supergraph schema.
	Name string
	// URL is the resolved HTTP endpoint.
	URL string
	// SchemaPath is the rover-usable schema path (may be empty).
	SchemaPath string
}

// SubgraphStatus is the runtime health record for a single subgraph, returned
// by StatusReport.
type SubgraphStatus struct {
	// Name is the subgraph name.
	Name string `json:"name"`
	// URL is the subgraph endpoint.
	URL string `json:"url"`
	// SchemaHash is the SHA-256 hex digest of the last composed schema.
	SchemaHash string `json:"schema_hash"`
	// LastComposed is the RFC 3339 timestamp of the last successful composition.
	LastComposed string `json:"last_composed"`
	// Status is one of "active", "inactive", "error".
	Status string `json:"status"`
}
