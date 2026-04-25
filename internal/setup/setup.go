// Package setup implements the nself init wizard: project scaffolding,
// secret generation, .env file creation, and .nself/ directory setup.
package setup

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nself-org/cli/internal/errs"
)

// Options holds all flags and settings for the init command.
type Options struct {
	Fast           bool
	Interactive    bool
	NonInteractive bool
	Template       string
	SkipValidation bool
	Wizard         bool
	Demo           bool
	Full           bool
	Force          bool
	Quiet          bool
	WorkDir        string
	Name           string
	// Domain is the BASE_DOMAIN value. When set (e.g. via --domain flag or
	// interactive wizard), it takes precedence over env vars and defaults.
	Domain string
	// DomainComment is written as a comment above BASE_DOMAIN in the generated
	// .env. It describes the chosen domain pattern for operator clarity.
	DomainComment string
	// NoPgvector skips pgvector extension and RAG scaffold on init (sets
	// PGVECTOR_ENABLED=false). Default (false) enables pgvector.
	NoPgvector bool
}

// Result holds the outcome of a successful init run.
type Result struct {
	ProjectName  string
	BaseDomain   string
	Env          string
	FilesCreated []string
	Demo         bool
}

// projectNameRe validates PROJECT_NAME: lowercase alphanumeric + hyphens,
// must start and end with alphanumeric, 2-30 chars.
var projectNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// Initialize runs the full init flow: validate inputs, generate secrets,
// write .env files, create .nself/ directory, and return a Result.
func Initialize(opts Options) (*Result, error) {
	if opts.WorkDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("determining working directory: %w", err)
		}
		opts.WorkDir = wd
	}

	// Check for existing .env unless --force.
	envPath := filepath.Join(opts.WorkDir, ".env")
	if !opts.Force {
		if _, err := os.Stat(envPath); err == nil {
			return nil, fmt.Errorf(".env already exists in %s — use --force to overwrite", opts.WorkDir)
		}
	}

	// Determine project name: env var > directory name > default.
	projectName := resolveProjectName(opts)
	if !opts.SkipValidation {
		if err := validateProjectName(projectName); err != nil {
			return nil, err
		}
	}

	// Determine environment and domain.
	env := resolveEnv(opts)
	baseDomain := resolveBaseDomain(opts, env)

	// Generate secrets.
	postgresPassword, err := GenerateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating postgres password: %w", err)
	}
	hasuraAdminSecret, err := GenerateSecret(44)
	if err != nil {
		return nil, fmt.Errorf("generating hasura admin secret: %w", err)
	}
	hasuraJWTKey, err := GenerateSecret(44)
	if err != nil {
		return nil, fmt.Errorf("generating hasura JWT key: %w", err)
	}

	// Generate a secure MinIO password (never use the known-bad defaults).
	minioPassword, err := resolveMinioPassword("")
	if err != nil {
		return nil, fmt.Errorf("generating minio password: %w", err)
	}

	// Generate plugin secrets so they are persisted to .env on first init.
	notifySecret, err := GenerateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating notify secret: %w", err)
	}
	cronSecret, err := GenerateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating cron secret: %w", err)
	}
	pluginInternalSecret, err := GenerateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating plugin internal secret: %w", err)
	}

	// Build .env content.
	pgvectorEnabled := !opts.NoPgvector
	var envContent string
	if opts.Demo {
		envContent = buildDemoEnv(projectName, baseDomain, env, postgresPassword, hasuraAdminSecret, hasuraJWTKey, minioPassword, notifySecret, cronSecret, pluginInternalSecret, opts.DomainComment, pgvectorEnabled)
	} else {
		envContent = buildStandardEnv(projectName, baseDomain, env, postgresPassword, hasuraAdminSecret, hasuraJWTKey, notifySecret, cronSecret, pluginInternalSecret, opts.DomainComment, pgvectorEnabled)
	}

	// Write .env.
	if err := writeFile(envPath, envContent, 0600); err != nil {
		return nil, fmt.Errorf("writing .env: %w", err)
	}
	filesCreated := []string{".env"}

	// Write .env.example (safe to commit, no secrets).
	examplePath := filepath.Join(opts.WorkDir, ".env.example")
	exampleContent := buildExampleEnv(projectName, baseDomain, env, opts.Demo)
	if err := writeFile(examplePath, exampleContent, 0644); err != nil {
		return nil, fmt.Errorf("writing .env.example: %w", err)
	}
	filesCreated = append(filesCreated, ".env.example")

	// --full: write environment-specific files.
	if opts.Full {
		fullFiles, err := writeFullEnvFiles(opts.WorkDir, projectName, baseDomain)
		if err != nil {
			return nil, fmt.Errorf("writing environment files: %w", err)
		}
		filesCreated = append(filesCreated, fullFiles...)
	}

	// P88 Sprint 01 T-01-09: write .env.ai (AI tier config + master secret).
	// O_EXCL: preserved across re-runs — the master secret must never rotate.
	if created, err := writeEnvAI(opts.WorkDir); err != nil {
		return nil, fmt.Errorf("writing .env.ai: %w", err)
	} else if created {
		filesCreated = append(filesCreated, ".env.ai")
	}

	// Append to .gitignore.
	if err := ensureGitignore(opts.WorkDir); err != nil {
		return nil, fmt.Errorf("updating .gitignore: %w", err)
	}
	filesCreated = append(filesCreated, ".gitignore")

	// Create .nself/ working directory.
	nelfDir := filepath.Join(opts.WorkDir, ".nself")
	if err := os.MkdirAll(nelfDir, 0755); err != nil {
		return nil, fmt.Errorf("creating .nself directory: %w", err)
	}

	return &Result{
		ProjectName:  projectName,
		BaseDomain:   baseDomain,
		Env:          env,
		FilesCreated: filesCreated,
		Demo:         opts.Demo,
	}, nil
}

// resolveProjectName picks the project name from flag, env, directory, or default.
func resolveProjectName(opts Options) string {
	if opts.Name != "" {
		return opts.Name
	}
	if v := os.Getenv("PROJECT_NAME"); v != "" {
		return v
	}
	// Use directory name, sanitized.
	dir := filepath.Base(opts.WorkDir)
	name := strings.ToLower(dir)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Strip non-alphanumeric/hyphen chars.
	clean := regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "")
	if clean == "" || len(clean) < 2 {
		return "myproject"
	}
	if len(clean) > 30 {
		clean = clean[:30]
	}
	return clean
}

// validateProjectName checks the name against the required pattern.
func validateProjectName(name string) error {
	if len(name) < 2 || len(name) > 30 {
		return fmt.Errorf("project name must be 2-30 characters, got %d: %q: %w", len(name), name, errs.ErrInvalidProjectName)
	}
	if !projectNameRe.MatchString(name) {
		return fmt.Errorf("project name must be lowercase alphanumeric with hyphens (no leading/trailing hyphen): %q: %w", name, errs.ErrInvalidProjectName)
	}
	return nil
}

// resolveEnv determines the environment from env vars or defaults.
func resolveEnv(opts Options) string {
	if v := os.Getenv("ENV"); v != "" {
		switch strings.ToLower(v) {
		case "production", "prod":
			return "prod"
		case "staging", "stage":
			return "staging"
		default:
			return "dev"
		}
	}
	return "dev"
}

// resolveBaseDomain determines the base domain.
// Priority: opts.Domain (flag/wizard) > BASE_DOMAIN env var > default.
func resolveBaseDomain(opts Options, env string) string {
	if opts.Domain != "" {
		return opts.Domain
	}
	if v := os.Getenv("BASE_DOMAIN"); v != "" {
		return v
	}
	if env == "dev" {
		return "local.nself.org"
	}
	return "localhost"
}

// GenerateSecret returns a crypto/rand base64url string of the given length,
// with URL-unsafe characters stripped. Suitable for passwords and API keys.
func GenerateSecret(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(b)
	// Strip characters that can cause issues in shell/.env.
	encoded = strings.NewReplacer("+", "", "/", "", "=", "").Replace(encoded)
	if len(encoded) < length {
		return encoded, nil
	}
	return encoded[:length], nil
}

// generateSecurePassword returns a cryptographically random alphanumeric
// password of the given length using crypto/rand.
func generateSecurePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

// knownBadMinioPasswords is the set of well-known default MinIO credentials
// that must be replaced with a secure random password on init.
var knownBadMinioPasswords = map[string]bool{
	"minioadmin": true,
	"minio123":   true,
	"admin":      true,
}

// resolveMinioPassword returns a secure MinIO password. If the provided
// value is empty or a known-bad default, a new 32-char random alphanumeric
// password is generated. Any other value (user-set) is returned unchanged.
func resolveMinioPassword(current string) (string, error) {
	if current == "" || knownBadMinioPasswords[current] {
		return generateSecurePassword(32)
	}
	return current, nil
}

// WriteEnvFile writes env file content to path with restrictive 0600
// permissions (owner read/write only). Parent directories are created
// as needed.
func WriteEnvFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory for env file: %w", err)
	}
	if err := os.WriteFile(path, content, 0600); err != nil {
		return fmt.Errorf("write env file %s: %w", path, err)
	}
	return nil
}

// EnsureEnvFilePermissions fixes the permissions of an existing env file to
// 0600 if they are more permissive. It is a no-op if the file does not exist.
func EnsureEnvFilePermissions(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat env file %s: %w", path, err)
	}
	if info.Mode().Perm() != 0600 {
		if err := os.Chmod(path, 0600); err != nil {
			return fmt.Errorf("chmod env file %s: %w", path, err)
		}
	}
	return nil
}

// buildDomainLine builds the BASE_DOMAIN line with an optional preceding comment.
func buildDomainLine(baseDomain, domainComment string) string {
	if domainComment != "" {
		return domainComment + "\nBASE_DOMAIN=" + baseDomain
	}
	return "BASE_DOMAIN=" + baseDomain
}

// buildStandardEnv generates the default .env content.
func buildStandardEnv(projectName, baseDomain, env, pgPass, hasuraSecret, jwtKey, notifySecret, cronSecret, pluginInternalSecret, domainComment string, pgvectorEnabled bool) string {
	pgvectorVal := "true"
	if !pgvectorEnabled {
		pgvectorVal = "false"
	}
	return fmt.Sprintf(`# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# nSelf Project Configuration
# Generated by: nself init
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# ── Core ──────────────────────────────────────────────────
PROJECT_NAME=%s
%s
ENV=%s

# ── PostgreSQL ────────────────────────────────────────────
POSTGRES_VERSION=16-alpine
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=%s
POSTGRES_USER=postgres
POSTGRES_PASSWORD=%s

# ── pgvector (RAG / vector search) ───────────────────────
PGVECTOR_ENABLED=%s
PGVECTOR_DIMENSIONS=1536
PGVECTOR_HNSW_M=16
PGVECTOR_HNSW_EF_CONSTRUCTION=64

# ── Hasura GraphQL ────────────────────────────────────────
HASURA_VERSION=v2.44.0
HASURA_GRAPHQL_ADMIN_SECRET=%s
HASURA_JWT_KEY=%s
HASURA_JWT_TYPE=HS256

# ── Auth ──────────────────────────────────────────────────
AUTH_ENABLED=true
AUTH_VERSION=0.36.0

# ── Nginx ─────────────────────────────────────────────────
NGINX_VERSION=alpine

# ── Optional Services (enable as needed) ──────────────────
REDIS_ENABLED=false
MINIO_ENABLED=false
MAILPIT_ENABLED=false
MONITORING_ENABLED=false
FUNCTIONS_ENABLED=false
ADMIN_ENABLED=false
SEARCH_ENABLED=false

# ── Plugin Secrets (auto-generated) ──────────────────────
NOTIFY_INTERNAL_SECRET=%s
CRON_INTERNAL_SECRET=%s
PLUGIN_INTERNAL_SECRET=%s
`, projectName, buildDomainLine(baseDomain, domainComment), env, projectName, pgPass, pgvectorVal, hasuraSecret, jwtKey, notifySecret, cronSecret, pluginInternalSecret)
}

// buildDemoEnv generates .env for --demo mode with all services enabled.
func buildDemoEnv(projectName, baseDomain, env, pgPass, hasuraSecret, jwtKey, minioPassword, notifySecret, cronSecret, pluginInternalSecret, domainComment string, pgvectorEnabled bool) string {
	pgvectorVal := "true"
	if !pgvectorEnabled {
		pgvectorVal = "false"
	}
	_ = pgvectorVal // used in format string below
	return fmt.Sprintf(`# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# nSelf Project Configuration (Demo Mode)
# Generated by: nself init --demo
# All services enabled with pre-configured defaults.
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# ── Core ──────────────────────────────────────────────────
PROJECT_NAME=%s
%s
ENV=%s

# ── PostgreSQL ────────────────────────────────────────────
POSTGRES_VERSION=16-alpine
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=%s
POSTGRES_USER=postgres
POSTGRES_PASSWORD=%s

# ── Hasura GraphQL ────────────────────────────────────────
HASURA_VERSION=v2.44.0
HASURA_GRAPHQL_ADMIN_SECRET=%s
HASURA_JWT_KEY=%s
HASURA_JWT_TYPE=HS256
HASURA_GRAPHQL_ENABLE_CONSOLE=true
HASURA_GRAPHQL_DEV_MODE=true

# ── Auth ──────────────────────────────────────────────────
AUTH_ENABLED=true
AUTH_VERSION=0.36.0

# ── Nginx ─────────────────────────────────────────────────
NGINX_VERSION=alpine

# ── Redis ─────────────────────────────────────────────────
REDIS_ENABLED=true
REDIS_VERSION=7-alpine

# ── MinIO / Storage ───────────────────────────────────────
MINIO_ENABLED=true
MINIO_ROOT_USER=nself
MINIO_ROOT_PASSWORD=%s

# ── Mailpit (Dev Email) ──────────────────────────────────
MAILPIT_ENABLED=true

# ── Monitoring Stack ──────────────────────────────────────
# Install monitoring plugin: nself plugin install monitoring
MONITORING_ENABLED=false

# ── Functions Runtime ─────────────────────────────────────
FUNCTIONS_ENABLED=true

# ── Admin Dashboard ───────────────────────────────────────
ADMIN_ENABLED=true

# ── Search (MeiliSearch) ──────────────────────────────────
SEARCH_ENABLED=true
SEARCH_ENGINE=meilisearch

# ── Example Custom Services ──────────────────────────────
CS_1_NAME=ping-api
CS_1_FRAMEWORK=node
CS_1_PORT=8001
CS_1_ROUTE=ping

CS_2_NAME=worker
CS_2_FRAMEWORK=node
CS_2_PORT=8002
CS_2_ROUTE=worker

# ── Example Frontend Apps ─────────────────────────────────
FRONTEND_APP_1_NAME=web
FRONTEND_APP_1_PORT=3000
FRONTEND_APP_1_ROUTE=app

FRONTEND_APP_2_NAME=admin-ui
FRONTEND_APP_2_PORT=3001
FRONTEND_APP_2_ROUTE=dashboard

# ── Plugin Secrets (auto-generated) ──────────────────────
NOTIFY_INTERNAL_SECRET=%s
CRON_INTERNAL_SECRET=%s
PLUGIN_INTERNAL_SECRET=%s
`, projectName, buildDomainLine(baseDomain, domainComment), env, projectName, pgPass, hasuraSecret, jwtKey, minioPassword, notifySecret, cronSecret, pluginInternalSecret)
}

// buildExampleEnv generates the .env.example (no real secrets).
func buildExampleEnv(projectName, baseDomain, env string, demo bool) string {
	mode := ""
	if demo {
		mode = " (Demo Mode)"
	}
	return fmt.Sprintf(`# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# nSelf Project Configuration%s
# Copy to .env and fill in your secrets.
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PROJECT_NAME=%s
BASE_DOMAIN=%s
ENV=%s

POSTGRES_PASSWORD=CHANGE_ME
HASURA_GRAPHQL_ADMIN_SECRET=CHANGE_ME
HASURA_JWT_KEY=CHANGE_ME
`, mode, projectName, baseDomain, env)
}

// writeFullEnvFiles creates .env.dev, .env.staging, .env.prod, .env.secrets.
func writeFullEnvFiles(workDir, projectName, baseDomain string) ([]string, error) {
	var created []string

	// .env.dev
	devContent := fmt.Sprintf(`# Development environment overrides
ENV=dev
BASE_DOMAIN=local.nself.org
HASURA_GRAPHQL_ENABLE_CONSOLE=true
HASURA_GRAPHQL_DEV_MODE=true
`)
	if err := writeFile(filepath.Join(workDir, ".env.dev"), devContent, 0600); err != nil {
		return nil, err
	}
	created = append(created, ".env.dev")

	// .env.staging
	stagingContent := fmt.Sprintf(`# Staging environment overrides
ENV=staging
BASE_DOMAIN=staging.%s
HASURA_GRAPHQL_ENABLE_CONSOLE=false
HASURA_GRAPHQL_DEV_MODE=false
`, baseDomain)
	if err := writeFile(filepath.Join(workDir, ".env.staging"), stagingContent, 0600); err != nil {
		return nil, err
	}
	created = append(created, ".env.staging")

	// .env.prod
	prodContent := fmt.Sprintf(`# Production environment overrides
ENV=prod
BASE_DOMAIN=%s
HASURA_GRAPHQL_ENABLE_CONSOLE=false
HASURA_GRAPHQL_DEV_MODE=false
`, baseDomain)
	if err := writeFile(filepath.Join(workDir, ".env.prod"), prodContent, 0600); err != nil {
		return nil, err
	}
	created = append(created, ".env.prod")

	// .env.secrets (600 perms, gitignored)
	pgPass, err := GenerateSecret(32)
	if err != nil {
		return nil, fmt.Errorf("generating postgres password for secrets file: %w", err)
	}
	hasuraSecret, err := GenerateSecret(44)
	if err != nil {
		return nil, fmt.Errorf("generating hasura secret for secrets file: %w", err)
	}
	jwtKey, err := GenerateSecret(44)
	if err != nil {
		return nil, fmt.Errorf("generating JWT key for secrets file: %w", err)
	}
	secretsContent := fmt.Sprintf(`# Secrets — DO NOT COMMIT
# Auto-generated by nself init --full
POSTGRES_PASSWORD=%s
HASURA_GRAPHQL_ADMIN_SECRET=%s
HASURA_JWT_KEY=%s
`, pgPass, hasuraSecret, jwtKey)
	if err := writeFile(filepath.Join(workDir, ".env.secrets"), secretsContent, 0600); err != nil {
		return nil, err
	}
	created = append(created, ".env.secrets")

	return created, nil
}

// gitignoreEntries are appended to .gitignore if not already present.
var gitignoreEntries = []string{
	".env",
	".env.local",
	".env.*.local",
	".env.secrets",
	".env.ai",
	".volumes/",
	"logs/",
	"*.log",
	"node_modules/",
	".DS_Store",
	".nself/",
}

// ensureGitignore creates or appends to .gitignore with required entries.
func ensureGitignore(workDir string) error {
	giPath := filepath.Join(workDir, ".gitignore")
	existing := ""
	if data, err := os.ReadFile(giPath); err == nil {
		existing = string(data)
	}

	var toAdd []string
	for _, entry := range gitignoreEntries {
		if !strings.Contains(existing, entry) {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) == 0 {
		return nil // Nothing to add.
	}

	f, err := os.OpenFile(giPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add a header comment if the file is new or we're appending.
	header := "\n# nSelf\n"
	if existing == "" {
		header = "# nSelf\n"
	}
	if _, err := f.WriteString(header); err != nil {
		return err
	}
	for _, entry := range toAdd {
		if _, err := f.WriteString(entry + "\n"); err != nil {
			return err
		}
	}
	return nil
}

// writeFile writes content to path with the given permissions.
// Parent directories are created as needed.
func writeFile(path, content string, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), perm)
}
