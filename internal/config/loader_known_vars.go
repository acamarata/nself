package config

// loader_known_vars.go — authoritative list of known environment variable names.
//
// Purpose: Enumerate every env var name that the nSelf config loader and
//          ApplyDefaults recognise. Used by warnUnknownEnvVars to surface
//          typos in user .env files (any key not in this list and not matching
//          a dynamic prefix emits a slog.Warn — it never fails the load).
// Inputs:  none (package-level var, referenced by loader.go and defaults.go).
// Outputs: knownEnvVars []string — consumed by warnUnknownEnvVars in warn.go.
// Constraints: Keep in sync with parseEnvToConfig (loader_parse_env.go) and
//              ApplyDefaults (defaults.go). Plugin-managed vars that the CLI
//              loader does NOT read are included at the bottom to suppress false
//              "unknown env var" warnings from compose-injected config.
// SPORT:   cli/internal/config — decomposed from loader.go (T-E2-06).

var knownEnvVars = []string{
	// Core
	"PROJECT_NAME",
	"BASE_DOMAIN",
	"PROJECT_DOMAIN",
	"ENV",
	"PROJECT_DESCRIPTION",
	"ADMIN_EMAIL",
	"DB_ENV_SEEDS",
	// APP_NAME: opt-in subdomain prefix for multi-app nginx routing (gap #5).
	"APP_NAME",
	// DOMAIN/ACME_EMAIL are the Traefik/staging-prod convention some app repos
	// use alongside (or instead of) BASE_DOMAIN — see ntask backend/.env.example.
	// Not read by the CLI loader directly; listed to suppress false warnings.
	"DOMAIN",
	"ACME_EMAIL",
	// App-level demo/seed toggle (read by app seed scripts, not the CLI loader).
	"DEMO_SEED",

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
	// HASURA_ADMIN_SECRET is a commonly-used alias for HASURA_GRAPHQL_ADMIN_SECRET
	// seen in real app .env files (e.g. ntask/backend/.env.example). The loader
	// does not read it directly today; listed here to suppress false warnings.
	"HASURA_ADMIN_SECRET",
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
	// Real-world Hasura tuning vars declared by apps (not yet read by the
	// loader as typed fields, but legitimate Hasura engine config surface).
	"HASURA_GRAPHQL_ENABLE_TELEMETRY",
	"HASURA_GRAPHQL_ENABLE_ALLOWLIST",
	"HASURA_GRAPHQL_UNAUTHORIZED_ROLE",
	"HASURA_GRAPHQL_NODE_LIMIT",
	"HASURA_GRAPHQL_DEPTH_LIMIT",
	"HASURA_GRAPHQL_BATCH_SIZE",
	"HASURA_GRAPHQL_LIVE_QUERIES_MULTIPLEXED_BATCH_SIZE",
	"HASURA_GRAPHQL_LIVE_QUERIES_MULTIPLEXED_REFETCH_INTERVAL",
	// Hasura Actions/cron webhook callback base URL (gap #11: wired into the
	// generated Hasura service env so Actions can call back into `functions`).
	"ACTION_HANDLER_URL",

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
	// Auth-mode toggle (bundled vs external shared auth — see AUTH_MODE doctrine
	// in app .env files) and the shared-auth vars used in external mode.
	"AUTH_MODE",
	"SHARED_AUTH_JWT_SECRET",
	"SHARED_AUTH_SERVER_URL",
	// hasura-auth container feature flags declared by real app .env files.
	"AUTH_ANONYMOUS_USERS_ENABLED",
	"AUTH_DISABLE_NEW_USERS",
	"AUTH_EMAIL_PASSWORDLESS_ENABLED",
	"AUTH_REQUIRE_EMAIL_VERIFICATION",
	"AUTH_EMAIL_TEMPLATE_FETCH_URL",
	"AUTH_GRAVATAR_ENABLED",
	"AUTH_LOCALE_DEFAULT",
	// hasura-auth internal URL (used by app functions to call the auth admin API).
	"HASURA_AUTH_URL",
	// hasura-auth SMTP vars (distinct prefix from AUTH_SMTP_* — hasura-auth
	// reads these directly; the CLI passes them through unmodified).
	"HASURA_AUTH_SMTP_HOST",
	"HASURA_AUTH_SMTP_PORT",
	"HASURA_AUTH_SMTP_USER",
	"HASURA_AUTH_SMTP_PASS",
	"HASURA_AUTH_SMTP_SENDER",
	"HASURA_AUTH_SMTP_SECURE",
	"HASURA_AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED",

	// Nginx
	"NGINX_VERSION",
	"NGINX_HTTP_PORT",
	"NGINX_PORT",
	"NGINX_HTTPS_PORT",
	"NGINX_SSL_PORT",
	"NGINX_CLIENT_MAX_BODY_SIZE",
	"NGINX_BIND_IP",
	// Per-zone rate limit overrides — read by parseEnvToConfig into
	// NginxConfig.RateLimitAPI/RateLimitAuth/RateLimitAI but were missing from
	// this schema list (pre-existing gap, fixed alongside gap #1).
	"RATE_LIMIT_API_RPS",
	"RATE_LIMIT_AUTH_RPS",
	"RATE_LIMIT_AI_RPS",
	// RPM-suffixed variants seen in real app .env files (e.g. ntask). Not read
	// by the CLI loader (the RPS vars above are canonical); listed to suppress
	// false warnings.
	"RATE_LIMIT_AUTH_RPM",
	"RATE_LIMIT_GRAPHQL_RPM",
	"RATE_LIMIT_UPLOADS_RPM",

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
	// MINIO_ACCESS_KEY/MINIO_SECRET_KEY are documented aliases for
	// MINIO_ROOT_USER/MINIO_ROOT_PASSWORD (gap #2). The loader aliases them in
	// parseEnvToConfig when the ROOT_* vars are not already set.
	"MINIO_ACCESS_KEY",
	"MINIO_SECRET_KEY",
	"MINIO_DEFAULT_BUCKETS",
	"MINIO_REGION",
	"S3_ACCESS_KEY",
	"S3_SECRET_KEY",
	"S3_BUCKET",
	// MINIO_BUCKET is an app-level alias for S3_BUCKET seen in real .env files;
	// not read by the CLI loader (apps use it directly), listed to suppress
	// false warnings.
	"MINIO_BUCKET",
	// MINIO_ENDPOINT/MINIO_ENDPOINT_PUBLIC are app-level MinIO endpoint URLs
	// (internal Docker network + public presigned-URL host). Compose-computed
	// equivalents exist as S3_ENDPOINT; these are read directly by apps.
	"MINIO_ENDPOINT",
	"MINIO_ENDPOINT_PUBLIC",
	"STORAGE_VERSION",
	"STORAGE_ROUTE",
	"STORAGE_CONSOLE_ROUTE",
	// STORAGE_PUBLIC_URL is the externally-reachable storage URL apps use for
	// generating download links; STORAGE_PORT is already compose-computed below.
	"STORAGE_PUBLIC_URL",
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
	// Mailhog — legacy predecessor to Mailpit, still referenced by older app
	// .env files (e.g. ntask). Not read by the CLI loader.
	"MAILHOG_SMTP_PORT",
	"MAILHOG_UI_PORT",

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
	"MEILISEARCH_WARMUP_QUERIES",
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
	// SMTP_SENDER is the "From" address companion to SMTP_* above, used
	// directly by app mailers (distinct from the CLI's EMAIL_FROM field).
	"SMTP_SENDER",

	// Backup
	"BACKUP_ENABLED",
	"BACKUP_DIR",
	"BACKUP_SCHEDULE",
	"BACKUP_RETENTION_DAYS",
	"BACKUP_CLOUD_PROVIDER",
	"BACKUP_REMOTE",
	"BACKUP_ENCRYPTION",
	"BACKUP_AGE_RECIPIENTS",
	"BACKUP_SCHEDULE_FULL",
	"BACKUP_WAL_INTERVAL_SECONDS",
	"BACKUP_RETENTION_DAILY",
	"BACKUP_RETENTION_WEEKLY",
	"BACKUP_RETENTION_MONTHLY",
	"BACKUP_RESTORE_TEST_SCHEDULE",
	"BACKUP_ALERT_ON_FAILURE",
	"BACKUP_S3_ACCESS_KEY_ID",
	"BACKUP_S3_SECRET_ACCESS_KEY",
	"BACKUP_S3_REGION",
	"BACKUP_S3_ENDPOINT",
	// App-level backup credential/target aliases seen in real .env files
	// (e.g. ntask). Not read by the CLI loader (app backup scripts use them).
	"BACKUP_ACCESS_KEY",
	"BACKUP_SECRET_KEY",
	"BACKUP_S3_BUCKET",
	"BACKUP_S3_PREFIX",

	// Disaster Recovery
	"DR_SECONDARY_REGION",
	"DR_STANDBY_HOST",
	"DR_DRILL_SCHEDULE",
	// DR_DATABASE_URL is the standby/replica connection string used by app-level
	// DR tooling; not read by the CLI loader.
	"DR_DATABASE_URL",

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
	// NOTE: AUTH_JWT_SECRET and AUTH_JWT_KEY ARE read directly by
	// parseEnvToConfig (gap #4 fix) as fallback sources for cfg.Hasura.JWTKey,
	// so a user-declared value survives every rebuild instead of being
	// silently ignored and regenerated.
	"AUTH_HOST",
	"AUTH_SERVER_URL",
	"AUTH_JWT_SECRET",
	"AUTH_JWT_KEY",
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
	// Social OAuth env vars — auth server (P2-E2-W3-S5-T16 + P2-E2-W3-S6-T17)
	// Shared
	"OAUTH_REDIRECT_ALLOWLIST",
	"OAUTH_TOKEN_ENCRYPTION_KEY",
	// Tier-1 (T16)
	"GOOGLE_OAUTH_CLIENT_ID",
	"GOOGLE_OAUTH_CLIENT_SECRET",
	"APPLE_OAUTH_CLIENT_ID",
	"APPLE_OAUTH_TEAM_ID",
	"APPLE_OAUTH_KEY_ID",
	"APPLE_OAUTH_PRIVATE_KEY",
	// Note: Apple does not use a static client_secret; the above key material
	// is used to generate a short-lived ES256 JWT per RFC 7636 on every token exchange.
	"GITHUB_OAUTH_CLIENT_ID",
	"GITHUB_OAUTH_CLIENT_SECRET",
	"FACEBOOK_OAUTH_CLIENT_ID",
	"FACEBOOK_OAUTH_CLIENT_SECRET",
	"TWITTER_OAUTH_CLIENT_ID",
	"TWITTER_OAUTH_CLIENT_SECRET",
	// Tier-2 Group A (T17a) — LinkedIn, Discord, Slack, Twitch
	"LINKEDIN_OAUTH_CLIENT_ID",
	"LINKEDIN_OAUTH_CLIENT_SECRET",
	"DISCORD_OAUTH_CLIENT_ID",
	"DISCORD_OAUTH_CLIENT_SECRET",
	"SLACK_OAUTH_CLIENT_ID",
	"SLACK_OAUTH_CLIENT_SECRET",
	"TWITCH_OAUTH_CLIENT_ID",
	"TWITCH_OAUTH_CLIENT_SECRET",
	// Tier-2 Group B (T17b) — Spotify, TikTok, Reddit, Microsoft
	"SPOTIFY_OAUTH_CLIENT_ID",
	"SPOTIFY_OAUTH_CLIENT_SECRET",
	"TIKTOK_OAUTH_CLIENT_ID",
	"TIKTOK_OAUTH_CLIENT_SECRET",
	"REDDIT_OAUTH_CLIENT_ID",
	"REDDIT_OAUTH_CLIENT_SECRET",
	"MICROSOFT_OAUTH_CLIENT_ID",
	"MICROSOFT_OAUTH_CLIENT_SECRET",
	// Tier-2 Gap-list (T17d) — reaching 21 total
	"PINTEREST_OAUTH_CLIENT_ID",
	"PINTEREST_OAUTH_CLIENT_SECRET",
	"DROPBOX_OAUTH_CLIENT_ID",
	"DROPBOX_OAUTH_CLIENT_SECRET",
	"ZOOM_OAUTH_CLIENT_ID",
	"ZOOM_OAUTH_CLIENT_SECRET",
	"ATLASSIAN_OAUTH_CLIENT_ID",
	"ATLASSIAN_OAUTH_CLIENT_SECRET",
	"GITLAB_OAUTH_CLIENT_ID",
	"GITLAB_OAUTH_CLIENT_SECRET",
	"BITBUCKET_OAUTH_CLIENT_ID",
	"BITBUCKET_OAUTH_CLIENT_SECRET",
	"FIGMA_OAUTH_CLIENT_ID",
	"FIGMA_OAUTH_CLIENT_SECRET",
	"NOTION_OAUTH_CLIENT_ID",
	"NOTION_OAUTH_CLIENT_SECRET",
	// Total: 5 Tier-1 + 16 Tier-2 = 21 OAuth providers (full Zernio parity)

	// App-level push notification credentials (mobile/RN apps talking directly
	// to APNs/FCM, distinct from the notify plugin's VAPID web-push keys above).
	// Not read by the CLI loader — listed to suppress false warnings.
	"APNS_KEY_ID",
	"APNS_KEY_PATH",
	"APNS_TEAM_ID",
	"FCM_SERVER_KEY",

	// App-level observability DSN (e.g. Sentry backend project). Not read by
	// the CLI loader — apps wire this into their own error reporting.
	"SENTRY_DSN_BACKEND",

	// App-owned vars commonly present in real app .env files (read by the
	// app's own code / Node runtime / dev scripts, not by the CLI loader).
	// Listed to suppress false "unknown env var" warnings (ntask dogfood
	// gap #19). Project-specific vars beyond these belong in ENV_ALLOWLIST.
	"NODE_ENV",
	"JWT_SECRET",
	"SSL_AUTO_TRUST",
	"COOKIE_SECRET",
	"ENABLE_DEBUG",
	"LOG_LEVEL",
	"NSELF_PROJECT_NAME",

	// ENV_ALLOWLIST itself: comma-separated var names (or prefixes ending
	// in *) that warnUnknownEnvVars treats as app-owned and never warns
	// about. Example: ENV_ALLOWLIST=MY_APP_TOKEN,FEATURE_*
	"ENV_ALLOWLIST",
}
