package migrationai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GenerateRequest is the request sent to the AI migration endpoint.
type GenerateRequest struct {
	Prompt     string             `json:"prompt"`
	Deltas     []SchemaFieldDelta `json:"deltas,omitempty"`
	SchemaHint string             `json:"schema_hint,omitempty"`
}

// GenerateResponse is the response from the AI migration endpoint.
type GenerateResponse struct {
	SQL        string  `json:"sql"`
	WarningSQL string  `json:"warning_sql,omitempty"`
	Confidence float64 `json:"confidence"`
	AIModel    string  `json:"ai_model"`
}

// Config holds all parameters for the migration generator.
type Config struct {
	// AIEndpoint is the base URL of the nself-ai plugin service.
	// Defaults to http://localhost:3001 (standard plugin-ai port).
	AIEndpoint string

	// ConfidenceMin is the minimum AI confidence required to accept a proposal.
	// Proposals below this threshold are rejected. Defaults to 0.75.
	ConfidenceMin float64

	// MigrationsDir is the directory where migration files are written.
	// Defaults to "migrations" in the current working directory.
	MigrationsDir string

	// AutoApply controls whether to apply the migration automatically.
	// This MUST always be false in production. The field exists only for
	// programmatic use in tests. The CLI enforces this via the env guard.
	AutoApply bool

	// Timeout for AI calls. Defaults to 30s.
	Timeout time.Duration
}

// DefaultConfig returns a Config populated from environment variables.
// NSELF_MIGRATION_WATCH_ENABLED, NSELF_MIGRATION_AI_CONFIDENCE_MIN, and
// NSELF_MIGRATION_AUTO_APPLY are read but the CLI enforces AutoApply=false
// regardless of environment in production profiles.
func DefaultConfig() Config {
	endpoint := os.Getenv("NSELF_AI_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:3001"
	}

	confidenceMin := 0.75
	migrationsDir := "migrations"
	timeout := 30 * time.Second

	return Config{
		AIEndpoint:    endpoint,
		ConfidenceMin: confidenceMin,
		MigrationsDir: migrationsDir,
		AutoApply:     false, // hard default — never overridden from env in production
		Timeout:       timeout,
	}
}

// Generator produces AI-assisted SQL migration proposals from natural-language prompts.
type Generator struct {
	cfg    Config
	client *http.Client
}

// NewGenerator creates a Generator with the given config.
func NewGenerator(cfg Config) *Generator {
	return &Generator{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Generate calls the AI plugin to produce a migration proposal for the given prompt.
// deltas may be nil when the prompt is the sole input (nself migrate generate mode).
//
// Returns ErrLowConfidence when the AI model's confidence is below ConfidenceMin.
func (g *Generator) Generate(ctx context.Context, prompt string, deltas []SchemaFieldDelta) (*MigrationProposal, error) {
	req := GenerateRequest{
		Prompt: prompt,
		Deltas: deltas,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal generate request: %w", err)
	}

	url := strings.TrimRight(g.cfg.AIEndpoint, "/") + "/v1/migration/generate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai generate call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai generate: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("decode generate response: %w", err)
	}

	if genResp.Confidence < g.cfg.ConfidenceMin {
		return nil, &ErrLowConfidence{
			Got:      genResp.Confidence,
			Required: g.cfg.ConfidenceMin,
		}
	}

	slug := sanitizeSlug(prompt)
	filePath, err := WriteMigrationFile(g.cfg.MigrationsDir, slug, genResp.SQL, genResp.WarningSQL)
	if err != nil {
		return nil, fmt.Errorf("write migration proposal: %w", err)
	}

	return &MigrationProposal{
		Slug:       slug,
		SQL:        genResp.SQL,
		WarningSQL: genResp.WarningSQL,
		Confidence: genResp.Confidence,
		AIModel:    genResp.AIModel,
		FilePath:   filePath,
	}, nil
}

// ErrLowConfidence is returned when the AI confidence score is below the minimum threshold.
type ErrLowConfidence struct {
	Got      float64
	Required float64
}

func (e *ErrLowConfidence) Error() string {
	return fmt.Sprintf("AI confidence %.2f is below minimum %.2f; proposal rejected", e.Got, e.Required)
}
