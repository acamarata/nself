package build

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/apidocs"
	"github.com/nself-org/cli/internal/compose"
	"github.com/nself-org/cli/internal/compose/monitoring"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/nginx"
	"github.com/nself-org/cli/internal/postgres"
	"github.com/nself-org/cli/internal/setup"
	"github.com/nself-org/cli/internal/ssl"
	"github.com/nself-org/cli/internal/version"
)

// BuildOptions controls build behavior via CLI flags.
type BuildOptions struct {
	// Force rebuilds everything regardless of cache freshness.
	Force bool
	// Verbose enables detailed build progress output.
	Verbose bool
	// Check validates configuration and exits without generating files.
	Check bool
	// SecurityReport prints a detailed security audit after validation.
	SecurityReport bool
	// NoAutoRedis disables automatic Redis enablement when a BullMQ-backed
	// plugin (ai, claw, mux, cron, notify, push) is detected. Pass
	// --no-auto-redis from the CLI to opt out of this behaviour.
	NoAutoRedis bool
	// Profile selects a curated subset of services for the generated
	// docker-compose.yml.  Empty string or "app" preserves today's full
	// behaviour (no regression).  Use "ops" for an observability + CI server.
	// Valid values: "app" (default), "ops".  See internal/compose/profiles.go.
	Profile compose.ProfileName
}

// BuildResult summarizes what the build produced.
type BuildResult struct {
	// ProjectName is the sanitized project name from config.
	ProjectName string
	// ComposeFile is the path to the generated docker-compose.yml.
	ComposeFile string
	// NginxConfig is the path to the generated nginx/nginx.conf.
	NginxConfig string
	// SSLCerts is the number of SSL certificate sets generated.
	SSLCerts int
	// Duration is the wall-clock time the build took.
	Duration time.Duration
	// FilesGenerated is the total number of files written.
	FilesGenerated int
	// PluginComposeFiles lists the absolute paths to plugin compose files
	// discovered during build. Empty when no plugins with compose files
	// are installed.
	PluginComposeFiles []string
	// MissingPlugins lists plugins declared in nself.yaml that could not be
	// wired into the generated stack (not installed, not auto-installable,
	// and not satisfied by a core service). Non-empty means the generated
	// stack does NOT match the declared manifest.
	MissingPlugins []string
	// CAInstalled is true when the mkcert CA is trusted by the OS.
	CAInstalled bool
	// CAManualCmd is non-empty when the user must manually trust the CA.
	CAManualCmd string
	// HostsAdded is the number of new /etc/hosts entries written.
	HostsAdded int
	// HostsManualNote is non-empty when /etc/hosts could not be updated automatically.
	HostsManualNote string
}

// requiredDirs lists the directories that must exist before generation.
// Created relative to the project workdir.
var requiredDirs = []string{
	"nginx",
	"nginx/conf.d",
	"nginx/includes",
	"nginx/sites",
	"ssl",
	"ssl/certificates",
	"postgres",
	"monitoring",
	"services",
	".nself",
}

// buildLockFile is the name of the file used to prevent concurrent builds.
const buildLockFile = ".nself/build.lock"

// acquireBuildLock creates an exclusive build lock file using O_EXCL so that
// two concurrent builds never write conflicting compose artifacts. The caller
// must defer releaseBuildLock.
func acquireBuildLock(workdir string) (*os.File, error) {
	lockPath := filepath.Join(workdir, buildLockFile)
	_ = os.MkdirAll(filepath.Dir(lockPath), 0755)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("another build is already running (lock file exists: %s). If no other build is running, remove the lock file and retry", lockPath)
		}
		return nil, fmt.Errorf("acquiring build lock: %w", err)
	}
	return f, nil
}

// releaseBuildLock closes and removes the lock file returned by acquireBuildLock.
func releaseBuildLock(f *os.File, workdir string) {
	f.Close()
	os.Remove(filepath.Join(workdir, buildLockFile))
}

// Build orchestrates the full nself build pipeline.
//
// The sequence follows BUILD_SPEC.md:
//  1. Load config via env cascade
//  2. Validate config (security, passwords, ports, CORS)
//  3. If --check: return after validation
//  4. Check cache (skip rebuild if not --force and cache fresh)
//  5. Create required directories
//  6. Generate SSL certificates
//  7. Generate nginx configuration files
//  8. Generate docker-compose.yml
//  9. Write docker-compose.yml with 0600 permissions
//  10. Write .env.computed (DATABASE_URL + DOCKER_NETWORK)
//  11. Save build version to .nself/build-version
//  12. Return BuildResult with summary
func Build(workdir string, opts BuildOptions) (*BuildResult, error) {
	start := time.Now()

	// Acquire exclusive build lock to prevent concurrent builds from
	// producing inconsistent compose artifacts.
	buildLock, err := acquireBuildLock(workdir)
	if err != nil {
		return nil, err
	}
	defer releaseBuildLock(buildLock, workdir)

	// ── Step 1: Load config via env cascade ─────────────────────────
	cfg, err := config.Load(workdir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// ── Step 1.5: Persist auto-generated secrets to .env.secrets ────
	if err := persistGeneratedSecrets(workdir, cfg); err != nil {
		return nil, fmt.Errorf("persisting generated secrets: %w", err)
	}

	// Fix permissions on .env files — ensure they are owner-only (0600).
	for _, envFile := range []string{".env", ".env.local", ".env.secrets", ".env.computed"} {
		if err := setup.EnsureEnvFilePermissions(filepath.Join(workdir, envFile)); err != nil {
			return nil, fmt.Errorf("fixing env file permissions: %w", err)
		}
	}

	// ── Step 2: Validate config ─────────────────────────────────────
	if err := config.Validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	// ── Step 2.5: Preflight — nginx domain conflict check ────────────
	routes := buildNginxRoutes(cfg)
	if conflict, pairs := nginx.HasDomainConflict(routes); conflict {
		msg := "nginx domain conflict detected:\n"
		for _, p := range pairs {
			msg += "  " + p + "\n"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	// ── Step 3: If --check, return after validation ─────────────────
	if opts.Check {
		return &BuildResult{
			ProjectName: cfg.ProjectName,
			Duration:    time.Since(start),
		}, nil
	}

	// ── Step 4: Check cache (skip if not --force and cache fresh) ───
	if !opts.Force {
		needsRebuild, err := NeedsRebuild(workdir)
		if err != nil {
			return nil, fmt.Errorf("checking build cache: %w", err)
		}
		if !needsRebuild {
			return &BuildResult{
				ProjectName: cfg.ProjectName,
				ComposeFile: filepath.Join(workdir, "docker-compose.yml"),
				NginxConfig: filepath.Join(workdir, "nginx", "nginx.conf"),
				Duration:    time.Since(start),
			}, nil
		}
	}

	// ── Step 5: Create required directories ─────────────────────────
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(workdir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	filesGenerated := 0

	// ── Step 6: Generate SSL certificates ───────────────────────────
	sslDir := filepath.Join(workdir, "ssl")
	sslGen := ssl.NewGenerator(cfg)
	sslResult, err := sslGen.GenerateWithResult(sslDir)
	if err != nil {
		return nil, fmt.Errorf("generating SSL certificates: %w", err)
	}
	filesGenerated += sslResult.Count * 2 // fullchain.pem + privkey.pem per cert set

	// ── Step 7: Generate nginx configuration ────────────────────────
	// Clear nginx/sites/ before regenerating so stale configs from a previous
	// BASE_DOMAIN value don't persist. conf.d/ is hand-managed and is NOT cleared.
	nginxSitesClearDir := filepath.Join(workdir, "nginx", "sites")
	if entries, readErr := os.ReadDir(nginxSitesClearDir); readErr == nil {
		for _, e := range entries {
			if !e.IsDir() {
				_ = os.Remove(filepath.Join(nginxSitesClearDir, e.Name()))
			}
		}
	}
	nginxGen := nginx.NewGenerator(cfg, workdir)
	nginxFiles, err := nginxGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("generating nginx config: %w", err)
	}

	// Write each nginx config file.
	for relPath, content := range nginxFiles {
		absPath := filepath.Join(workdir, relPath)
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory for %s: %w", relPath, err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", relPath, err)
		}
		filesGenerated++
	}

	// ── Step 7.05: Resolve declared plugins (nself.yaml) ────────────
	// Declared plugins (nself.yaml plugins:/bundle:/bundles: blocks) are
	// resolved BEFORE plugin nginx routes, the np_plugins seed, and compose
	// discovery — auto-installing any that are missing — so that a manifest
	// declaration alone is sufficient to wire a plugin into the stack.
	// Declared plugins that cannot be wired are reported loudly (build
	// warning + BuildResult.MissingPlugins), never silently dropped.
	pluginDir := DefaultPluginDir()
	missingPlugins := ResolveDeclaredPlugins(context.Background(), cfg, workdir, pluginDir, expectedCoreServices(cfg))
	for _, p := range missingPlugins {
		slog.Warn("declared plugin is NOT wired into the generated stack — install it or remove it from nself.yaml",
			"plugin", p, "fix", fmt.Sprintf("nself plugin install %s", p))
	}

	// ── Step 7.1: Inject plugin nginx routes ────────────────────────
	pluginRoutes, err := InjectPluginNginxRoutes(workdir, "", cfg)
	if err != nil {
		return nil, fmt.Errorf("injecting plugin nginx routes: %w", err)
	}
	filesGenerated += pluginRoutes

	// Monitoring configs: handled by nself-monitoring plugin (RenderPluginConfigs above)

	// ── Step 7.5: Generate Postgres initialization scripts ───────────
	if err := postgres.GenerateInitScript(workdir); err != nil {
		return nil, fmt.Errorf("generating postgres init script: %w", err)
	}
	filesGenerated++

	// ── Step 7.5a: Seed np_plugins with one row per installed plugin ─
	// Idempotent: INSERT ... ON CONFLICT (name) DO NOTHING. Rerunning
	// `nself build` produces byte-equal SQL on the same install set, so
	// docker-compose volume hashes don't churn.
	if _, err := GenerateNpPluginsSeed(workdir, DefaultPluginDir()); err != nil {
		return nil, fmt.Errorf("generating np_plugins seed: %w", err)
	}
	filesGenerated++

	// ── Step 7.6: Auto-enable Redis when a BullMQ plugin is installed ─
	// If Redis is not explicitly enabled but a plugin that needs it (ai, claw,
	// mux, cron, notify, push) is installed, set cfg.Redis.Enabled = true so
	// that both DetectServices and compose.Generator emit a redis block.
	// Skipped when opts.NoAutoRedis is set (--no-auto-redis flag).
	if !cfg.Redis.Enabled && !opts.NoAutoRedis && ShouldAutoEnableRedis(DefaultPluginDir()) {
		cfg.Redis.Enabled = true
		slog.Info("Redis auto-enabled", "reason", "BullMQ-backed plugin detected (ai, claw, mux, cron, notify, or push)", "disable_flag", "--no-auto-redis")
	}

	// ── Step 8: Generate docker-compose.yml ─────────────────────────
	// Profile selection: empty or "app" → full service set (no regression).
	// "ops" → observability + CI + functions + registry, no app services.
	profile := opts.Profile
	if profile == "" {
		profile = compose.ProfileApp
	}
	composeGen := compose.NewGeneratorWithProfile(cfg, profile)
	composeYAML, err := composeGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("generating docker-compose.yml: %w", err)
	}

	// ── Step 8.5: Merge Ollama sidecar when AI_OLLAMA_ENABLED=true ──
	ollamaEnv := collectOllamaEnv()
	mergedYAML, ollamaInjected, err := MergeOllamaSidecar(composeYAML, ollamaEnv)
	if err != nil {
		return nil, fmt.Errorf("merging ollama sidecar: %w", err)
	}
	if ollamaInjected {
		composeYAML = mergedYAML
		if opts.Verbose {
			slog.Info("Ollama sidecar injected", "file", "docker-compose.yml", "trigger", "AI_OLLAMA_ENABLED=true")
		}
	}

	// ── Step 8.7: Template secrets out of the generated YAML ────────
	// Literal passwords/keys become ${VAR} references resolved at
	// container-start time from .nself/compose.env (written in Step 10 and
	// passed via --env-file by start/stop/restart). Secrets never land in
	// the generated docker-compose.yml (ASI generated-file-secret rule).
	secretMap := SecretEnvMap(cfg)
	composeYAML = TemplateSecrets(composeYAML, secretMap)
	for _, leak := range LiteralSecretLeaks(composeYAML, secretMap) {
		slog.Warn("secret value still appears literally in docker-compose.yml — do not commit this file",
			"var", leak)
	}

	// ── Step 9: Write docker-compose.yml with 0600 permissions ──────
	// Prepend the GENERATED marker so pre-commit hooks, auditors, and humans
	// can unambiguously detect a hand-edited compose file (S32-T12).
	composeYAML = prependGeneratedHeader(composeYAML)
	composePath := filepath.Join(workdir, "docker-compose.yml")
	if err := os.WriteFile(composePath, composeYAML, 0600); err != nil {
		return nil, fmt.Errorf("writing docker-compose.yml: %w", err)
	}
	filesGenerated++

	// ── Step 9.5: Discover plugin compose files ────────────────────
	pluginComposeFiles, err := DiscoverPluginComposeFiles(workdir, pluginDir)
	if err != nil {
		return nil, fmt.Errorf("discovering plugin compose files: %w", err)
	}

	// Write the compose manifest so start/stop/restart know which files
	// to pass as -f flags to docker compose.
	if err := WriteComposeManifest(workdir, composePath, pluginComposeFiles); err != nil {
		return nil, fmt.Errorf("writing compose manifest: %w", err)
	}
	filesGenerated++

	if opts.Verbose && len(pluginComposeFiles) > 0 {
		slog.Info("discovered plugin compose files", "count", len(pluginComposeFiles), "files", pluginComposeFiles)
	}

	// ── Step 9.6: Verify Go plugin Dockerfiles have HEALTHCHECK ─────
	for _, w := range CheckGoPluginDockerfiles(pluginDir) {
		slog.Warn(w)
	}

	// ── Step 9.7: Wire ɳSentry scrape targets + Loki configs ────────
	// When the monitoring bundle is enabled (MONITORING_ENABLED=true), emit
	// monitoring/prometheus.yml with builtin targets + any installed ɳSentry
	// plugin targets (S12.T01) and monitoring/loki.yml + promtail.yml (S9.T08).
	//
	// Both writes are idempotent: identical inputs produce byte-identical files.
	// When monitoring is disabled, this step is a no-op so projects that opt out
	// don't acquire stray monitoring/*.yml files.
	if cfg.Monitoring.Enabled {
		// Prometheus: build PrometheusConfig with builtin targets, append any
		// installed ɳSentry plugin targets, render and write to disk.
		promCfg := monitoring.Defaults()
		nsentryAdded := AppendNSentryTargets(promCfg, pluginDir)
		promYAML, err := monitoring.RenderPrometheusYAML(promCfg)
		if err != nil {
			return nil, fmt.Errorf("rendering monitoring/prometheus.yml: %w", err)
		}
		monDir := filepath.Join(workdir, "monitoring")
		if err := os.MkdirAll(monDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating monitoring dir: %w", err)
		}
		promPath := filepath.Join(monDir, "prometheus.yml")
		if err := atomicWrite(promPath, promYAML, 0o644); err != nil {
			return nil, fmt.Errorf("writing monitoring/prometheus.yml: %w", err)
		}
		filesGenerated++
		if opts.Verbose {
			if nsentryAdded > 0 {
				slog.Info("generated monitoring/prometheus.yml", "nsentry_targets", nsentryAdded)
			} else {
				slog.Info("generated monitoring/prometheus.yml", "nsentry_targets", 0)
			}
		}

		// Loki: render loki.yml + promtail.yml with project-name + retention
		// taken from the monitoring config (env-cascade applied).
		lokiOpts := LokiBuildOptions{
			ProjectName: cfg.ProjectName,
		}
		if r := strings.TrimSpace(os.Getenv("LOKI_RETENTION_PERIOD")); r != "" {
			lokiOpts.RetentionPeriod = r
		}
		nLoki, err := WriteLokiConfigs(workdir, lokiOpts)
		if err != nil {
			return nil, fmt.Errorf("writing Loki configs: %w", err)
		}
		filesGenerated += nLoki
		if opts.Verbose {
			slog.Info("generated monitoring configs", "files", nLoki)
		}
	}

	// ── Step 9.8: Refresh hasura/config.yaml from the resolved cascade ──
	// Only touches projects that already use the Hasura CLI project layout
	// (a hasura/ directory present) so repos that don't run hasura-cli by
	// hand are unaffected (T-gap-10, backward compatible).
	if info, statErr := os.Stat(filepath.Join(workdir, "hasura")); statErr == nil && info.IsDir() {
		n, err := WriteHasuraCLIConfig(workdir, cfg)
		if err != nil {
			return nil, fmt.Errorf("writing hasura/config.yaml: %w", err)
		}
		filesGenerated += n
		if opts.Verbose && n > 0 {
			slog.Info("generated hasura/config.yaml", "endpoint", fmt.Sprintf("http://localhost:%d", cfg.Hasura.Port))
		}
	}

	// ── Step 10: Write .env.computed ────────────────────────────────
	pluginEnvVars := ComputePluginEnvVars(workdir, pluginDir)
	computedPath := filepath.Join(workdir, ".env.computed")
	computedContent := buildEnvComputed(cfg, pluginEnvVars)
	if err := os.WriteFile(computedPath, []byte(computedContent), 0600); err != nil {
		return nil, fmt.Errorf("writing .env.computed: %w", err)
	}
	filesGenerated++

	// ── Step 10.1: Write .nself/compose.env (0600) ──────────────────
	// Resolves every ${VAR} reference the secret-templating pass (Step 8.7)
	// emitted, plus plugin fragment vars (DOCKER_NETWORK, NSELF_PLUGIN_DIR,
	// PLUGIN_*_INTERNAL_URL). Passed to docker compose via --env-file.
	if err := WriteComposeEnv(workdir, cfg, secretMap, pluginEnvVars); err != nil {
		return nil, fmt.Errorf("writing %s: %w", composeEnvFile, err)
	}
	filesGenerated++

	// ── Step 11: Save build version to .nself/build-version ─────────
	versionPath := filepath.Join(workdir, buildVersionFile)
	if err := os.WriteFile(versionPath, []byte(version.GetVersion()), 0644); err != nil {
		return nil, fmt.Errorf("writing build version: %w", err)
	}
	filesGenerated++

	// ── Step 11.6: Generate OpenAPI 3.1 spec + Scalar HTML page ─────
	// Only runs when api_docs.enabled is true (default). Writes two files:
	//   .nself/dist/openapi.json   — served at /api-docs by nginx
	//   .nself/dist/scalar.html    — served at /docs (or custom path)
	// Also writes nginx/conf.d/api-docs.conf with the location blocks.
	apiDocsCfg := apidocs.ApiDocsConfig{
		Enabled:         cfg.ApiDocs.Enabled,
		Path:            cfg.ApiDocs.Path,
		Title:           cfg.ApiDocs.Title,
		Theme:           cfg.ApiDocs.Theme,
		AuthEnvVar:      cfg.ApiDocs.AuthEnvVar,
		HideEndpoints:   cfg.ApiDocs.HideEndpoints,
		GraphQLEnabled:  cfg.ApiDocs.GraphQLEnabled,
		GraphQLEndpoint: cfg.ApiDocs.GraphQLEndpoint,
	}
	// Default-fill when the config section was left empty.
	if !apiDocsCfg.Enabled && cfg.ApiDocs.Path == "" {
		apiDocsCfg = apidocs.DefaultApiDocsConfig()
	}
	if apiDocsCfg.Enabled {
		pluginDir := DefaultPluginDir()
		pluginRoutes, err := apidocs.CollectPluginRoutes(pluginDir)
		if err != nil {
			slog.Warn("collecting plugin API routes", "err", err)
		}
		if _, err := apidocs.Generate(workdir, cfg.ProjectName, cfg.BaseDomain, apiDocsCfg, pluginRoutes); err != nil {
			return nil, fmt.Errorf("generating api docs: %w", err)
		}
		filesGenerated += 2 // openapi.json + scalar.html

		// Write the nginx site config (full server block, served on docs.<base>).
		apiDocsNginxConf := apidocs.NginxConf(apiDocsCfg.Path, cfg.BaseDomain)
		apiDocsConfPath := filepath.Join(workdir, "nginx", "sites", "api-docs.conf")
		if err := os.WriteFile(apiDocsConfPath, []byte(apiDocsNginxConf), 0644); err != nil {
			return nil, fmt.Errorf("writing api-docs nginx conf: %w", err)
		}
		// Best-effort cleanup of the legacy bare-location file, if present from a
		// prior build with the broken layout.
		_ = os.Remove(filepath.Join(workdir, "nginx", "conf.d", "api-docs.conf"))
		filesGenerated++
	}

	// ── Step 11.5: Post-build validation ────────────────────────────
	nginxSitesDir := filepath.Join(workdir, "nginx", "sites")
	pvResult := PostValidate(composePath, nginxSitesDir)

	// Print warnings — they do not fail the build.
	for _, w := range pvResult.Warnings {
		slog.Warn(w)
	}

	// Any errors from post-validation fail the build.
	if len(pvResult.Errors) > 0 {
		msg := "post-build validation failed:\n"
		for _, e := range pvResult.Errors {
			msg += "  - " + e + "\n"
		}
		return nil, fmt.Errorf("%s", msg)
	}

	// ── Step 12: Return BuildResult with summary ────────────────────
	return &BuildResult{
		ProjectName:        cfg.ProjectName,
		ComposeFile:        composePath,
		NginxConfig:        filepath.Join(workdir, "nginx", "nginx.conf"),
		SSLCerts:           sslResult.Count,
		Duration:           time.Since(start),
		FilesGenerated:     filesGenerated,
		PluginComposeFiles: pluginComposeFiles,
		MissingPlugins:     missingPlugins,
		CAInstalled:        sslResult.CAInstalled,
		CAManualCmd:        sslResult.CAManualCmd,
		HostsAdded:         sslResult.HostsAdded,
		HostsManualNote:    sslResult.HostsManualNote,
	}, nil
}
