package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// knownEnvVars is the authoritative list of every environment variable name
// that parseEnvToConfig (and ApplyDefaults) reads. Any key present in a .env
// file that is NOT in this list — and does not match a user-defined prefix
// (CS_*, FRONTEND_APP_*, etc.) — triggers a WarnUnknownEnvVars warning so
// users learn about typos early.
//
// Maintainers: keep this list in sync with parseEnvToConfig and defaults.go.
var knownEnvVars = []string{
	// Core
	"PROJECT_NAME",
	"BASE_DOMAIN",
	"PROJECT_DOMAIN",
	"ENV",
	"PROJECT_DESCRIPTION",
	"ADMIN_EMAIL",
	"DB_ENV_SEEDS",

	// PostgreSQL
	"POSTGRES_VERSION",
	"POSTGRES_HOST",
	"POSTGRES_PORT",
	"POSTGRES_DB",
	"POSTGRES_USER",
	"POSTGRES_PASSWORD",
	"POSTGRES_EXTENSIONS",
	"POSTGRES_EXPOSE_PORT",
	"POSTGRES_MEM_LIMIT",
	"POSTGRES_CPU_LIMIT",

	// Hasura
	"HASURA_VERSION",
	"HASURA_GRAPHQL_ADMIN_SECRET",
	"HASURA_JWT_KEY",
	"HASURA_JWT_TYPE",
	"HASURA_GRAPHQL_ENABLE_CONSOLE",
	"HASURA_GRAPHQL_DEV_MODE",
	"HASURA_DEV_MODE",
	"HASURA_GRAPHQL_CORS_DOMAIN",
	"HASURA_GRAPHQL_JWT_SECRET",
	"HASURA_ROUTE",
	"HASURA_PORT",
	"HASURA_MEM_LIMIT",
	"HASURA_CPU_LIMIT",
	"HASURA_GRAPHQL_LOG_LEVEL",

	// Auth
	"AUTH_VERSION",
	"AUTH_PORT",
	"AUTH_CLIENT_URL",
	"AUTH_ACCESS_TOKEN_EXPIRES_IN",
	"AUTH_REFRESH_TOKEN_EXPIRES_IN",
	"AUTH_ROUTE",
	"AUTH_SMTP_HOST",
	"AUTH_SMTP_PORT",
	"AUTH_SMTP_USER",
	"AUTH_SMTP_PASS",
	"AUTH_SMTP_SECURE",
	"AUTH_SMTP_SENDER",
	"AUTH_MEM_LIMIT",
	"AUTH_CPU_LIMIT",
	"AUTH_EXTRA_REDIRECT_URLS",
	"AUTH_RATE_LIMIT",
	"AUTH_WEBAUTHN_ENABLED",
	"AUTH_LOG_LEVEL",

	// Nginx
	"NGINX_VERSION",
	"NGINX_HTTP_PORT",
	"NGINX_PORT",
	"NGINX_HTTPS_PORT",
	"NGINX_SSL_PORT",
	"NGINX_CLIENT_MAX_BODY_SIZE",
	"NGINX_BIND_IP",

	// SSL
	"SSL_MODE",
	"EXTRA_SSL_DOMAINS",

	// Redis
	"REDIS_ENABLED",
	"REDIS_VERSION",
	"REDIS_PORT",
	"REDIS_PASSWORD",
	"REDIS_MEMORY",
	"REDIS_CPU",

	// MinIO / Storage
	"MINIO_ENABLED",
	"STORAGE_ENABLED",
	"MINIO_VERSION",
	"MINIO_PORT",
	"MINIO_CONSOLE_PORT",
	"MINIO_ROOT_USER",
	"MINIO_ROOT_PASSWORD",
	"MINIO_DEFAULT_BUCKETS",
	"MINIO_REGION",
	"S3_ACCESS_KEY",
	"S3_SECRET_KEY",
	"S3_BUCKET",
	"STORAGE_VERSION",
	"STORAGE_ROUTE",
	"STORAGE_CONSOLE_ROUTE",
	"MINIO_MEMORY",
	"MINIO_CPU",

	// Mailpit
	"MAILPIT_ENABLED",
	"MAILPIT_VERSION",
	"MAILPIT_SMTP_PORT",
	"MAILPIT_UI_PORT",
	"MAILPIT_MAX_MESSAGES",
	"MAILPIT_ROUTE",
	"MAIL_ROUTE",
	"MAILPIT_UI_USER",
	"MAILPIT_UI_PASSWORD",

	// Functions
	"FUNCTIONS_ENABLED",
	"FUNCTIONS_VERSION",
	"FUNCTIONS_PORT",
	"FUNCTIONS_ROUTE",

	// MLflow
	"MLFLOW_ENABLED",
	"MLFLOW_VERSION",
	"MLFLOW_PORT",
	"MLFLOW_ROUTE",
	"MLFLOW_DB_NAME",
	"MLFLOW_ARTIFACTS_BUCKET",
	"MLFLOW_AUTH_ENABLED",
	"MLFLOW_AUTH_USERNAME",
	"MLFLOW_AUTH_PASSWORD",

	// Admin
	"NSELF_ADMIN_ENABLED",
	"NSELF_ADMIN_VERSION",
	"NSELF_ADMIN_PORT",
	"NSELF_ADMIN_ROUTE",
	"NSELF_ADMIN_DEV",
	"NSELF_ADMIN_DEV_PORT",
	"ADMIN_SECRET_KEY",
	"ADMIN_PASSWORD_HASH",

	// Search
	"SEARCH_ENABLED",
	"SEARCH_ENGINE",
	"SEARCH_PROVIDER",
	"SEARCH_PORT",
	"SEARCH_API_KEY",
	"SEARCH_ROUTE",
	"SEARCH_INDEX_PREFIX",
	"SEARCH_AUTO_INDEX",
	"SEARCH_LANGUAGE",
	"MEILISEARCH_VERSION",
	"MEILISEARCH_MASTER_KEY",
	"MEILISEARCH_ENV",
	"MEILI_ENV",
	"TYPESENSE_VERSION",
	"TYPESENSE_API_KEY",
	"TYPESENSE_ENABLE_CORS",
	"TYPESENSE_LOG_LEVEL",
	"TYPESENSE_NUM_MEMORY_SHARDS",
	"TYPESENSE_SNAPSHOT_INTERVAL_SECONDS",
	"ELASTICSEARCH_VERSION",
	"ELASTICSEARCH_PORT",
	"ELASTICSEARCH_PASSWORD",
	"ELASTICSEARCH_MEMORY",

	// Monitoring
	"MONITORING_ENABLED",
	"PROMETHEUS_ENABLED",
	"PROMETHEUS_PORT",
	"GRAFANA_ENABLED",
	"GRAFANA_PORT",
	"GRAFANA_ADMIN_USER",
	"GRAFANA_ADMIN_PASSWORD",
	"GRAFANA_ROUTE",
	"LOKI_ENABLED",
	"LOKI_PORT",
	"PROMTAIL_ENABLED",
	"TEMPO_ENABLED",
	"TEMPO_PORT",
	"ALERTMANAGER_ENABLED",
	"ALERTMANAGER_PORT",
	"CADVISOR_ENABLED",
	"CADVISOR_PORT",
	"NODE_EXPORTER_ENABLED",
	"NODE_EXPORTER_PORT",
	"POSTGRES_EXPORTER_ENABLED",
	"POSTGRES_EXPORTER_PORT",
	"REDIS_EXPORTER_ENABLED",
	"REDIS_EXPORTER_PORT",

	// Email
	"EMAIL_PROVIDER",
	"EMAIL_FROM",
	"ELASTIC_EMAIL_API_KEY",
	"ELASTIC_EMAIL_ACCOUNT_EMAIL",
	"SENDGRID_API_KEY",
	"POSTMARK_API_KEY",
	"MAILGUN_API_KEY",
	"MAILGUN_DOMAIN",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_REGION",
	"SMTP_HOST",
	"SMTP_PORT",
	"SMTP_USER",
	"SMTP_PASS",
	"SMTP_SECURE",

	// Backup
	"BACKUP_ENABLED",
	"BACKUP_DIR",
	"BACKUP_SCHEDULE",
	"BACKUP_RETENTION_DAYS",
	"BACKUP_CLOUD_PROVIDER",

	// Plugin Pro
	"NOTIFY_INTERNAL_SECRET",
	"NOTIFY_PORT",
	"NOTIFY_VAPID_PUBLIC_KEY",
	"NOTIFY_VAPID_PRIVATE_KEY",
	"NOTIFY_ROUTE",
	"CRON_INTERNAL_SECRET",
	"CRON_PORT",
	"CRON_RETENTION_DAYS",
	"PLUGIN_AI_MEMORY_LIMIT",
	"PLUGIN_AI_CPU_LIMIT",
	"PLUGIN_MUX_MEMORY_LIMIT",
	"PLUGIN_MUX_CPU_LIMIT",
	"PLUGIN_CLAW_MEMORY_LIMIT",
	"PLUGIN_CLAW_CPU_LIMIT",
	"PLUGIN_DEFAULT_MEMORY_LIMIT",
	"PLUGIN_DEFAULT_CPU_LIMIT",
	"PLUGIN_INTERNAL_SECRET",

	// Plugin System
	"NSELF_PLUGIN_DIR",
	"NSELF_PLUGIN_CACHE",
	"NSELF_PLUGIN_REGISTRY",
	"NSELF_REGISTRY_CACHE_TTL",
	"NSELF_PLUGIN_LICENSE_KEY",
	"NSELF_LICENSE_SKIP_VERIFY",
	"NSELF_PING_API_URL",
	"NSELF_PRICING_URL",

	// Docker
	"DOCKER_NETWORK",
	"DOCKER_LOG_MAX_SIZE",
	"DOCKER_LOG_MAX_FILE",
	"DOCKER_STOP_GRACE_PERIOD",
	"NSELF_DOCKER_BUILD_TIMEOUT",

	// Start/Stop
	"NSELF_START_MODE",
	"NSELF_HEALTH_CHECK_TIMEOUT",
	"NSELF_HEALTH_CHECK_INTERVAL",
	"NSELF_HEALTH_CHECK_REQUIRED",
	"NSELF_CLEANUP_ON_START",
	"NSELF_ALLOW_EXPOSED_PORTS",
	"NSELF_PARALLEL_LIMIT",
	"NSELF_LOG_LEVEL",
	"NSELF_SKIP_HEALTH_CHECKS",
	"NSELF_STOP_TIMEOUT",

	// Plugin-managed: compose-injected vars that users may set in .env.
	// The CLI loader does not read these; they are listed here only to suppress
	// false "unknown env var" warnings from WarnUnknownEnvVars.
	// Auth service (nHost auth container) — passed through compose template.
	"AUTH_HOST",
	"AUTH_SERVER_URL",
	"AUTH_JWT_SECRET",
	"AUTH_REFRESH_TOKEN_SECRET",
	"AUTH_ACCESS_TOKEN_EXPIRY",
	"AUTH_REFRESH_TOKEN_EXPIRY",
	"AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED",
	// Hasura container — passed through compose template.
	"HASURA_GRAPHQL_ENABLE_TELEMETRY",
	"HASURA_GRAPHQL_UNAUTHORIZED_ROLE",
	"HASURA_CONSOLE_PORT",
	"HASURA_GRAPHQL_JWT_SECRET",
	"HASURA_GRAPHQL_DATABASE_URL",
	"HASURA_METADATA_DATABASE_URL",
	// Nginx compose template vars.
	"NGINX_GZIP_ENABLED",
	"NGINX_MODE",
	"NGINX_MEM_LIMIT",
	// MinIO/Storage — compose-computed.
	"S3_ENDPOINT",
	"STORAGE_PORT",
	"FILES_ROUTE",
	// nSelf Admin container.
	"NSELF_ADMIN_USER",
	"NSELF_ADMIN_PASSWORD",
	// Typesense search provider (partial: TYPESENSE_PORT/ROUTE not in knownEnvVars struct).
	"TYPESENSE_PORT",
	"TYPESENSE_ROUTE",
	// Notify plugin.
	"NOTIFY_VAPID_SUBJECT",
	// Docker Compose runtime vars (exported by shell wrapper, not read by loader).
	"COMPOSE_PROJECT_NAME",
	"DOCKER_BUILDKIT",
	// Phase 14 CLI-command vars (read by cmd handlers, not by loader).
	"NSELF_AUTO_TRUST_CA",
	"NSELF_AUTO_HOSTS_ENTRIES",
	"NSELF_MKCERT_CAROOT",
	"NSELF_NO_MONOREPO",
	// CLI tool behavior vars (read by main binary, not by loader).
	"DEBUG",
	"NO_COLOR",
	// Postgres internal port (documentation-only; always 5432).
	"POSTGRES_INTERNAL_PORT",
	// MeiliSearch search engine (plugin-managed: injected into search compose template).
	"MEILISEARCH_ENABLED",
	"MEILISEARCH_PORT",
	"MEILISEARCH_ROUTE",
	"MEILI_NO_ANALYTICS",
	// OpenSearch search provider (plugin-managed: opensearch plugin compose template).
	"OPENSEARCH_VERSION",
	"OPENSEARCH_PORT",
	"OPENSEARCH_PASSWORD",
	"OPENSEARCH_MEMORY",
	// Zinc search provider (plugin-managed: zinc plugin compose template).
	"ZINC_VERSION",
	"ZINC_PORT",
	"ZINC_ADMIN_USER",
	"ZINC_ADMIN_PASSWORD",
	// Sonic search provider (plugin-managed: sonic plugin compose template).
	"SONIC_VERSION",
	"SONIC_PORT",
	"SONIC_PASSWORD",
	// Dashboard plugin (plugin-managed: dashboard plugin compose template).
	"DASHBOARD_ENABLED",
	"DASHBOARD_VERSION",
	"DASHBOARD_ROUTE",
	"DASHBOARD_PORT",
	// Legacy microservice system (plugin-managed: pre-CS_N system; may appear in old .env files).
	"SERVICES_ENABLED",
	"NESTJS_ENABLED",
	"NESTJS_SERVICES",
	"NESTJS_USE_TYPESCRIPT",
	"NESTJS_PORT_START",
	"BULLMQ_ENABLED",
	"BULLMQ_WORKERS",
	"BULLMQ_DASHBOARD_ENABLED",
	"BULLMQ_DASHBOARD_PORT",
	"BULLMQ_DASHBOARD_ROUTE",
	"GOLANG_ENABLED",
	"GOLANG_SERVICES",
	"GOLANG_PORT_START",
	"PYTHON_ENABLED",
	"PYTHON_SERVICES",
	"PYTHON_FRAMEWORK",
	"PYTHON_PORT_START",
	// Plugin integration vars (plugin-managed: stripe, github, shopify plugin compose templates).
	"STRIPE_API_KEY",
	"STRIPE_WEBHOOK_SECRET",
	"STRIPE_SYNC_INTERVAL",
	"GITHUB_TOKEN",
	"GITHUB_WEBHOOK_SECRET",
	"GITHUB_ORG",
	"GITHUB_REPOS",
	"SHOPIFY_STORE",
	"SHOPIFY_ACCESS_TOKEN",
	"SHOPIFY_API_VERSION",
	"SHOPIFY_WEBHOOK_SECRET",
	"SHOPIFY_SYNC_INTERVAL",
}

// Load reads the .env cascade from projectDir, populates a Config struct from
// os.Getenv, applies smart defaults, and returns the complete configuration.
//
// Cascade order (later overrides earlier):
//
//	.env.dev → .env.{ENV} → .env.secrets → .env.local → .env
//
// Each file is optional. Missing files are silently skipped.
func Load(projectDir string) (*Config, error) {
	// 1. Detect ENV first (needed to pick the correct .env.{ENV} file).
	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}
	env = normalizeEnv(env)

	// 2. Build the file cascade.
	files := []string{
		filepath.Join(projectDir, ".env.dev"),
	}
	switch env {
	case "staging":
		files = append(files, filepath.Join(projectDir, ".env.staging"))
	case "prod":
		files = append(files, filepath.Join(projectDir, ".env.prod"))
	}
	files = append(files,
		filepath.Join(projectDir, ".env.secrets"),
		filepath.Join(projectDir, ".env.local"),
		filepath.Join(projectDir, ".env"),
	)

	// 3. Load each file (skip if not exists, later overrides earlier).
	// Simultaneously collect all keys that appear in any .env file so we can
	// warn about unknown vars after loading is complete.
	const maxEnvFileSize = 1 << 20 // 1 MB
	loadedFromFiles := make(map[string]string)
	for _, f := range files {
		info, statErr := os.Stat(f)
		if statErr != nil {
			continue // file doesn't exist, skip
		}
		if info.Size() > maxEnvFileSize {
			return nil, fmt.Errorf("config file too large (max 1MB): %s", f)
		}
		contents, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}
		if pos := bytes.IndexByte(contents, 0); pos >= 0 {
			return nil, fmt.Errorf("invalid .env file: contains null byte at position %d in %s", pos, f)
		}
		if err := godotenv.Overload(f); err != nil {
			return nil, fmt.Errorf("loading %s: %w", f, err)
		}
		// godotenv.Read parses the file without touching os.Environ, giving
		// us the raw key set to check for unknown vars.
		if m, err := godotenv.Read(f); err == nil {
			for k, v := range m {
				loadedFromFiles[k] = v
			}
		}
	}

	// Warn about env var names that are not in the known schema.
	// Warnings go to stderr via slog.Warn and never cause a non-zero exit.
	warnUnknownEnvVars(loadedFromFiles, knownEnvVars)

	// 4. Parse os.Environ into Config struct.
	cfg := parseEnvToConfig()

	// 5. Parse dynamic collections.
	customServices, err := parseCustomServices()
	if err != nil {
		return nil, err
	}
	cfg.CustomServices = customServices
	frontendApps, err := parseFrontendApps()
	if err != nil {
		return nil, err
	}
	cfg.FrontendApps = frontendApps
	remoteSchemas, err := parseRemoteSchemas()
	if err != nil {
		return nil, err
	}
	cfg.RemoteSchemas = remoteSchemas
	cfg.InternalRoutes = parseInternalRoutes()

	// 6. Collect passthrough vars (AUTH_PROVIDER_*, REMOTE_SCHEMA_*, HASURA_EXTRA_*).
	cfg.Passthrough = collectPassthrough(os.Environ())

	// 7. Apply smart defaults (fills every unset field).
	cfg, err = ApplyDefaults(cfg)
	if err != nil {
		return nil, fmt.Errorf("applying defaults: %w", err)
	}

	// 8. Sanitize user-supplied name and domain after defaults.
	if cfg.ProjectName != "" {
		sanitized, err := SanitizeName(cfg.ProjectName)
		if err != nil {
			return nil, fmt.Errorf("PROJECT_NAME: %w", err)
		}
		if sanitized != cfg.ProjectName {
			slog.Warn("PROJECT_NAME sanitized for Docker compatibility",
				"original", cfg.ProjectName,
				"sanitized", sanitized,
			)
		}
		cfg.ProjectName = sanitized
	}
	if cfg.BaseDomain != "" {
		sanitized, err := SanitizeDomain(cfg.BaseDomain)
		if err != nil {
			return nil, fmt.Errorf("BASE_DOMAIN: %w", err)
		}
		cfg.BaseDomain = sanitized
	}

	return cfg, nil
}

// parseEnvToConfig reads every Config field from os.Getenv using the helper
// functions (getEnvOr, getEnvInt, getEnvBool). This is the single place that
// maps environment variable names to struct fields.
func parseEnvToConfig() *Config {
	cfg := &Config{}

	// ── Core ─────────────────────────────────────────────────────────
	cfg.ProjectName = os.Getenv("PROJECT_NAME")
	cfg.BaseDomain = os.Getenv("BASE_DOMAIN")
	if cfg.BaseDomain == "" {
		cfg.BaseDomain = os.Getenv("PROJECT_DOMAIN")
	}
	cfg.Env = normalizeEnv(getEnvOr("ENV", "dev"))
	cfg.ProjectDescription = os.Getenv("PROJECT_DESCRIPTION")
	cfg.AdminEmail = os.Getenv("ADMIN_EMAIL")
	cfg.DBEnvSeeds = getEnvBool("DB_ENV_SEEDS", true)

	// ── PostgreSQL ───────────────────────────────────────────────────
	cfg.Postgres = PostgresConfig{
		Version:    os.Getenv("POSTGRES_VERSION"),
		Host:       os.Getenv("POSTGRES_HOST"),
		Port:       getEnvInt("POSTGRES_PORT", 0),
		DB:         os.Getenv("POSTGRES_DB"),
		User:       os.Getenv("POSTGRES_USER"),
		Password:   os.Getenv("POSTGRES_PASSWORD"),
		Extensions: parseExtensionList(getEnvOr("POSTGRES_EXTENSIONS", "uuid-ossp,pgcrypto,pg_trgm")),
		ExposePort: os.Getenv("POSTGRES_EXPOSE_PORT"),
		MemLimit:   os.Getenv("POSTGRES_MEM_LIMIT"),
		CPULimit:   os.Getenv("POSTGRES_CPU_LIMIT"),
	}

	// ── Hasura ───────────────────────────────────────────────────────
	cfg.Hasura = HasuraConfig{
		Version:     os.Getenv("HASURA_VERSION"),
		AdminSecret: os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET"),
		JWTKey:      os.Getenv("HASURA_JWT_KEY"),
		JWTType:     os.Getenv("HASURA_JWT_TYPE"),
		Console:     getEnvBool("HASURA_GRAPHQL_ENABLE_CONSOLE", false),
		DevMode:     getEnvBool("HASURA_GRAPHQL_DEV_MODE", false),
		CORSDomain:  os.Getenv("HASURA_GRAPHQL_CORS_DOMAIN"),
		Route:       os.Getenv("HASURA_ROUTE"),
		Port:        getEnvInt("HASURA_PORT", 0),
		MemLimit:    os.Getenv("HASURA_MEM_LIMIT"),
		CPULimit:    os.Getenv("HASURA_CPU_LIMIT"),
		LogLevel:    os.Getenv("HASURA_GRAPHQL_LOG_LEVEL"),
	}
	// HASURA_DEV_MODE backward-compat alias: v1 used HASURA_DEV_MODE, v2 uses HASURA_GRAPHQL_DEV_MODE.
	// Only apply alias if HASURA_GRAPHQL_DEV_MODE was not explicitly set.
	if alias := os.Getenv("HASURA_DEV_MODE"); alias != "" {
		if _, explicitly := os.LookupEnv("HASURA_GRAPHQL_DEV_MODE"); !explicitly {
			cfg.Hasura.DevMode = alias == "true" || alias == "1" || alias == "yes"
		}
	}

	// ── Auth ─────────────────────────────────────────────────────────
	cfg.Auth = AuthConfig{
		Version:            os.Getenv("AUTH_VERSION"),
		Port:               getEnvInt("AUTH_PORT", 0),
		ClientURL:          os.Getenv("AUTH_CLIENT_URL"),
		AccessTokenExpiry:  getEnvInt("AUTH_ACCESS_TOKEN_EXPIRES_IN", 0),
		RefreshTokenExpiry: getEnvInt("AUTH_REFRESH_TOKEN_EXPIRES_IN", 0),
		Route:              os.Getenv("AUTH_ROUTE"),
		SMTPHost:           os.Getenv("AUTH_SMTP_HOST"),
		SMTPPort:           getEnvInt("AUTH_SMTP_PORT", 0),
		SMTPUser:           os.Getenv("AUTH_SMTP_USER"),
		SMTPPass:           os.Getenv("AUTH_SMTP_PASS"),
		SMTPSecure:         getEnvBool("AUTH_SMTP_SECURE", false),
		SMTPSender:         os.Getenv("AUTH_SMTP_SENDER"),
		MemLimit:           os.Getenv("AUTH_MEM_LIMIT"),
		CPULimit:           os.Getenv("AUTH_CPU_LIMIT"),
		ExtraRedirectURLs:  os.Getenv("AUTH_EXTRA_REDIRECT_URLS"),
		WebAuthnEnabled:    getEnvBool("AUTH_WEBAUTHN_ENABLED", false),
		LogLevel:           os.Getenv("AUTH_LOG_LEVEL"),
	}

	// ── Nginx ────────────────────────────────────────────────────────
	cfg.Nginx = NginxConfig{
		Version:       os.Getenv("NGINX_VERSION"),
		HTTPPort:      getEnvInt("NGINX_HTTP_PORT", getEnvInt("NGINX_PORT", 0)),
		SSLPort:       getEnvInt("NGINX_HTTPS_PORT", getEnvInt("NGINX_SSL_PORT", 0)),
		MaxBody:       os.Getenv("NGINX_CLIENT_MAX_BODY_SIZE"),
		BindIP:        os.Getenv("NGINX_BIND_IP"),
		AuthRateLimit: os.Getenv("AUTH_RATE_LIMIT"),
	}

	// ── SSL ──────────────────────────────────────────────────────────
	cfg.SSLMode = os.Getenv("SSL_MODE")
	cfg.ExtraSSLDomains = os.Getenv("EXTRA_SSL_DOMAINS")

	// ── Redis ────────────────────────────────────────────────────────
	cfg.Redis = RedisConfig{
		Enabled:  getEnvBool("REDIS_ENABLED", false),
		Version:  os.Getenv("REDIS_VERSION"),
		Port:     getEnvInt("REDIS_PORT", 0),
		Password: os.Getenv("REDIS_PASSWORD"),
		Memory:   os.Getenv("REDIS_MEMORY"),
		CPU:      os.Getenv("REDIS_CPU"),
	}

	// ── MinIO / Storage ──────────────────────────────────────────────
	// Backward compat: STORAGE_ENABLED=true implies MINIO_ENABLED=true.
	minioEnabled := getEnvBool("MINIO_ENABLED", false) || getEnvBool("STORAGE_ENABLED", false)
	cfg.Minio = MinioConfig{
		Enabled:        minioEnabled,
		Version:        os.Getenv("MINIO_VERSION"),
		Port:           getEnvInt("MINIO_PORT", 0),
		ConsolePort:    getEnvInt("MINIO_CONSOLE_PORT", 0),
		RootUser:       os.Getenv("MINIO_ROOT_USER"),
		RootPassword:   os.Getenv("MINIO_ROOT_PASSWORD"),
		DefaultBuckets: os.Getenv("MINIO_DEFAULT_BUCKETS"),
		Region:         os.Getenv("MINIO_REGION"),
		S3AccessKey:    os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:    os.Getenv("S3_SECRET_KEY"),
		S3Bucket:       os.Getenv("S3_BUCKET"),
		StorageVersion: os.Getenv("STORAGE_VERSION"),
		StorageRoute:   os.Getenv("STORAGE_ROUTE"),
		ConsoleRoute:   os.Getenv("STORAGE_CONSOLE_ROUTE"),
		MemLimit:       os.Getenv("MINIO_MEMORY"),
		CPULimit:       os.Getenv("MINIO_CPU"),
	}

	// ── Mailpit ──────────────────────────────────────────────────────
	cfg.Mailpit = MailpitConfig{
		Enabled:     getEnvBool("MAILPIT_ENABLED", false),
		Version:     os.Getenv("MAILPIT_VERSION"),
		SMTPPort:    getEnvInt("MAILPIT_SMTP_PORT", 0),
		UIPort:      getEnvInt("MAILPIT_UI_PORT", 0),
		MaxMessages: getEnvInt("MAILPIT_MAX_MESSAGES", 0),
		Route:       getEnvOr("MAILPIT_ROUTE", os.Getenv("MAIL_ROUTE")),
		UIUser:      getEnvOr("MAILPIT_UI_USER", "admin"),
		UIPassword:  os.Getenv("MAILPIT_UI_PASSWORD"),
	}

	// ── Functions ────────────────────────────────────────────────────
	cfg.Functions = FunctionsConfig{
		Enabled: getEnvBool("FUNCTIONS_ENABLED", false),
		Version: os.Getenv("FUNCTIONS_VERSION"),
		Port:    getEnvInt("FUNCTIONS_PORT", 0),
		Route:   os.Getenv("FUNCTIONS_ROUTE"),
	}

	// ── MLflow ───────────────────────────────────────────────────────
	cfg.MLflow = MLflowConfig{
		Enabled:         getEnvBool("MLFLOW_ENABLED", false),
		Version:         os.Getenv("MLFLOW_VERSION"),
		Port:            getEnvInt("MLFLOW_PORT", 0),
		Route:           os.Getenv("MLFLOW_ROUTE"),
		DBName:          os.Getenv("MLFLOW_DB_NAME"),
		ArtifactsBucket: os.Getenv("MLFLOW_ARTIFACTS_BUCKET"),
		AuthEnabled:     getEnvBool("MLFLOW_AUTH_ENABLED", false),
		AuthUsername:    os.Getenv("MLFLOW_AUTH_USERNAME"),
		AuthPassword:    os.Getenv("MLFLOW_AUTH_PASSWORD"),
	}

	// ── Admin ────────────────────────────────────────────────────────
	cfg.Admin = AdminConfig{
		Enabled:      getEnvBool("NSELF_ADMIN_ENABLED", false),
		Version:      os.Getenv("NSELF_ADMIN_VERSION"),
		Port:         getEnvInt("NSELF_ADMIN_PORT", 0),
		Route:        os.Getenv("NSELF_ADMIN_ROUTE"),
		DevMode:      getEnvBool("NSELF_ADMIN_DEV", false),
		DevPort:      getEnvInt("NSELF_ADMIN_DEV_PORT", 0),
		SecretKey:    os.Getenv("ADMIN_SECRET_KEY"),
		PasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),
	}

	// ── Search (provider-agnostic) ───────────────────────────────────
	cfg.Search = SearchConfig{
		Enabled:     getEnvBool("SEARCH_ENABLED", false),
		Engine:      getEnvOr("SEARCH_ENGINE", os.Getenv("SEARCH_PROVIDER")),
		Port:        getEnvInt("SEARCH_PORT", 0),
		APIKey:      os.Getenv("SEARCH_API_KEY"),
		Route:       os.Getenv("SEARCH_ROUTE"),
		IndexPrefix: os.Getenv("SEARCH_INDEX_PREFIX"),
		AutoIndex:   getEnvBool("SEARCH_AUTO_INDEX", true),
		Language:    os.Getenv("SEARCH_LANGUAGE"),
		MeiliSearch: MeiliSearchConfig{
			Version:   os.Getenv("MEILISEARCH_VERSION"),
			MasterKey: os.Getenv("MEILISEARCH_MASTER_KEY"),
			Env:       getEnvOr("MEILISEARCH_ENV", os.Getenv("MEILI_ENV")),
		},
		Typesense: TypesenseConfig{
			Version:           os.Getenv("TYPESENSE_VERSION"),
			APIKey:            os.Getenv("TYPESENSE_API_KEY"),
			EnableCORS:        getEnvBool("TYPESENSE_ENABLE_CORS", false),
			LogLevel:          os.Getenv("TYPESENSE_LOG_LEVEL"),
			NumMemoryShards:   getEnvInt("TYPESENSE_NUM_MEMORY_SHARDS", 0),
			SnapshotIntervalS: getEnvInt("TYPESENSE_SNAPSHOT_INTERVAL_SECONDS", 0),
		},
		Elasticsearch: ElasticsearchConfig{
			Version:  os.Getenv("ELASTICSEARCH_VERSION"),
			Port:     getEnvInt("ELASTICSEARCH_PORT", 0),
			Password: os.Getenv("ELASTICSEARCH_PASSWORD"),
			Memory:   os.Getenv("ELASTICSEARCH_MEMORY"),
		},
	}

	// ── Monitoring ───────────────────────────────────────────────────
	cfg.Monitoring = MonitoringConfig{
		Enabled:              getEnvBool("MONITORING_ENABLED", false),
		PrometheusEnabled:    getEnvBool("PROMETHEUS_ENABLED", false),
		PrometheusPort:       getEnvInt("PROMETHEUS_PORT", 0),
		GrafanaEnabled:       getEnvBool("GRAFANA_ENABLED", false),
		GrafanaPort:          getEnvInt("GRAFANA_PORT", 0),
		GrafanaAdminUser:     os.Getenv("GRAFANA_ADMIN_USER"),
		GrafanaAdminPassword: os.Getenv("GRAFANA_ADMIN_PASSWORD"),
		GrafanaRoute:         os.Getenv("GRAFANA_ROUTE"),
		LokiEnabled:          getEnvBool("LOKI_ENABLED", false),
		LokiPort:             getEnvInt("LOKI_PORT", 0),
		PromtailEnabled:      getEnvBool("PROMTAIL_ENABLED", false),
		TempoEnabled:         getEnvBool("TEMPO_ENABLED", false),
		TempoPort:            getEnvInt("TEMPO_PORT", 0),
		AlertmanagerEnabled:  getEnvBool("ALERTMANAGER_ENABLED", false),
		AlertmanagerPort:     getEnvInt("ALERTMANAGER_PORT", 0),
		CadvisorEnabled:      getEnvBool("CADVISOR_ENABLED", false),
		CadvisorPort:         getEnvInt("CADVISOR_PORT", 0),
		NodeExporterEnabled:  getEnvBool("NODE_EXPORTER_ENABLED", false),
		NodeExporterPort:     getEnvInt("NODE_EXPORTER_PORT", 0),
		PGExporterEnabled:    getEnvBool("POSTGRES_EXPORTER_ENABLED", false),
		PGExporterPort:       getEnvInt("POSTGRES_EXPORTER_PORT", 0),
		RedisExporterEnabled: getEnvBool("REDIS_EXPORTER_ENABLED", false),
		RedisExporterPort:    getEnvInt("REDIS_EXPORTER_PORT", 0),
	}

	// ── Email ────────────────────────────────────────────────────────
	cfg.Email = EmailConfig{
		Provider:            os.Getenv("EMAIL_PROVIDER"),
		From:                os.Getenv("EMAIL_FROM"),
		ElasticEmailAPIKey:  os.Getenv("ELASTIC_EMAIL_API_KEY"),
		ElasticEmailAccount: os.Getenv("ELASTIC_EMAIL_ACCOUNT_EMAIL"),
		SendGridAPIKey:      os.Getenv("SENDGRID_API_KEY"),
		PostmarkAPIKey:      os.Getenv("POSTMARK_API_KEY"),
		MailgunAPIKey:       os.Getenv("MAILGUN_API_KEY"),
		MailgunDomain:       os.Getenv("MAILGUN_DOMAIN"),
		AWSAccessKeyID:      os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey:  os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSRegion:           os.Getenv("AWS_REGION"),
		SMTPHost:            os.Getenv("SMTP_HOST"),
		SMTPPort:            getEnvInt("SMTP_PORT", 0),
		SMTPUser:            os.Getenv("SMTP_USER"),
		SMTPPass:            os.Getenv("SMTP_PASS"),
		SMTPSecure:          getEnvBool("SMTP_SECURE", false),
	}

	// ── Backup ───────────────────────────────────────────────────────
	cfg.Backup = BackupConfig{
		Enabled:       getEnvBool("BACKUP_ENABLED", false),
		Dir:           os.Getenv("BACKUP_DIR"),
		Schedule:      os.Getenv("BACKUP_SCHEDULE"),
		RetentionDays: getEnvInt("BACKUP_RETENTION_DAYS", 0),
		CloudProvider: os.Getenv("BACKUP_CLOUD_PROVIDER"),
	}

	// Plugin port defaults (3712=notify, 3713=cron) are set in ApplyDefaults() when port==0.
	// ── Plugin Pro Configuration ─────────────────────────────────────
	cfg.PluginConfig = PluginProConfig{
		NotifySecret:    os.Getenv("NOTIFY_INTERNAL_SECRET"),
		NotifyPort:      getEnvInt("NOTIFY_PORT", 0),
		NotifyVAPIDPub:  os.Getenv("NOTIFY_VAPID_PUBLIC_KEY"),
		NotifyVAPIDPriv: os.Getenv("NOTIFY_VAPID_PRIVATE_KEY"),
		NotifyRoute:     os.Getenv("NOTIFY_ROUTE"),
		CronSecret:      os.Getenv("CRON_INTERNAL_SECRET"),
		CronPort:        getEnvInt("CRON_PORT", 0),
		CronRetention:   getEnvInt("CRON_RETENTION_DAYS", 0),
		AIMemLimit:      os.Getenv("PLUGIN_AI_MEMORY_LIMIT"),
		AICPULimit:      os.Getenv("PLUGIN_AI_CPU_LIMIT"),
		MuxMemLimit:     os.Getenv("PLUGIN_MUX_MEMORY_LIMIT"),
		MuxCPULimit:     os.Getenv("PLUGIN_MUX_CPU_LIMIT"),
		ClawMemLimit:    os.Getenv("PLUGIN_CLAW_MEMORY_LIMIT"),
		ClawCPULimit:    os.Getenv("PLUGIN_CLAW_CPU_LIMIT"),
		DefaultMemLimit: os.Getenv("PLUGIN_DEFAULT_MEMORY_LIMIT"),
		DefaultCPULimit: os.Getenv("PLUGIN_DEFAULT_CPU_LIMIT"),
	}

	// ── Plugin System ────────────────────────────────────────────────
	cfg.PluginSystem = PluginSystemConfig{
		Dir:            os.Getenv("NSELF_PLUGIN_DIR"),
		Cache:          os.Getenv("NSELF_PLUGIN_CACHE"),
		Registry:       os.Getenv("NSELF_PLUGIN_REGISTRY"),
		CacheTTL:       getEnvInt("NSELF_REGISTRY_CACHE_TTL", 0),
		LicenseKey:     os.Getenv("NSELF_PLUGIN_LICENSE_KEY"),
		SkipVerify:     getEnvBool("NSELF_LICENSE_SKIP_VERIFY", false),
		PingURL:        os.Getenv("NSELF_PING_API_URL"),
		PricingURL:     os.Getenv("NSELF_PRICING_URL"),
		InternalSecret: os.Getenv("PLUGIN_INTERNAL_SECRET"),
	}

	// ── Docker ───────────────────────────────────────────────────────
	cfg.DockerNetwork = os.Getenv("DOCKER_NETWORK")
	cfg.DockerLogMaxSize = os.Getenv("DOCKER_LOG_MAX_SIZE")
	cfg.DockerLogMaxFile = os.Getenv("DOCKER_LOG_MAX_FILE")
	cfg.DockerStopGrace = os.Getenv("DOCKER_STOP_GRACE_PERIOD")
	cfg.DockerBuildTimeout = getEnvInt("NSELF_DOCKER_BUILD_TIMEOUT", 0)

	// ── Start/Stop ───────────────────────────────────────────────────
	cfg.StartMode = os.Getenv("NSELF_START_MODE")
	cfg.HealthCheckTimeout = getEnvInt("NSELF_HEALTH_CHECK_TIMEOUT", 0)
	cfg.HealthCheckInterval = getEnvInt("NSELF_HEALTH_CHECK_INTERVAL", 0)
	cfg.HealthCheckRequired = getEnvInt("NSELF_HEALTH_CHECK_REQUIRED", 0)
	cfg.CleanupOnStart = os.Getenv("NSELF_CLEANUP_ON_START")
	cfg.AllowExposedPorts = getEnvBool("NSELF_ALLOW_EXPOSED_PORTS", false)
	cfg.ParallelLimit = getEnvInt("NSELF_PARALLEL_LIMIT", 0)
	cfg.LogLevel = os.Getenv("NSELF_LOG_LEVEL")
	cfg.SkipHealthChecks = getEnvBool("NSELF_SKIP_HEALTH_CHECKS", false)
	cfg.StopTimeout = getEnvInt("NSELF_STOP_TIMEOUT", 0)

	return cfg
}

// collectPassthrough scans the full environment for dynamic env vars matching
// known prefixes (AUTH_PROVIDER_*, REMOTE_SCHEMA_*, HASURA_EXTRA_*) and returns
// them as a key-value map. These variables cannot be predefined in structs
// because users add them dynamically for OAuth providers, remote schemas, etc.
func collectPassthrough(environ []string) map[string]string {
	prefixes := []string{
		"AUTH_PROVIDER_",
		"REMOTE_SCHEMA_",
		"HASURA_EXTRA_",
	}
	result := make(map[string]string)
	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(parts[0], prefix) {
				result[parts[0]] = parts[1]
			}
		}
	}
	return result
}

// parseInternalRoutes parses INTERNAL_ROUTE_1 through INTERNAL_ROUTE_20
// environment variables into InternalRoute structs. Each route is defined by:
//
//	INTERNAL_ROUTE_N_NAME       — required (skip if empty)
//	INTERNAL_ROUTE_N_SUBDOMAIN
//	INTERNAL_ROUTE_N_TARGET     — e.g., hasura:8080
//	INTERNAL_ROUTE_N_RATE_ZONE  — default: general
//	INTERNAL_ROUTE_N_WEBSOCKET  — bool
func parseInternalRoutes() []InternalRoute {
	var routes []InternalRoute
	for i := 1; i <= 20; i++ {
		prefix := fmt.Sprintf("INTERNAL_ROUTE_%d_", i)
		name := os.Getenv(prefix + "NAME")
		if name == "" {
			continue
		}

		route := InternalRoute{
			Index:     i,
			Name:      name,
			Subdomain: os.Getenv(prefix + "SUBDOMAIN"),
			Target:    os.Getenv(prefix + "TARGET"),
			RateZone:  getEnvOr(prefix+"RATE_ZONE", "general"),
			WebSocket: getEnvBool(prefix+"WEBSOCKET", false),
		}
		routes = append(routes, route)
	}
	return routes
}

// parseExtensionList parses a comma-separated extension list string into a slice.
// Trims whitespace from each element and removes empty entries.
func parseExtensionList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
