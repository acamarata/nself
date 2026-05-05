package commands

// nself model — local model management.
//
// Provides a focused surface for day-to-day local model operations:
//   list      Show all pulled Ollama models with size + modified date
//   pull      Pull (download) a model from the Ollama registry
//   remove    Delete a pulled model to free disk space
//   update    Re-pull a model to pick up the latest tag
//   benchmark Run a standard prompt and report tokens/s + p99 latency
//
// All commands delegate to the Ollama API (B38 plugin). The Ollama host is
// resolved via NSELF_OLLAMA_HOST or PLUGIN_AI_OLLAMA_URL (same env vars as
// `nself ollama`), defaulting to http://localhost:11434.
//
// Sprint P95-AA10 (#897 / #213).

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Root command
// ---------------------------------------------------------------------------

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage local AI models via Ollama",
	Long: `List, pull, remove, update, and benchmark local AI models.

All operations require the ollama plugin to be installed and running:

  nself plugin install ollama
  nself start

Examples:
  nself model list
  nself model pull llama3.2:3b
  nself model remove gemma-3-4b
  nself model update llama3.2:3b
  nself model benchmark llama3.2:3b`,
	RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

// ---------------------------------------------------------------------------
// `nself model list`
// ---------------------------------------------------------------------------

var modelListJSON bool

var modelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all pulled Ollama models",
	Long: `Show every model currently downloaded in the Ollama store.

Columns:
  NAME         Tag name (e.g. llama3.2:3b)
  SIZE         Compressed disk size
  MODIFIED     Date the model was pulled or updated
  DEFAULT      Marked when NSELF_OLLAMA_DEFAULT_MODEL matches the name`,
	RunE: runModelList,
}

func runModelList(_ *cobra.Command, _ []string) error {
	var resp struct {
		Models []struct {
			Name       string    `json:"name"`
			Size       int64     `json:"size"`
			ModifiedAt time.Time `json:"modified_at"`
		} `json:"models"`
	}
	if err := modelOllamaGet("/api/tags", &resp); err != nil {
		return fmt.Errorf("model list: %w\nhint: is the ollama plugin installed? run: nself plugin install ollama", err)
	}

	if modelListJSON {
		return json.NewEncoder(os.Stdout).Encode(resp.Models)
	}

	if len(resp.Models) == 0 {
		fmt.Println("No models pulled yet.")
		fmt.Println("Run: nself model pull <name>   (e.g. nself model pull gemma-3-4b)")
		return nil
	}

	defaultModel := os.Getenv("NSELF_OLLAMA_DEFAULT_MODEL")

	fmt.Printf("%-42s  %9s  %-12s  %s\n", "NAME", "SIZE", "MODIFIED", "")
	fmt.Println(strings.Repeat("-", 72))
	for _, m := range resp.Models {
		tag := ""
		if defaultModel != "" && (m.Name == defaultModel || strings.HasPrefix(m.Name, defaultModel+":")) {
			tag = "[default]"
		}
		fmt.Printf("%-42s  %9s  %-12s  %s\n",
			m.Name,
			formatBytes(m.Size),
			m.ModifiedAt.Format("2006-01-02"),
			tag,
		)
	}
	return nil
}

// ---------------------------------------------------------------------------
// `nself model pull`
// ---------------------------------------------------------------------------

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

	client := &http.Client{Timeout: 30 * time.Minute}
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

// ---------------------------------------------------------------------------
// `nself model remove`
// ---------------------------------------------------------------------------

var modelRemoveCmd = &cobra.Command{
	Use:     "remove <model>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a pulled model from Ollama to free disk space",
	Args:    cobra.ExactArgs(1),
	RunE:    runModelRemove,
}

func runModelRemove(_ *cobra.Command, args []string) error {
	model := args[0]
	b, _ := json.Marshal(map[string]string{"name": model})
	req, err := http.NewRequest("DELETE", modelOllamaURL()+"/api/delete", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := modelHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("remove %s: %w", model, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("remove %s: HTTP %d: %s", model, resp.StatusCode, body)
	}

	fmt.Printf("Removed %s.\n", model)
	return nil
}

// ---------------------------------------------------------------------------
// `nself model update`
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// `nself model benchmark`
// ---------------------------------------------------------------------------

var (
	modelBenchPrompt string
	modelBenchRuns   int
	modelBenchJSON   bool
)

// benchResult holds metrics for one model benchmark run.
type benchResult struct {
	Model     string        `json:"model"`
	Runs      int           `json:"runs"`
	AvgTokPS  float64       `json:"avg_tok_s"`
	P99Ms     float64       `json:"p99_ms"`
	TotalToks int           `json:"total_tokens"`
	Errors    int           `json:"errors"`
	Elapsed   time.Duration `json:"-"`
}

var modelBenchmarkCmd = &cobra.Command{
	Use:   "benchmark <model>",
	Short: "Run a standard prompt and report tokens/s + p99 latency",
	Long: `Send the benchmark prompt to the model N times and measure:

  tok/s      Average tokens per second across all successful runs
  p99 ms     99th-percentile latency (time to full response)
  errors     Count of failed runs

The default prompt is:
  "Explain what a Merkle tree is in two sentences."

Override with --prompt. Increase --runs for a more stable measurement.`,
	Args: cobra.ExactArgs(1),
	RunE: runModelBenchmark,
}

func runModelBenchmark(_ *cobra.Command, args []string) error {
	model := args[0]
	prompt := modelBenchPrompt
	if prompt == "" {
		prompt = "Explain what a Merkle tree is in two sentences."
	}

	fmt.Printf("Benchmarking %s  (%d runs)...\n", model, modelBenchRuns)

	latencies := make([]float64, 0, modelBenchRuns)
	totalTokens := 0
	errCount := 0
	start := time.Now()

	for i := 0; i < modelBenchRuns; i++ {
		toks, ms, err := modelBenchRun(model, prompt)
		if err != nil {
			errCount++
			continue
		}
		latencies = append(latencies, ms)
		totalTokens += toks
	}

	elapsed := time.Since(start)

	successRuns := len(latencies)
	if successRuns == 0 {
		return fmt.Errorf("all %d runs failed — is Ollama running and the model pulled?", modelBenchRuns)
	}

	// Compute avg tok/s and p99 latency.
	totalMS := 0.0
	for _, ms := range latencies {
		totalMS += ms
	}
	avgMS := totalMS / float64(successRuns)

	sort.Float64s(latencies)
	p99idx := int(math.Ceil(float64(successRuns)*0.99)) - 1
	if p99idx < 0 {
		p99idx = 0
	}
	if p99idx >= successRuns {
		p99idx = successRuns - 1
	}
	p99 := latencies[p99idx]

	// tok/s: total tokens across all runs / total wall time for those runs (s).
	totalElapsedS := elapsed.Seconds()
	var avgTokPS float64
	if totalElapsedS > 0 {
		avgTokPS = float64(totalTokens) / totalElapsedS
	}

	res := benchResult{
		Model:     model,
		Runs:      modelBenchRuns,
		AvgTokPS:  avgTokPS,
		P99Ms:     p99,
		TotalToks: totalTokens,
		Errors:    errCount,
		Elapsed:   elapsed,
	}

	if modelBenchJSON {
		return json.NewEncoder(os.Stdout).Encode(res)
	}

	fmt.Println()
	fmt.Printf("  Model:        %s\n", res.Model)
	fmt.Printf("  Runs:         %d / %d succeeded\n", successRuns, modelBenchRuns)
	fmt.Printf("  Avg latency:  %.0f ms\n", avgMS)
	fmt.Printf("  p99 latency:  %.0f ms\n", res.P99Ms)
	fmt.Printf("  Tok/s:        %.1f\n", res.AvgTokPS)
	fmt.Printf("  Total tokens: %d\n", res.TotalToks)
	fmt.Printf("  Elapsed:      %s\n", elapsed.Round(time.Millisecond))
	if errCount > 0 {
		fmt.Printf("  Errors:       %d\n", errCount)
	}
	return nil
}

// modelBenchRun sends one chat request and returns (tokenCount, latencyMs, err).
func modelBenchRun(model, prompt string) (int, float64, error) {
	payload, _ := json.Marshal(map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	})

	req, err := http.NewRequest("POST", modelOllamaURL()+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	t0 := time.Now()
	resp, err := modelHTTPClient().Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	latencyMs := float64(time.Since(t0).Milliseconds())

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return 0, 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var result struct {
		EvalCount int `json:"eval_count"` // tokens in response
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, latencyMs, nil // still count the run even if parse fails
	}

	return result.EvalCount, latencyMs, nil
}

// ---------------------------------------------------------------------------
// HTTP helpers (model command scope — mirrors ollama.go helpers)
// ---------------------------------------------------------------------------

// modelOllamaURL returns the base URL for the Ollama API. Reads the same env
// vars as the `nself ollama` command tree so both share configuration.
func modelOllamaURL() string {
	if u := os.Getenv("NSELF_OLLAMA_HOST"); u != "" {
		return u
	}
	if u := os.Getenv("PLUGIN_AI_OLLAMA_URL"); u != "" {
		return u
	}
	return "http://localhost:11434"
}

func modelHTTPClient() *http.Client {
	timeout := 120 * time.Second
	if t := os.Getenv("NSELF_OLLAMA_TIMEOUT_SECONDS"); t != "" {
		var secs int
		if _, err := fmt.Sscanf(t, "%d", &secs); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}
	return &http.Client{Timeout: timeout}
}

// modelOllamaGet performs a GET against the Ollama API and decodes JSON.
func modelOllamaGet(path string, dst any) error {
	resp, err := modelHTTPClient().Get(modelOllamaURL() + path)
	if err != nil {
		return fmt.Errorf("ollama GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ollama GET %s HTTP %d: %s", path, resp.StatusCode, body)
	}
	return json.Unmarshal(body, dst)
}

// ---------------------------------------------------------------------------
// init — cobra registration
// ---------------------------------------------------------------------------

func init() {
	// list
	modelListCmd.Flags().BoolVar(&modelListJSON, "json", false, "Emit JSON output")

	// benchmark
	modelBenchmarkCmd.Flags().StringVar(&modelBenchPrompt, "prompt", "",
		"Custom prompt to use for the benchmark (default: Merkle tree question)")
	modelBenchmarkCmd.Flags().IntVar(&modelBenchRuns, "runs", 5,
		"Number of inference runs (higher = more stable p99)")
	modelBenchmarkCmd.Flags().BoolVar(&modelBenchJSON, "json", false, "Emit JSON output")

	// wire
	modelCmd.AddCommand(modelListCmd)
	modelCmd.AddCommand(modelPullCmd)
	modelCmd.AddCommand(modelRemoveCmd)
	modelCmd.AddCommand(modelUpdateCmd)
	modelCmd.AddCommand(modelBenchmarkCmd)

	RootCmd.AddCommand(modelCmd)
}
