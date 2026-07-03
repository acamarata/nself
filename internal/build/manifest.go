package build

// Purpose: Read the project's nself.yaml manifest and guarantee that every
//          declared plugin is wired into the generated stack (auto-installing
//          missing plugins when possible) — or loudly reported, never silently
//          dropped (PCI plugin-injection-dropped, 2026-07-03).
// Inputs:  workdir (project root containing nself.yaml), pluginDir (global
//          plugin install dir), cfg (resolved env-cascade config), generated
//          compose service names.
// Outputs: Declared plugin list, missing-plugin report, best-effort installs.
// Constraints: Backward compatible — projects without nself.yaml keep the
//              install-then-discover flow unchanged. Auto-install is
//              best-effort and can be disabled with
//              NSELF_AUTO_INSTALL_PLUGINS=false (offline/CI).
// SPORT: cli/internal/build — manifest injection (gap #8, ntask dogfood).

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nself-org/cli/internal/bundle"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/plugin"
)

// manifestFilenames are the accepted names for the project manifest, checked
// in order. The canonical name is nself.yaml.
var manifestFilenames = []string{"nself.yaml", "nself.yml"}

// ProjectManifest is the parsed subset of nself.yaml that the build pipeline
// consumes. Unknown keys are ignored so app repos can keep richer metadata
// (display_name, tier, auth_mode, ...) in the same file.
type ProjectManifest struct {
	App     string          `yaml:"app"`
	Bundle  string          `yaml:"bundle"`
	Bundles []string        `yaml:"bundles"`
	Plugins ManifestPlugins `yaml:"plugins"`
}

// ManifestPlugins accepts both manifest shapes:
//
//	plugins:            # flat list
//	  - cron
//	  - notify
//
//	plugins:            # tiered map
//	  free: [cron, notify]
//	  pro:  [ai-gateway]
type ManifestPlugins struct {
	Free []string
	Pro  []string
	Flat []string
}

// UnmarshalYAML implements custom decoding for the two supported shapes.
func (mp *ManifestPlugins) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		return node.Decode(&mp.Flat)
	case yaml.MappingNode:
		var aux struct {
			Free []string `yaml:"free"`
			Pro  []string `yaml:"pro"`
		}
		if err := node.Decode(&aux); err != nil {
			return fmt.Errorf("parsing plugins block: %w", err)
		}
		mp.Free = aux.Free
		mp.Pro = aux.Pro
		return nil
	}
	// Null/absent — leave zero value.
	return nil
}

// LoadProjectManifest reads nself.yaml (or nself.yml) from workdir.
// Returns (nil, nil) when no manifest exists — callers treat that as
// "no declarations, legacy discover-only flow".
func LoadProjectManifest(workdir string) (*ProjectManifest, error) {
	for _, name := range manifestFilenames {
		path := filepath.Join(workdir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		var m ProjectManifest
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		return &m, nil
	}
	return nil, nil
}

// DeclaredPlugins returns the deduplicated, sorted set of plugin slugs the
// manifest declares: plugins.free + plugins.pro + flat list + the expansion
// of bundle/bundles via the canonical bundle catalog. Placeholder entries
// (anything that is not a bare slug, e.g. "(free plugins only ...)") are
// dropped.
func (m *ProjectManifest) DeclaredPlugins() []string {
	if m == nil {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	add := func(name string) {
		slug := strings.ToLower(strings.TrimSpace(name))
		if !isPluginSlug(slug) || seen[slug] {
			return
		}
		seen[slug] = true
		out = append(out, slug)
	}

	for _, n := range m.Plugins.Flat {
		add(n)
	}
	for _, n := range m.Plugins.Free {
		add(n)
	}
	for _, n := range m.Plugins.Pro {
		add(n)
	}

	bundles := m.Bundles
	if m.Bundle != "" {
		bundles = append(bundles, m.Bundle)
	}
	for _, slug := range bundles {
		b, ok := bundle.Get(slug)
		if !ok {
			continue
		}
		for _, p := range b.Plugins {
			add(p)
		}
	}

	sort.Strings(out)
	return out
}

// isPluginSlug reports whether s looks like a valid plugin slug:
// lowercase letters, digits, and hyphens only.
func isPluginSlug(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
		default:
			return false
		}
	}
	return true
}

// coreServiceAliases maps declared plugin names that are satisfied by core
// compose services rather than plugin containers. "auth" and "storage" are
// wired by the compose generator directly (hasura-auth container + MinIO),
// so declaring them must not produce a false "missing plugin" report.
var coreServiceAliases = map[string][]string{
	"auth":    {"auth"},
	"storage": {"minio", "storage"},
}

// autoInstallEnabled reports whether missing declared plugins may be
// auto-installed from the registry. Disabled with
// NSELF_AUTO_INSTALL_PLUGINS=false (offline builds, hermetic CI).
func autoInstallEnabled() bool {
	return !strings.EqualFold(os.Getenv("NSELF_AUTO_INSTALL_PLUGINS"), "false")
}

// autoInstallTimeout bounds a single best-effort plugin auto-install so an
// unreachable registry can never hang `nself build`.
const autoInstallTimeout = 60 * time.Second

// ResolveDeclaredPlugins guarantees every plugin declared in nself.yaml is
// wired into the build, in three steps per declared plugin:
//
//  1. Already installed in pluginDir → satisfied (its compose fragment, if
//     any, is picked up by DiscoverPluginComposeFiles).
//  2. Provided by a core compose service (auth, storage) → satisfied.
//  3. Otherwise attempt a best-effort auto-install from the registry, then
//     re-check. Still absent → reported in missing.
//
// Returns the list of declared plugins that could not be wired. Callers MUST
// surface missing plugins to the user (build warning + BuildResult) — a
// declared plugin is never silently dropped.
func ResolveDeclaredPlugins(ctx context.Context, cfg *config.Config, workdir, pluginDir string, composeServices []string) (missing []string) {
	manifest, err := LoadProjectManifest(workdir)
	if err != nil {
		slog.Warn("could not parse project manifest — declared plugins not resolved", "err", err)
		return nil
	}
	declared := manifest.DeclaredPlugins()
	if len(declared) == 0 {
		return nil
	}

	serviceSet := make(map[string]bool, len(composeServices))
	for _, s := range composeServices {
		serviceSet[s] = true
	}

	for _, name := range declared {
		if pluginInstalled(pluginDir, name) {
			continue
		}
		if satisfiedByCoreService(name, serviceSet) {
			continue
		}
		if autoInstallEnabled() {
			installCtx, cancel := context.WithTimeout(ctx, autoInstallTimeout)
			err := plugin.Install(installCtx, cfg, name, pluginDir)
			cancel()
			if err == nil && pluginInstalled(pluginDir, name) {
				slog.Info("auto-installed declared plugin", "plugin", name, "source", "nself.yaml")
				continue
			}
			slog.Warn("declared plugin auto-install failed", "plugin", name, "err", err)
		}
		missing = append(missing, name)
	}

	return missing
}

// pluginInstalled reports whether the named plugin has an install directory
// (with at least a plugin.json or compose fragment) under pluginDir.
func pluginInstalled(pluginDir, name string) bool {
	dir := filepath.Join(pluginDir, name)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	// A directory with a manifest or compose fragment counts as installed.
	for _, f := range []string{"plugin.json", pluginComposeFilename} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}
	return false
}

// satisfiedByCoreService reports whether a declared plugin name is provided
// by a generated core compose service instead of a plugin container.
func satisfiedByCoreService(name string, serviceSet map[string]bool) bool {
	if serviceSet[name] {
		return true
	}
	for _, alias := range coreServiceAliases[name] {
		if serviceSet[alias] {
			return true
		}
	}
	return false
}

// expectedCoreServices returns the compose service names the generator will
// emit for core/optional services, derived from config. Used by
// ResolveDeclaredPlugins to recognise declared names (auth, storage) that a
// core service satisfies — the compose YAML is generated after plugin
// resolution, so the names are derived rather than parsed.
func expectedCoreServices(cfg *config.Config) []string {
	services := []string{"postgres", "hasura", "auth", "nginx"}
	if cfg.Minio.Enabled {
		services = append(services, "minio")
	}
	if cfg.Redis.Enabled {
		services = append(services, "redis")
	}
	if cfg.Functions.Enabled {
		services = append(services, "functions")
	}
	if cfg.Mailpit.Enabled {
		services = append(services, "mailpit")
	}
	return services
}
