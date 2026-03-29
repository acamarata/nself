package compose

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// remoteSchemaEnvVars returns env var map for Hasura remote schema bootstrapping.
func remoteSchemaEnvVars(schemas []config.RemoteSchema) map[string]string {
	env := make(map[string]string, len(schemas)*3)
	for i, rs := range schemas {
		n := i + 1
		key := fmt.Sprintf("HASURA_GRAPHQL_REMOTE_SCHEMA_%d", n)
		env[key+"_NAME"] = rs.Name
		env[key+"_URL"] = rs.URL
		if rs.Headers != "" {
			env[key+"_HEADERS"] = rs.Headers
		}
	}
	return env
}

// buildPostgresService returns the PostgreSQL service configuration.
func (g *Generator) buildPostgresService() ServiceConfig {
	cfg := g.cfg
	return ServiceConfig{
		Image:         ResolveImage("postgres", fmt.Sprintf("postgres:%s", cfg.Postgres.Version)),
		ContainerName: fmt.Sprintf("%s_postgres", cfg.ProjectName),
		Restart:       "unless-stopped",
		Networks:      []string{cfg.DockerNetwork},
		User: "999:999",
		Environment: map[string]string{
			"POSTGRES_USER":             cfg.Postgres.User,
			"POSTGRES_PASSWORD":         cfg.Postgres.Password,
			"POSTGRES_DB":               cfg.Postgres.DB,
			"POSTGRES_HOST_AUTH_METHOD": "scram-sha-256",
			"PGDATA":                    "/var/lib/postgresql/data/pgdata",
		},
		Volumes: []string{
			"postgres_data:/var/lib/postgresql/data",
			"./postgres/init:/docker-entrypoint-initdb.d:ro",
		},
		Ports: []string{
			fmt.Sprintf("127.0.0.1:%d:5432", cfg.Postgres.Port),
		},
		ShmSize: "256mb",
		Healthcheck: &Healthcheck{
			Test:     []string{"CMD-SHELL", fmt.Sprintf("pg_isready -U %s", cfg.Postgres.User)},
			Interval: "10s",
			Timeout:  "5s",
			Retries:  5,
		},
		Deploy: &DeployConfig{
			Resources: &Resources{
				Limits: &ResourceLimits{
					Memory: cfg.Postgres.MemLimit,
					CPUs:   cfg.Postgres.CPULimit,
				},
			},
		},
	}
}

// buildHasuraService returns the Hasura GraphQL engine service configuration.
func (g *Generator) buildHasuraService() ServiceConfig {
	cfg := g.cfg
	consoleStr := "false"
	if cfg.Hasura.Console {
		consoleStr = "true"
	}
	devModeStr := "false"
	if cfg.Hasura.DevMode {
		devModeStr = "true"
	}

	env := map[string]string{
		"HASURA_GRAPHQL_DATABASE_URL":      cfg.DatabaseURL(),
		"HASURA_GRAPHQL_ADMIN_SECRET":      cfg.Hasura.AdminSecret,
		"HASURA_GRAPHQL_ENABLE_CONSOLE":    consoleStr,
		"HASURA_GRAPHQL_DEV_MODE":          devModeStr,
		"HASURA_GRAPHQL_ENABLE_TELEMETRY":  "false",
		"HASURA_GRAPHQL_CORS_DOMAIN":       cfg.Hasura.CORSDomain,
		"HASURA_GRAPHQL_LOG_LEVEL":         cfg.Hasura.LogLevel,
		"HASURA_GRAPHQL_JWT_SECRET":        config.BuildJWTSecret(cfg),
		"HASURA_GRAPHQL_UNAUTHORIZED_ROLE": "public",
	}

	// Merge remote schema env vars
	for k, v := range remoteSchemaEnvVars(cfg.RemoteSchemas) {
		env[k] = v
	}

	// Passthrough: forward REMOTE_SCHEMA_* and HASURA_EXTRA_* from .env to the container.
	// Hasura's url_from_env feature reads Remote Schema URLs directly from the container
	// environment, so these vars must be present at runtime.
	if cfg.Passthrough != nil {
		keys := make([]string, 0, len(cfg.Passthrough))
		for k := range cfg.Passthrough {
			if strings.HasPrefix(k, "REMOTE_SCHEMA_") || strings.HasPrefix(k, "HASURA_EXTRA_") {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			env[k] = cfg.Passthrough[k]
		}
	}

	return ServiceConfig{
		Image:         ResolveImage("hasura", fmt.Sprintf("hasura/graphql-engine:%s", cfg.Hasura.Version)),
		ContainerName: fmt.Sprintf("%s_hasura", cfg.ProjectName),
		Restart:       "unless-stopped",
		User:          "1001:1001",
		Networks:      []string{cfg.DockerNetwork},
		DependsOn: map[string]DepOn{
			"postgres": {Condition: "service_healthy"},
		},
		Environment: env,
		Ports: []string{
			fmt.Sprintf("127.0.0.1:%d:8080", cfg.Hasura.Port),
		},
		Healthcheck: &Healthcheck{
			Test:        []string{"CMD", "curl", "-f", "http://localhost:8080/healthz"},
			Interval:    "10s",
			Timeout:     "5s",
			Retries:     5,
			StartPeriod: "15s",
		},
		Deploy: &DeployConfig{
			Resources: &Resources{
				Limits: &ResourceLimits{
					Memory: cfg.Hasura.MemLimit,
					CPUs:   cfg.Hasura.CPULimit,
				},
			},
		},
	}
}

// buildAuthService returns the authentication service configuration.
// Auth depends on BOTH postgres (database) and hasura (metadata validation at startup).
func (g *Generator) buildAuthService() ServiceConfig {
	cfg := g.cfg
	smtpSecureStr := "false"
	if cfg.Auth.SMTPSecure {
		smtpSecureStr = "true"
	}
	webAuthnStr := "false"
	if cfg.Auth.WebAuthnEnabled {
		webAuthnStr = "true"
	}

	env := map[string]string{
		"AUTH_HOST":                                  "0.0.0.0",
		"AUTH_PORT":                                  fmt.Sprintf("%d", cfg.Auth.Port),
		"AUTH_LOG_LEVEL":                             cfg.Auth.LogLevel,
		"AUTH_WEBAUTHN_ENABLED":                      webAuthnStr,
		"DATABASE_URL":                               cfg.DatabaseURL(),
		"AUTH_DATABASE_URL":                          cfg.DatabaseURL(),
		"HASURA_GRAPHQL_DATABASE_URL":                cfg.DatabaseURL(),
		"POSTGRES_HOST":                              cfg.Postgres.Host,
		"POSTGRES_PORT":                              "5432",
		"AUTH_SERVER_URL":                            fmt.Sprintf("http://localhost:%d", cfg.Auth.Port),
		"AUTH_CLIENT_URL":                            cfg.Auth.ClientURL,
		"AUTH_JWT_SECRET":                            cfg.Hasura.JWTKey,
		"AUTH_JWT_TYPE":                              cfg.Hasura.JWTType,
		"HASURA_GRAPHQL_JWT_SECRET":                  config.BuildJWTSecret(cfg),
		"HASURA_GRAPHQL_GRAPHQL_URL":                 "http://hasura:8080/v1/graphql",
		"HASURA_GRAPHQL_ADMIN_SECRET":                cfg.Hasura.AdminSecret,
		"AUTH_ACCESS_TOKEN_EXPIRES_IN":               fmt.Sprintf("%d", cfg.Auth.AccessTokenExpiry),
		"AUTH_REFRESH_TOKEN_EXPIRES_IN":              fmt.Sprintf("%d", cfg.Auth.RefreshTokenExpiry),
		"AUTH_SMTP_HOST":                             cfg.Auth.SMTPHost,
		"AUTH_SMTP_PORT":                             fmt.Sprintf("%d", cfg.Auth.SMTPPort),
		"AUTH_SMTP_USER":                             cfg.Auth.SMTPUser,
		"AUTH_SMTP_PASS":                             cfg.Auth.SMTPPass,
		"AUTH_SMTP_SECURE":                           smtpSecureStr,
		"AUTH_SMTP_SENDER":                           cfg.Auth.SMTPSender,
		"AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED":  "false",
	}

	// T06: Add AUTH_DB_* alias vars required by Nhost Auth.
	for k, v := range authPGAliasVars(cfg.Postgres) {
		env[k] = v
	}

	// T07: Inject AUTH_ALLOWED_REDIRECT_URLS for OAuth and magic link support.
	env["AUTH_ALLOWED_REDIRECT_URLS"] = computeAllowedRedirectURLs(cfg)

	// Passthrough: iterate for AUTH_PROVIDER_* keys
	if cfg.Passthrough != nil {
		keys := make([]string, 0, len(cfg.Passthrough))
		for k := range cfg.Passthrough {
			if strings.HasPrefix(k, "AUTH_PROVIDER_") {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			env[k] = cfg.Passthrough[k]
		}
	}

	return ServiceConfig{
		Image:         ResolveImage("auth", fmt.Sprintf("nhost/hasura-auth:%s", cfg.Auth.Version)),
		ContainerName: fmt.Sprintf("%s_auth", cfg.ProjectName),
		Restart:       "unless-stopped",
		Networks:      []string{cfg.DockerNetwork},
		DependsOn: map[string]DepOn{
			"postgres": {Condition: "service_healthy"},
			"hasura":   {Condition: "service_healthy"},
		},
		Environment: env,
		Ports: []string{
			fmt.Sprintf("127.0.0.1:%d:4002", cfg.Auth.Port),
		},
		Healthcheck: &Healthcheck{
			Test:        []string{"CMD", "wget", "-qO-", "http://localhost:4002/healthz"},
			Interval:    "30s",
			Timeout:     "10s",
			Retries:     3,
			StartPeriod: "30s",
		},
		Deploy: &DeployConfig{
			Resources: &Resources{
				Limits: &ResourceLimits{
					Memory: cfg.Auth.MemLimit,
					CPUs:   cfg.Auth.CPULimit,
				},
			},
		},
	}
}

// buildNginxService returns the Nginx reverse proxy service configuration.
// It receives the full DockerCompose to compute depends_on from existing services.
func (g *Generator) buildNginxService(dc *DockerCompose) ServiceConfig {
	cfg := g.cfg

	// Build depends_on: always hasura + auth, plus any custom services
	dependsOn := map[string]DepOn{
		"hasura": {Condition: "service_healthy"},
		"auth":   {Condition: "service_healthy"},
	}
	// Add custom services that exist in dc.Services
	for _, cs := range cfg.CustomServices {
		if _, exists := dc.Services[cs.Name]; exists {
			dependsOn[cs.Name] = DepOn{Condition: "service_started"}
		}
	}

	return ServiceConfig{
		Image:         ResolveImage("nginx", "nginx:alpine"),
		ContainerName: fmt.Sprintf("%s_nginx", cfg.ProjectName),
		Restart:       "unless-stopped",
		User:          "101:101",
		Networks:      []string{cfg.DockerNetwork},
		DependsOn:     dependsOn,
		Environment: map[string]string{
			"BASE_DOMAIN":  cfg.BaseDomain,
			"PROJECT_NAME": cfg.ProjectName,
			"ENV":          cfg.Env,
		},
		Ports: []string{
			fmt.Sprintf("%s:%d:80", cfg.Nginx.BindIP, cfg.Nginx.HTTPPort),
			fmt.Sprintf("%s:%d:443", cfg.Nginx.BindIP, cfg.Nginx.SSLPort),
		},
		Volumes: []string{
			"./nginx/nginx.conf:/etc/nginx/nginx.conf:ro",
			fmt.Sprintf("./%s:/etc/nginx/conf.d:ro", NginxConfDDir),
			fmt.Sprintf("./nginx/conf.d-%s:/etc/nginx/conf.d-%s:ro", cfg.Env, cfg.Env),
			fmt.Sprintf("./%s:/etc/nginx/sites:ro", NginxSitesDir),
			"./nginx/includes:/etc/nginx/includes:ro",
			"./ssl/certificates:/etc/nginx/ssl:ro",
		},
		Tmpfs: []string{
			"/var/cache/nginx:uid=101,gid=101",
			"/var/run:uid=101,gid=101",
		},
		Healthcheck: &Healthcheck{
			Test:        []string{"CMD-SHELL", "wget --no-check-certificate --no-verbose --tries=1 -O /dev/null https://127.0.0.1/health 2>/dev/null || exit 1"},
			Interval:    "30s",
			Timeout:     "5s",
			Retries:     3,
			StartPeriod: "10s",
		},
		Deploy: &DeployConfig{
			Resources: &Resources{
				Limits: &ResourceLimits{
					Memory: "256m",
					CPUs:   "0.5",
				},
			},
		},
	}
}
