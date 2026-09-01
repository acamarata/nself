package database

// Purpose: reads live Hasura metadata for the drift/reconcile/verify trio,
// with the Hasura admin secret sourced from the RUNNING container rather
// than cfg.Hasura.AdminSecret / .env on disk.
// Inputs: context, *config.Config.
// Outputs: parsed table metadata keyed by "schema.name", or an error.
// Constraints: split out of metadata_drift.go to respect the 300-line file
// cap (ASI Policy 3) — pure move, no behavior change.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/httptimeout"
)

// hasuraContainer returns the Hasura container name for the project. This
// MUST match the container_name nself build actually writes
// (internal/compose/core_services.go: "%s_hasura") — see the postgres
// equivalent's cautionary note in internal/plugin/schema.go.
func hasuraContainer(cfg *config.Config) string {
	return cfg.ProjectName + "_hasura"
}

// readHasuraAdminSecretFromContainer reads HASURA_GRAPHQL_ADMIN_SECRET from
// the RUNNING Hasura container's environment via `docker inspect`, never
// from cfg.Hasura.AdminSecret / .env on disk.
//
// Both the 2026-08-21 Unity report and the 2026-08-22 ntask report hit the
// same footgun independently: a stale .env admin secret silently downgrades
// the request to the anonymous role instead of failing auth, surfacing as a
// confusing "field not found in type: 'query_root'" error rather than an
// authentication error. Reading the secret the container actually has
// closes that hole for the drift/reconcile/verify commands built here.
func readHasuraAdminSecretFromContainer(ctx context.Context, cfg *config.Config) (string, error) {
	container := hasuraContainer(cfg)
	cmd := exec.CommandContext(ctx, "docker", "inspect", container,
		"--format", "{{range .Config.Env}}{{println .}}{{end}}")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker inspect %s (is the stack running? try 'nself start'): %w", container, err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(line, "HASURA_GRAPHQL_ADMIN_SECRET="); ok {
			if v == "" {
				return "", fmt.Errorf("HASURA_GRAPHQL_ADMIN_SECRET is empty in the running %s container", container)
			}
			return v, nil
		}
	}
	return "", fmt.Errorf("HASURA_GRAPHQL_ADMIN_SECRET not found in %s container environment", container)
}

// postMetadataAdmin sends a metadata API request authenticated with an
// explicit admin secret (rather than hasura.go's postMetadata, which reads
// cfg.Hasura.AdminSecret). Shares metadataRequest/hasuraMetadataURL from
// hasura.go — same package, no duplication of the URL/request shape.
func postMetadataAdmin(ctx context.Context, cfg *config.Config, secret, reqType string, args interface{}) ([]byte, error) {
	body, err := json.Marshal(metadataRequest{Type: reqType, Args: args})
	if err != nil {
		return nil, fmt.Errorf("marshal metadata request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hasuraMetadataURL(cfg), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create metadata request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hasura-Admin-Secret", secret)

	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hasura metadata request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody := new(bytes.Buffer)
	if _, err := respBody.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("read metadata response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hasura metadata API returned %d: %s", resp.StatusCode, respBody.String())
	}
	return respBody.Bytes(), nil
}

// fetchLiveTables exports live Hasura metadata (admin secret read from the
// running container) and returns its tracked tables keyed by "schema.name".
func fetchLiveTables(ctx context.Context, cfg *config.Config) (map[string]hasuraTable, error) {
	secret, err := readHasuraAdminSecretFromContainer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	body, err := postMetadataAdmin(ctx, cfg, secret, "export_metadata", struct{}{})
	if err != nil {
		return nil, fmt.Errorf("export live metadata: %w", err)
	}
	var doc hasuraMetadataDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse live metadata: %w", err)
	}
	var all []hasuraTable
	for _, src := range doc.Sources {
		all = append(all, src.Tables...)
	}
	return tablesByKey(all), nil
}
