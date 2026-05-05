package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

var (
	clawChatTopic   string
	clawChatModel   string
	clawChatResume  bool
	clawChatSession string
)

var clawChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session with nClaw",
	Long: `Open an interactive terminal chat session with your nClaw AI assistant.

Type messages at the ɳ> prompt. AI responses stream with markdown rendering.

REPL commands:
  /exit, /quit   Exit the session
  /topic <name>  Switch topic context
  /model <name>  Switch model
  /memory        Show recent memories
  /clear         Clear screen
  /help          Show available commands

Ctrl+C cancels the current generation.
Ctrl+D exits the session.`,
	RunE: runClawChat,
}

func init() {
	clawChatCmd.Flags().StringVar(&clawChatTopic, "topic", "", "Start in specific topic")
	clawChatCmd.Flags().StringVar(&clawChatModel, "model", "", "Use specific model")
	clawChatCmd.Flags().BoolVar(&clawChatResume, "resume", false, "Resume last conversation")
	clawChatCmd.Flags().StringVar(&clawChatSession, "session", "", "Resume specific session ID")
}

func runClawChat(cmd *cobra.Command, args []string) error {
	client, baseURL, err := clawClient()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	// Ensure history directory exists
	histPath := clawHistoryPath()
	if err := os.MkdirAll(filepath.Dir(histPath), 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create history dir: %v\n", err)
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "\033[1;35mɳ>\033[0m ",
		HistoryFile:       histPath,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return fmt.Errorf("initializing readline: %w", err)
	}
	defer rl.Close()

	// Set up glamour renderer for markdown
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		// Fall back to no rendering
		renderer = nil
	}

	// Track conversation history
	var history []map[string]string
	activeTopic := clawChatTopic
	activeModel := clawChatModel
	sessionID := clawChatSession

	fmt.Println("nClaw Interactive Chat")
	fmt.Println("Type /help for commands, Ctrl+D to exit")
	fmt.Println()

	// Persistent signal channel for Ctrl+C — avoids Notify/Stop per iteration
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	for {
		line, err := rl.Readline()
		if err != nil {
			// Ctrl+D or EOF
			if err == readline.ErrInterrupt {
				continue
			}
			fmt.Println("\nGoodbye!")
			return nil
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle REPL commands
		if strings.HasPrefix(line, "/") {
			if handleChatCommand(line, &activeTopic, &activeModel, client, baseURL, renderer) {
				return nil // /exit
			}
			continue
		}

		// Add user message to history
		history = append(history, map[string]string{"role": "user", "content": line})

		// Build request
		body := map[string]interface{}{
			"messages": history,
			"stream":   true,
		}
		if activeModel != "" {
			body["model"] = activeModel
		}
		metadata := map[string]interface{}{}
		if activeTopic != "" {
			metadata["topic"] = activeTopic
		}
		if sessionID != "" {
			metadata["session_id"] = sessionID
		}
		if clawChatResume && sessionID == "" {
			metadata["resume"] = true
		}
		if len(metadata) > 0 {
			body["metadata"] = metadata
		}

		jsonBody, _ := json.Marshal(body)
		url := baseURL + "/claw/v1/chat/completions"

		// Create cancellable context for Ctrl+C
		ctx, cancel := context.WithCancel(cmd.Context())

		go func() {
			select {
			case <-sigCh:
				cancel()
			case <-ctx.Done():
			}
		}()

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
		if err != nil {
			cancel()
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			if ctx.Err() != nil {
				fmt.Println("\n(generation cancelled)")
				continue
			}
			fmt.Fprintf(os.Stderr, "Connection error: %v\n", err)
			continue
		}

		// Stream response
		content := streamChatResponse(resp, renderer, ctx)
		resp.Body.Close()
		cancel()

		if content != "" {
			history = append(history, map[string]string{"role": "assistant", "content": content})
		}
	}
}

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
