package config

// loader_parse_env.go — maps every environment variable to its Config struct field.
//
// Purpose: Single canonical mapping from env var name strings to typed Config
//          struct fields. Every section (Core, Postgres, Hasura, Auth, Nginx,
//          SSL, WAF, Redis, Minio, Mailpit, Functions, MLflow, Admin, Search,
//          Monitoring, Email, Backup, DR, PluginPro, PluginSystem, Docker,
//          Start/Stop) is mapped in one place so the connection between env var
//          name and field is unambiguous and grep-friendly.
// Inputs:  os.Environ (read via os.Getenv, getEnvOr, getEnvInt, getEnvBool).
// Outputs: *Config — fully populated struct (no defaults yet; ApplyDefaults
//          fills zero values after this function returns).
// Constraints: Must not call ApplyDefaults or touch the filesystem. Pure
//              os.Getenv reads only. Keep in sync with loader_known_vars.go.
// SPORT:   cli/internal/config — decomposed from loader.go (T-E2-06).

import "os"

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
	// JWT-ALGO-01 / gap #4: populate cfg.Hasura.JWTKey/JWTType from whatever
	// source already has the value, so a previously-generated (or user-supplied)
	// key survives every rebuild — including --force — instead of ApplyDefaults
	// generating a brand new one because HASURA_JWT_KEY specifically was unset.
	// Priority (highest first): HASURA_JWT_KEY/HASURA_JWT_TYPE (already applied
	// above) > HASURA_GRAPHQL_JWT_SECRET JSON (the full persisted secret written
	// to .env.secrets by persistGeneratedSecrets) > AUTH_JWT_SECRET/AUTH_JWT_TYPE
	// and AUTH_JWT_KEY (the real-world var names apps like ntask declare in
	// their .env for the same underlying key material).
	if cfg.Hasura.JWTKey == "" {
		if key, typ, ok := parseHasuraJWTSecretJSON(os.Getenv("HASURA_GRAPHQL_JWT_SECRET")); ok {
			cfg.Hasura.JWTKey = key
			if cfg.Hasura.JWTType == "" {
				cfg.Hasura.JWTType = typ
			}
		}
	}
	if cfg.Hasura.JWTKey == "" {
		cfg.Hasura.JWTKey = os.Getenv("AUTH_JWT_SECRET")
	}
	if cfg.Hasura.JWTKey == "" {
		cfg.Hasura.JWTKey = os.Getenv("AUTH_JWT_KEY")
	}
	if cfg.Hasura.JWTType == "" {
		cfg.Hasura.JWTType = os.Getenv("AUTH_JWT_TYPE")
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
		RateLimitAPI:  os.Getenv("RATE_LIMIT_API_RPS"),
		RateLimitAuth: os.Getenv("RATE_LIMIT_AUTH_RPS"),
		RateLimitAI:   os.Getenv("RATE_LIMIT_AI_RPS"),
	}

	// ── SSL ──────────────────────────────────────────────────────────
	cfg.SSLMode = os.Getenv("SSL_MODE")
	cfg.SSLProvider = os.Getenv("SSL_PROVIDER")
	cfg.SSLWildcardDomain = os.Getenv("SSL_WILDCARD_DOMAIN")
	cfg.ExtraSSLDomains = os.Getenv("EXTRA_SSL_DOMAINS")
	cfg.CloudflareAPIKey = os.Getenv("CLOUDFLARE_API_KEY")

	// ── WAF ──────────────────────────────────────────────────────────
	cfg.WAFMode = os.Getenv("WAF_MODE")

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
	// Gap #8: also infer intent to enable storage when the user has explicitly
	// set MinIO-specific credentials/config (MINIO_ROOT_USER/PASSWORD, their
	// MINIO_ACCESS_KEY/MINIO_SECRET_KEY aliases, or S3_ACCESS_KEY/S3_SECRET_KEY)
	// without also setting MINIO_ENABLED/STORAGE_ENABLED. This matches how apps
	// like ntask declare a full MinIO credential surface in .env.example but
	// never set the ENABLED toggle — the storage service must still generate.
	_, minioEnabledSet := os.LookupEnv("MINIO_ENABLED")
	_, storageEnabledSet := os.LookupEnv("STORAGE_ENABLED")
	minioIntentVars := []string{
		"MINIO_ROOT_USER", "MINIO_ROOT_PASSWORD",
		"MINIO_ACCESS_KEY", "MINIO_SECRET_KEY",
		"S3_ACCESS_KEY", "S3_SECRET_KEY",
	}
	hasMinioIntent := false
	for _, k := range minioIntentVars {
		if v, ok := os.LookupEnv(k); ok && v != "" {
			hasMinioIntent = true
			break
		}
	}
	minioEnabled := getEnvBool("MINIO_ENABLED", false) || getEnvBool("STORAGE_ENABLED", false)
	if !minioEnabled && !minioEnabledSet && !storageEnabledSet && hasMinioIntent {
		minioEnabled = true
	}

	// MINIO_ACCESS_KEY/MINIO_SECRET_KEY alias to MINIO_ROOT_USER/MINIO_ROOT_PASSWORD
	// (gap #2). The internal MinIO container always reads MINIO_ROOT_USER/
	// MINIO_ROOT_PASSWORD; apps commonly document the S3-style ACCESS/SECRET
	// key names instead. ROOT_* wins when both are set, for back-compat with
	// anyone already using ROOT_* directly.
	rootUser := os.Getenv("MINIO_ROOT_USER")
	if rootUser == "" {
		rootUser = os.Getenv("MINIO_ACCESS_KEY")
	}
	rootPassword := os.Getenv("MINIO_ROOT_PASSWORD")
	if rootPassword == "" {
		rootPassword = os.Getenv("MINIO_SECRET_KEY")
	}

	cfg.Minio = MinioConfig{
		Enabled:        minioEnabled,
		Version:        os.Getenv("MINIO_VERSION"),
		Port:           getEnvInt("MINIO_PORT", 0),
		ConsolePort:    getEnvInt("MINIO_CONSOLE_PORT", 0),
		RootUser:       rootUser,
		RootPassword:   rootPassword,
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
		Enabled:             getEnvBool("BACKUP_ENABLED", false),
		Dir:                 os.Getenv("BACKUP_DIR"),
		Schedule:            os.Getenv("BACKUP_SCHEDULE"),
		RetentionDays:       getEnvInt("BACKUP_RETENTION_DAYS", 0),
		CloudProvider:       os.Getenv("BACKUP_CLOUD_PROVIDER"),
		Remote:              os.Getenv("BACKUP_REMOTE"),
		Encryption:          getEnvBool("BACKUP_ENCRYPTION", false),
		AgeRecipients:       os.Getenv("BACKUP_AGE_RECIPIENTS"),
		ScheduleFull:        os.Getenv("BACKUP_SCHEDULE_FULL"),
		WALInterval:         getEnvInt("BACKUP_WAL_INTERVAL_SECONDS", 0),
		RetentionDaily:      getEnvInt("BACKUP_RETENTION_DAILY", 0),
		RetentionWeekly:     getEnvInt("BACKUP_RETENTION_WEEKLY", 0),
		RetentionMonthly:    getEnvInt("BACKUP_RETENTION_MONTHLY", 0),
		RestoreTestSchedule: os.Getenv("BACKUP_RESTORE_TEST_SCHEDULE"),
		AlertOnFailure:      getEnvBool("BACKUP_ALERT_ON_FAILURE", true),
		S3AccessKeyID:       os.Getenv("BACKUP_S3_ACCESS_KEY_ID"),
		S3SecretAccessKey:   os.Getenv("BACKUP_S3_SECRET_ACCESS_KEY"),
		S3Region:            os.Getenv("BACKUP_S3_REGION"),
		S3Endpoint:          os.Getenv("BACKUP_S3_ENDPOINT"),
	}

	cfg.DR = DRConfig{
		SecondaryRegion: os.Getenv("DR_SECONDARY_REGION"),
		StandbyHost:     os.Getenv("DR_STANDBY_HOST"),
		DrillSchedule:   os.Getenv("DR_DRILL_SCHEDULE"),
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
