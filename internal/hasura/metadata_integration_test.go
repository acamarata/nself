//go:build integration

// Package hasura integration test: boots real postgres + hasura containers
// via `docker compose`, applies a 2-table metadata fixture through
// ApplyIfPresent, and asserts both a clean apply and zero inconsistencies —
// the end-to-end path unit tests (metadata_test.go, database/hasura_apply_test.go)
// fake out entirely.
//
// Run with:
//
//	INTEGRATION=1 go test -tags integration -timeout 180s ./internal/hasura/...
package hasura

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/database"
)

const integrationComposeYAML = `
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: nself_dev_password
      POSTGRES_DB: nself
    ports: ["15532:5432"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 2s
      timeout: 3s
      retries: 20
  hasura:
    image: hasura/graphql-engine:v2.44.0
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      HASURA_GRAPHQL_DATABASE_URL: postgres://postgres:nself_dev_password@postgres:5432/nself
      HASURA_GRAPHQL_ADMIN_SECRET: integration_test_admin_secret_32c
      HASURA_GRAPHQL_ENABLE_CONSOLE: "false"
    ports: ["18080:8080"]
`

func TestApplyIfPresent_Integration_TwoTableFixture(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("set INTEGRATION=1 to run (requires Docker)")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not found in PATH")
	}

	dir := t.TempDir()
	composeFile := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(integrationComposeYAML), 0o644); err != nil {
		t.Fatalf("write compose fixture: %v", err)
	}

	up := exec.Command("docker", "compose", "-f", composeFile, "-p", "nself-hasura-integ", "up", "-d")
	if out, err := up.CombinedOutput(); err != nil {
		t.Fatalf("docker compose up: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		down := exec.Command("docker", "compose", "-f", composeFile, "-p", "nself-hasura-integ", "down", "-v")
		_, _ = down.CombinedOutput()
	})

	cfg := &config.Config{Env: "dev"}
	cfg.Hasura.Port = 18080
	cfg.Hasura.AdminSecret = "integration_test_admin_secret_32c"

	if err := waitHasuraHealthy(t.Context(), cfg, 60*time.Second); err != nil {
		t.Fatalf("hasura never became healthy: %v", err)
	}

	// Two-table fixture: metadata/tables.yaml !include-ing two bare-mapping
	// table files — the well-formed shape TestResolveIncludes_WellFormedTableTree
	// unit-tests in isolation, exercised here end-to-end against a live Hasura.
	metadataDir := filepath.Join(dir, "hasura", "metadata")
	writeIntegFixtureFile(t, metadataDir, "tables.yaml", "- !include public_np_integ_a.yaml\n- !include public_np_integ_b.yaml\n")
	writeIntegFixtureFile(t, metadataDir, "public_np_integ_a.yaml", "table:\n  schema: public\n  name: np_integ_a\n")
	writeIntegFixtureFile(t, metadataDir, "public_np_integ_b.yaml", "table:\n  schema: public\n  name: np_integ_b\n")

	if err := ApplyIfPresent(t.Context(), cfg, dir); err != nil {
		t.Fatalf("ApplyIfPresent: %v", err)
	}

	names, err := database.HasuraGetInconsistentMetadata(t.Context(), cfg)
	if err != nil {
		t.Fatalf("HasuraGetInconsistentMetadata: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected zero inconsistent objects after applying a fixture with no underlying tables tracked as relations, got %v", names)
	}
}

func waitHasuraHealthy(ctx context.Context, cfg *config.Config, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://localhost:%d/healthz", cfg.Hasura.Port)
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("healthz returned %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out waiting for hasura healthz: %w", lastErr)
}

func writeIntegFixtureFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s/%s: %v", dir, name, err)
	}
}
