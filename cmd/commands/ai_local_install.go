package commands

// Purpose: the "nself ai local install" subcommand and its RunE, which
// bootstraps a local Ollama-backed LLM setup. Inputs are the cobra
// command/args; outputs are an installed local model runtime or an error.
// Constraints: split out of ai.go (CLI-R12) as a pure move, no behavior change.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/installer"
	"github.com/spf13/cobra"
)

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
