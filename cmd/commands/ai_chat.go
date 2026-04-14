// P88 Sprint 05: `nself ai chat` — quick chat for verification.
// Acceptance criteria: `nself ai chat "hello"` returns coherent local response <3s.
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	aiChatModel string
	aiChatJSON  bool
)

var aiChatCmd = &cobra.Command{
	Use:   "chat <message>",
	Short: "Send a quick chat message to the local AI",
	Long: `Send a message to the local Ollama instance and print the response.
Useful for verifying that local AI is working after setup.

Example:
  nself ai chat "hello"
  nself ai chat --model llama3.2:3b "explain recursion"`,
	Args: cobra.ExactArgs(1),
	RunE: runAIChat,
}

func runAIChat(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	message := args[0]
	model := aiChatModel
	if model == "" {
		model = os.Getenv("AI_DEFAULT_MODEL")
	}
	if model == "" {
		model = "gemma2:2b"
	}

	payload, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": message},
		},
		"stream": false,
	})

	req, err := http.NewRequestWithContext(ctx, "POST",
		ollamaBaseURL()+"/api/chat",
		strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama unreachable: %w (run 'nself doctor --ai' to set up)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Ollama error %d: %s", resp.StatusCode, string(b))
	}

	var chatResp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		TotalDuration int64 `json:"total_duration"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if aiChatJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"model":    model,
			"response": chatResp.Message.Content,
			"duration": fmt.Sprintf("%dms", chatResp.TotalDuration/1e6),
		})
	}

	fmt.Println(chatResp.Message.Content)
	return nil
}

func init() {
	aiChatCmd.Flags().StringVar(&aiChatModel, "model", "", "Model to use (default: AI_DEFAULT_MODEL or gemma2:2b)")
	aiChatCmd.Flags().BoolVar(&aiChatJSON, "json", false, "Emit JSON output")
	aiCmd.AddCommand(aiChatCmd)
}
