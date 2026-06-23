package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ollamaCmd is the top-level `nself ollama` command.
//
// Deprecated: use `nself model` for day-to-day model management. `nself ollama`
// is preserved for compatibility but the `model` command is the current surface.
var ollamaCmd = &cobra.Command{
	Use:        "ollama",
	Short:      "[Deprecated] Manage local Ollama LLM stack — use `nself model` instead",
	Deprecated: "use `nself model` for model management (e.g. nself model list, nself model pull <model>). `nself ollama` is kept for compatibility only.",
	Long: `[DEPRECATED] Manage the local Ollama container and its model library.

Prefer ` + "`nself model`" + ` for all day-to-day model operations. This command is
preserved for backwards compatibility only.

After installing the ollama plugin (` + "`nself plugin install ollama`" + `), Ollama
runs as a service alongside the nSelf stack. Use these commands to pull, list,
and remove models, and to check Ollama status.

Examples:
  nself model list                        (preferred)
  nself model pull llama3.2:3b            (preferred)
  nself model remove gemma-3-4b           (preferred)
  nself ollama models list                (compatibility)
  nself ollama models pull llama3.2:3b    (compatibility)
  nself ollama models remove gemma-3-4b   (compatibility)
  nself ollama status`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ollamaModelsCmd groups model management subcommands.
var ollamaModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage Ollama models",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var ollamaModelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pulled Ollama models",
	RunE:  runOllamaModelsList,
}

var ollamaModelsPullCmd = &cobra.Command{
	Use:   "pull <model>",
	Short: "Pull an Ollama model (e.g. llama3.2:3b, gemma-3-4b)",
	Args:  cobra.ExactArgs(1),
	RunE:  runOllamaModelsPull,
}

var ollamaModelsRemoveCmd = &cobra.Command{
	Use:     "remove <model>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove an Ollama model",
	Args:    cobra.ExactArgs(1),
	RunE:    runOllamaModelsRemove,
}

var ollamaStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Ollama service status and default model",
	RunE:  runOllamaStatus,
}

func init() {
	ollamaModelsCmd.AddCommand(ollamaModelsListCmd)
	ollamaModelsCmd.AddCommand(ollamaModelsPullCmd)
	ollamaModelsCmd.AddCommand(ollamaModelsRemoveCmd)
	ollamaCmd.AddCommand(ollamaModelsCmd)
	ollamaCmd.AddCommand(ollamaStatusCmd)
	RootCmd.AddCommand(ollamaCmd)
}

// --- helpers ---

func ollamaURL() string {
	if u := os.Getenv("NSELF_OLLAMA_HOST"); u != "" {
		return u
	}
	if u := os.Getenv("PLUGIN_AI_OLLAMA_URL"); u != "" {
		return u
	}
	return "http://localhost:11434"
}

func ollamaHTTPClient() *http.Client {
	timeout := 120 * time.Second
	if t := os.Getenv("NSELF_OLLAMA_TIMEOUT_SECONDS"); t != "" {
		// Parse manually to avoid import of strconv for a simple case.
		var secs int
		if _, err := fmt.Sscanf(t, "%d", &secs); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}
	return &http.Client{Timeout: timeout}
}

// ollamaGet performs a GET against the Ollama API and decodes JSON into dst.
func ollamaGet(path string, dst any) error {
	resp, err := ollamaHTTPClient().Get(ollamaURL() + path)
	if err != nil {
		return fmt.Errorf("ollama: GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ollama: GET %s HTTP %d: %s", path, resp.StatusCode, body)
	}
	return json.Unmarshal(body, dst)
}

// ollamaPost performs a POST against the Ollama API, streaming lines to stdout.
func ollamaPost(path string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := ollamaHTTPClient().Post(ollamaURL()+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("ollama: POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ollama: POST %s HTTP %d: %s", path, resp.StatusCode, body)
	}
	// Pretty-print progress lines
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	for _, line := range lines {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			if status, ok := m["status"].(string); ok {
				fmt.Println(status)
			}
		}
	}
	return nil
}

// --- command implementations ---

func runOllamaModelsList(_ *cobra.Command, _ []string) error {
	var resp struct {
		Models []struct {
			Name       string    `json:"name"`
			Size       int64     `json:"size"`
			ModifiedAt time.Time `json:"modified_at"`
		} `json:"models"`
	}
	if err := ollamaGet("/api/tags", &resp); err != nil {
		return fmt.Errorf("list models: %w (is the ollama plugin installed and running?)", err)
	}

	if len(resp.Models) == 0 {
		fmt.Println("No models pulled yet. Run: nself ollama models pull gemma-3-4b")
		return nil
	}

	defaultModel := os.Getenv("NSELF_OLLAMA_DEFAULT_MODEL")
	if defaultModel == "" {
		defaultModel = "gemma-3-4b"
	}

	fmt.Printf("%-40s  %10s  %s\n", "NAME", "SIZE", "MODIFIED")
	fmt.Println(strings.Repeat("-", 70))
	for _, m := range resp.Models {
		tag := ""
		if m.Name == defaultModel || strings.HasPrefix(m.Name, defaultModel+":") {
			tag = " [default]"
		}
		fmt.Printf("%-40s  %10s  %s%s\n",
			m.Name,
			formatBytes(m.Size),
			m.ModifiedAt.Format("2006-01-02"),
			tag,
		)
	}
	return nil
}

func runOllamaModelsPull(_ *cobra.Command, args []string) error {
	model := args[0]
	fmt.Printf("Pulling %s...\n", model)
	if err := ollamaPost("/api/pull", map[string]string{"name": model}); err != nil {
		return fmt.Errorf("pull %s: %w", model, err)
	}
	fmt.Printf("Model %s ready\n", model)
	return nil
}

func runOllamaModelsRemove(_ *cobra.Command, args []string) error {
	model := args[0]
	b, _ := json.Marshal(map[string]string{"name": model})
	req, err := http.NewRequest("DELETE", ollamaURL()+"/api/delete", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := ollamaHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("remove %s: %w", model, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove %s: HTTP %d: %s", model, resp.StatusCode, body)
	}
	fmt.Printf("Removed %s\n", model)
	return nil
}

func runOllamaStatus(_ *cobra.Command, _ []string) error {
	var version struct {
		Version string `json:"version"`
	}
	if err := ollamaGet("/api/version", &version); err != nil {
		return fmt.Errorf("ollama not reachable: %w\nIs `nself plugin install ollama` complete and `nself start` running?", err)
	}

	defaultModel := os.Getenv("NSELF_OLLAMA_DEFAULT_MODEL")
	if defaultModel == "" {
		defaultModel = "gemma-3-4b"
	}

	fmt.Printf("ollama: ready\n")
	fmt.Printf("version: %s\n", version.Version)
	fmt.Printf("url: %s\n", ollamaURL())
	fmt.Printf("default model: %s\n", defaultModel)
	fmt.Printf("vram: n/a (CPU mode unless NSELF_OLLAMA_GPU=true)\n")
	return nil
}

// formatBytes is defined in migrate.go (shared across the commands package).
