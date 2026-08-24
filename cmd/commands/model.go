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
//
// Each subcommand's flags, cobra.Command var, and RunE handler now live in
// their own file (CLI-R12 Batch B mechanical file-size split):
// model_list.go, model_pull.go (also update), model_remove.go,
// model_benchmark.go. Shared HTTP helpers are in model_http.go.

import (
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
