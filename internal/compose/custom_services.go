package compose

import (
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// coreEnvVars returns the standard env vars injected into every custom service.
//
// By design, custom services receive only this fixed generic set (project
// identity, Postgres, Hasura, Auth, and the optional Redis/Minio blocks) plus
// whatever the service opts into via CS_N_ENV_PASSTHROUGH or CS_N_ENV — never
// the full project .env. This keeps each service's declared surface area
// explicit and auditable instead of silently coupling it to every var another
// part of the stack happens to define. See
// .github/wiki/Config-Custom-Services.md for the user-facing version of this
// decision.
//
// Precedence (lowest to highest): fixed defaults → CS_N_ENV_PASSTHROUGH
// (named allowlist forwarded from the project's resolved env) → CS_N_ENV
// (explicit overrides, always win).
func coreEnvVars(cfg *config.Config, svc config.CustomService) map[string]string {
	env := map[string]string{
		"PROJECT_NAME":      cfg.ProjectName,
		"BASE_DOMAIN":       cfg.BaseDomain,
		"ENV":               cfg.Env,
		"POSTGRES_HOST":     "postgres",
		"POSTGRES_PORT":     "5432",
		"POSTGRES_DB":       cfg.Postgres.DB,
		"POSTGRES_USER":     cfg.Postgres.User,
		"POSTGRES_PASSWORD": cfg.Postgres.Password,
		"DATABASE_URL":      cfg.DatabaseURL(),
		// Gap #7: Hasura always listens on 8080 INSIDE its container regardless
		// of the host-mapped cfg.Hasura.Port (compose maps
		// "127.0.0.1:<Hasura.Port>:8080" — see buildHasuraService). Custom
		// services reach Hasura over the same Docker network, so this must be
		// the fixed container port, not the host-exposed one.
		"HASURA_GRAPHQL_ENDPOINT":     "http://hasura:8080/v1/graphql",
		"HASURA_GRAPHQL_ADMIN_SECRET": cfg.Hasura.AdminSecret,
		"AUTH_SERVER_URL":             fmt.Sprintf("http://auth:%d", cfg.Auth.Port),
		"AUTH_CLIENT_URL":             cfg.Auth.ClientURL,
		"SERVICE_NAME":                svc.Name,
		"SERVICE_PORT":                fmt.Sprintf("%d", svc.Port),
		"SERVICE_ROUTE":               svc.Route,
		"TABLE_PREFIX":                svc.TablePrefix,
	}
	if cfg.Redis.Enabled {
		env["REDIS_URL"] = fmt.Sprintf("redis://:%s@redis:%d", cfg.Redis.Password, cfg.Redis.Port)
	}
	if cfg.Minio.Enabled {
		env["S3_ENDPOINT"] = fmt.Sprintf("http://minio:%d", cfg.Minio.Port)
		env["S3_ACCESS_KEY"] = cfg.Minio.RootUser
		env["S3_SECRET_KEY"] = cfg.Minio.RootPassword
		env["S3_BUCKET"] = cfg.Minio.DefaultBuckets
	}
	// CS_N_ENV_PASSTHROUGH — explicit allowlist of extra project env vars to
	// forward into this container beyond the fixed core set above. Applied
	// before CS_N_ENV so an explicit override still wins on conflict. Names
	// not present in the resolved env are silently skipped (not an error) so
	// an allowlist can be shared across environments where a var may be
	// optional.
	if svc.EnvPassthrough != "" {
		for _, name := range strings.Split(svc.EnvPassthrough, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if val, ok := os.LookupEnv(name); ok {
				env[name] = val
			}
		}
	}
	// CS_N_ENV overrides applied last — user wins
	if svc.ExtraEnv != "" {
		for _, pair := range strings.Split(svc.ExtraEnv, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				env[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return env
}

// buildHealthcheck renders the Docker healthcheck for a custom service,
// honoring CS_N_HEALTHCHECK. The raw value may be:
//
//   - empty                                — default: GET /health on the
//     service's own port via wget (the prior hardcoded behavior).
//   - a path (e.g. "/auth/health")         — GET that path instead of /health,
//     still via wget on the service's own port.
//   - a full command starting with "CMD"
//     or "CMD-SHELL" (e.g. "CMD-SHELL curl
//     -f http://localhost:4002/status")    — passed through verbatim as the
//     compose healthcheck test, split on whitespace. Use this when the
//     service needs curl, a non-HTTP probe, or a port other than its own.
//   - "disabled" / "none" / "false"
//     (case-insensitive)                   — no healthcheck is emitted.
//
// Without this, a service whose health endpoint isn't literally /health on
// its own port (e.g. auth_server serving /auth/health) is reported unhealthy
// forever regardless of its actual state.
func buildHealthcheck(cs config.CustomService) *Healthcheck {
	raw := strings.TrimSpace(cs.HealthCheck)
	switch strings.ToLower(raw) {
	case "disabled", "none", "false":
		return nil
	}

	var test []string
	if strings.HasPrefix(raw, "CMD") {
		test = strings.Fields(raw)
	} else {
		path := raw
		if path == "" {
			path = "/health"
		} else if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		test = []string{"CMD", "wget", "-qO-", fmt.Sprintf("http://localhost:%d%s", cs.Port, path)}
	}

	return &Healthcheck{
		Test:        test,
		Interval:    "30s",
		Timeout:     "10s",
		Retries:     3,
		StartPeriod: "60s",
	}
}

// buildCustomService returns the service configuration for a user-defined
// custom service (CS_1..CS_10). Each custom service is built from a Dockerfile
// in ./services/{name}/ by default, or from CS_N_PATH when set.
func (g *Generator) buildCustomService(cs config.CustomService) ServiceConfig {
	cfg := g.cfg

	buildContext := cs.BuildPath
	if buildContext == "" {
		buildContext = fmt.Sprintf("./services/%s", cs.Name)
	}

	return ServiceConfig{
		Build: &BuildConfig{
			Context:    buildContext,
			Dockerfile: "Dockerfile",
		},
		ContainerName: fmt.Sprintf("%s_%s", cfg.ProjectName, cs.Name),
		Restart:       "unless-stopped",
		Networks:      []string{cfg.DockerNetwork},
		DependsOn: map[string]DepOn{
			"postgres": {Condition: "service_healthy"},
		},
		Environment: coreEnvVars(cfg, cs),
		Ports: []string{
			fmt.Sprintf("127.0.0.1:%d:%d", cs.Port, cs.Port),
		},
		Healthcheck: buildHealthcheck(cs),
		Deploy: &DeployConfig{
			Resources: &Resources{
				Limits: &ResourceLimits{
					Memory: cs.Memory,
					CPUs:   cs.CPU,
				},
			},
		},
	}
}
