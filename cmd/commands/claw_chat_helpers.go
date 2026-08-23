package commands

// Purpose: helpers used by the chat REPL: in-chat command handling, memory
// fetching, and streaming/rendering the model's response. Inputs are the
// current line/topic/model state or an http.Response; outputs are updated
// state or the rendered response text.
// Constraints: split out of claw_chat.go (CLI-R12) as a pure move, no behavior change.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
)

// handleChatCommand processes /commands. Returns true if should exit.
func handleChatCommand(line string, topic, model *string, client *http.Client, baseURL string, renderer *glamour.TermRenderer) bool {
	parts := strings.Fields(line)
	cmd := parts[0]

	switch cmd {
	case "/exit", "/quit":
		fmt.Println("Goodbye!")
		return true

	case "/topic":
		if len(parts) < 2 {
			if *topic == "" {
				fmt.Println("No topic set. Usage: /topic <name>")
			} else {
				fmt.Printf("Current topic: %s\n", *topic)
			}
		} else {
			*topic = strings.Join(parts[1:], " ")
			fmt.Printf("Topic set to: %s\n", *topic)
		}

	case "/model":
		if len(parts) < 2 {
			if *model == "" {
				fmt.Println("Using default model. Usage: /model <name>")
			} else {
				fmt.Printf("Current model: %s\n", *model)
			}
		} else {
			*model = parts[1]
			fmt.Printf("Model set to: %s\n", *model)
		}

	case "/memory":
		fmt.Println("Fetching recent memories...")
		fetchMemories(client, baseURL)

	case "/clear":
		fmt.Print("\033[H\033[2J")

	case "/help":
		fmt.Println("Commands:")
		fmt.Println("  /exit, /quit   Exit the session")
		fmt.Println("  /topic <name>  Switch topic context")
		fmt.Println("  /model <name>  Switch model")
		fmt.Println("  /memory        Show recent memories")
		fmt.Println("  /clear         Clear screen")
		fmt.Println("  /help          Show this help")

	default:
		fmt.Printf("Unknown command: %s (type /help)\n", cmd)
	}

	return false
}

// fetchMemories calls the server to get recent memories and prints them.
func fetchMemories(_ *http.Client, baseURL string) {
	authClient, authBaseURL, err := clawClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	if authBaseURL != "" {
		baseURL = authBaseURL
	}
	req, err := http.NewRequest("GET", baseURL+"/claw/v1/memories?limit=10", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}
	resp, err := authClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not fetch memories: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Server returned HTTP %d\n", resp.StatusCode)
		return
	}

	var result struct {
		Memories []struct {
			Content   string `json:"content"`
			CreatedAt string `json:"created_at"`
		} `json:"memories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		return
	}

	if len(result.Memories) == 0 {
		fmt.Println("No memories found.")
		return
	}

	for _, m := range result.Memories {
		fmt.Printf("  [%s] %s\n", m.CreatedAt, m.Content)
	}
}

// streamChatResponse reads SSE and prints content, optionally rendering markdown.
func streamChatResponse(resp *http.Response, renderer *glamour.TermRenderer, ctx context.Context) string {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	var fullContent strings.Builder

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return fullContent.String()
		default:
		}

		line := scanner.Text()
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
				fullContent.WriteString(content)
				// Print tokens as they arrive (raw during stream)
				fmt.Print(content)
			}
		}
	}

	fmt.Println() // newline after stream

	// Render the full response as markdown if a renderer is available
	full := fullContent.String()
	if full != "" && renderer != nil {
		rendered, err := renderer.Render(full)
		if err == nil && rendered != "" {
			fmt.Print("\r")
			fmt.Print(rendered)
		}
	}

	return full
}

// streamChatResponseReader is a helper for reading SSE from an io.Reader.
func streamChatResponseReader(r io.Reader, ctx context.Context) string {
	resp := &http.Response{Body: io.NopCloser(r)}
	return streamChatResponse(resp, nil, ctx)
}
