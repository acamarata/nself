package commands

import (
	"bufio"
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

var (
	clawPromptJSON     bool
	clawPromptModel    string
	clawPromptTopic    string
	clawPromptNoMemory bool
	clawPromptSystem   string
	clawPromptRaw      bool
	clawPromptTimeout  time.Duration
	clawPromptFile     string
	clawPromptNoStream bool
)

var clawPromptCmd = &cobra.Command{
	Use:   "prompt [question]",
	Short: "Send a single prompt to nClaw and print the response",
	Long: `Send a one-shot prompt to the nClaw server and print the AI response.

By default, tokens stream to stdout as they arrive (SSE). Use --no-stream
to wait for the complete response before printing.

Supports stdin piping:
  echo "some text" | nself claw prompt "Summarize this"

Exit codes:
  0  Success
  1  API error
  2  Auth error (no API key)
  3  Connection error`,
	RunE: runClawPrompt,
}

func init() {
	clawPromptCmd.Flags().BoolVar(&clawPromptJSON, "json", false, "Output full JSON response")
	clawPromptCmd.Flags().StringVar(&clawPromptModel, "model", "", "Override model name")
	clawPromptCmd.Flags().StringVar(&clawPromptTopic, "topic", "", "Topic context")
	clawPromptCmd.Flags().BoolVar(&clawPromptNoMemory, "no-memory", false, "Skip memory injection")
	clawPromptCmd.Flags().StringVar(&clawPromptSystem, "system", "", "Custom system prompt")
	clawPromptCmd.Flags().BoolVar(&clawPromptRaw, "raw", false, "No formatting, raw text only")
	clawPromptCmd.Flags().DurationVar(&clawPromptTimeout, "timeout", 120*time.Second, "Request timeout")
	clawPromptCmd.Flags().StringVarP(&clawPromptFile, "file", "f", "", "Read prompt from file")
	clawPromptCmd.Flags().BoolVar(&clawPromptNoStream, "no-stream", false, "Wait for complete response")
}

func runClawPrompt(cmd *cobra.Command, args []string) error {
	client, baseURL, err := clawClient()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}
	client.Timeout = clawPromptTimeout

	// Build the prompt from args + file + stdin
	var parts []string

	// Read from file if -f given
	if clawPromptFile != "" {
		data, err := os.ReadFile(clawPromptFile)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		parts = append(parts, string(data))
	}

	// Read from stdin if piped
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		if len(data) > 0 {
			parts = append(parts, string(data))
		}
	}

	// Positional args
	if len(args) > 0 {
		parts = append(parts, strings.Join(args, " "))
	}

	if len(parts) == 0 {
		return fmt.Errorf("no prompt provided; pass a question as argument, pipe stdin, or use -f")
	}

	prompt := strings.Join(parts, "\n\n")

	// Build request body (OpenAI-compatible)
	messages := []map[string]string{}
	if clawPromptSystem != "" {
		messages = append(messages, map[string]string{"role": "system", "content": clawPromptSystem})
	}
	messages = append(messages, map[string]string{"role": "user", "content": prompt})

	body := map[string]interface{}{
		"messages": messages,
		"stream":   !clawPromptNoStream,
	}
	if clawPromptModel != "" {
		body["model"] = clawPromptModel
	}

	// Metadata for topic/memory
	metadata := map[string]interface{}{}
	if clawPromptTopic != "" {
		metadata["topic"] = clawPromptTopic
	}
	if clawPromptNoMemory {
		metadata["no_memory"] = true
	}
	if len(metadata) > 0 {
		body["metadata"] = metadata
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	url := baseURL + "/claw/v1/chat/completions"
	req, err := http.NewRequestWithContext(cmd.Context(), "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("authentication failed; check your API key")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if clawPromptNoStream {
		return handleNonStreamResponse(resp)
	}
	return handleStreamResponse(resp)
}

func handleNonStreamResponse(resp *http.Response) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if clawPromptJSON {
		fmt.Println(string(bodyBytes))
		return nil
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Choices) > 0 {
		fmt.Println(result.Choices[0].Message.Content)
	}
	return nil
}

func handleStreamResponse(resp *http.Response) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	var fullContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// SSE format: lines starting with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				if clawPromptJSON {
					fullContent.WriteString(content)
				} else {
					fmt.Print(content)
				}
			}
		}
	}

	if clawPromptJSON {
		// Reconstruct a JSON response from accumulated content
		result := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": fullContent.String(),
					},
					"finish_reason": "stop",
				},
			},
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println() // final newline after streaming
	}

	return nil
}
