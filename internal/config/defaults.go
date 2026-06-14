package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
)

// ApplyDefaults fills every empty/zero field in cfg with the canonical default
// value. It never overrides a non-empty string, non-zero int, or explicitly-set
// boolean. Empty string "" is considered unset for string fields; zero is
// considered unset for int fields.
//
// Environment-specific overrides (Console, DevMode, CORS, BindIP, SSL) are
// applied after all static defaults.
func ApplyDefaults(cfg *Config) (*Config, error) {
	// ── Core ──────────────────────────────────────────────────────────
	if cfg.ProjectName == "" {
		cfg.ProjectName = "myproject"
		slog.Debug("default", "key", "PROJECT_NAME", "value", cfg.ProjectName)
	}
	if cfg.BaseDomain == "" {
		cfg.BaseDomain = "local.nself.org"
		slog.Debug("default", "key", "BASE_DOMAIN", "value", cfg.BaseDomain)
	}
	if cfg.Env == "" {
		cfg.Env = "dev"
		slog.Debug("default", "key", "NSELF_ENV", "value", cfg.Env)
	}
	// Normalize env aliases (development->dev, production->prod, stage->staging)
	cfg.Env = normalizeEnv(cfg.Env)

	// ── PostgreSQL ────────────────────────────────────────────────────
	if cfg.Postgres.Version == "" {
		cfg.Postgres.Version = "16-alpine"
		slog.Debug("default", "key", "POSTGRES_VERSION", "value", cfg.Postgres.Version)
	}
	if cfg.Postgres.Host == "" {
		cfg.Postgres.Host = "postgres"
		slog.Debug("default", "key", "POSTGRES_HOST", "value", cfg.Postgres.Host)
	}
	if cfg.Postgres.Port == 0 {
		cfg.Postgres.Port = 5432
		slog.Debug("default", "key", "POSTGRES_PORT", "value", fmt.Sprintf("%d", cfg.Postgres.Port))
	}
	if cfg.Postgres.DB == "" {
		cfg.Postgres.DB = "nself"
		slog.Debug("default", "key", "POSTGRES_DB", "value", cfg.Postgres.DB)
	}
	if cfg.Postgres.User == "" {
		cfg.Postgres.User = "postgres"
		slog.Debug("default", "key", "POSTGRES_USER", "value", cfg.Postgres.User)
	}
	if cfg.Postgres.Password == "" {
		secret, err := generateSecureRandom(24)
		if err != nil {
			return nil, fmt.Errorf("generating POSTGRES_PASSWORD: %w", err)
		}
		cfg.Postgres.Password = secret
		slog.Debug("default", "key", "POSTGRES_PASSWORD", "value", "[generated]")
	}
	if len(cfg.Postgres.Extensions) == 0 {
		cfg.Postgres.Extensions = []string{"uuid-ossp"}
		slog.Debug("default", "key", "POSTGRES_EXTENSIONS", "value", cfg.Postgres.Extensions)
	}
	if cfg.Postgres.ExposePort == "" {
		cfg.Postgres.ExposePort = "auto"
		slog.Debug("default", "key", "POSTGRES_EXPOSE_PORT", "value", cfg.Postgres.ExposePort)
	}
	if cfg.Postgres.MemLimit == "" {
		cfg.Postgres.MemLimit = "2g"
		slog.Debug("default", "key", "POSTGRES_MEM_LIMIT", "value", cfg.Postgres.MemLimit)
	}
	if cfg.Postgres.CPULimit == "" {
		cfg.Postgres.CPULimit = "2.0"
		slog.Debug("default", "key", "POSTGRES_CPU_LIMIT", "value", cfg.Postgres.CPULimit)
	}

	// ── Hasura ────────────────────────────────────────────────────────
	if cfg.Hasura.Version == "" {
		cfg.Hasura.Version = "v2.44.0"
		slog.Debug("default", "key", "HASURA_VERSION", "value", cfg.Hasura.Version)
	}
	if cfg.Hasura.AdminSecret == "" {
		secret, err := generateSecureRandom(44)
		if err != nil {
			return nil, fmt.Errorf("generating HASURA_GRAPHQL_ADMIN_SECRET: %w", err)
		}
		cfg.Hasura.AdminSecret = secret
		slog.Debug("default", "key", "HASURA_GRAPHQL_ADMIN_SECRET", "value", "[generated]")
	}
	if cfg.Hasura.JWTKey == "" {
		secret, err := generateSecureRandom(44)
		if err != nil {
			return nil, fmt.Errorf("generating HASURA_GRAPHQL_JWT_KEY: %w", err)
		}
		cfg.Hasura.JWTKey = secret
		slog.Debug("default", "key", "HASURA_GRAPHQL_JWT_KEY", "value", "[generated]")
	}
	if cfg.Hasura.JWTType == "" {
		// SEC-JWT-01: RS256 is the default for new installs (asymmetric — private key
		// never leaves the auth service; public key shared with Hasura).
		// HS256 is still supported for existing deployments but emits a deprecation
		// warning at startup. Migrate guide: docs.nself.org/security/jwt-migration.
		cfg.Hasura.JWTType = "RS256"
		slog.Debug("default", "key", "HASURA_GRAPHQL_JWT_TYPE", "value", cfg.Hasura.JWTType)
	} else if strings.EqualFold(cfg.Hasura.JWTType, "HS256") {
		// SEC-JWT-01: HS256 uses a shared secret — if the secret leaks, all tokens
		// are forgeable. Upgrade to RS256 for new deployments.
		slog.Warn("SEC-JWT-01: HASURA_JWT_TYPE=HS256 is deprecated and will be removed in v2.0. " +
			"Migrate to RS256. See docs.nself.org/security/jwt-migration.")
	}
	if cfg.Hasura.Route == "" {
		cfg.Hasura.Route = "api"
		slog.Debug("default", "key", "HASURA_ROUTE", "value", cfg.Hasura.Route)
	}
	if cfg.Hasura.Port == 0 {
		cfg.Hasura.Port = 8080
		slog.Debug("default", "key", "HASURA_PORT", "value", fmt.Sprintf("%d", cfg.Hasura.Port))
	}
	if cfg.Hasura.MemLimit == "" {
		cfg.Hasura.MemLimit = "1g"
		slog.Debug("default", "key", "HASURA_MEM_LIMIT", "value", cfg.Hasura.MemLimit)
	}
	if cfg.Hasura.CPULimit == "" {
		cfg.Hasura.CPULimit = "1.0"
		slog.Debug("default", "key", "HASURA_CPU_LIMIT", "value", cfg.Hasura.CPULimit)
	}
	if cfg.Hasura.LogLevel == "" {
		cfg.Hasura.LogLevel = "warn"
		slog.Debug("default", "key", "HASURA_GRAPHQL_LOG_LEVEL", "value", cfg.Hasura.LogLevel)
	}

	// Hasura env-specific: dev/staging get console+devmode, prod forces them off.
	if cfg.Env == "prod" {
		cfg.Hasura.Console = false
		cfg.Hasura.DevMode = false
	} else {
		// In dev/staging, enable by default (bool zero = unset = enable).
		// If the user explicitly loaded true from .env, this is a no-op.
		if !cfg.Hasura.Console {
			cfg.Hasura.Console = true
		}
		if !cfg.Hasura.DevMode {
			cfg.Hasura.DevMode = true
		}
	}

	// Hasura CORS: dev gets localhost wildcard, non-dev gets domain wildcard.
	if cfg.Hasura.CORSDomain == "" {
		if cfg.Env == "dev" {
			cfg.Hasura.CORSDomain = "http://localhost:*"
		} else {
			cfg.Hasura.CORSDomain = "https://*." + cfg.BaseDomain
		}
		slog.Debug("default", "key", "HASURA_GRAPHQL_CORS_DOMAIN", "value", cfg.Hasura.CORSDomain)
	}

	// ── Auth ──────────────────────────────────────────────────────────
	if cfg.Auth.Version == "" {
		cfg.Auth.Version = "0.36.0"
		slog.Debug("default", "key", "AUTH_VERSION", "value", cfg.Auth.Version)
	}
	if cfg.Auth.Port == 0 {
		cfg.Auth.Port = 4000
		slog.Debug("default", "key", "AUTH_PORT", "value", fmt.Sprintf("%d", cfg.Auth.Port))
	}
	if cfg.Auth.ClientURL == "" {
		cfg.Auth.ClientURL = "http://localhost:3000"
		slog.Debug("default", "key", "AUTH_CLIENT_URL", "value", cfg.Auth.ClientURL)
	}
	if cfg.Auth.AccessTokenExpiry == 0 {
		cfg.Auth.AccessTokenExpiry = 900
		slog.Debug("default", "key", "AUTH_ACCESS_TOKEN_EXPIRES_IN", "value", fmt.Sprintf("%d", cfg.Auth.AccessTokenExpiry))
	}
	if cfg.Auth.RefreshTokenExpiry == 0 {
		cfg.Auth.RefreshTokenExpiry = 2592000
		slog.Debug("default", "key", "AUTH_REFRESH_TOKEN_EXPIRES_IN", "value", fmt.Sprintf("%d", cfg.Auth.RefreshTokenExpiry))
	}
	if cfg.Auth.Route == "" {
		cfg.Auth.Route = "auth"
		slog.Debug("default", "key", "AUTH_ROUTE", "value", cfg.Auth.Route)
	}
	if cfg.Auth.MemLimit == "" {
		cfg.Auth.MemLimit = "256m"
		slog.Debug("default", "key", "AUTH_MEM_LIMIT", "value", cfg.Auth.MemLimit)
	}
	if cfg.Auth.CPULimit == "" {
		cfg.Auth.CPULimit = "0.25"
		slog.Debug("default", "key", "AUTH_CPU_LIMIT", "value", cfg.Auth.CPULimit)
	}
	if cfg.Auth.SMTPHost == "" {
		cfg.Auth.SMTPHost = "mailpit"
		slog.Debug("default", "key", "AUTH_SMTP_HOST", "value", cfg.Auth.SMTPHost)
	}
	if cfg.Auth.SMTPPort == 0 {
		cfg.Auth.SMTPPort = 1025
		slog.Debug("default", "key", "AUTH_SMTP_PORT", "value", fmt.Sprintf("%d", cfg.Auth.SMTPPort))
	}
	if cfg.Auth.SMTPSender == "" {
		cfg.Auth.SMTPSender = "noreply@" + cfg.BaseDomain
		slog.Debug("default", "key", "AUTH_SMTP_SENDER", "value", cfg.Auth.SMTPSender)
	}
	if cfg.Auth.LogLevel == "" {
		cfg.Auth.LogLevel = "info"
		slog.Debug("default", "key", "AUTH_LOG_LEVEL", "value", cfg.Auth.LogLevel)
	}

	// ── Nginx ─────────────────────────────────────────────────────────
	if cfg.Nginx.Version == "" {
		cfg.Nginx.Version = "alpine"
		slog.Debug("default", "key", "NGINX_VERSION", "value", cfg.Nginx.Version)
	}
	if cfg.Nginx.HTTPPort == 0 {
		cfg.Nginx.HTTPPort = 80
		slog.Debug("default", "key", "NGINX_HTTP_PORT", "value", fmt.Sprintf("%d", cfg.Nginx.HTTPPort))
	}
	if cfg.Nginx.SSLPort == 0 {
		cfg.Nginx.SSLPort = 443
		slog.Debug("default", "key", "NGINX_SSL_PORT", "value", fmt.Sprintf("%d", cfg.Nginx.SSLPort))
	}
	if cfg.Nginx.MaxBody == "" {
		cfg.Nginx.MaxBody = "100M"
		slog.Debug("default", "key", "NGINX_MAX_BODY", "value", cfg.Nginx.MaxBody)
	}
	if cfg.Nginx.AuthRateLimit == "" {
		cfg.Nginx.AuthRateLimit = "30r/m"
		slog.Debug("default", "key", "AUTH_RATE_LIMIT", "value", cfg.Nginx.AuthRateLimit)
	}

	// Nginx BindIP: only default when unset. Preserves any user-set value (TRAP-03).
	if cfg.Nginx.BindIP == "" {
		if cfg.Env == "dev" {
			cfg.Nginx.BindIP = "127.0.0.1"
		} else {
			cfg.Nginx.BindIP = "0.0.0.0"
		}
		slog.Debug("default", "key", "NGINX_BIND_IP", "value", cfg.Nginx.BindIP)
	}

	// ── SSL ───────────────────────────────────────────────────────────
	if cfg.SSLMode == "" {
		if cfg.Env == "dev" {
			cfg.SSLMode = "local"
		} else {
			cfg.SSLMode = "letsencrypt"
		}
		slog.Debug("default", "key", "SSL_MODE", "value", cfg.SSLMode)
	}
	if cfg.SSLProvider == "" {
		cfg.SSLProvider = "cloudflare"
		slog.Debug("default", "key", "SSL_PROVIDER", "value", cfg.SSLProvider)
	}
	if cfg.WAFMode == "" {
		cfg.WAFMode = "off"
		slog.Debug("default", "key", "WAF_MODE", "value", cfg.WAFMode)
	}

	// ── Redis ─────────────────────────────────────────────────────────
	if cfg.Redis.Version == "" {
		cfg.Redis.Version = "7-alpine"
		slog.Debug("default", "key", "REDIS_VERSION", "value", cfg.Redis.Version)
	}
	if cfg.Redis.Port == 0 {
		cfg.Redis.Port = 6379
		slog.Debug("default", "key", "REDIS_PORT", "value", fmt.Sprintf("%d", cfg.Redis.Port))
	}
	if cfg.Redis.Memory == "" {
		cfg.Redis.Memory = "512M"
		slog.Debug("default", "key", "REDIS_MEMORY", "value", cfg.Redis.Memory)
	}
	if cfg.Redis.CPU == "" {
		cfg.Redis.CPU = "0.5"
		slog.Debug("default", "key", "REDIS_CPU", "value", cfg.Redis.CPU)
	}
	if cfg.Redis.PoolSize == 0 {
		if cfg.Env == "dev" {
			cfg.Redis.PoolSize = 20
		} else {
			cfg.Redis.PoolSize = 50
		}
		slog.Debug("default", "key", "REDIS_POOL_SIZE", "value", fmt.Sprintf("%d", cfg.Redis.PoolSize))
	}

	// ── MinIO / Storage ───────────────────────────────────────────────
	if cfg.Minio.Version == "" {
		cfg.Minio.Version = "latest"
		slog.Debug("default", "key", "MINIO_VERSION", "value", cfg.Minio.Version)
	}
	if cfg.Minio.Port == 0 {
		cfg.Minio.Port = 9000
		slog.Debug("default", "key", "MINIO_PORT", "value", fmt.Sprintf("%d", cfg.Minio.Port))
	}
	if cfg.Minio.ConsolePort == 0 {
		cfg.Minio.ConsolePort = 9001
		slog.Debug("default", "key", "MINIO_CONSOLE_PORT", "value", fmt.Sprintf("%d", cfg.Minio.ConsolePort))
	}
	if cfg.Minio.RootUser == "" {
		cfg.Minio.RootUser = "minioadmin"
		slog.Debug("default", "key", "MINIO_ROOT_USER", "value", cfg.Minio.RootUser)
	}
	if cfg.Minio.RootPassword == "" {
		cfg.Minio.RootPassword = "minioadmin"
		slog.Debug("default", "key", "MINIO_ROOT_PASSWORD", "value", "[default]")
	}
	// Guard: reject minioadmin defaults in staging/prod immediately after
	// they are set so that nself start exits non-zero with a clear message
	// before any container is started. Dev is intentionally unblocked.
	if err := ValidateMinioCredentials(cfg); err != nil {
		return nil, err
	}
	if cfg.Minio.DefaultBuckets == "" {
		cfg.Minio.DefaultBuckets = "uploads,public,private,temp"
		slog.Debug("default", "key", "MINIO_DEFAULT_BUCKETS", "value", cfg.Minio.DefaultBuckets)
	}
	if cfg.Minio.Region == "" {
		cfg.Minio.Region = "us-east-1"
		slog.Debug("default", "key", "MINIO_REGION", "value", cfg.Minio.Region)
	}
	if cfg.Minio.S3Bucket == "" {
		cfg.Minio.S3Bucket = "nself"
		slog.Debug("default", "key", "MINIO_S3_BUCKET", "value", cfg.Minio.S3Bucket)
	}
	if cfg.Minio.StorageVersion == "" {
		cfg.Minio.StorageVersion = "0.6.1"
		slog.Debug("default", "key", "STORAGE_VERSION", "value", cfg.Minio.StorageVersion)
	}
	if cfg.Minio.StorageRoute == "" {
		cfg.Minio.StorageRoute = "storage"
		slog.Debug("default", "key", "STORAGE_ROUTE", "value", cfg.Minio.StorageRoute)
	}
	if cfg.Minio.ConsoleRoute == "" {
		cfg.Minio.ConsoleRoute = "storage-console"
		slog.Debug("default", "key", "MINIO_CONSOLE_ROUTE", "value", cfg.Minio.ConsoleRoute)
	}
	if cfg.Minio.MemLimit == "" {
		cfg.Minio.MemLimit = "1G"
		slog.Debug("default", "key", "MINIO_MEM_LIMIT", "value", cfg.Minio.MemLimit)
	}
	if cfg.Minio.CPULimit == "" {
		cfg.Minio.CPULimit = "0.5"
		slog.Debug("default", "key", "MINIO_CPU_LIMIT", "value", cfg.Minio.CPULimit)
	}

	// ── Mailpit ───────────────────────────────────────────────────────
	if cfg.Mailpit.Version == "" {
		cfg.Mailpit.Version = "latest"
		slog.Debug("default", "key", "MAILPIT_VERSION", "value", cfg.Mailpit.Version)
	}
	if cfg.Mailpit.SMTPPort == 0 {
		cfg.Mailpit.SMTPPort = 1025
		slog.Debug("default", "key", "MAILPIT_SMTP_PORT", "value", fmt.Sprintf("%d", cfg.Mailpit.SMTPPort))
	}
	if cfg.Mailpit.UIPort == 0 {
		cfg.Mailpit.UIPort = 8025
		slog.Debug("default", "key", "MAILPIT_UI_PORT", "value", fmt.Sprintf("%d", cfg.Mailpit.UIPort))
	}
	if cfg.Mailpit.MaxMessages == 0 {
		cfg.Mailpit.MaxMessages = 500
		slog.Debug("default", "key", "MAILPIT_MAX_MESSAGES", "value", fmt.Sprintf("%d", cfg.Mailpit.MaxMessages))
	}
	if cfg.Mailpit.Route == "" {
		cfg.Mailpit.Route = "mail"
		slog.Debug("default", "key", "MAILPIT_ROUTE", "value", cfg.Mailpit.Route)
	}

	// ── Functions ─────────────────────────────────────────────────────
	if cfg.Functions.Version == "" {
		cfg.Functions.Version = "latest"
		slog.Debug("default", "key", "FUNCTIONS_VERSION", "value", cfg.Functions.Version)
	}
	if cfg.Functions.Port == 0 {
		cfg.Functions.Port = 3008
		slog.Debug("default", "key", "FUNCTIONS_PORT", "value", fmt.Sprintf("%d", cfg.Functions.Port))
	}
	if cfg.Functions.Route == "" {
		cfg.Functions.Route = "functions"
		slog.Debug("default", "key", "FUNCTIONS_ROUTE", "value", cfg.Functions.Route)
	}

	// ── MLflow ────────────────────────────────────────────────────────
	if cfg.MLflow.Version == "" {
		cfg.MLflow.Version = "2.9.2"
		slog.Debug("default", "key", "MLFLOW_VERSION", "value", cfg.MLflow.Version)
	}
	if cfg.MLflow.Port == 0 {
		cfg.MLflow.Port = 5000
		slog.Debug("default", "key", "MLFLOW_PORT", "value", fmt.Sprintf("%d", cfg.MLflow.Port))
	}
	if cfg.MLflow.Route == "" {
		cfg.MLflow.Route = "mlflow"
		slog.Debug("default", "key", "MLFLOW_ROUTE", "value", cfg.MLflow.Route)
	}
	if cfg.MLflow.DBName == "" {
		cfg.MLflow.DBName = "mlflow"
		slog.Debug("default", "key", "MLFLOW_DB_NAME", "value", cfg.MLflow.DBName)
	}
	if cfg.MLflow.ArtifactsBucket == "" {
		cfg.MLflow.ArtifactsBucket = "mlflow-artifacts"
		slog.Debug("default", "key", "MLFLOW_ARTIFACTS_BUCKET", "value", cfg.MLflow.ArtifactsBucket)
	}
	if cfg.MLflow.AuthUsername == "" {
		cfg.MLflow.AuthUsername = "admin"
		slog.Debug("default", "key", "MLFLOW_AUTH_USERNAME", "value", cfg.MLflow.AuthUsername)
	}

	// ── Admin ─────────────────────────────────────────────────────────
	if cfg.Admin.Version == "" {
		cfg.Admin.Version = "latest"
		slog.Debug("default", "key", "ADMIN_VERSION", "value", cfg.Admin.Version)
	}
	if cfg.Admin.Port == 0 {
		cfg.Admin.Port = 3021
		slog.Debug("default", "key", "ADMIN_PORT", "value", fmt.Sprintf("%d", cfg.Admin.Port))
	}
	if cfg.Admin.Route == "" {
		cfg.Admin.Route = "admin"
		slog.Debug("default", "key", "ADMIN_ROUTE", "value", cfg.Admin.Route)
	}

	// ── Search ────────────────────────────────────────────────────────
	if cfg.Search.Engine == "" {
		cfg.Search.Engine = "meilisearch"
		slog.Debug("default", "key", "SEARCH_ENGINE", "value", cfg.Search.Engine)
	}
	if cfg.Search.Port == 0 {
		cfg.Search.Port = 7700
		slog.Debug("default", "key", "SEARCH_PORT", "value", "7700")
	}
	if cfg.Search.Route == "" {
		cfg.Search.Route = "search"
		slog.Debug("default", "key", "SEARCH_ROUTE", "value", cfg.Search.Route)
	}
	if cfg.Search.Language == "" {
		cfg.Search.Language = "en"
		slog.Debug("default", "key", "SEARCH_LANGUAGE", "value", cfg.Search.Language)
	}
	// MeiliSearch defaults
	if cfg.Search.MeiliSearch.Version == "" {
		cfg.Search.MeiliSearch.Version = "v1.6"
		slog.Debug("default", "key", "MEILISEARCH_VERSION", "value", cfg.Search.MeiliSearch.Version)
	}
	if cfg.Search.MeiliSearch.Env == "" {
		cfg.Search.MeiliSearch.Env = "development"
		slog.Debug("default", "key", "MEILISEARCH_ENV", "value", cfg.Search.MeiliSearch.Env)
	}

	// Typesense defaults
	if cfg.Search.Typesense.Version == "" {
		cfg.Search.Typesense.Version = "27.1"
		slog.Debug("default", "key", "TYPESENSE_VERSION", "value", cfg.Search.Typesense.Version)
	}
	if cfg.Search.Typesense.LogLevel == "" {
		cfg.Search.Typesense.LogLevel = "info"
		slog.Debug("default", "key", "TYPESENSE_LOG_LEVEL", "value", cfg.Search.Typesense.LogLevel)
	}
	if cfg.Search.Typesense.NumMemoryShards == 0 {
		cfg.Search.Typesense.NumMemoryShards = 4
		slog.Debug("default", "key", "TYPESENSE_NUM_MEMORY_SHARDS", "value", fmt.Sprintf("%d", cfg.Search.Typesense.NumMemoryShards))
	}
	if cfg.Search.Typesense.SnapshotIntervalS == 0 {
		cfg.Search.Typesense.SnapshotIntervalS = 3600
		slog.Debug("default", "key", "TYPESENSE_SNAPSHOT_INTERVAL_SECONDS", "value", fmt.Sprintf("%d", cfg.Search.Typesense.SnapshotIntervalS))
	}
	if !cfg.Search.Typesense.EnableCORS {
		cfg.Search.Typesense.EnableCORS = true
	}

	// Elasticsearch defaults
	if cfg.Search.Elasticsearch.Version == "" {
		cfg.Search.Elasticsearch.Version = "8.11.3"
		slog.Debug("default", "key", "ELASTICSEARCH_VERSION", "value", cfg.Search.Elasticsearch.Version)
	}
	if cfg.Search.Elasticsearch.Port == 0 {
		cfg.Search.Elasticsearch.Port = 9200
		slog.Debug("default", "key", "ELASTICSEARCH_PORT", "value", fmt.Sprintf("%d", cfg.Search.Elasticsearch.Port))
	}
	if cfg.Search.Elasticsearch.Memory == "" {
		cfg.Search.Elasticsearch.Memory = "1Gi"
		slog.Debug("default", "key", "ELASTICSEARCH_MEMORY", "value", cfg.Search.Elasticsearch.Memory)
	}

	// ── Monitoring ────────────────────────────────────────────────────
	// Auto-enable all 10 sub-booleans when monitoring master toggle is on.
	if cfg.Monitoring.Enabled {
		if !cfg.Monitoring.PrometheusEnabled {
			cfg.Monitoring.PrometheusEnabled = true
		}
		if !cfg.Monitoring.GrafanaEnabled {
			cfg.Monitoring.GrafanaEnabled = true
		}
		if !cfg.Monitoring.LokiEnabled {
			cfg.Monitoring.LokiEnabled = true
		}
		if !cfg.Monitoring.PromtailEnabled {
			cfg.Monitoring.PromtailEnabled = true
		}
		if !cfg.Monitoring.TempoEnabled {
			cfg.Monitoring.TempoEnabled = true
		}
		if !cfg.Monitoring.AlertmanagerEnabled {
			cfg.Monitoring.AlertmanagerEnabled = true
		}
		if !cfg.Monitoring.CadvisorEnabled {
			cfg.Monitoring.CadvisorEnabled = true
		}
		if !cfg.Monitoring.NodeExporterEnabled {
			cfg.Monitoring.NodeExporterEnabled = true
		}
		if !cfg.Monitoring.PGExporterEnabled {
			cfg.Monitoring.PGExporterEnabled = true
		}
		if !cfg.Monitoring.RedisExporterEnabled {
			cfg.Monitoring.RedisExporterEnabled = true
		}
	}

	// Monitoring ports (always fill regardless of enabled state)
	if cfg.Monitoring.PrometheusPort == 0 {
		cfg.Monitoring.PrometheusPort = 9090
		slog.Debug("default", "key", "PROMETHEUS_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.PrometheusPort))
	}
	if cfg.Monitoring.GrafanaPort == 0 {
		cfg.Monitoring.GrafanaPort = 3001
		slog.Debug("default", "key", "GRAFANA_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.GrafanaPort))
	}
	if cfg.Monitoring.GrafanaAdminUser == "" {
		cfg.Monitoring.GrafanaAdminUser = "admin"
		slog.Debug("default", "key", "GRAFANA_ADMIN_USER", "value", cfg.Monitoring.GrafanaAdminUser)
	}
	if cfg.Monitoring.GrafanaRoute == "" {
		cfg.Monitoring.GrafanaRoute = "grafana"
		slog.Debug("default", "key", "GRAFANA_ROUTE", "value", cfg.Monitoring.GrafanaRoute)
	}
	if cfg.Monitoring.LokiPort == 0 {
		cfg.Monitoring.LokiPort = 3100
		slog.Debug("default", "key", "LOKI_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.LokiPort))
	}
	if cfg.Monitoring.TempoPort == 0 {
		cfg.Monitoring.TempoPort = 3200
		slog.Debug("default", "key", "TEMPO_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.TempoPort))
	}
	if cfg.Monitoring.AlertmanagerPort == 0 {
		cfg.Monitoring.AlertmanagerPort = 9093
		slog.Debug("default", "key", "ALERTMANAGER_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.AlertmanagerPort))
	}
	if cfg.Monitoring.CadvisorPort == 0 {
		cfg.Monitoring.CadvisorPort = 8082
		slog.Debug("default", "key", "CADVISOR_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.CadvisorPort))
	}
	if cfg.Monitoring.NodeExporterPort == 0 {
		cfg.Monitoring.NodeExporterPort = 9100
		slog.Debug("default", "key", "NODE_EXPORTER_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.NodeExporterPort))
	}
	if cfg.Monitoring.PGExporterPort == 0 {
		cfg.Monitoring.PGExporterPort = 9187
		slog.Debug("default", "key", "PG_EXPORTER_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.PGExporterPort))
	}
	if cfg.Monitoring.RedisExporterPort == 0 {
		cfg.Monitoring.RedisExporterPort = 9121
		slog.Debug("default", "key", "REDIS_EXPORTER_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.RedisExporterPort))
	}

	// ── Docker ────────────────────────────────────────────────────────
	// DockerNetwork is always computed from ProjectName.
	cfg.DockerNetwork = cfg.ProjectName + "_network"

	if cfg.DockerLogMaxSize == "" {
		cfg.DockerLogMaxSize = "10m"
		slog.Debug("default", "key", "DOCKER_LOG_MAX_SIZE", "value", cfg.DockerLogMaxSize)
	}
	if cfg.DockerLogMaxFile == "" {
		cfg.DockerLogMaxFile = "3"
		slog.Debug("default", "key", "DOCKER_LOG_MAX_FILE", "value", cfg.DockerLogMaxFile)
	}
	if cfg.DockerStopGrace == "" {
		cfg.DockerStopGrace = "30s"
		slog.Debug("default", "key", "DOCKER_STOP_GRACE", "value", cfg.DockerStopGrace)
	}
	if cfg.DockerBuildTimeout == 0 {
		cfg.DockerBuildTimeout = 300
		slog.Debug("default", "key", "DOCKER_BUILD_TIMEOUT", "value", fmt.Sprintf("%d", cfg.DockerBuildTimeout))
	}

	// ── Start/Stop ────────────────────────────────────────────────────
	if cfg.StartMode == "" {
		cfg.StartMode = "smart"
		slog.Debug("default", "key", "START_MODE", "value", cfg.StartMode)
	}
	if cfg.HealthCheckTimeout == 0 {
		cfg.HealthCheckTimeout = 120
		slog.Debug("default", "key", "HEALTH_CHECK_TIMEOUT", "value", fmt.Sprintf("%d", cfg.HealthCheckTimeout))
	}
	if cfg.HealthCheckInterval == 0 {
		cfg.HealthCheckInterval = 2
		slog.Debug("default", "key", "HEALTH_CHECK_INTERVAL", "value", fmt.Sprintf("%d", cfg.HealthCheckInterval))
	}
	if cfg.HealthCheckRequired == 0 {
		cfg.HealthCheckRequired = 80
		slog.Debug("default", "key", "HEALTH_CHECK_REQUIRED", "value", fmt.Sprintf("%d", cfg.HealthCheckRequired))
	}
	if cfg.CleanupOnStart == "" {
		cfg.CleanupOnStart = "auto"
		slog.Debug("default", "key", "CLEANUP_ON_START", "value", cfg.CleanupOnStart)
	}
	if cfg.ParallelLimit == 0 {
		cfg.ParallelLimit = 5
		slog.Debug("default", "key", "PARALLEL_LIMIT", "value", fmt.Sprintf("%d", cfg.ParallelLimit))
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
		slog.Debug("default", "key", "NSELF_LOG_LEVEL", "value", cfg.LogLevel)
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = 30
		slog.Debug("default", "key", "STOP_TIMEOUT", "value", fmt.Sprintf("%d", cfg.StopTimeout))
	}

	// ── Plugin System ─────────────────────────────────────────────────
	if cfg.PluginSystem.Dir == "" {
		cfg.PluginSystem.Dir = "~/.nself/plugins"
		slog.Debug("default", "key", "PLUGIN_DIR", "value", cfg.PluginSystem.Dir)
	}
	if cfg.PluginSystem.Cache == "" {
		cfg.PluginSystem.Cache = "~/.nself/cache/plugins"
		slog.Debug("default", "key", "PLUGIN_CACHE", "value", cfg.PluginSystem.Cache)
	}
	if cfg.PluginSystem.Registry == "" {
		cfg.PluginSystem.Registry = "https://plugins.nself.org"
		slog.Debug("default", "key", "PLUGIN_REGISTRY", "value", cfg.PluginSystem.Registry)
	}
	if cfg.PluginSystem.CacheTTL == 0 {
		cfg.PluginSystem.CacheTTL = 300
		slog.Debug("default", "key", "PLUGIN_CACHE_TTL", "value", fmt.Sprintf("%d", cfg.PluginSystem.CacheTTL))
	}
	if cfg.PluginSystem.PingURL == "" {
		cfg.PluginSystem.PingURL = "https://ping.nself.org"
		slog.Debug("default", "key", "PLUGIN_PING_URL", "value", cfg.PluginSystem.PingURL)
	}
	if cfg.PluginSystem.PricingURL == "" {
		cfg.PluginSystem.PricingURL = "https://nself.org/pricing"
		slog.Debug("default", "key", "PLUGIN_PRICING_URL", "value", cfg.PluginSystem.PricingURL)
	}

	// ── Plugin Pro Config ─────────────────────────────────────────────
	if cfg.PluginConfig.NotifyPort == 0 {
		cfg.PluginConfig.NotifyPort = 3712
		slog.Debug("default", "key", "PLUGIN_NOTIFY_PORT", "value", fmt.Sprintf("%d", cfg.PluginConfig.NotifyPort))
	}
	if cfg.PluginConfig.NotifySecret == "" {
		secret, err := generateSecureRandom(32)
		if err != nil {
			return nil, fmt.Errorf("generating NOTIFY_INTERNAL_SECRET: %w", err)
		}
		cfg.PluginConfig.NotifySecret = secret
		slog.Debug("default", "key", "NOTIFY_INTERNAL_SECRET", "value", "[generated]")
	}
	if cfg.PluginConfig.CronSecret == "" {
		secret, err := generateSecureRandom(32)
		if err != nil {
			return nil, fmt.Errorf("generating CRON_INTERNAL_SECRET: %w", err)
		}
		cfg.PluginConfig.CronSecret = secret
		slog.Debug("default", "key", "CRON_INTERNAL_SECRET", "value", "[generated]")
	}
	if cfg.PluginSystem.InternalSecret == "" {
		secret, err := generateSecureRandom(32)
		if err != nil {
			return nil, fmt.Errorf("generating PLUGIN_INTERNAL_SECRET: %w", err)
		}
		cfg.PluginSystem.InternalSecret = secret
		slog.Debug("default", "key", "PLUGIN_INTERNAL_SECRET", "value", "[generated]")
	}
	if cfg.PluginConfig.NotifyRoute == "" {
		cfg.PluginConfig.NotifyRoute = "notify"
		slog.Debug("default", "key", "PLUGIN_NOTIFY_ROUTE", "value", cfg.PluginConfig.NotifyRoute)
	}
	if cfg.PluginConfig.CronPort == 0 {
		cfg.PluginConfig.CronPort = 3713
		slog.Debug("default", "key", "PLUGIN_CRON_PORT", "value", fmt.Sprintf("%d", cfg.PluginConfig.CronPort))
	}
	if cfg.PluginConfig.CronRetention == 0 {
		cfg.PluginConfig.CronRetention = 90
		slog.Debug("default", "key", "PLUGIN_CRON_RETENTION", "value", fmt.Sprintf("%d", cfg.PluginConfig.CronRetention))
	}
	if cfg.PluginConfig.AIMemLimit == "" {
		cfg.PluginConfig.AIMemLimit = "1g"
		slog.Debug("default", "key", "PLUGIN_AI_MEM_LIMIT", "value", cfg.PluginConfig.AIMemLimit)
	}
	if cfg.PluginConfig.AICPULimit == "" {
		cfg.PluginConfig.AICPULimit = "1.0"
		slog.Debug("default", "key", "PLUGIN_AI_CPU_LIMIT", "value", cfg.PluginConfig.AICPULimit)
	}
	if cfg.PluginConfig.MuxMemLimit == "" {
		cfg.PluginConfig.MuxMemLimit = "512m"
		slog.Debug("default", "key", "PLUGIN_MUX_MEM_LIMIT", "value", cfg.PluginConfig.MuxMemLimit)
	}
	if cfg.PluginConfig.MuxCPULimit == "" {
		cfg.PluginConfig.MuxCPULimit = "0.5"
		slog.Debug("default", "key", "PLUGIN_MUX_CPU_LIMIT", "value", cfg.PluginConfig.MuxCPULimit)
	}
	if cfg.PluginConfig.ClawMemLimit == "" {
		cfg.PluginConfig.ClawMemLimit = "512m"
		slog.Debug("default", "key", "PLUGIN_CLAW_MEM_LIMIT", "value", cfg.PluginConfig.ClawMemLimit)
	}
	if cfg.PluginConfig.ClawCPULimit == "" {
		cfg.PluginConfig.ClawCPULimit = "0.5"
		slog.Debug("default", "key", "PLUGIN_CLAW_CPU_LIMIT", "value", cfg.PluginConfig.ClawCPULimit)
	}
	if cfg.PluginConfig.DefaultMemLimit == "" {
		cfg.PluginConfig.DefaultMemLimit = "512m"
		slog.Debug("default", "key", "PLUGIN_DEFAULT_MEM_LIMIT", "value", cfg.PluginConfig.DefaultMemLimit)
	}
	if cfg.PluginConfig.DefaultCPULimit == "" {
		cfg.PluginConfig.DefaultCPULimit = "0.5"
		slog.Debug("default", "key", "PLUGIN_DEFAULT_CPU_LIMIT", "value", cfg.PluginConfig.DefaultCPULimit)
	}

	// ── Backup ────────────────────────────────────────────────────────
	if cfg.Backup.Dir == "" {
		cfg.Backup.Dir = "./backups"
		slog.Debug("default", "key", "BACKUP_DIR", "value", cfg.Backup.Dir)
	}
	if cfg.Backup.RetentionDays == 0 {
		cfg.Backup.RetentionDays = 30
		slog.Debug("default", "key", "BACKUP_RETENTION_DAYS", "value", fmt.Sprintf("%d", cfg.Backup.RetentionDays))
	}
	if cfg.Backup.Schedule == "" {
		cfg.Backup.Schedule = "0 2 * * *"
		slog.Debug("default", "key", "BACKUP_SCHEDULE", "value", cfg.Backup.Schedule)
	}
	if cfg.Backup.ScheduleFull == "" {
		cfg.Backup.ScheduleFull = "0 3 * * *"
		slog.Debug("default", "key", "BACKUP_SCHEDULE_FULL", "value", cfg.Backup.ScheduleFull)
	}
	if cfg.Backup.WALInterval == 0 {
		cfg.Backup.WALInterval = 60
		slog.Debug("default", "key", "BACKUP_WAL_INTERVAL_SECONDS", "value", "60")
	}
	if cfg.Backup.RetentionDaily == 0 {
		cfg.Backup.RetentionDaily = 7
		slog.Debug("default", "key", "BACKUP_RETENTION_DAILY", "value", "7")
	}
	if cfg.Backup.RetentionWeekly == 0 {
		cfg.Backup.RetentionWeekly = 4
		slog.Debug("default", "key", "BACKUP_RETENTION_WEEKLY", "value", "4")
	}
	if cfg.Backup.RetentionMonthly == 0 {
		cfg.Backup.RetentionMonthly = 12
		slog.Debug("default", "key", "BACKUP_RETENTION_MONTHLY", "value", "12")
	}
	if cfg.Backup.RestoreTestSchedule == "" {
		cfg.Backup.RestoreTestSchedule = "0 5 * * 0"
		slog.Debug("default", "key", "BACKUP_RESTORE_TEST_SCHEDULE", "value", cfg.Backup.RestoreTestSchedule)
	}

	// ── Email ─────────────────────────────────────────────────────────
	if cfg.Email.Provider == "" {
		cfg.Email.Provider = "mailpit"
		slog.Debug("default", "key", "EMAIL_PROVIDER", "value", cfg.Email.Provider)
	}
	if cfg.Email.From == "" {
		cfg.Email.From = "noreply@" + cfg.BaseDomain
		slog.Debug("default", "key", "EMAIL_FROM", "value", cfg.Email.From)
	}
	if cfg.Email.AWSRegion == "" {
		cfg.Email.AWSRegion = "us-east-1"
		slog.Debug("default", "key", "EMAIL_AWS_REGION", "value", cfg.Email.AWSRegion)
	}
	if cfg.Email.SMTPPort == 0 {
		cfg.Email.SMTPPort = 587
		slog.Debug("default", "key", "SMTP_PORT", "value", "587")
	}

	return cfg, nil
}

// BuildJWTSecret constructs the HASURA_GRAPHQL_JWT_SECRET JSON string.
// If the environment variable HASURA_GRAPHQL_JWT_SECRET is already set,
// it is returned directly. Otherwise the secret is constructed from
// cfg.Hasura.JWTKey and cfg.Hasura.JWTType. In dev mode, a missing
// JWTKey is auto-generated. In non-dev modes, an empty JWTKey produces
// an empty string (the caller must validate).
func BuildJWTSecret(cfg *Config) (string, error) {
	// If the full JSON is already set in the environment, use it as-is.
	if existing := os.Getenv("HASURA_GRAPHQL_JWT_SECRET"); existing != "" {
		return existing, nil
	}

	jwtType := cfg.Hasura.JWTType
	if jwtType == "" {
		jwtType = "RS256" // SEC-JWT-01: RS256 is the new default
	}
	if strings.EqualFold(jwtType, "HS256") {
		slog.Warn("SEC-JWT-01: HASURA_JWT_TYPE=HS256 is deprecated. Migrate to RS256. See docs.nself.org/security/jwt-migration.")
	}

	jwtKey := cfg.Hasura.JWTKey
	if jwtKey == "" {
		// Auto-generate if not provided in any environment.
		// Staging/prod should ideally set this explicitly, but auto-gen
		// is safer than failing — the user can always override in .env.
		secret, err := generateSecureRandom(44)
		if err != nil {
			return "", fmt.Errorf("generating JWT key: %w", err)
		}
		jwtKey = secret
		cfg.Hasura.JWTKey = jwtKey
	}

	obj := map[string]string{"type": jwtType, "key": jwtKey}
	b, _ := json.Marshal(obj)
	return string(b), nil
}

// DatabaseURL returns the computed PostgreSQL connection string using internal
// container networking (always port 5432, host "postgres"). The password is
// percent-encoded per RFC 3986 for safe URL inclusion.
func (cfg *Config) DatabaseURL() string {
	password := url.PathEscape(cfg.Postgres.Password)
	return fmt.Sprintf("postgresql://%s:%s@%s:5432/%s",
		cfg.Postgres.User,
		password,
		cfg.Postgres.Host,
		cfg.Postgres.DB,
	)
}

// EmbeddedPGDatabaseURL returns a PostgreSQL DSN that connects via the
// Unix-domain socket bridge created by the pglite/wasmtime embedded runtime.
// The host field is the runtimeDir path; sslmode=disable is required because
// the embedded runtime does not perform TLS termination.
//
// Use this DSN only when cfg.EmbeddedPG is true and runtimeDir is the
// directory passed to embedded.NewEmbeddedPGRuntime.
func (cfg *Config) EmbeddedPGDatabaseURL(runtimeDir string) string {
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}
	// The host= DSN key accepts a directory path for UDS connections in libpq.
	return fmt.Sprintf("host=%s dbname=%s sslmode=disable", runtimeDir, db)
}

// BuildServiceURL constructs a full HTTPS URL for a service subdomain against
// the given baseDomain. If baseDomain already starts with "subdomain.", that
// prefix is stripped first to avoid double-prefixing (e.g. "auth.auth.example.com").
//
// Examples:
//
//	BuildServiceURL("auth", "auth.example.com") → "https://auth.example.com"
//	BuildServiceURL("auth", "example.com")      → "https://auth.example.com"
func BuildServiceURL(subdomain, baseDomain string) string {
	cleanDomain := strings.TrimPrefix(baseDomain, subdomain+".")
	return "https://" + subdomain + "." + cleanDomain
}

// generateSecureRandom returns a crypto/rand base64url string of the given
// length. Used for auto-generating JWT keys and other secrets in dev mode.
func generateSecureRandom(length int) (string, error) {
	// Allocate enough bytes to guarantee the encoded output is >= length.
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating secret: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(b)
	if len(encoded) < length {
		return encoded, nil
	}
	return encoded[:length], nil
}
