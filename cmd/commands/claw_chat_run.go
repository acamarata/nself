package commands

// Purpose: runClawChat, the RunE for "nself claw chat" (the interactive
// chat REPL loop). Inputs are the cobra command/args; outputs are the
// interactive session's exit result or an error.
// Constraints: split out of claw_chat.go (CLI-R12) as a pure move, no behavior change.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

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
