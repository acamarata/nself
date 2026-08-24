package commands

// Purpose: `nself model remove` split out of model.go (CLI-R12 Batch B
// mechanical file-size split). Deletes a pulled model from Ollama.
// Inputs: the positional model name.
// Outputs: stdout confirmation; errors wrap Ollama API failures.
// Constraints: pure move, no behavior change. modelCmd (parent) and the
// modelOllamaURL/modelHTTPClient helpers remain in model.go/model_http.go.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

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
