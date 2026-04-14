package commands

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

	"github.com/nself-org/cli/internal/installer"
	"github.com/spf13/cobra"
)

// -----------------------------------------------------------------------------
// `nself ai` root + `nself ai local` subtree
// Sprint P88-S02: zero-config local LLM automation.
// -----------------------------------------------------------------------------

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Manage the nSelf AI plugin and local LLM stack",
	Long: `Commands for managing the nSelf AI plugin (providers, models, routing).

Subcommands:
  local    Manage the local Ollama runtime and models`,
	RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

var aiLocalCmd = &cobra.Command{
	Use:   "local",
	Short: "Manage local Ollama runtime and models",
	Long: `Install, inspect, and manage a local Ollama runtime for zero-config AI.

Subcommands:
  install     Install Ollama, systemd service, firewall, and recommended models
  models      List / add / remove / recommend local models
  health      Show Ollama + plugin-ai health
  swap        Hot-swap the default model for a task
  benchmark   Run benchmark prompts against one or more models`,
	RunE: func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

// -----------------------------------------------------------------------------
// `nself ai local install`  (T-02-03)
// -----------------------------------------------------------------------------

var (
	aiInstallYes       bool
	aiInstallNoModels  bool
	aiInstallModelFlag string
	aiInstallBind      string
	aiInstallJSON      bool
)

var aiLocalInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Ollama + systemd + firewall + recommended models",
	RunE:  runAILocalInstall,
}

func runAILocalInstall(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	opts := installer.InstallOptions{
		Yes:        aiInstallYes,
		SkipModels: aiInstallNoModels,
		Model:      aiInstallModelFlag,
		Bind:       aiInstallBind,
		JSON:       aiInstallJSON,
		LogFn: func(level, msg string, kv map[string]any) {
			if aiInstallJSON {
				return
			}
			fmt.Fprintf(os.Stderr, "[%s] %s", level, msg)
			if len(kv) > 0 {
				b, _ := json.Marshal(kv)
				fmt.Fprintf(os.Stderr, " %s", string(b))
			}
			fmt.Fprintln(os.Stderr)
		},
	}
	res, err := installer.Install(ctx, opts)
	if err != nil {
		if aiInstallJSON {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]string{
				"status": "error", "error": err.Error(),
			})
		}
		return err
	}
	if aiInstallJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"status": "ok",
			"result": res,
		})
	}
	fmt.Println()
	fmt.Println("  ✓ Ollama installed and reachable")
	fmt.Printf("    Version:       %s\n", res.OllamaVersion)
	fmt.Printf("    Bind:          %s\n", res.Bind)
	fmt.Printf("    RAM tier:      %s\n", res.Tier)
	fmt.Printf("    Models pulled: %s\n", strings.Join(res.ModelsPulled, ", "))
	fmt.Println()
	return nil
}

// -----------------------------------------------------------------------------
// `nself ai local models ...`  (T-02-04)
// -----------------------------------------------------------------------------

var aiLocalModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List, add, remove, or recommend local models",
	RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
}

var (
	modelsListInstalled  bool
	modelsListRegistered bool
	modelsListJSON       bool
)

var aiLocalModelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed + registered local models with diff",
	RunE:  runModelsList,
}

func runModelsList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	installed, err := ollamaListInstalled(ctx)
	if err != nil {
		installed = nil
	}
	registered, _ := aiPluginGET(ctx, "/ai/local/models")

	if modelsListJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"installed":  installed,
			"registered": json.RawMessage(registered),
		})
	}
	fmt.Printf("%-28s %-10s %-10s %-20s\n", "NAME", "SIZE_MB", "STATE", "TASKS")
	for _, m := range installed {
		fmt.Printf("%-28s %-10d %-10s %-20s\n", m.Name, m.SizeMB, "installed", "-")
	}
	return nil
}

var (
	modelAddTask    string
	modelAddDefault bool
)

var aiLocalModelsAddCmd = &cobra.Command{
	Use:   "add <model>",
	Short: "Pull a model via Ollama and register with plugin-ai",
	Args:  cobra.ExactArgs(1),
	RunE:  runModelsAdd,
}

func runModelsAdd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]
	fmt.Printf("Pulling %s from Ollama...\n", name)
	if err := ollamaPull(ctx, name); err != nil {
		return fmt.Errorf("ollama pull: %w", err)
	}
	tasks := strings.Split(strings.TrimSpace(modelAddTask), ",")
	if modelAddTask == "" {
		tasks = []string{"chat"}
	}
	payload, _ := json.Marshal(map[string]any{
		"name":        name,
		"tasks":       tasks,
		"set_default": modelAddDefault,
	})
	body, status, err := aiPluginRequest(ctx, "POST", "/ai/local/models", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(body))
	}
	fmt.Printf("Registered %s (tasks=%s, default=%t)\n", name, strings.Join(tasks, ","), modelAddDefault)
	return nil
}

var modelRemoveForce bool

var aiLocalModelsRemoveCmd = &cobra.Command{
	Use:   "remove <model>",
	Short: "Soft-delete a local model and uninstall from Ollama",
	Args:  cobra.ExactArgs(1),
	RunE:  runModelsRemove,
}

func runModelsRemove(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]
	path := "/ai/local/models/" + name
	if modelRemoveForce {
		path += "?force=true"
	}
	body, status, err := aiPluginRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	if status == 409 {
		return fmt.Errorf("%s is a default model; pass --force to remove", name)
	}
	if status >= 400 {
		return fmt.Errorf("plugin-ai %d: %s", status, string(body))
	}
	_ = ollamaDelete(ctx, name)
	fmt.Printf("Removed %s\n", name)
	return nil
}

var modelRecommendTier string

var aiLocalModelsRecommendCmd = &cobra.Command{
	Use:   "recommend",
	Short: "Print recommended models for this host",
	RunE:  runModelsRecommend,
}

func runModelsRecommend(_ *cobra.Command, _ []string) error {
	tier, recs := installer.RecommendForHost()
	fmt.Printf("Detected RAM tier: %s\n", tier)
	if len(recs) == 0 {
		fmt.Println("No models recommended for this tier.")
		return nil
	}
	fmt.Printf("%-28s %-20s\n", "MODEL", "TASKS")
	for _, r := range recs {
		fmt.Printf("%-28s %-20s\n", r.Name, strings.Join(r.Tasks, ","))
	}
	return nil
}

// -----------------------------------------------------------------------------
// `nself ai local health|swap|benchmark`  (T-02-05)
// -----------------------------------------------------------------------------

var (
	aiHealthWatch bool
	aiHealthJSON  bool
)

var aiLocalHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show Ollama + plugin-ai health",
	RunE:  runAILocalHealth,
}

func runAILocalHealth(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	for {
		body, status, err := aiPluginRequest(ctx, "GET", "/ai/local/status", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin-ai unreachable: %v\n", err)
			if !aiHealthWatch {
				return err
			}
		} else {
			if aiHealthJSON {
				fmt.Println(string(body))
			} else {
				fmt.Printf("HTTP %d\n%s\n", status, string(body))
			}
		}
		if !aiHealthWatch {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

var (
	swapTask   string
	swapReason string
)

var aiLocalSwapCmd = &cobra.Command{
	Use:   "swap <model>",
	Short: "Hot-swap the default model for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runAILocalSwap,
}

func runAILocalSwap(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	tasks := []string{"chat"}
	if swapTask != "" && swapTask != "all" {
		tasks = strings.Split(swapTask, ",")
	}
	if swapTask == "all" {
		tasks = []string{"chat", "embed", "classify"}
	}
	payload, _ := json.Marshal(map[string]any{
		"model":  args[0],
		"tasks":  tasks,
		"reason": swapReason,
	})
	body, status, err := aiPluginRequest(ctx, "POST", "/ai/local/swap-model", payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("swap failed %d: %s", status, string(body))
	}
	fmt.Printf("Swapped to %s for tasks=%s\n", args[0], strings.Join(tasks, ","))
	return nil
}

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
	resp, err := http.DefaultClient.Do(req)
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func aiPluginURL() string {
	if u := os.Getenv("PLUGIN_AI_INTERNAL_URL"); u != "" {
		return u
	}
	return "http://ai:3680"
}

func aiPluginRequest(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, aiPluginURL()+path, rdr)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok := os.Getenv("PLUGIN_INTERNAL_SECRET"); tok != "" {
		req.Header.Set("X-Internal-Token", tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return b, resp.StatusCode, nil
}

func aiPluginGET(ctx context.Context, path string) ([]byte, error) {
	body, status, err := aiPluginRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("plugin-ai %s %d: %s", path, status, string(body))
	}
	return body, nil
}

// -----------------------------------------------------------------------------
// init — wire commands + flags
// -----------------------------------------------------------------------------

func init() {
	// install flags
	aiLocalInstallCmd.Flags().BoolVar(&aiInstallYes, "yes", false, "Non-interactive mode")
	aiLocalInstallCmd.Flags().BoolVar(&aiInstallNoModels, "no-models", false, "Skip model pulls")
	aiLocalInstallCmd.Flags().StringVar(&aiInstallModelFlag, "model", "", "Pull only this model")
	aiLocalInstallCmd.Flags().StringVar(&aiInstallBind, "bind", "", "host:port to bind Ollama to")
	aiLocalInstallCmd.Flags().BoolVar(&aiInstallJSON, "json", false, "Emit JSON output")

	// models flags
	aiLocalModelsListCmd.Flags().BoolVar(&modelsListInstalled, "installed", false, "Show only installed")
	aiLocalModelsListCmd.Flags().BoolVar(&modelsListRegistered, "registered", false, "Show only registered")
	aiLocalModelsListCmd.Flags().BoolVar(&modelsListJSON, "json", false, "Emit JSON output")
	aiLocalModelsAddCmd.Flags().StringVar(&modelAddTask, "task", "chat", "Comma-separated task classes")
	aiLocalModelsAddCmd.Flags().BoolVar(&modelAddDefault, "default", false, "Set as default for the tasks")
	aiLocalModelsRemoveCmd.Flags().BoolVar(&modelRemoveForce, "force", false, "Remove even if default")
	aiLocalModelsRecommendCmd.Flags().StringVar(&modelRecommendTier, "tier", "auto", "Force a tier (auto|minimal|balanced|max)")

	aiLocalModelsCmd.AddCommand(aiLocalModelsListCmd)
	aiLocalModelsCmd.AddCommand(aiLocalModelsAddCmd)
	aiLocalModelsCmd.AddCommand(aiLocalModelsRemoveCmd)
	aiLocalModelsCmd.AddCommand(aiLocalModelsRecommendCmd)

	// health/swap/benchmark
	aiLocalHealthCmd.Flags().BoolVar(&aiHealthWatch, "watch", false, "Re-poll every 2s")
	aiLocalHealthCmd.Flags().BoolVar(&aiHealthJSON, "json", false, "Emit JSON output")
	aiLocalSwapCmd.Flags().StringVar(&swapTask, "task", "chat", "Task: chat|embed|classify|all")
	aiLocalSwapCmd.Flags().StringVar(&swapReason, "reason", "", "Free-text reason (audit log)")
	aiLocalBenchmarkCmd.Flags().StringVar(&benchTasks, "tasks", "chat", "Comma-separated tasks")
	aiLocalBenchmarkCmd.Flags().IntVar(&benchIterations, "iterations", 5, "Iterations per task")

	aiLocalCmd.AddCommand(aiLocalInstallCmd)
	aiLocalCmd.AddCommand(aiLocalModelsCmd)
	aiLocalCmd.AddCommand(aiLocalHealthCmd)
	aiLocalCmd.AddCommand(aiLocalSwapCmd)
	aiLocalCmd.AddCommand(aiLocalBenchmarkCmd)

	aiCmd.AddCommand(aiLocalCmd)
	RootCmd.AddCommand(aiCmd)
}
