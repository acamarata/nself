package build

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/compose"
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

	// ── Step 8: Generate docker-compose.yml ─────────────────────────
	composeGen := compose.NewGenerator(cfg)
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
			fmt.Println("Ollama sidecar injected into docker-compose.yml (AI_OLLAMA_ENABLED=true)")
		}
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
	pluginDir := DefaultPluginDir()
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
		fmt.Printf("Discovered %d plugin compose file(s):\n", len(pluginComposeFiles))
		for _, pf := range pluginComposeFiles {
			fmt.Printf("  - %s\n", pf)
		}
	}

	// ── Step 9.6: Verify Go plugin Dockerfiles have HEALTHCHECK ─────
	for _, w := range CheckGoPluginDockerfiles(pluginDir) {
		fmt.Printf("warning: %s\n", w)
	}

	// ── Step 10: Write .env.computed ────────────────────────────────
	pluginEnvVars := ComputePluginEnvVars(workdir, pluginDir)
	computedPath := filepath.Join(workdir, ".env.computed")
	computedContent := buildEnvComputed(cfg, pluginEnvVars)
	if err := os.WriteFile(computedPath, []byte(computedContent), 0600); err != nil {
		return nil, fmt.Errorf("writing .env.computed: %w", err)
	}
	filesGenerated++

	// ── Step 11: Save build version to .nself/build-version ─────────
	versionPath := filepath.Join(workdir, buildVersionFile)
	if err := os.WriteFile(versionPath, []byte(version.GetVersion()), 0644); err != nil {
		return nil, fmt.Errorf("writing build version: %w", err)
	}
	filesGenerated++

	// ── Step 11.5: Post-build validation ────────────────────────────
	nginxSitesDir := filepath.Join(workdir, "nginx", "sites")
	pvResult := PostValidate(composePath, nginxSitesDir)

	// Print warnings — they do not fail the build.
	for _, w := range pvResult.Warnings {
		fmt.Printf("warning: %s\n", w)
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
		CAInstalled:        sslResult.CAInstalled,
		CAManualCmd:        sslResult.CAManualCmd,
		HostsAdded:         sslResult.HostsAdded,
		HostsManualNote:    sslResult.HostsManualNote,
	}, nil
}

// persistGeneratedSecrets checks whether key secrets exist in any .env file on
// disk. If a secret was auto-generated by ApplyDefaults (i.e. it is set in cfg
// but absent from all .env files), it is appended to .env.secrets so subsequent
// builds and restarts reuse the same value.
//
// HASURA_GRAPHQL_JWT_SECRET is a special case: Hasura is always a core service
// in nSelf, so if no JWT secret is on disk we synthesize the full JSON via
// config.BuildJWTSecret and persist it.
func persistGeneratedSecrets(workdir string, cfg *config.Config) error {
	type secretEntry struct {
		envKey string
		value  string
	}

	candidates := []secretEntry{
		{"PLUGIN_INTERNAL_SECRET", cfg.PluginSystem.InternalSecret},
		{"NOTIFY_INTERNAL_SECRET", cfg.PluginConfig.NotifySecret},
		{"CRON_INTERNAL_SECRET", cfg.PluginConfig.CronSecret},
	}

	// Collect env keys already present in any .env file on disk.
	onDisk := make(map[string]bool)
	for _, name := range []string{".env", ".env.dev", ".env.staging", ".env.prod", ".env.secrets", ".env.local"} {
		p := filepath.Join(workdir, name)
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || line[0] == '#' {
				continue
			}
			if idx := strings.IndexByte(line, '='); idx > 0 {
				onDisk[line[:idx]] = true
			}
		}
		f.Close()
	}

	// Determine which secrets need persisting.
	var toWrite []secretEntry
	for _, s := range candidates {
		if s.value != "" && !onDisk[s.envKey] {
			toWrite = append(toWrite, s)
		}
	}

	// HASURA_GRAPHQL_JWT_SECRET: Hasura is a core service, so if the JWT
	// secret is absent from every env file on disk, synthesize and persist it.
	// BuildJWTSecret auto-generates a crypto/rand key and returns the full
	// {"type":"HS256","key":"..."} JSON that Hasura expects.
	if !onDisk["HASURA_GRAPHQL_JWT_SECRET"] {
		jwtJSON, err := config.BuildJWTSecret(cfg)
		if err != nil {
			return fmt.Errorf("building JWT secret for persistence: %w", err)
		}
		if jwtJSON != "" {
			toWrite = append(toWrite, secretEntry{"HASURA_GRAPHQL_JWT_SECRET", jwtJSON})
		}
	}

	if len(toWrite) == 0 {
		return nil
	}

	// Append to .env.secrets. OpenFile with 0600 sets the mode only if the
	// file is newly created; an explicit Chmod after ensures owner-only
	// permissions even when the file already exists with a looser mode.
	secretsPath := filepath.Join(workdir, ".env.secrets")
	f, err := os.OpenFile(secretsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening %s: %w", secretsPath, err)
	}
	defer f.Close()

	for _, s := range toWrite {
		if _, err := fmt.Fprintf(f, "%s=%s\n", s.envKey, s.value); err != nil {
			return fmt.Errorf("writing %s: %w", s.envKey, err)
		}
		slog.Info("Persisted auto-generated secret to .env.secrets", "key", s.envKey)
	}

	// Enforce 0600 unconditionally — covers the case where the file existed
	// with 0644 before we appended to it.
	if err := os.Chmod(secretsPath, 0600); err != nil {
		return fmt.Errorf("chmod %s: %w", secretsPath, err)
	}

	return nil
}

// buildNginxRoutes collects all nginx routes that will be generated for the
// given config. The list is used for preflight conflict detection via
// nginx.HasDomainConflict before any files are written.
//
// The server_name for each route is route + "." + baseDomain, matching the
// logic in the nginx Generator. Location is always "/" in current templates.
func buildNginxRoutes(cfg *config.Config) []nginx.NginxRoute {
	bd := cfg.BaseDomain
	loc := "/"
	var routes []nginx.NginxRoute

	// Core: hasura
	hasuraRoute := cfg.Hasura.Route
	if hasuraRoute == "" {
		hasuraRoute = "api"
	}
	routes = append(routes, nginx.NginxRoute{ServerName: hasuraRoute + "." + bd, Location: loc})

	// Core: auth
	authRoute := cfg.Auth.Route
	if authRoute == "" {
		authRoute = "auth"
	}
	routes = append(routes, nginx.NginxRoute{ServerName: authRoute + "." + bd, Location: loc})

	// Core: storage + storage-console (when MinIO enabled)
	if cfg.Minio.Enabled {
		storageRoute := cfg.Minio.StorageRoute
		if storageRoute == "" {
			storageRoute = "storage"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: storageRoute + "." + bd, Location: loc})

		consoleRoute := cfg.Minio.ConsoleRoute
		if consoleRoute == "" {
			consoleRoute = "storage-console"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: consoleRoute + "." + bd, Location: loc})
	}

	// Optional: admin
	if cfg.Admin.Enabled {
		adminRoute := cfg.Admin.Route
		if adminRoute == "" {
			adminRoute = "admin"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: adminRoute + "." + bd, Location: loc})
	}

	// Optional: search
	if cfg.Search.Enabled {
		searchRoute := cfg.Search.Route
		if searchRoute == "" {
			searchRoute = "search"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: searchRoute + "." + bd, Location: loc})
	}

	// Optional: mail
	if cfg.Mailpit.Enabled {
		mailRoute := cfg.Mailpit.Route
		if mailRoute == "" {
			mailRoute = "mail"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: mailRoute + "." + bd, Location: loc})
	}

	// Optional: functions
	if cfg.Functions.Enabled {
		fnRoute := cfg.Functions.Route
		if fnRoute == "" {
			fnRoute = "functions"
		}
		routes = append(routes, nginx.NginxRoute{ServerName: fnRoute + "." + bd, Location: loc})
	}

	// Custom services (CS_1..CS_10)
	for _, cs := range cfg.CustomServices {
		if cs.Route == "" || cs.Port == 0 {
			continue
		}
		routes = append(routes, nginx.NginxRoute{ServerName: cs.Route + "." + bd, Location: loc})
	}

	// Frontend apps
	for _, fe := range cfg.FrontendApps {
		if fe.Route == "" || fe.Port == 0 {
			continue
		}
		routes = append(routes, nginx.NginxRoute{ServerName: fe.Route + "." + bd, Location: loc})
	}

	// Internal routes
	for _, ir := range cfg.InternalRoutes {
		if ir.Subdomain == "" || ir.Target == "" {
			continue
		}
		routes = append(routes, nginx.NginxRoute{ServerName: ir.Subdomain + "." + bd, Location: loc})
	}

	return routes
}

// buildEnvComputed generates the .env.computed file content with computed
// values that other tools (e.g., Hasura migrations) depend on. The extra
// map holds additional key=value pairs contributed by plugins or other
// build steps.
func buildEnvComputed(cfg *config.Config, extra map[string]string) string {
	network := cfg.DockerNetwork
	if network == "" {
		network = cfg.ProjectName + "_network"
	}
	content := fmt.Sprintf("# Generated by nself build - DO NOT EDIT MANUALLY\n"+
		"DATABASE_URL=%s\n"+
		"DOCKER_NETWORK=%s\n",
		cfg.DatabaseURL(),
		network,
	)
	for k, v := range extra {
		content += fmt.Sprintf("%s=%s\n", k, v)
	}
	return content
}

// ComposeGeneratedHeader is the canonical marker prepended to every generated
// docker-compose.yml. Pre-commit hooks and auditors grep for this exact line
// to confirm a compose file was produced by `nself build` and not hand-edited.
// See S32-T12 and the PPI nSelf-First Doctrine.
const ComposeGeneratedHeader = "# GENERATED BY nself build - DO NOT HAND EDIT\n" +
	"# Run 'nself build' to regenerate this file.\n"

// prependGeneratedHeader returns composeYAML with ComposeGeneratedHeader
// prefixed. YAML treats `#` as a comment so the resulting file remains valid.
func prependGeneratedHeader(composeYAML []byte) []byte {
	out := make([]byte, 0, len(ComposeGeneratedHeader)+len(composeYAML))
	out = append(out, []byte(ComposeGeneratedHeader)...)
	out = append(out, composeYAML...)
	return out
}
