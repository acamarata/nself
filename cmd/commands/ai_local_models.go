package commands

// Purpose: the "nself ai local models" subtree (list/add/remove/recommend)
// and its RunE functions. Inputs are the cobra command/args; outputs are
// model listings/changes printed to the user or an error.
// Constraints: split out of ai.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/installer"
	"github.com/spf13/cobra"
)

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
