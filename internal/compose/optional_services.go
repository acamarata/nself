package compose

import (
	"fmt"
	"log/slog"
)

// buildRedisService returns the Redis cache/queue service configuration.
// Command and health check are conditional on whether a password is set.
func (g *Generator) buildRedisService() ServiceConfig {
	rc := g.cfg.Redis

	version := rc.Version
	if version == "" {
		version = "7-alpine"
	}
	port := rc.Port
	if port == 0 {
		port = 6379
	}
	memory := rc.Memory
	if memory == "" {
		memory = "512M"
	}
	cpu := rc.CPU
	if cpu == "" {
		cpu = "0.5"
	}

	var command interface{}
	var healthTest []string

	if rc.Password != "" {
		command = fmt.Sprintf(
			"redis-server --appendonly yes --protected-mode yes --requirepass %s",
			rc.Password,
		)
		healthTest = []string{"CMD", "redis-cli", "-a", rc.Password, "ping"}
	} else {
		command = "redis-server --appendonly yes --protected-mode no"
		healthTest = []string{"CMD", "redis-cli", "ping"}
	}

	return ServiceConfig{
		Image:         ResolveImage("redis", fmt.Sprintf("redis:%s", version)),
		ContainerName: fmt.Sprintf("%s_redis", g.cfg.ProjectName),
		Restart:       "unless-stopped",
		User:          "999:999",
		Networks:      []string{g.cfg.DockerNetwork},
		Command:       command,
		Volumes:       []string{"redis_data:/data"},
		Ports:         []string{fmt.Sprintf("127.0.0.1:%d:6379", port)},
		Healthcheck: &Healthcheck{
			Test:     healthTest,
			Interval: "10s",
			Timeout:  "5s",
			Retries:  5,
		},
		Deploy: &DeployConfig{
			Resources: &Resources{
				Limits: &ResourceLimits{
					Memory: memory,
					CPUs:   cpu,
				},
			},
		},
	}
}

// buildMinioInitService returns the MinIO initialization service configuration.
// Uses busybox to chown the data directory before MinIO starts.
func (g *Generator) buildMinioInitService() ServiceConfig {
	return ServiceConfig{
		Image:         "busybox:1.36",
		ContainerName: fmt.Sprintf("%s_minio_init", g.cfg.ProjectName),
		Restart:       "no",
		User:          "root",
		Networks:      []string{g.cfg.DockerNetwork},
		Volumes:       []string{"minio_data:/data"},
		Command:       `sh -c "chown -R 1000:1000 /data; chmod -R 755 /data"`,
		Labels: map[string]string{
			"nself.type":        "init-container",
			"nself.service":     "minio",
			"nself.auto-remove": "true",
		},
		// No Profiles — init containers must be visible to depends_on
	}
}

// buildMinioService returns the MinIO object storage service configuration.
func (g *Generator) buildMinioService() ServiceConfig {
	mc := g.cfg.Minio

	version := mc.Version
	if version == "" {
		version = "latest"
	}
	port := mc.Port
	if port == 0 {
		port = 9000
	}
	consolePort := mc.ConsolePort
	if consolePort == 0 {
		consolePort = 9001
	}

	return ServiceConfig{
		Image:         ResolveImage("minio", fmt.Sprintf("minio/minio:%s", version)),
		ContainerName: fmt.Sprintf("%s_minio", g.cfg.ProjectName),
		Restart:       "unless-stopped",
		User:          "1000:1000",
		Networks:      []string{g.cfg.DockerNetwork},
		DependsOn: map[string]DepOn{
			"minio-init": {Condition: "service_completed_successfully"},
		},
		Environment: map[string]string{
			"MINIO_ROOT_USER":       mc.RootUser,
			"MINIO_ROOT_PASSWORD":   mc.RootPassword,
			"MINIO_DEFAULT_BUCKETS": mc.DefaultBuckets,
		},
		Command: `server /data --console-address ":9001"`,
		Volumes: []string{"minio_data:/data"},
		Ports: []string{
			fmt.Sprintf("127.0.0.1:%d:9000", port),
			fmt.Sprintf("127.0.0.1:%d:9001", consolePort),
		},
		Healthcheck: &Healthcheck{
			Test:     []string{"CMD", "curl", "-f", "http://localhost:9000/minio/health/live"},
			Interval: "30s",
			Timeout:  "10s",
			Retries:  3,
		},
	}
}

// buildMailpitService returns the Mailpit email testing service configuration.
func (g *Generator) buildMailpitService() ServiceConfig {
	mp := g.cfg.Mailpit

	version := mp.Version
	if version == "" {
		version = "latest"
	}
	smtpPort := mp.SMTPPort
	if smtpPort == 0 {
		smtpPort = 1025
	}
	uiPort := mp.UIPort
	if uiPort == 0 {
		uiPort = 8025
	}
	maxMessages := mp.MaxMessages
	if maxMessages == 0 {
		maxMessages = 500
	}

	env := map[string]string{
		"MP_UI_BIND_ADDR":             "0.0.0.0:8025",
		"MP_SMTP_BIND_ADDR":           "0.0.0.0:1025",
		"MP_SMTP_AUTH_ACCEPT_ANY":     "1",
		"MP_SMTP_AUTH_ALLOW_INSECURE": "1",
		"MP_MAX_MESSAGES":             fmt.Sprintf("%d", maxMessages),
	}
	// Add UI basic auth for non-dev environments when password is configured
	if g.cfg.Env != "dev" && g.cfg.Mailpit.UIPassword != "" {
		// Format: "user:password" — Mailpit supports MP_UI_BASICAUTH for HTTP basic auth
		env["MP_UI_BASICAUTH"] = fmt.Sprintf("%s:%s", g.cfg.Mailpit.UIUser, g.cfg.Mailpit.UIPassword)
	}

	return ServiceConfig{
		Image:         ResolveImage("mailpit", fmt.Sprintf("axllent/mailpit:%s", version)),
		ContainerName: fmt.Sprintf("%s_mailpit", g.cfg.ProjectName),
		Restart:       "unless-stopped",
		User:          "1000:1000",
		Networks:      []string{g.cfg.DockerNetwork},
		Environment:   env,
		Ports: []string{
			fmt.Sprintf("127.0.0.1:%d:1025", smtpPort),
			fmt.Sprintf("127.0.0.1:%d:8025", uiPort),
		},
		Healthcheck: &Healthcheck{
			Test:     []string{"CMD", "nc", "-z", "localhost", "8025"},
			Interval: "30s",
			Timeout:  "10s",
			Retries:  3,
		},
	}
}

// buildAdminService returns the nSelf Admin GUI service configuration.
func (g *Generator) buildAdminService() ServiceConfig {
	ac := g.cfg.Admin

	version := ac.Version
	if version == "" {
		version = "latest"
	}
	port := ac.Port
	if port == 0 {
		port = 3021
	}
	if ac.SecretKey == "" {
		slog.Warn("ADMIN_SECRET_KEY is not set — admin UI will start without authentication")
	}
	if ac.PasswordHash == "" {
		slog.Warn("ADMIN_PASSWORD_HASH is not set — admin UI will start without authentication")
	}

	return ServiceConfig{
		Image:         ResolveImage("admin", fmt.Sprintf("nself/nself-admin:%s", version)),
		ContainerName: fmt.Sprintf("%s_admin", g.cfg.ProjectName),
		Restart:       "unless-stopped",
		User:          "1000:1000",
		Networks:      []string{g.cfg.DockerNetwork},
		DependsOn: map[string]DepOn{
			"postgres": {Condition: "service_healthy"},
			"hasura":   {Condition: "service_healthy"},
		},
		Environment: map[string]string{
			"NODE_ENV":                    "production",
			"PROJECT_NAME":               g.cfg.ProjectName,
			"BASE_DOMAIN":                g.cfg.BaseDomain,
			"DATABASE_URL":               fmt.Sprintf("postgres://%s:%s@postgres:5432/%s", g.cfg.Postgres.User, g.cfg.Postgres.Password, g.cfg.Postgres.DB),
			"HASURA_GRAPHQL_ENDPOINT":    "http://hasura:8080/v1/graphql",
			"HASURA_GRAPHQL_ADMIN_SECRET": g.cfg.Hasura.AdminSecret,
			"DOCKER_HOST":                "unix:///var/run/docker.sock",
			"ADMIN_SECRET_KEY":           ac.SecretKey,
			"ADMIN_PASSWORD_HASH":        ac.PasswordHash,
		},
		Ports: []string{fmt.Sprintf("127.0.0.1:%d:3021", port)},
		Volumes: []string{
			"./:/workspace:rw",
			"nself_admin_data:/app/data",
			"/var/run/docker.sock:/var/run/docker.sock:ro",
		},
		Healthcheck: &Healthcheck{
			Test:        []string{"CMD", "curl", "-f", "http://localhost:3021/api/health"},
			Interval:    "30s",
			Timeout:     "10s",
			Retries:     3,
			StartPeriod: "30s",
		},
	}
}

// buildFunctionsService returns the serverless functions service configuration.
func (g *Generator) buildFunctionsService() ServiceConfig {
	fc := g.cfg.Functions

	version := fc.Version
	if version == "" {
		version = "latest"
	}
	port := fc.Port
	if port == 0 {
		port = 3008
	}

	return ServiceConfig{
		Image:         ResolveImage("functions", fmt.Sprintf("nhost/functions:%s", version)),
		ContainerName: fmt.Sprintf("%s_functions", g.cfg.ProjectName),
		Restart:       "unless-stopped",
		Networks:      []string{g.cfg.DockerNetwork},
		DependsOn: map[string]DepOn{
			"postgres": {Condition: "service_healthy"},
			"hasura":   {Condition: "service_healthy"},
		},
		Environment: map[string]string{
			"DATABASE_URL":               fmt.Sprintf("postgres://%s:%s@postgres:5432/%s", g.cfg.Postgres.User, g.cfg.Postgres.Password, g.cfg.Postgres.DB),
			"HASURA_GRAPHQL_ENDPOINT":    "http://hasura:8080/v1/graphql",
			"HASURA_GRAPHQL_ADMIN_SECRET": g.cfg.Hasura.AdminSecret,
			"PORT":                       fmt.Sprintf("%d", port),
		},
		Volumes: []string{"./functions:/opt/project"},
		Ports:   []string{fmt.Sprintf("127.0.0.1:%d:3008", port)},
		Healthcheck: &Healthcheck{
			Test:        []string{"CMD-SHELL", fmt.Sprintf("curl -f http://localhost:%d/healthz", port)},
			Interval:    "30s",
			Timeout:     "10s",
			Retries:     3,
			StartPeriod: "40s",
		},
	}
}

// buildMeiliInitService returns the MeiliSearch initialization service configuration.
// Uses the same init-container pattern as MinIO.
func (g *Generator) buildMeiliInitService() ServiceConfig {
	return ServiceConfig{
		Image:         "busybox:1.36",
		ContainerName: fmt.Sprintf("%s_meilisearch_init", g.cfg.ProjectName),
		Restart:       "no",
		User:          "root",
		Networks:      []string{g.cfg.DockerNetwork},
		Volumes:       []string{"meilisearch_data:/data"},
		Command:       `sh -c "chown -R 1000:1000 /data; chmod -R 755 /data"`,
		Labels: map[string]string{
			"nself.type":        "init-container",
			"nself.service":     "meilisearch",
			"nself.auto-remove": "true",
		},
		// No Profiles — init containers must be visible to depends_on
	}
}

// buildMeiliService returns the MeiliSearch service configuration.
func (g *Generator) buildMeiliService() ServiceConfig {
	sc := g.cfg.Search
	mc := sc.MeiliSearch

	version := mc.Version
	if version == "" {
		version = "latest"
	}
	port := sc.Port
	if port == 0 {
		port = 7700
	}
	env := mc.Env
	if env == "" {
		env = "development"
	}

	return ServiceConfig{
		Image:         ResolveImage("meilisearch", fmt.Sprintf("getmeili/meilisearch:%s", version)),
		ContainerName: fmt.Sprintf("%s_meilisearch", g.cfg.ProjectName),
		Restart:       "unless-stopped",
		User:          "1000:1000",
		Networks:      []string{g.cfg.DockerNetwork},
		DependsOn: map[string]DepOn{
			"meilisearch-init": {Condition: "service_completed_successfully"},
		},
		Environment: map[string]string{
			"MEILI_ENV":        env,
			"MEILI_MASTER_KEY": mc.MasterKey,
		},
		Command: "meilisearch --db-path=/meili_data",
		Volumes: []string{"meilisearch_data:/meili_data"},
		Ports:   []string{fmt.Sprintf("127.0.0.1:%d:7700", port)},
		Healthcheck: &Healthcheck{
			Test:     []string{"CMD", "curl", "-f", "http://localhost:7700/health"},
			Interval: "30s",
			Timeout:  "10s",
			Retries:  3,
		},
	}
}

// MLflow: moved to nself-mlflow free plugin (not in core)

// buildTypesenseService returns the Typesense search service configuration.
func (g *Generator) buildTypesenseService() ServiceConfig {
	sc := g.cfg.Search
	tc := sc.Typesense

	version := tc.Version
	if version == "" {
		version = "27.1"
	}
	port := sc.Port
	if port == 0 {
		port = 8108
	}
	enableCORS := "true"
	if !tc.EnableCORS {
		enableCORS = "false"
	}
	logLevel := tc.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	return ServiceConfig{
		Image:         ResolveImage("typesense", fmt.Sprintf("typesense/typesense:%s", version)),
		ContainerName: fmt.Sprintf("%s_typesense", g.cfg.ProjectName),
		Restart:       "unless-stopped",
		Networks:      []string{g.cfg.DockerNetwork},
		Environment: map[string]string{
			"TYPESENSE_API_KEY":     tc.APIKey,
			"TYPESENSE_ENABLE_CORS": enableCORS,
			"TYPESENSE_LOG_LEVEL":   logLevel,
		},
		Command: fmt.Sprintf("--data-dir /data --api-key=%s --enable-cors", tc.APIKey),
		Volumes: []string{"typesense_data:/data"},
		Ports:   []string{fmt.Sprintf("127.0.0.1:%d:8108", port)},
		Healthcheck: &Healthcheck{
			Test:     []string{"CMD", "curl", "-f", "-H", fmt.Sprintf("X-TYPESENSE-API-KEY: %s", tc.APIKey), "http://localhost:8108/health"},
			Interval: "30s",
			Timeout:  "10s",
			Retries:  3,
		},
	}
}
