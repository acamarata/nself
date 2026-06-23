package commands

// Purpose: Implements `nself claw ingest` — data-portability surface for feeding
// documents, URLs, or files into the nClaw knowledge base.
//
// Inputs:  one positional argument (file path or URL) + flags (--type, --topic,
//          --tag, --status)
// Outputs: JSON progress report or human-readable confirmation on stdout.
// Constraints: exit 0 = success; exit 1 = server/io error; exit 2 = bad args/flags.
// SPORT: F02-COMMAND-INVENTORY row: claw ingest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	clawIngestType   string
	clawIngestTopic  string
	clawIngestTag    string
	clawIngestStatus string
	clawIngestJSON   bool
)

var clawIngestCmd = &cobra.Command{
	Use:   "ingest <file-or-url>",
	Short: "Ingest a document or URL into nClaw knowledge base",
	Long: `Submit a file or URL for ingestion into the nClaw knowledge base.

Accepted sources:
  File path   A local file (text, markdown, PDF)
  URL         An http/https URL to fetch and ingest

Flags control metadata applied to the ingested item.

Exit codes:
  0  success
  1  server or I/O error
  2  invalid arguments or flags

Examples:
  nself claw ingest notes.md
  nself claw ingest https://example.com/article
  nself claw ingest notes.md --topic "work" --tag "meeting"
  nself claw ingest docs.pdf --type document --json`,
	Args: cobra.ExactArgs(1),
	RunE: runClawIngest,
}

func init() {
	clawIngestCmd.Flags().StringVar(&clawIngestType, "type", "", "Source type: document, url, text (auto-detected if omitted)")
	clawIngestCmd.Flags().StringVar(&clawIngestTopic, "topic", "", "Associate with an existing topic name")
	clawIngestCmd.Flags().StringVar(&clawIngestTag, "tag", "", "Tag to apply to the ingested item")
	clawIngestCmd.Flags().StringVar(&clawIngestStatus, "status", "", "Check status of a prior ingest job by ID")
	clawIngestCmd.Flags().BoolVar(&clawIngestJSON, "json", false, "Output result as JSON")
}

// clawIngestSourceType resolves the source type from a flag or heuristic.
func clawIngestSourceType(source, flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	lower := strings.ToLower(source)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return "url"
	}
	switch {
	case strings.HasSuffix(lower, ".pdf"):
		return "document"
	case strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".txt"):
		return "text"
	}
	return "document"
}

func runClawIngest(cmd *cobra.Command, args []string) error {
	// --status mode: poll an existing job by ID
	if clawIngestStatus != "" {
		return runClawIngestStatus(cmd, clawIngestStatus)
	}

	source := args[0]
	sourceType := clawIngestSourceType(source, clawIngestType)

	// Validate type flag if provided
	validTypes := map[string]bool{"document": true, "url": true, "text": true}
	if clawIngestType != "" && !validTypes[clawIngestType] {
		return fmt.Errorf("unknown --type %q (use: document, url, text)", clawIngestType)
	}

	client, baseURL, err := clawClient()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	// Build the ingest payload
	payload := map[string]interface{}{
		"source":      source,
		"source_type": sourceType,
	}
	if clawIngestTopic != "" {
		payload["topic"] = clawIngestTopic
	}
	if clawIngestTag != "" {
		payload["tag"] = clawIngestTag
	}

	// For file sources, read the file content
	if sourceType != "url" {
		data, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("reading file %q: %w", source, err)
		}
		payload["content"] = string(data)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(cmd.Context(), "POST", baseURL+"/claw/v1/ingest", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ingest request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if clawIngestJSON {
		fmt.Println(string(respBody))
		return nil
	}

	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.ID != "" {
		fmt.Printf("Ingestion submitted. Job ID: %s\n", result.ID)
		if result.Status != "" {
			fmt.Printf("Status: %s\n", result.Status)
		}
		fmt.Printf("Check progress: nself claw ingest --status %s\n", result.ID)
	} else {
		fmt.Printf("Ingestion submitted.\n%s\n", string(respBody))
	}

	return nil
}

// runClawIngestStatus polls the ingest job status by ID.
func runClawIngestStatus(cmd *cobra.Command, jobID string) error {
	if jobID == "" {
		return fmt.Errorf("--status requires a job ID")
	}

	client, baseURL, err := clawClient()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	req, err := http.NewRequestWithContext(cmd.Context(), "GET", baseURL+"/claw/v1/ingest/"+jobID+"/status", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("status request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("job %q not found", jobID)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	if clawIngestJSON {
		fmt.Println(string(body))
		return nil
	}

	var result struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Progress int    `json:"progress"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println(string(body))
		return nil
	}

	fmt.Printf("Job:      %s\n", result.ID)
	fmt.Printf("Status:   %s\n", result.Status)
	if result.Progress > 0 {
		fmt.Printf("Progress: %d%%\n", result.Progress)
	}
	if result.Error != "" {
		fmt.Printf("Error:    %s\n", result.Error)
	}

	return nil
}
