package compose

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

// minimalConfigWithCS returns a Config set up for custom service testing.
func minimalConfigWithCS() *config.Config {
	cfg := minimalConfig()
	cfg.Postgres.Password = "pgpass"
	cfg.Hasura.Port = 8080
	cfg.Auth.Port = 4000
	cfg.Auth.ClientURL = "http://localhost:3000"
	return cfg
}

func testCS() config.CustomService {
	return config.CustomService{
		Index:    1,
		Name:     "ping-api",
		Template: "go",
		Port:     8001,
		Route:    "ping",
	}
}

// ── coreEnvVars ────────────────────────────────────────────────────────────────

// TestCoreEnvVars_ProjectFields verifies that PROJECT_NAME, BASE_DOMAIN, and ENV
// are injected from the Config.
func TestCoreEnvVars_ProjectFields(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	if env["PROJECT_NAME"] != cfg.ProjectName {
		t.Errorf("PROJECT_NAME = %q, want %q", env["PROJECT_NAME"], cfg.ProjectName)
	}
	if env["BASE_DOMAIN"] != cfg.BaseDomain {
		t.Errorf("BASE_DOMAIN = %q, want %q", env["BASE_DOMAIN"], cfg.BaseDomain)
	}
	if env["ENV"] != cfg.Env {
		t.Errorf("ENV = %q, want %q", env["ENV"], cfg.Env)
	}
}

// TestCoreEnvVars_PostgresVars verifies that POSTGRES_* env vars use the
// internal container address (host = "postgres", port = "5432").
func TestCoreEnvVars_PostgresVars(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	if env["POSTGRES_HOST"] != "postgres" {
		t.Errorf("POSTGRES_HOST = %q, want %q", env["POSTGRES_HOST"], "postgres")
	}
	if env["POSTGRES_PORT"] != "5432" {
		t.Errorf("POSTGRES_PORT = %q, want %q", env["POSTGRES_PORT"], "5432")
	}
	if env["POSTGRES_DB"] != cfg.Postgres.DB {
		t.Errorf("POSTGRES_DB = %q, want %q", env["POSTGRES_DB"], cfg.Postgres.DB)
	}
	if env["POSTGRES_USER"] != cfg.Postgres.User {
		t.Errorf("POSTGRES_USER = %q, want %q", env["POSTGRES_USER"], cfg.Postgres.User)
	}
	if env["POSTGRES_PASSWORD"] != cfg.Postgres.Password {
		t.Errorf("POSTGRES_PASSWORD = %q, want %q", env["POSTGRES_PASSWORD"], cfg.Postgres.Password)
	}
}

// TestCoreEnvVars_DatabaseURL verifies that DATABASE_URL is a valid postgresql DSN.
func TestCoreEnvVars_DatabaseURL(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	dbURL := env["DATABASE_URL"]
	if !strings.HasPrefix(dbURL, "postgresql://") {
		t.Errorf("DATABASE_URL should start with 'postgresql://', got %q", dbURL)
	}
	if !strings.Contains(dbURL, "@postgres:5432/") {
		t.Errorf("DATABASE_URL should contain '@postgres:5432/', got %q", dbURL)
	}
}

// TestCoreEnvVars_HasuraEndpoint verifies that HASURA_GRAPHQL_ENDPOINT uses
// the internal container address and Hasura's fixed container-internal port.
func TestCoreEnvVars_HasuraEndpoint(t *testing.T) {
	cfg := minimalConfigWithCS()
	cfg.Hasura.Port = 8080
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	want := "http://hasura:8080/v1/graphql"
	if env["HASURA_GRAPHQL_ENDPOINT"] != want {
		t.Errorf("HASURA_GRAPHQL_ENDPOINT = %q, want %q", env["HASURA_GRAPHQL_ENDPOINT"], want)
	}
}

// TestCoreEnvVars_HasuraEndpoint_IgnoresHostPortOverride is the gap #7
// regression test: HASURA_GRAPHQL_ENDPOINT must always target Hasura's fixed
// container-internal port (8080), never cfg.Hasura.Port, which is the
// HOST-mapped port and may be overridden (e.g. to 8181) to avoid a host port
// collision without changing how in-network services reach Hasura.
func TestCoreEnvVars_HasuraEndpoint_IgnoresHostPortOverride(t *testing.T) {
	cfg := minimalConfigWithCS()
	cfg.Hasura.Port = 8181 // host-mapped port override
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	want := "http://hasura:8080/v1/graphql"
	if env["HASURA_GRAPHQL_ENDPOINT"] != want {
		t.Errorf("HASURA_GRAPHQL_ENDPOINT = %q, want %q (must ignore HASURA_PORT host override)", env["HASURA_GRAPHQL_ENDPOINT"], want)
	}
}

// TestCoreEnvVars_AuthServerURL verifies that AUTH_SERVER_URL uses the internal
// container address and the configured Auth port.
func TestCoreEnvVars_AuthServerURL(t *testing.T) {
	cfg := minimalConfigWithCS()
	cfg.Auth.Port = 4000
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	want := "http://auth:4000"
	if env["AUTH_SERVER_URL"] != want {
		t.Errorf("AUTH_SERVER_URL = %q, want %q", env["AUTH_SERVER_URL"], want)
	}
}

// TestCoreEnvVars_ServiceFields verifies that SERVICE_NAME, SERVICE_PORT, and
// SERVICE_ROUTE are taken from the CustomService struct.
func TestCoreEnvVars_ServiceFields(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	if env["SERVICE_NAME"] != cs.Name {
		t.Errorf("SERVICE_NAME = %q, want %q", env["SERVICE_NAME"], cs.Name)
	}
	if env["SERVICE_PORT"] != fmt.Sprintf("%d", cs.Port) {
		t.Errorf("SERVICE_PORT = %q, want %q", env["SERVICE_PORT"], fmt.Sprintf("%d", cs.Port))
	}
	if env["SERVICE_ROUTE"] != cs.Route {
		t.Errorf("SERVICE_ROUTE = %q, want %q", env["SERVICE_ROUTE"], cs.Route)
	}
}

// TestCoreEnvVars_RedisAbsent verifies that REDIS_URL is not injected when
// Redis is disabled.
func TestCoreEnvVars_RedisAbsent(t *testing.T) {
	cfg := minimalConfigWithCS()
	cfg.Redis.Enabled = false
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	if _, ok := env["REDIS_URL"]; ok {
		t.Error("REDIS_URL should not be present when Redis is disabled")
	}
}

// TestCoreEnvVars_RedisPresent verifies that REDIS_URL is injected when
// Redis is enabled and uses the internal container address.
func TestCoreEnvVars_RedisPresent(t *testing.T) {
	cfg := minimalConfigWithCS()
	cfg.Redis.Enabled = true
	cfg.Redis.Password = "redispass"
	cfg.Redis.Port = 6379
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	redisURL, ok := env["REDIS_URL"]
	if !ok {
		t.Fatal("REDIS_URL should be present when Redis is enabled")
	}
	if !strings.Contains(redisURL, "redis:6379") {
		t.Errorf("REDIS_URL should reference redis:6379, got %q", redisURL)
	}
}

// TestCoreEnvVars_MinioAbsent verifies that S3_* vars are not injected when
// Minio is disabled.
func TestCoreEnvVars_MinioAbsent(t *testing.T) {
	cfg := minimalConfigWithCS()
	cfg.Minio.Enabled = false
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	for _, key := range []string{"S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		if _, ok := env[key]; ok {
			t.Errorf("%q should not be present when Minio is disabled", key)
		}
	}
}

// TestCoreEnvVars_MinioPresent verifies that S3_* vars are injected when Minio
// is enabled and reference the internal container address.
func TestCoreEnvVars_MinioPresent(t *testing.T) {
	cfg := minimalConfigWithCS()
	cfg.Minio.Enabled = true
	cfg.Minio.Port = 9000
	cfg.Minio.RootUser = "minioadmin"
	cfg.Minio.RootPassword = "minioadminpass"
	cfg.Minio.DefaultBuckets = "uploads"
	cs := testCS()

	env := coreEnvVars(cfg, cs)

	if !strings.Contains(env["S3_ENDPOINT"], "minio:9000") {
		t.Errorf("S3_ENDPOINT should reference minio:9000, got %q", env["S3_ENDPOINT"])
	}
	if env["S3_ACCESS_KEY"] != "minioadmin" {
		t.Errorf("S3_ACCESS_KEY = %q, want %q", env["S3_ACCESS_KEY"], "minioadmin")
	}
	if env["S3_SECRET_KEY"] != "minioadminpass" {
		t.Errorf("S3_SECRET_KEY = %q, want %q", env["S3_SECRET_KEY"], "minioadminpass")
	}
	if env["S3_BUCKET"] != "uploads" {
		t.Errorf("S3_BUCKET = %q, want %q", env["S3_BUCKET"], "uploads")
	}
}

// TestCoreEnvVars_ExtraEnvOverrides verifies that CS_N_ENV key=value pairs are
// parsed and applied last, overriding any default value.
func TestCoreEnvVars_ExtraEnvOverrides(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()
	cs.ExtraEnv = "CUSTOM_KEY=custom_value,ENV=override"

	env := coreEnvVars(cfg, cs)

	if env["CUSTOM_KEY"] != "custom_value" {
		t.Errorf("CUSTOM_KEY = %q, want %q", env["CUSTOM_KEY"], "custom_value")
	}
	// ExtraEnv must win over the built-in default.
	if env["ENV"] != "override" {
		t.Errorf("ExtraEnv override: ENV = %q, want %q", env["ENV"], "override")
	}
}

// TestCoreEnvVars_ExtraEnvMalformed verifies that a malformed ExtraEnv entry
// (missing '=') is silently skipped rather than causing a panic.
func TestCoreEnvVars_ExtraEnvMalformed(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()
	cs.ExtraEnv = "NOEQUALS,VALID_KEY=val"

	env := coreEnvVars(cfg, cs)

	// NOEQUALS has no '=' so it should be ignored.
	if _, ok := env["NOEQUALS"]; ok {
		t.Error("malformed ExtraEnv entry without '=' should be ignored")
	}
	// Valid entry must still be applied.
	if env["VALID_KEY"] != "val" {
		t.Errorf("VALID_KEY = %q, want %q", env["VALID_KEY"], "val")
	}
}

// ── buildCustomService ────────────────────────────────────────────────────────

// TestBuildCustomService_ContainerName verifies that the container name is
// "{project}_{service}" following the nself naming convention.
func TestBuildCustomService_ContainerName(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	g := NewGenerator(cfg)
	svc := g.buildCustomService(cs)

	want := cfg.ProjectName + "_" + cs.Name
	if svc.ContainerName != want {
		t.Errorf("ContainerName = %q, want %q", svc.ContainerName, want)
	}
}

// TestBuildCustomService_BuildContext verifies that the build context points to
// ./services/{name} with a Dockerfile.
func TestBuildCustomService_BuildContext(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	g := NewGenerator(cfg)
	svc := g.buildCustomService(cs)

	if svc.Build == nil {
		t.Fatal("buildCustomService: Build config is nil")
	}

	wantCtx := "./services/" + cs.Name
	if svc.Build.Context != wantCtx {
		t.Errorf("Build.Context = %q, want %q", svc.Build.Context, wantCtx)
	}
	if svc.Build.Dockerfile != "Dockerfile" {
		t.Errorf("Build.Dockerfile = %q, want %q", svc.Build.Dockerfile, "Dockerfile")
	}
}

// TestBuildCustomService_PortMapping verifies that the port is bound to
// 127.0.0.1 (not 0.0.0.0) per the service-binding hard rule.
func TestBuildCustomService_PortMapping(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	g := NewGenerator(cfg)
	svc := g.buildCustomService(cs)

	want := fmt.Sprintf("127.0.0.1:%d:%d", cs.Port, cs.Port)
	found := false
	for _, p := range svc.Ports {
		if p == want {
			found = true
		}
		if strings.Contains(p, "0.0.0.0") {
			t.Errorf("port binding must use 127.0.0.1, not 0.0.0.0; got %q", p)
		}
	}
	if !found {
		t.Errorf("expected port mapping %q, got: %v", want, svc.Ports)
	}
}

// TestBuildCustomService_HealthcheckPort verifies that the healthcheck URL
// references the correct port.
func TestBuildCustomService_HealthcheckPort(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	g := NewGenerator(cfg)
	svc := g.buildCustomService(cs)

	if svc.Healthcheck == nil {
		t.Fatal("buildCustomService: Healthcheck is nil")
	}

	hcStr := strings.Join(svc.Healthcheck.Test, " ")
	wantPort := fmt.Sprintf("%d", cs.Port)
	if !strings.Contains(hcStr, wantPort) {
		t.Errorf("healthcheck must reference port %s, got: %s", wantPort, hcStr)
	}
}

// TestBuildCustomService_DependsOnPostgres verifies that every custom service
// depends on postgres being healthy before starting.
func TestBuildCustomService_DependsOnPostgres(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	g := NewGenerator(cfg)
	svc := g.buildCustomService(cs)

	dep, ok := svc.DependsOn["postgres"]
	if !ok {
		t.Fatal("buildCustomService: DependsOn must include 'postgres'")
	}
	if dep.Condition != "service_healthy" {
		t.Errorf("postgres dependency condition = %q, want %q", dep.Condition, "service_healthy")
	}
}

// TestBuildCustomService_Restart verifies that the restart policy is
// "unless-stopped" (the nself standard for long-running services).
func TestBuildCustomService_Restart(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	g := NewGenerator(cfg)
	svc := g.buildCustomService(cs)

	if svc.Restart != "unless-stopped" {
		t.Errorf("Restart = %q, want %q", svc.Restart, "unless-stopped")
	}
}

// TestBuildCustomService_ResourceLimits verifies that Memory and CPU limits
// from the CustomService struct are carried through to the deploy config.
func TestBuildCustomService_ResourceLimits(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()
	cs.Memory = "512m"
	cs.CPU = "0.5"

	g := NewGenerator(cfg)
	svc := g.buildCustomService(cs)

	if svc.Deploy == nil || svc.Deploy.Resources == nil || svc.Deploy.Resources.Limits == nil {
		t.Fatal("buildCustomService: Deploy resource limits are nil")
	}

	if svc.Deploy.Resources.Limits.Memory != "512m" {
		t.Errorf("Deploy.Resources.Limits.Memory = %q, want %q", svc.Deploy.Resources.Limits.Memory, "512m")
	}
	if svc.Deploy.Resources.Limits.CPUs != "0.5" {
		t.Errorf("Deploy.Resources.Limits.CPUs = %q, want %q", svc.Deploy.Resources.Limits.CPUs, "0.5")
	}
}

// TestBuildCustomService_Network verifies that the service is attached to
// the project's Docker network.
func TestBuildCustomService_Network(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	g := NewGenerator(cfg)
	svc := g.buildCustomService(cs)

	found := false
	for _, n := range svc.Networks {
		if n == cfg.DockerNetwork {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("service must be on network %q, got: %v", cfg.DockerNetwork, svc.Networks)
	}
}

// TestBuildCustomService_CoreEnvInjected verifies that core env vars like
// PROJECT_NAME and POSTGRES_HOST are present in the generated service Environment.
func TestBuildCustomService_CoreEnvInjected(t *testing.T) {
	cfg := minimalConfigWithCS()
	cs := testCS()

	g := NewGenerator(cfg)
	svc := g.buildCustomService(cs)

	for _, key := range []string{
		"PROJECT_NAME",
		"BASE_DOMAIN",
		"ENV",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"DATABASE_URL",
		"HASURA_GRAPHQL_ENDPOINT",
		"AUTH_SERVER_URL",
		"SERVICE_NAME",
		"SERVICE_PORT",
	} {
		if val, ok := svc.Environment[key]; !ok {
			t.Errorf("buildCustomService: Environment missing key %q", key)
		} else if val == "" {
			t.Errorf("buildCustomService: Environment key %q is empty", key)
		}
	}
}
