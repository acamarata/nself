package build

// orchestrator_build_ssl.go — Build() Steps 5-7.6: create required
// directories, generate SSL certificates, generate + write nginx
// configuration, resolve declared plugins (nself.yaml), inject plugin
// nginx routes, generate the Postgres init script, seed np_plugins, and
// auto-enable Redis for BullMQ-backed plugins. Split from orchestrator.go
// (T-P6-E2-W1-S1-T3).
// Inputs:  st.workdir, st.opts, st.cfg (already loaded by loadValidateConfig).
// Outputs: sets st.filesGenerated (initialized here), st.sslResult,
//          st.pluginDir, st.missingPlugins; may mutate st.cfg.Redis.Enabled.
//          Returns error on any generation failure.
// Constraints: pure move, same checks/output/errors/order, no behavior
//              change.

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/nginx"
	"github.com/nself-org/cli/internal/postgres"
	"github.com/nself-org/cli/internal/ssl"
)

// generateSSLAndNginx runs Steps 5-7.6 of Build().
func (st *buildState) generateSSLAndNginx() error {
	// ── Step 5: Create required directories ─────────────────────────
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(st.workdir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	st.filesGenerated = 0

	// ── Step 6: Generate SSL certificates ───────────────────────────
	sslDir := filepath.Join(st.workdir, "ssl")
	sslGen := ssl.NewGenerator(st.cfg)
	sslResult, err := sslGen.GenerateWithResult(sslDir)
	if err != nil {
		return fmt.Errorf("generating SSL certificates: %w", err)
	}
	st.sslResult = sslResult
	st.filesGenerated += sslResult.Count * 2 // fullchain.pem + privkey.pem per cert set

	// ── Step 7: Generate nginx configuration ────────────────────────
	// sitesDir is normally <workdir>/nginx/sites. When NGINX_FRONTED_BY is
	// set this project generates no nginx container of its own, so that
	// directory is never read by any running nginx — resolveNginxSitesDir
	// (cli#385) targets the fronting stack's own nginx/sites/ instead, or
	// refuses the build when that stack's directory cannot be confirmed.
	// See nginx_sites_dir.go.
	sitesDir, err := resolveNginxSitesDir(st.workdir, st.cfg.Nginx.FrontedBy)
	if err != nil {
		return err
	}

	// Clear stale *.conf files from a previous BASE_DOMAIN before
	// regenerating — but only in sitesDir's own (unfronted) case. When
	// fronted, sitesDir belongs to another project's stack, which writes
	// its own per-service confs into the same directory; blanket-clearing
	// it would delete that stack's own routes. conf.d/ is hand-managed and
	// is NOT cleared either way.
	if sitesDir == filepath.Join(st.workdir, "nginx", "sites") {
		if entries, readErr := os.ReadDir(sitesDir); readErr == nil {
			for _, e := range entries {
				if !e.IsDir() {
					_ = os.Remove(filepath.Join(sitesDir, e.Name()))
				}
			}
		}
	}

	nginxGen := nginx.NewGenerator(st.cfg, st.workdir)
	nginxFiles, err := nginxGen.Generate()
	if err != nil {
		return fmt.Errorf("generating nginx config: %w", err)
	}

	// Write each nginx config file. Per-service site route confs
	// ("nginx/sites/...") are redirected to sitesDir; everything else
	// (nginx.conf, conf.d/default.conf, includes/rate-limits.conf) stays
	// under this project's own workdir/nginx regardless of FrontedBy.
	const sitesPrefix = "nginx/sites/"
	for relPath, content := range nginxFiles {
		var absPath string
		if strings.HasPrefix(relPath, sitesPrefix) {
			absPath = filepath.Join(sitesDir, strings.TrimPrefix(relPath, sitesPrefix))
		} else {
			absPath = filepath.Join(st.workdir, relPath)
		}
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", relPath, err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", relPath, err)
		}
		st.filesGenerated++
	}

	// ── Step 7.05: Resolve declared plugins (nself.yaml) ────────────
	// Declared plugins (nself.yaml plugins:/bundle:/bundles: blocks) are
	// resolved BEFORE plugin nginx routes, the np_plugins seed, and compose
	// discovery — auto-installing any that are missing — so that a manifest
	// declaration alone is sufficient to wire a plugin into the stack.
	// Declared plugins that cannot be wired are reported loudly (build
	// warning + BuildResult.MissingPlugins), never silently dropped.
	st.pluginDir = DefaultPluginDir()
	st.missingPlugins = ResolveDeclaredPlugins(context.Background(), st.cfg, st.workdir, st.pluginDir, expectedCoreServices(st.cfg))
	for _, p := range st.missingPlugins {
		slog.Warn("declared plugin is NOT wired into the generated stack — install it or remove it from nself.yaml",
			"plugin", p, "fix", fmt.Sprintf("nself plugin install %s", p))
	}

	// ── Step 7.1: Inject plugin nginx routes ────────────────────────
	pluginRoutes, err := InjectPluginNginxRoutes(st.workdir, "", st.cfg)
	if err != nil {
		return fmt.Errorf("injecting plugin nginx routes: %w", err)
	}
	st.filesGenerated += pluginRoutes

	// Monitoring configs: handled by nself-monitoring plugin (RenderPluginConfigs above)

	// ── Step 7.5: Generate Postgres initialization scripts ───────────
	if err := postgres.GenerateInitScript(st.workdir); err != nil {
		return fmt.Errorf("generating postgres init script: %w", err)
	}
	st.filesGenerated++

	// ── Step 7.5a: Seed np_plugins with one row per installed plugin ─
	// Idempotent: INSERT ... ON CONFLICT (name) DO NOTHING. Rerunning
	// `nself build` produces byte-equal SQL on the same install set, so
	// docker-compose volume hashes don't churn.
	if _, err := GenerateNpPluginsSeed(st.workdir, DefaultPluginDir()); err != nil {
		return fmt.Errorf("generating np_plugins seed: %w", err)
	}
	st.filesGenerated++

	// ── Step 7.6: Auto-enable Redis when a BullMQ plugin is installed ─
	// If Redis is not explicitly enabled but a plugin that needs it (ai, claw,
	// mux, cron, notify, push) is installed, set cfg.Redis.Enabled = true so
	// that both DetectServices and compose.Generator emit a redis block.
	// Skipped when opts.NoAutoRedis is set (--no-auto-redis flag).
	if !st.cfg.Redis.Enabled && !st.opts.NoAutoRedis && ShouldAutoEnableRedis(DefaultPluginDir()) {
		st.cfg.Redis.Enabled = true
		slog.Info("Redis auto-enabled", "reason", "BullMQ-backed plugin detected (ai, claw, mux, cron, notify, or push)", "disable_flag", "--no-auto-redis")
	}

	return nil
}
