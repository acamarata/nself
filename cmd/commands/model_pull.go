package commands

// Purpose: `nself model pull` and `nself model update` split out of
// model.go (CLI-R12 Batch B mechanical file-size split). Update is a thin
// wrapper that re-invokes the same pull logic.
// Inputs: the positional model name; NSELF_OLLAMA_TIMEOUT_SECONDS overrides
// the default 30-minute download timeout.
// Outputs: stdout progress messages; errors wrap Ollama API failures.
// Constraints: pure move, no behavior change. modelCmd (parent) and the
// modelOllamaURL helper remain in model.go/model_http.go.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var modelPullCmd = &cobra.Command{
	Use:   "pull <model>",
	Short: "Pull (download) a model from the Ollama registry",
	Long: `Download a model from the Ollama library.

Model names follow the Ollama tag format:
  <name>          Latest tag (e.g. llama3.2)
  <name>:<tag>    Specific tag (e.g. llama3.2:3b)

Common models:
  gemma-3-4b        ~2.5 GB  — good for CPU-only inference
  llama3.2:3b       ~2.0 GB  — fast general chat
  llama3.2:7b       ~4.7 GB  — higher quality, needs 8 GB RAM
  mistral           ~4.1 GB  — good instruct model`,
	Args: cobra.ExactArgs(1),
	RunE: runModelPull,
}

func runModelPull(_ *cobra.Command, args []string) error {
	model := args[0]
	fmt.Printf("Pulling %s...\n", model)

	payload, _ := json.Marshal(map[string]any{"name": model, "stream": false})
	req, err := http.NewRequest("POST", modelOllamaURL()+"/api/pull", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Default to 30 minutes for large model downloads; honour NSELF_OLLAMA_TIMEOUT_SECONDS
	// when set so that tests can fail fast against unreachable endpoints.
	pullTimeout := 30 * time.Minute
	if t := os.Getenv("NSELF_OLLAMA_TIMEOUT_SECONDS"); t != "" {
		var secs int
		if _, scanErr := fmt.Sscanf(t, "%d", &secs); scanErr == nil && secs > 0 {
			pullTimeout = time.Duration(secs) * time.Second
		}
	}
	client := &http.Client{Timeout: pullTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pull %s: %w", model, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("pull %s: HTTP %d: %s", model, resp.StatusCode, body)
	}

	fmt.Printf("Model %s ready.\n", model)
	return nil
}

var modelUpdateCmd = &cobra.Command{
	Use:   "update <model>",
	Short: "Re-pull a model to fetch the latest version of its tag",
	Long: `Re-download a model to pick up the newest weights for its tag.

This is equivalent to running:
  nself model pull <model>

but makes the intent explicit.  The local copy is replaced in-place; Ollama
only downloads layers that have changed.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Updating %s...\n", args[0])
		return runModelPull(cmd, args)
	},
}
