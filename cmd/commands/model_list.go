package commands

// Purpose: `nself model list` split out of model.go (CLI-R12 Batch B
// mechanical file-size split). Lists every model in the local Ollama
// store.
// Inputs: cobra command flag --json.
// Outputs: a formatted table or JSON array of pulled models.
// Constraints: pure move, no behavior change. modelCmd (parent) and the
// modelOllamaGet/modelOllamaURL helpers remain in model.go/model_http.go.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

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
