package commands

// Purpose: `nself model benchmark` split out of model.go (CLI-R12 Batch B
// mechanical file-size split). Sends a fixed prompt to a model N times and
// reports tokens/s and p99 latency.
// Inputs: cobra command flags (--prompt, --runs, --json) and the
// positional model name.
// Outputs: a formatted report or JSON benchResult; errors when every run
// fails.
// Constraints: pure move, no behavior change. modelCmd (parent) and the
// modelOllamaURL/modelHTTPClient helpers remain in model.go/model_http.go.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

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
