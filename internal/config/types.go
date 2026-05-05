package config

// Config is the top-level configuration struct for an nSelf project.
// All fields are populated from the .env cascade and environment variables.
type Config struct {
	// Core
	ProjectName        string `env:"PROJECT_NAME"`
	BaseDomain         string `env:"BASE_DOMAIN"`
	Env                string `env:"ENV"` // dev, staging, prod
	ProjectDescription string `env:"PROJECT_DESCRIPTION"`
	AdminEmail         string `env:"ADMIN_EMAIL"`
	DBEnvSeeds         bool   `env:"DB_ENV_SEEDS"`

	// PostgreSQL
	Postgres PostgresConfig

	// Hasura
	Hasura HasuraConfig

	// Auth
	Auth AuthConfig

	// Nginx
	Nginx NginxConfig

	// SSL
	SSLMode           string `env:"SSL_MODE"`            // local, letsencrypt, custom, none
	SSLProvider       string `env:"SSL_PROVIDER"`        // cloudflare, route53, digitalocean, custom
	SSLWildcardDomain string `env:"SSL_WILDCARD_DOMAIN"` // *.example.com
	ExtraSSLDomains   string `env:"EXTRA_SSL_DOMAINS"`   // comma-separated
	CloudflareAPIKey  string `env:"CLOUDFLARE_API_KEY"`  // DNS-01 challenge

	// WAF
	WAFMode string `env:"WAF_MODE"` // off, detection, blocking

	// Optional Services
	Redis      RedisConfig
	Minio      MinioConfig
	Mailpit    MailpitConfig
	Functions  FunctionsConfig
	MLflow     MLflowConfig
	Admin      AdminConfig
	Monitoring MonitoringConfig

	// Search (provider-agnostic)
	Search SearchConfig

	// Email Provider
	Email EmailConfig

	// PgBouncer connection pooler
	PgBouncer PgBouncerConfig

	// Backup & Recovery
	Backup BackupConfig

	// Disaster Recovery
	DR DRConfig

	// Multi-Tenancy & Billing
	Tenant TenantConfig

	// License
	License LicenseConfig

	// Secrets Management
	Secrets SecretsConfig

	// Plugin Pro Configuration
	PluginConfig PluginProConfig

	// Plugin System
	PluginSystem PluginSystemConfig

	// API Docs (Scalar)
	ApiDocs ApiDocsConfig

	// Custom Services
	CustomServices []CustomService // CS_1..CS_10

	// Frontend Apps
	FrontendApps []FrontendApp // FRONTEND_APP_1..FRONTEND_APP_20

	// Remote Schemas
	RemoteSchemas []RemoteSchema

	// Internal Routes (up to 20)
	InternalRoutes []InternalRoute

	// Docker
	DockerNetwork      string `env:"DOCKER_NETWORK"`
	DockerLogMaxSize   string `env:"DOCKER_LOG_MAX_SIZE"`        // 10m
	DockerLogMaxFile   string `env:"DOCKER_LOG_MAX_FILE"`        // 3
	DockerStopGrace    string `env:"DOCKER_STOP_GRACE_PERIOD"`   // 30s
	DockerBuildTimeout int    `env:"NSELF_DOCKER_BUILD_TIMEOUT"` // 300

	// Start/Stop behavior
	StartMode           string `env:"NSELF_START_MODE"`           // smart, fresh, force
	HealthCheckTimeout  int    `env:"NSELF_HEALTH_CHECK_TIMEOUT"` // seconds
	HealthCheckInterval int    `env:"NSELF_HEALTH_CHECK_INTERVAL"`
	HealthCheckRequired int    `env:"NSELF_HEALTH_CHECK_REQUIRED"` // percentage
	CleanupOnStart      string `env:"NSELF_CLEANUP_ON_START"`      // auto/always/never
	AllowExposedPorts   bool   `env:"NSELF_ALLOW_EXPOSED_PORTS"`
	ParallelLimit       int    `env:"NSELF_PARALLEL_LIMIT"` // 5
	LogLevel            string `env:"NSELF_LOG_LEVEL"`      // info
	SkipHealthChecks    bool   `env:"NSELF_SKIP_HEALTH_CHECKS"`
	StopTimeout         int    `env:"NSELF_STOP_TIMEOUT"` // 30

	// Federation — GraphQL Federation via Apollo Router (G05).
	// When true, nself build injects Apollo Router (CS_7) and composes a
	// supergraph schema from installed plugin subgraphs. Default: false.
	FederationEnabled bool `env:"NSELF_FEDERATION"`

	// Passthrough: arbitrary env vars matching patterns (AUTH_PROVIDER_*, REMOTE_SCHEMA_*, etc.)
	Passthrough map[string]string
}

// PostgresConfig holds PostgreSQL database configuration.
type PostgresConfig struct {
	Version    string   `env:"POSTGRES_VERSION"` // 16-alpine
	Host       string   `env:"POSTGRES_HOST"`    // postgres (container name)
	Port       int      `env:"POSTGRES_PORT"`    // 5432
	DB         string   `env:"POSTGRES_DB"`      // nself
	User       string   `env:"POSTGRES_USER"`    // postgres
	Password   string   `env:"POSTGRES_PASSWORD"`
	Extensions []string `env:"POSTGRES_EXTENSIONS"`  // comma-separated list
	ExposePort string   `env:"POSTGRES_EXPOSE_PORT"` // auto, true, false
	MemLimit   string   `env:"POSTGRES_MEM_LIMIT"`   // 2g
	CPULimit   string   `env:"POSTGRES_CPU_LIMIT"`   // 2.0
}

// PgBouncerConfig holds connection pooler configuration.
type PgBouncerConfig struct {
	Enabled           bool   `env:"PGBOUNCER_ENABLED"`
	Port              int    `env:"PGBOUNCER_PORT"`                // 6432
	PoolMode          string `env:"PGBOUNCER_POOL_MODE"`           // session, transaction, statement
	MaxClientConn     int    `env:"PGBOUNCER_MAX_CLIENT_CONN"`     // 100
	DefaultPoolSize   int    `env:"PGBOUNCER_DEFAULT_POOL_SIZE"`   // 25
	MinPoolSize       int    `env:"PGBOUNCER_MIN_POOL_SIZE"`       // 5
	ReservePoolSize   int    `env:"PGBOUNCER_RESERVE_POOL_SIZE"`   // 5
	ServerIdleTimeout int    `env:"PGBOUNCER_SERVER_IDLE_TIMEOUT"` // 600
	LogConnections    bool   `env:"PGBOUNCER_LOG_CONNECTIONS"`     // false
	LogDisconnections bool   `env:"PGBOUNCER_LOG_DISCONNECTIONS"`  // false
	AdminUsers        string `env:"PGBOUNCER_ADMIN_USERS"`         // postgres
	StatsUsers        string `env:"PGBOUNCER_STATS_USERS"`         // postgres
}

// HasuraConfig holds Hasura GraphQL engine configuration.
type HasuraConfig struct {
	Version     string `env:"HASURA_VERSION"`
	AdminSecret string `env:"HASURA_GRAPHQL_ADMIN_SECRET"`
	JWTKey      string `env:"HASURA_JWT_KEY"`
	JWTType     string `env:"HASURA_JWT_TYPE"` // HS256
	Console     bool   `env:"HASURA_GRAPHQL_ENABLE_CONSOLE"`
	DevMode     bool   `env:"HASURA_GRAPHQL_DEV_MODE"`
	CORSDomain  string `env:"HASURA_GRAPHQL_CORS_DOMAIN"`
	Route       string `env:"HASURA_ROUTE"` // api.{BASE_DOMAIN}
	Port        int    `env:"HASURA_PORT"`  // 8080
	MemLimit    string `env:"HASURA_MEM_LIMIT"`
	CPULimit    string `env:"HASURA_CPU_LIMIT"`
	LogLevel    string `env:"HASURA_GRAPHQL_LOG_LEVEL"` // warn
}

// AuthConfig holds authentication service configuration.
type AuthConfig struct {
	Version            string `env:"AUTH_VERSION"` // 0.36.0
	Port               int    `env:"AUTH_PORT"`    // 4000
	ClientURL          string `env:"AUTH_CLIENT_URL"`
	AccessTokenExpiry  int    `env:"AUTH_ACCESS_TOKEN_EXPIRES_IN"`  // seconds
	RefreshTokenExpiry int    `env:"AUTH_REFRESH_TOKEN_EXPIRES_IN"` // seconds
	Route              string `env:"AUTH_ROUTE"`
	SMTPHost           string `env:"AUTH_SMTP_HOST"`
	SMTPPort           int    `env:"AUTH_SMTP_PORT"`
	SMTPUser           string `env:"AUTH_SMTP_USER"`
	SMTPPass           string `env:"AUTH_SMTP_PASS"`
	SMTPSecure         bool   `env:"AUTH_SMTP_SECURE"`
	SMTPSender         string `env:"AUTH_SMTP_SENDER"`
	MemLimit           string `env:"AUTH_MEM_LIMIT"`           // 256m
	CPULimit           string `env:"AUTH_CPU_LIMIT"`           // 0.25
	ExtraRedirectURLs  string `env:"AUTH_EXTRA_REDIRECT_URLS"` // comma-separated extra redirect URLs
	WebAuthnEnabled    bool   `env:"AUTH_WEBAUTHN_ENABLED"`
	MFATOTPEnabled     bool   `env:"AUTH_MFA_TOTP_ENABLED"` // false
	LogLevel           string `env:"AUTH_LOG_LEVEL"`        // info
}

// NginxConfig holds Nginx reverse proxy configuration.
type NginxConfig struct {
	Version       string `env:"NGINX_VERSION"`              // alpine
	HTTPPort      int    `env:"NGINX_HTTP_PORT"`            // 80
	SSLPort       int    `env:"NGINX_HTTPS_PORT"`           // 443
	MaxBody       string `env:"NGINX_CLIENT_MAX_BODY_SIZE"` // 100M
	BindIP        string `env:"NGINX_BIND_IP"`              // computed: 127.0.0.1 (dev) or 0.0.0.0 (prod) — overridable
	AuthRateLimit string `env:"AUTH_RATE_LIMIT"`            // 30r/m
	RateLimitAPI  string `env:"RATE_LIMIT_API_RPS"`         // 30
	RateLimitAuth string `env:"RATE_LIMIT_AUTH_RPS"`        // 5
	RateLimitAI   string `env:"RATE_LIMIT_AI_RPS"`          // 10
}

// RedisConfig holds Redis cache/queue configuration.
type RedisConfig struct {
	Enabled  bool   `env:"REDIS_ENABLED"`
	Version  string `env:"REDIS_VERSION"`   // 7-alpine
	Port     int    `env:"REDIS_PORT"`      // 6379
	Password string `env:"REDIS_PASSWORD"`  // empty = no auth
	Memory   string `env:"REDIS_MEMORY"`    // 512M
	CPU      string `env:"REDIS_CPU"`       // 0.5
	PoolSize int    `env:"REDIS_POOL_SIZE"` // 50 (prod default); use 20 for dev
}

// MinioConfig holds MinIO S3-compatible object storage configuration.
type MinioConfig struct {
	Enabled        bool   `env:"MINIO_ENABLED"`
	Version        string `env:"MINIO_VERSION"`         // latest
	Port           int    `env:"MINIO_PORT"`            // 9000
	ConsolePort    int    `env:"MINIO_CONSOLE_PORT"`    // 9001
	RootUser       string `env:"MINIO_ROOT_USER"`       // minioadmin
	RootPassword   string `env:"MINIO_ROOT_PASSWORD"`   // minioadmin
	DefaultBuckets string `env:"MINIO_DEFAULT_BUCKETS"` // uploads,public,private,temp
	Region         string `env:"MINIO_REGION"`          // us-east-1
	S3AccessKey    string `env:"S3_ACCESS_KEY"`
	S3SecretKey    string `env:"S3_SECRET_KEY"`
	S3Bucket       string `env:"S3_BUCKET"`             // nself
	StorageVersion string `env:"STORAGE_VERSION"`       // 0.6.1
	StorageRoute   string `env:"STORAGE_ROUTE"`         // storage.{BD}
	ConsoleRoute   string `env:"STORAGE_CONSOLE_ROUTE"` // storage-console.{BD}
	MemLimit       string `env:"MINIO_MEMORY"`          // 1G
	CPULimit       string `env:"MINIO_CPU"`             // 0.5
}

// MailpitConfig holds Mailpit local email testing configuration.
type MailpitConfig struct {
	Enabled     bool   `env:"MAILPIT_ENABLED"`
	Version     string `env:"MAILPIT_VERSION"`      // latest
	SMTPPort    int    `env:"MAILPIT_SMTP_PORT"`    // 1025
	UIPort      int    `env:"MAILPIT_UI_PORT"`      // 8025
	MaxMessages int    `env:"MAILPIT_MAX_MESSAGES"` // 500
	Route       string `env:"MAILPIT_ROUTE"`        // mail.{BD}
	UIUser      string `env:"MAILPIT_UI_USER"`      // admin (default)
	UIPassword  string `env:"MAILPIT_UI_PASSWORD"`
}

// FunctionsConfig holds serverless functions runtime configuration.
type FunctionsConfig struct {
	Enabled bool   `env:"FUNCTIONS_ENABLED"`
	Version string `env:"FUNCTIONS_VERSION"` // latest
	Port    int    `env:"FUNCTIONS_PORT"`    // 3008
	Route   string `env:"FUNCTIONS_ROUTE"`   // functions.{BD}

	// Runtime selects the container image: node (default), deno, python.
	Runtime string `env:"FUNCTIONS_RUNTIME"` // node|deno|python

	// Resource limits for the functions container.
	Memory  string `env:"FUNCTIONS_MEMORY"`  // 256M
	CPU     string `env:"FUNCTIONS_CPU"`     // 0.5
	Timeout string `env:"FUNCTIONS_TIMEOUT"` // 30s
}

// MLflowConfig holds MLflow experiment tracking configuration.
// Compose generation is plugin-managed: nself plugin install mlflow
// Enabled, Route, and Port are read by ssl/domains.go, urls.go, and doctor.go.
// All other fields are consumed exclusively by the nself-mlflow plugin at install time.
type MLflowConfig struct {
	Enabled         bool   `env:"MLFLOW_ENABLED"`
	Route           string `env:"MLFLOW_ROUTE"`            // mlflow.{BD} — read by ssl/domains.go, urls.go, doctor.go
	Version         string `env:"MLFLOW_VERSION"`          // plugin-managed: populated by nself plugin install mlflow
	Port            int    `env:"MLFLOW_PORT"`             // read by doctor.go for port-conflict checks; plugin-managed: populated by nself plugin install mlflow
	DBName          string `env:"MLFLOW_DB_NAME"`          // plugin-managed: populated by nself plugin install mlflow
	ArtifactsBucket string `env:"MLFLOW_ARTIFACTS_BUCKET"` // plugin-managed: populated by nself plugin install mlflow
	AuthEnabled     bool   `env:"MLFLOW_AUTH_ENABLED"`     // plugin-managed: populated by nself plugin install mlflow
	AuthUsername    string `env:"MLFLOW_AUTH_USERNAME"`    // plugin-managed: populated by nself plugin install mlflow
	AuthPassword    string `env:"MLFLOW_AUTH_PASSWORD"`    // plugin-managed: populated by nself plugin install mlflow
}

// AdminConfig holds nSelf Admin GUI configuration.
type AdminConfig struct {
	Enabled      bool   `env:"NSELF_ADMIN_ENABLED"`
	Version      string `env:"NSELF_ADMIN_VERSION"`  // latest
	Port         int    `env:"NSELF_ADMIN_PORT"`     // 3021
	Route        string `env:"NSELF_ADMIN_ROUTE"`    // admin.{BD}
	DevMode      bool   `env:"NSELF_ADMIN_DEV"`      // false
	DevPort      int    `env:"NSELF_ADMIN_DEV_PORT"` // 3000
	SecretKey    string `env:"ADMIN_SECRET_KEY"`
	PasswordHash string `env:"ADMIN_PASSWORD_HASH"`
}

// SearchConfig holds search engine configuration (provider-agnostic).
type SearchConfig struct {
	Enabled     bool   `env:"SEARCH_ENABLED"`
	Engine      string `env:"SEARCH_ENGINE"`  // meilisearch, typesense, etc.
	Port        int    `env:"SEARCH_PORT"`    // auto from provider
	APIKey      string `env:"SEARCH_API_KEY"` // auto-generated if unset
	Route       string `env:"SEARCH_ROUTE"`   // search.{BD}
	IndexPrefix string `env:"SEARCH_INDEX_PREFIX"`
	AutoIndex   bool   `env:"SEARCH_AUTO_INDEX"` // true
	Language    string `env:"SEARCH_LANGUAGE"`   // en

	// Provider-specific (only populated for active provider)
	MeiliSearch   MeiliSearchConfig
	Typesense     TypesenseConfig
	Elasticsearch ElasticsearchConfig
}

// MeiliSearchConfig holds MeiliSearch-specific configuration.
type MeiliSearchConfig struct {
	Version   string `env:"MEILISEARCH_VERSION"` // v1.6
	MasterKey string `env:"MEILISEARCH_MASTER_KEY"`
	Env       string `env:"MEILISEARCH_ENV"` // development
}

// TypesenseConfig holds Typesense-specific configuration.
type TypesenseConfig struct {
	Version           string `env:"TYPESENSE_VERSION"` // 27.1
	APIKey            string `env:"TYPESENSE_API_KEY"`
	EnableCORS        bool   `env:"TYPESENSE_ENABLE_CORS"`
	LogLevel          string `env:"TYPESENSE_LOG_LEVEL"`
	NumMemoryShards   int    `env:"TYPESENSE_NUM_MEMORY_SHARDS"`
	SnapshotIntervalS int    `env:"TYPESENSE_SNAPSHOT_INTERVAL_SECONDS"`
}

// ElasticsearchConfig holds Elasticsearch-specific configuration.
type ElasticsearchConfig struct {
	Version  string `env:"ELASTICSEARCH_VERSION"` // 8.11.3
	Port     int    `env:"ELASTICSEARCH_PORT"`    // 9200
	Password string `env:"ELASTICSEARCH_PASSWORD"`
	Memory   string `env:"ELASTICSEARCH_MEMORY"` // 1Gi
}

// MonitoringConfig holds monitoring stack configuration.
// Compose generation is plugin-managed: nself plugin install monitoring
// Enabled, GrafanaEnabled, GrafanaRoute, GrafanaAdminPassword, and GrafanaPort are read by
// ssl/domains.go (SSL SANs), config/validator.go (password check), urls.go, and doctor.go.
// All other fields are consumed exclusively by the nself-monitoring plugin at install time.
type MonitoringConfig struct {
	Enabled              bool   `env:"MONITORING_ENABLED"`
	GrafanaEnabled       bool   `env:"GRAFANA_ENABLED"`
	GrafanaRoute         string `env:"GRAFANA_ROUTE"`             // read by ssl/domains.go, urls.go, doctor.go
	GrafanaAdminPassword string `env:"GRAFANA_ADMIN_PASSWORD"`    // read by config/validator.go
	PrometheusEnabled    bool   `env:"PROMETHEUS_ENABLED"`        // plugin-managed: populated by nself plugin install monitoring
	PrometheusPort       int    `env:"PROMETHEUS_PORT"`           // plugin-managed: populated by nself plugin install monitoring
	GrafanaPort          int    `env:"GRAFANA_PORT"`              // read by urls.go and doctor.go for port display; plugin-managed: populated by nself plugin install monitoring
	GrafanaAdminUser     string `env:"GRAFANA_ADMIN_USER"`        // plugin-managed: populated by nself plugin install monitoring
	LokiEnabled          bool   `env:"LOKI_ENABLED"`              // plugin-managed: populated by nself plugin install monitoring
	LokiPort             int    `env:"LOKI_PORT"`                 // plugin-managed: populated by nself plugin install monitoring
	PromtailEnabled      bool   `env:"PROMTAIL_ENABLED"`          // plugin-managed: populated by nself plugin install monitoring
	TempoEnabled         bool   `env:"TEMPO_ENABLED"`             // plugin-managed: populated by nself plugin install monitoring
	TempoPort            int    `env:"TEMPO_PORT"`                // plugin-managed: populated by nself plugin install monitoring
	AlertmanagerEnabled  bool   `env:"ALERTMANAGER_ENABLED"`      // plugin-managed: populated by nself plugin install monitoring
	AlertmanagerPort     int    `env:"ALERTMANAGER_PORT"`         // plugin-managed: populated by nself plugin install monitoring
	CadvisorEnabled      bool   `env:"CADVISOR_ENABLED"`          // plugin-managed: populated by nself plugin install monitoring
	CadvisorPort         int    `env:"CADVISOR_PORT"`             // plugin-managed: populated by nself plugin install monitoring
	NodeExporterEnabled  bool   `env:"NODE_EXPORTER_ENABLED"`     // plugin-managed: populated by nself plugin install monitoring
	NodeExporterPort     int    `env:"NODE_EXPORTER_PORT"`        // plugin-managed: populated by nself plugin install monitoring
	PGExporterEnabled    bool   `env:"POSTGRES_EXPORTER_ENABLED"` // plugin-managed: populated by nself plugin install monitoring
	PGExporterPort       int    `env:"POSTGRES_EXPORTER_PORT"`    // plugin-managed: populated by nself plugin install monitoring
	RedisExporterEnabled bool   `env:"REDIS_EXPORTER_ENABLED"`    // plugin-managed: populated by nself plugin install monitoring
	RedisExporterPort    int    `env:"REDIS_EXPORTER_PORT"`       // plugin-managed: populated by nself plugin install monitoring

	// S34 additions
	PrometheusRetention      string `env:"PROMETHEUS_RETENTION"`       // e.g. "30d"
	LokiHotDays              int    `env:"LOKI_HOT_DAYS"`              // default 30
	LokiColdDays             int    `env:"LOKI_COLD_DAYS"`             // default 365
	AlertmanagerPagerdutyKey string `env:"ALERTMANAGER_PAGERDUTY_KEY"` // PagerDuty integration key

	// Watchdog
	WatchdogEnabled                bool   `env:"WATCHDOG_ENABLED"`
	WatchdogCircuitBreakerAttempts int    `env:"WATCHDOG_CIRCUIT_BREAKER_ATTEMPTS"` // default 3
	WatchdogCircuitBreakerWindow   string `env:"WATCHDOG_CIRCUIT_BREAKER_WINDOW"`   // default 10m
	WatchdogEscalationWebhook      string `env:"WATCHDOG_ESCALATION_WEBHOOK"`

	// Queue/Jobs
	QueueWorkersPerQueue   int `env:"QUEUE_WORKERS_PER_QUEUE"`   // default 2
	QueueDLQAlertThreshold int `env:"QUEUE_DLQ_ALERT_THRESHOLD"` // default 100

	// Promotion
	PromoteRequiresTwoApprovers bool `env:"PROMOTE_REQUIRES_TWO_APPROVERS"`
}

// EmailConfig holds email provider configuration.
type EmailConfig struct {
	Provider            string `env:"EMAIL_PROVIDER"` // mailpit/elasticemail/sendgrid/postmark/mailgun/ses/smtp
	From                string `env:"EMAIL_FROM"`
	ElasticEmailAPIKey  string `env:"ELASTIC_EMAIL_API_KEY"`
	ElasticEmailAccount string `env:"ELASTIC_EMAIL_ACCOUNT_EMAIL"`
	SendGridAPIKey      string `env:"SENDGRID_API_KEY"`
	PostmarkAPIKey      string `env:"POSTMARK_API_KEY"`
	MailgunAPIKey       string `env:"MAILGUN_API_KEY"`
	MailgunDomain       string `env:"MAILGUN_DOMAIN"`
	AWSAccessKeyID      string `env:"AWS_ACCESS_KEY_ID"`
	AWSSecretAccessKey  string `env:"AWS_SECRET_ACCESS_KEY"`
	AWSRegion           string `env:"AWS_REGION"`
	SMTPHost            string `env:"SMTP_HOST"`
	SMTPPort            int    `env:"SMTP_PORT"`
	SMTPUser            string `env:"SMTP_USER"`
	SMTPPass            string `env:"SMTP_PASS"`
	SMTPSecure          bool   `env:"SMTP_SECURE"`
}

// BackupConfig holds backup and recovery configuration.
// Dir is read by internal/database/backup.go and restore.go for ad-hoc pg_dump/pg_restore.
// Scheduled, cloud, and retention features are managed via BACKUP_* env vars.
type BackupConfig struct {
	Dir           string `env:"BACKUP_DIR"` // ./backups — read by database/backup.go and restore.go
	Enabled       bool   `env:"BACKUP_ENABLED"`
	Schedule      string `env:"BACKUP_SCHEDULE"`       // legacy alias for BACKUP_SCHEDULE_FULL
	RetentionDays int    `env:"BACKUP_RETENTION_DAYS"` // legacy — use Daily/Weekly/Monthly instead
	CloudProvider string `env:"BACKUP_CLOUD_PROVIDER"` // legacy — use Remote instead

	// Cloud/remote storage
	Remote              string `env:"BACKUP_REMOTE"`                // rclone remote path, e.g. s3://bucket/path
	Encryption          bool   `env:"BACKUP_ENCRYPTION"`            // enable age encryption
	AgeRecipients       string `env:"BACKUP_AGE_RECIPIENTS"`        // age public key for encryption
	ScheduleFull        string `env:"BACKUP_SCHEDULE_FULL"`         // cron expr for full backups (default: 0 3 * * *)
	WALInterval         int    `env:"BACKUP_WAL_INTERVAL_SECONDS"`  // WAL archive interval (default: 60)
	RetentionDaily      int    `env:"BACKUP_RETENTION_DAILY"`       // keep last N daily backups (default: 7)
	RetentionWeekly     int    `env:"BACKUP_RETENTION_WEEKLY"`      // keep last N weekly backups (default: 4)
	RetentionMonthly    int    `env:"BACKUP_RETENTION_MONTHLY"`     // keep last N monthly backups (default: 12)
	RestoreTestSchedule string `env:"BACKUP_RESTORE_TEST_SCHEDULE"` // cron for restore tests (default: 0 5 * * 0)
	AlertOnFailure      bool   `env:"BACKUP_ALERT_ON_FAILURE"`      // send alert on backup failure
	S3AccessKeyID       string `env:"BACKUP_S3_ACCESS_KEY_ID"`
	S3SecretAccessKey   string `env:"BACKUP_S3_SECRET_ACCESS_KEY"`
	S3Region            string `env:"BACKUP_S3_REGION"`
	S3Endpoint          string `env:"BACKUP_S3_ENDPOINT"`
}

// LicenseConfig holds license validation and grace period configuration.
type LicenseConfig struct {
	PingURL           string `env:"LICENSE_PING_URL"`            // https://ping.nself.org
	CachePath         string `env:"LICENSE_CACHE_PATH"`          // ~/.cache/nself/license.json
	GraceDays         int    `env:"LICENSE_GRACE_DAYS"`          // 7
	CheckInterval     string `env:"LICENSE_CHECK_INTERVAL"`      // 6h
	OfflineMode       bool   `env:"LICENSE_OFFLINE_MODE"`        // false
	PublicKeyOverride string `env:"LICENSE_PUBLIC_KEY_OVERRIDE"` // hex-encoded Ed25519 pubkey for testing
}

// SecretsConfig holds secrets management configuration.
type SecretsConfig struct {
	AgeKeyPath   string `env:"SECRETS_AGE_KEY_PATH"` // ~/.config/nself/age-key.txt
	DeployAgeKey string `env:"DEPLOY_AGE_KEY"`       // raw age private key for CI/CD
}

// TenantConfig holds multi-tenancy and billing configuration.
type TenantConfig struct {
	DefaultPlan             string `env:"TENANT_DEFAULT_PLAN"`               // basic
	DestroyBackupRetainDays int    `env:"TENANT_DESTROY_BACKUP_RETAIN_DAYS"` // 90
	StripeSecretKey         string `env:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret     string `env:"STRIPE_WEBHOOK_SECRET"`
	StripeAPIVersion        string `env:"STRIPE_API_VERSION"` // 2024-04-10
}

// DRConfig holds disaster recovery configuration.
type DRConfig struct {
	SecondaryRegion string `env:"DR_SECONDARY_REGION"` // Hetzner region for standby
	StandbyHost     string `env:"DR_STANDBY_HOST"`     // IP/hostname of warm standby
	DrillSchedule   string `env:"DR_DRILL_SCHEDULE"`   // cron for DR drills (default: off)
}

// PluginProConfig holds per-plugin configuration for Pro plugins.
type PluginProConfig struct {
	NotifySecret    string `env:"NOTIFY_INTERNAL_SECRET"`
	NotifyPort      int    `env:"NOTIFY_PORT"` // 3712
	NotifyVAPIDPub  string `env:"NOTIFY_VAPID_PUBLIC_KEY"`
	NotifyVAPIDPriv string `env:"NOTIFY_VAPID_PRIVATE_KEY"`
	NotifyRoute     string `env:"NOTIFY_ROUTE"`
	CronSecret      string `env:"CRON_INTERNAL_SECRET"`
	CronPort        int    `env:"CRON_PORT"`                   // 3713
	CronRetention   int    `env:"CRON_RETENTION_DAYS"`         // 90
	AIMemLimit      string `env:"PLUGIN_AI_MEMORY_LIMIT"`      // 1g
	AICPULimit      string `env:"PLUGIN_AI_CPU_LIMIT"`         // 1.0
	MuxMemLimit     string `env:"PLUGIN_MUX_MEMORY_LIMIT"`     // 512m
	MuxCPULimit     string `env:"PLUGIN_MUX_CPU_LIMIT"`        // 0.5
	ClawMemLimit    string `env:"PLUGIN_CLAW_MEMORY_LIMIT"`    // 512m
	ClawCPULimit    string `env:"PLUGIN_CLAW_CPU_LIMIT"`       // 0.5
	DefaultMemLimit string `env:"PLUGIN_DEFAULT_MEMORY_LIMIT"` // 512m
	DefaultCPULimit string `env:"PLUGIN_DEFAULT_CPU_LIMIT"`    // 0.5
}

// PluginSystemConfig holds plugin system management configuration.
type PluginSystemConfig struct {
	Dir            string `env:"NSELF_PLUGIN_DIR"`         // ~/.nself/plugins
	Cache          string `env:"NSELF_PLUGIN_CACHE"`       // ~/.nself/cache/plugins
	Registry       string `env:"NSELF_PLUGIN_REGISTRY"`    // https://plugins.nself.org
	CacheTTL       int    `env:"NSELF_REGISTRY_CACHE_TTL"` // 300
	LicenseKey     string `env:"NSELF_PLUGIN_LICENSE_KEY"`
	SkipVerify     bool   `env:"NSELF_LICENSE_SKIP_VERIFY"`
	PingURL        string `env:"NSELF_PING_API_URL"` // https://ping.nself.org
	PricingURL     string `env:"NSELF_PRICING_URL"`  // https://nself.org/pricing
	InternalSecret string `env:"PLUGIN_INTERNAL_SECRET"`
}

// CustomService represents a user-defined custom service (CS_1..CS_10).
type CustomService struct {
	Index       int    // 1-10
	Name        string // parsed from CS_N
	Template    string // express-ts, fastapi, etc.
	Port        int
	Route       string // empty = internal only
	Public      bool
	Memory      string
	CPU         string
	TablePrefix string // CS_N_TABLE_PREFIX
	ExtraEnv    string // CS_N_ENV (raw key=val pairs, comma-separated)
}

// FrontendApp represents a frontend application (FRONTEND_APP_1..FRONTEND_APP_20).
type FrontendApp struct {
	Index       int
	DisplayName string
	SystemName  string
	Port        int
	Route       string
	Framework   string
	TablePrefix string
	Image       string // FRONTEND_APP_N_IMAGE (optional docker image reference)
}

// RemoteSchema represents a Hasura Remote Schema configuration.
type RemoteSchema struct {
	Index   int
	Name    string
	URL     string
	Headers string // key:val,key:val
}

// InternalRoute represents an internal Nginx route (INTERNAL_ROUTE_1..INTERNAL_ROUTE_20).
type InternalRoute struct {
	Index     int
	Name      string // INTERNAL_ROUTE_N_NAME
	Subdomain string // INTERNAL_ROUTE_N_SUBDOMAIN
	Target    string // INTERNAL_ROUTE_N_TARGET (e.g., hasura:8080)
	RateZone  string // INTERNAL_ROUTE_N_RATE_ZONE (default: general)
	WebSocket bool   // INTERNAL_ROUTE_N_WEBSOCKET
}

// ApiDocsConfig holds the api_docs section from nself.yaml.
// Controls generation of the OpenAPI 3.1 spec and Scalar interactive docs page.
type ApiDocsConfig struct {
	Enabled         bool     `env:"API_DOCS_ENABLED"`      // default: true
	Path            string   `env:"API_DOCS_PATH"`         // serve path, default: /docs
	Title           string   `env:"API_DOCS_TITLE"`        // defaults to "<ProjectName> API"
	Theme           string   `env:"API_DOCS_THEME"`        // default | moon | purple | solarized
	AuthEnvVar      string   `env:"API_DOCS_AUTH_ENV_VAR"` // env var with bearer token for try-out
	HideEndpoints   []string // paths to exclude from the spec
	GraphQLEnabled  bool     `env:"API_DOCS_GRAPHQL_ENABLED"`  // default: true
	GraphQLEndpoint string   `env:"API_DOCS_GRAPHQL_ENDPOINT"` // default: /v1/graphql
}

// IsProduction reports whether the project environment is production.
// Both "prod" and "production" are treated as production; the loader normalises
// "production" → "prod" via normalizeEnv, so only "prod" is checked here.
func (c *Config) IsProduction() bool {
	return c.Env == "prod" || c.Env == "production"
}
