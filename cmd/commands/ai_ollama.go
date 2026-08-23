package commands

// Purpose: the "nself ai local benchmark" subcommand plus the low-level
// Ollama client helpers (list/pull/delete installed models). Inputs are a
// context and model name; outputs are benchmark results or ollama API
// responses, or an error.
// Constraints: split out of ai.go (CLI-R12) as a pure move, no behavior change.

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

	"github.com/nself-org/cli/internal/httptimeout"
	"github.com/spf13/cobra"
)

var (
	benchTasks      string
	benchIterations int
)

var aiLocalBenchmarkCmd = &cobra.Command{
	Use:   "benchmark [model]",
	Short: "Run benchmark suite against one or more models",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAILocalBenchmark,
}

func runAILocalBenchmark(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	var models []string
	if len(args) > 0 {
		models = []string{args[0]}
	}
	tasks := []string{"chat"}
	if benchTasks != "" {
		tasks = strings.Split(benchTasks, ",")
	}
	payload, _ := json.Marshal(map[string]any{
		"models":     models,
		"tasks":      tasks,
		"iterations": benchIterations,
	})
	body, status, err := aiPluginRequest(ctx, "POST", "/ai/local/benchmark", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("benchmark failed %d: %s", status, string(body))
	}
	fmt.Println(string(body))
	return nil
}

// -----------------------------------------------------------------------------
// HTTP + Ollama helpers
// -----------------------------------------------------------------------------

type ollamaTag struct {
	Name   string `json:"name"`
	SizeMB int64  `json:"size_mb"`
	Digest string `json:"digest"`
}

func ollamaBaseURL() string {
	if u := os.Getenv("OLLAMA_BASE_URL"); u != "" {
		return u
	}
	if u := os.Getenv("OLLAMA_HOST"); u != "" {
		if !strings.HasPrefix(u, "http") {
			return "http://" + u
		}
		return u
	}
	return "http://127.0.0.1:11434"
}

func ollamaListInstalled(ctx context.Context) ([]ollamaTag, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", ollamaBaseURL()+"/api/tags", nil)
	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body struct {
		Models []struct {
			Name   string `json:"name"`
			Size   int64  `json:"size"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]ollamaTag, 0, len(body.Models))
	for _, m := range body.Models {
		out = append(out, ollamaTag{Name: m.Name, SizeMB: m.Size / 1024 / 1024, Digest: m.Digest})
	}
	return out, nil
}

func ollamaPull(ctx context.Context, name string) error {
	payload, _ := json.Marshal(map[string]any{"name": name, "stream": false})
	req, _ := http.NewRequestWithContext(ctx, "POST",
		ollamaBaseURL()+"/api/pull", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func ollamaDelete(ctx context.Context, name string) error {
	payload, _ := json.Marshal(map[string]any{"name": name})
	req, _ := http.NewRequestWithContext(ctx, "DELETE",
		ollamaBaseURL()+"/api/delete", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httptimeout.Default.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
