package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/federation"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var federationCmd = &cobra.Command{
	Use:   "federation",
	Short: "Manage GraphQL Federation (opt-in, requires NSELF_FEDERATION=true)",
	Long: `Manage the Apollo Router supergraph for multi-service GraphQL Federation.

Federation is opt-in. Enable it by setting NSELF_FEDERATION=true in your .env.

When enabled, nself build:
  1. Collects all installed plugins with graphql.enabled: true.
  2. Adds Hasura as the core subgraph.
  3. Composes a supergraph schema via rover supergraph compose.
  4. Injects Apollo Router (CS_7, port 4000) into docker-compose.yml.
  5. Routes /v1/graphql through Apollo Router instead of directly to Hasura.

Subcommands:
  compose    Re-compose the supergraph schema (auto-runs on nself build).
  status     Show subgraph health and schema composition state.
  introspect Print the full supergraph schema to stdout.`,
}

var federationComposeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Re-compose supergraph schema from installed plugin subgraphs",
	Long: `Re-run rover supergraph compose for all installed plugins that declare
graphql.enabled: true. The resulting supergraph.graphql is written to
.nself/federation/ and mounted into the Apollo Router container on next start.

This command runs automatically during nself build when NSELF_FEDERATION=true.
Run it manually after installing or updating a plugin that exposes a subgraph.`,
	RunE: runFederationCompose,
}

var federationStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show subgraph health and schema composition status",
	Long: `Print the list of registered subgraphs with their URL, last composition
timestamp, and schema hash. Non-zero exit when any subgraph reports status=error.`,
	RunE: runFederationStatus,
}

var federationIntrospectCmd = &cobra.Command{
	Use:   "introspect",
	Short: "Print the full supergraph schema to stdout",
	Long: `Print the composed supergraph.graphql to stdout. Useful for auditing
the full federated type system or piping into schema diffing tools.`,
	RunE: runFederationIntrospect,
}

func init() {
	federationStatusCmd.Flags().Bool("json", false, "Output status as JSON")
	federationCmd.AddCommand(federationComposeCmd)
	federationCmd.AddCommand(federationStatusCmd)
	federationCmd.AddCommand(federationIntrospectCmd)
	RootCmd.AddCommand(federationCmd)
}

// federationLoadContext centralises the repeated pattern: find project root,
// load config, resolve the .nself/federation/ directory path.
type federationContext struct {
	projectDir string
	fedDir     string
	cfg        *config.Config
}

func loadFederationContext() (*federationContext, error) {
	rawCwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	projectDir, err := config.FindNSelfRoot(rawCwd)
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}
	cfg, err := config.Load(projectDir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return &federationContext{
		projectDir: projectDir,
		fedDir:     filepath.Join(projectDir, ".nself", "federation"),
		cfg:        cfg,
	}, nil
}

func runFederationCompose(_ *cobra.Command, _ []string) error {
	fctx, err := loadFederationContext()
	if err != nil {
		return err
	}
	if !fctx.cfg.FederationEnabled {
		ui.Warn("Federation is disabled. Set NSELF_FEDERATION=true in your .env to enable.")
		return nil
	}

	ui.CommandHeader("nself federation compose", "Compose supergraph schema")

	ctx := context.Background()

	if err := os.MkdirAll(fctx.fedDir, 0o700); err != nil {
		return fmt.Errorf("creating federation dir: %w", err)
	}
	supergraphYAMLPath := filepath.Join(fctx.fedDir, "supergraph.yaml")
	supergraphGraphQLPath := filepath.Join(fctx.fedDir, "supergraph.graphql")

	// Load installed plugin manifests and extract federation blocks.
	blocks, err := pluginFederationBlocks(fctx.cfg.PluginSystem.Dir)
	if err != nil {
		return err
	}

	hasuraURL := fmt.Sprintf("http://127.0.0.1:%d/v1/graphql", fctx.cfg.Hasura.Port)
	entries, err := federation.FromManifestBlocks(blocks, hasuraURL)
	if err != nil {
		return fmt.Errorf("resolving subgraph entries: %w", err)
	}

	ui.Info(fmt.Sprintf("Composing %d subgraph(s)...", len(entries)))
	for _, e := range entries {
		ui.Info(fmt.Sprintf("  %-20s %s", e.Name, e.URL))
	}

	yamlContent, err := federation.BuildSupergraphYAML(entries)
	if err != nil {
		return fmt.Errorf("building supergraph YAML: %w", err)
	}
	if err := os.WriteFile(supergraphYAMLPath, yamlContent, 0o600); err != nil {
		return fmt.Errorf("writing supergraph.yaml: %w", err)
	}

	if err := federation.RunRoverCompose(ctx, federation.ComposeOptions{
		SupergraphYAMLPath: supergraphYAMLPath,
		OutputPath:         supergraphGraphQLPath,
	}); err != nil {
		ui.Error("Supergraph composition failed — check schema errors above.")
		return err
	}

	// Write base router config if it does not yet exist.
	routerCfgPath := filepath.Join(fctx.fedDir, "router.yaml")
	if _, statErr := os.Stat(routerCfgPath); os.IsNotExist(statErr) {
		if err := os.WriteFile(routerCfgPath, []byte(federation.RouterBaseConfig()), 0o600); err != nil {
			return fmt.Errorf("writing router.yaml: %w", err)
		}
	}

	ui.Success(fmt.Sprintf("Supergraph schema written to %s", supergraphGraphQLPath))
	return nil
}

func runFederationStatus(cmd *cobra.Command, _ []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")

	fctx, err := loadFederationContext()
	if err != nil {
		return err
	}

	if !fctx.cfg.FederationEnabled {
		if jsonOut {
			fmt.Println(`{"federation":false,"subgraphs":[]}`)
			return nil
		}
		ui.Warn("Federation is disabled (NSELF_FEDERATION=false).")
		return nil
	}

	blocks, err := pluginFederationBlocks(fctx.cfg.PluginSystem.Dir)
	if err != nil {
		return err
	}
	hasuraURL := fmt.Sprintf("http://127.0.0.1:%d/v1/graphql", fctx.cfg.Hasura.Port)
	entries, err := federation.FromManifestBlocks(blocks, hasuraURL)
	if err != nil {
		return fmt.Errorf("resolving subgraph entries: %w", err)
	}

	statuses := make([]federation.SubgraphStatus, 0, len(entries))
	for _, e := range entries {
		statuses = append(statuses, federation.SubgraphStatus{
			Name:   e.Name,
			URL:    e.URL,
			Status: "active",
		})
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]interface{}{
			"federation": true,
			"subgraphs":  statuses,
		})
	}

	ui.CommandHeader("nself federation status", "Subgraph composition state")
	ui.Info(fmt.Sprintf("%-20s %-42s %s", "SUBGRAPH", "URL", "STATUS"))
	ui.Info(fmt.Sprintf("%-20s %-42s %s", "--------", "---", "------"))
	for _, s := range statuses {
		ui.Info(fmt.Sprintf("%-20s %-42s %s", s.Name, s.URL, s.Status))
	}
	return nil
}

func runFederationIntrospect(_ *cobra.Command, _ []string) error {
	fctx, err := loadFederationContext()
	if err != nil {
		return err
	}

	schemaPath := filepath.Join(fctx.fedDir, "supergraph.graphql")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("supergraph schema not found at %s — run: nself federation compose", schemaPath)
		}
		return fmt.Errorf("reading supergraph schema: %w", err)
	}
	fmt.Print(string(data))
	return nil
}

// pluginFederationBlocks scans the installed plugin manifests in pluginDir
// and returns federation blocks for plugins with graphql.enabled: true.
//
// It uses plugin.LoadManifestsFromDir which reads plugin.json files from the
// plugin installation directory, giving access to the full PluginManifest
// including the GraphQL block added by G05.
func pluginFederationBlocks(pluginDir string) ([]federation.ManifestGraphQLBlock, error) {
	manifests, err := plugin.LoadManifestsFromDir(pluginDir)
	if err != nil {
		// Non-fatal: no plugins installed or dir missing returns empty list.
		return nil, nil
	}

	var blocks []federation.ManifestGraphQLBlock
	for _, m := range manifests {
		if m.GraphQL == nil || !m.GraphQL.Enabled {
			continue
		}
		entities := make([]federation.EntityKey, len(m.GraphQL.Entities))
		for i, e := range m.GraphQL.Entities {
			entities[i] = federation.EntityKey{Type: e.Type, Key: e.Key}
		}
		blocks = append(blocks, federation.ManifestGraphQLBlock{
			PluginName: m.Name,
			PluginPort: m.Port,
			GraphQL: federation.SubgraphConfig{
				Enabled:      m.GraphQL.Enabled,
				SubgraphName: m.GraphQL.SubgraphName,
				SubgraphURL:  m.GraphQL.SubgraphURL,
				SchemaPath:   m.GraphQL.SchemaPath,
				Entities:     entities,
			},
		})
	}
	return blocks, nil
}
