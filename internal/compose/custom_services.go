package compose

import (
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// coreEnvVars returns the standard env vars injected into every custom service.
// User overrides via CS_N_ENV are applied last and win over defaults.
func coreEnvVars(cfg *config.Config, svc config.CustomService) map[string]string {
	env := map[string]string{
		"PROJECT_NAME":                cfg.ProjectName,
		"BASE_DOMAIN":                 cfg.BaseDomain,
		"ENV":                         cfg.Env,
		"POSTGRES_HOST":               "postgres",
		"POSTGRES_PORT":               "5432",
		"POSTGRES_DB":                 cfg.Postgres.DB,
		"POSTGRES_USER":               cfg.Postgres.User,
		"POSTGRES_PASSWORD":           cfg.Postgres.Password,
		"DATABASE_URL":                cfg.DatabaseURL(),
		"HASURA_GRAPHQL_ENDPOINT":     fmt.Sprintf("http://hasura:%d/v1/graphql", cfg.Hasura.Port),
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

// buildCustomService returns the service configuration for a user-defined
// custom service (CS_1..CS_10). Each custom service is built from a Dockerfile
// in ./services/{name}/ and exposes a single port with a /health endpoint.
func (g *Generator) buildCustomService(cs config.CustomService) ServiceConfig {
	cfg := g.cfg

	return ServiceConfig{
		Build: &BuildConfig{
			Context:    fmt.Sprintf("./services/%s", cs.Name),
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
		Healthcheck: &Healthcheck{
			Test:        []string{"CMD", "wget", "-qO-", fmt.Sprintf("http://localhost:%d/health", cs.Port)},
			Interval:    "30s",
			Timeout:     "10s",
			Retries:     3,
			StartPeriod: "60s",
		},
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
