package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var (
	clawMCPTransport string
	clawMCPPort      int
)

var clawMCPCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start an MCP server for nClaw",
	Long: `Start a Model Context Protocol (MCP) server that exposes nClaw tools.

By default, runs on stdio (for use as a Claude Code MCP server).
Use --transport sse --port 8899 for SSE transport.

Available tools:
  chat              Send a message and get a response
  search_memories   Search user memories
  search_topics     Search topics
  list_topics       List all topics
  create_memory     Store a new memory

Example Claude Code config (.claude/settings.json):
  "mcpServers": {
    "nclaw": {
      "command": "nself",
      "args": ["claw", "mcp"]
    }
  }`,
	RunE: runClawMCP,
}

func init() {
	clawMCPCmd.Flags().StringVar(&clawMCPTransport, "transport", "stdio", "Transport mode: stdio or sse")
	clawMCPCmd.Flags().IntVar(&clawMCPPort, "port", 8899, "Port for SSE transport")
}

func runClawMCP(cmd *cobra.Command, args []string) error {
	_, baseURL, err := clawClient()
	if err != nil {
		return err
	}
	apiKey := clawAPIKey()

	mcpServer := server.NewMCPServer(
		"nClaw MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tools
	mcpServer.AddTool(
		mcp.NewTool("chat",
			mcp.WithDescription("Send a message to nClaw and get a response"),
			mcp.WithString("message", mcp.Required(), mcp.Description("The message to send")),
			mcp.WithString("topic", mcp.Description("Optional topic context")),
			mcp.WithString("model", mcp.Description("Optional model override")),
		),
		mcpChatHandler(baseURL, apiKey),
	)

	mcpServer.AddTool(
		mcp.NewTool("search_memories",
			mcp.WithDescription("Search user memories by content"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		),
		mcpSearchMemoriesHandler(baseURL, apiKey),
	)

	mcpServer.AddTool(
		mcp.NewTool("search_topics",
			mcp.WithDescription("Search topics by name"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		),
		mcpSearchTopicsHandler(baseURL, apiKey),
	)

	mcpServer.AddTool(
		mcp.NewTool("list_topics",
			mcp.WithDescription("List all conversation topics"),
		),
		mcpListTopicsHandler(baseURL, apiKey),
	)

	mcpServer.AddTool(
		mcp.NewTool("create_memory",
			mcp.WithDescription("Store a new memory"),
			mcp.WithString("content", mcp.Required(), mcp.Description("Memory content to store")),
			mcp.WithString("type", mcp.Description("Memory type (fact, decision, preference, etc.)")),
		),
		mcpCreateMemoryHandler(baseURL, apiKey),
	)

	switch clawMCPTransport {
	case "stdio":
		return server.ServeStdio(mcpServer)
	case "sse":
		addr := fmt.Sprintf("127.0.0.1:%d", clawMCPPort)
		fmt.Fprintf(os.Stderr, "nClaw MCP server listening on %s (SSE)\n", addr)
		sseServer := server.NewSSEServer(mcpServer)
		return sseServer.Start(addr)
	default:
		return fmt.Errorf("unknown transport %q (use stdio or sse)", clawMCPTransport)
	}
}

// mcpDoRequest is a helper that makes an authenticated request to the claw API.
func mcpDoRequest(ctx context.Context, method, urlStr, apiKey string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func mcpChatHandler(baseURL, apiKey string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		message, _ := args["message"].(string)
		if message == "" {
			return mcp.NewToolResultText("Error: message is required"), nil
		}

		body := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": message},
			},
			"stream": false,
		}
		if model, ok := args["model"].(string); ok && model != "" {
			body["model"] = model
		}
		metadata := map[string]interface{}{}
		if topic, ok := args["topic"].(string); ok && topic != "" {
			metadata["topic"] = topic
		}
		if len(metadata) > 0 {
			body["metadata"] = metadata
		}

		respBody, err := mcpDoRequest(ctx, "POST", baseURL+"/claw/v1/chat/completions", apiKey, body)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}

		var result struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error parsing response: %v", err)), nil
		}

		if len(result.Choices) > 0 {
			return mcp.NewToolResultText(result.Choices[0].Message.Content), nil
		}
		return mcp.NewToolResultText("No response received"), nil
	}
}

func mcpSearchMemoriesHandler(baseURL, apiKey string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		query, _ := args["query"].(string)
		if query == "" {
			return mcp.NewToolResultText("Error: query is required"), nil
		}

		respBody, err := mcpDoRequest(ctx, "GET", baseURL+"/claw/memories/search?q="+url.QueryEscape(query), apiKey, nil)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}

		return mcp.NewToolResultText(string(respBody)), nil
	}
}

func mcpSearchTopicsHandler(baseURL, apiKey string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		query, _ := args["query"].(string)
		if query == "" {
			return mcp.NewToolResultText("Error: query is required"), nil
		}

		respBody, err := mcpDoRequest(ctx, "GET", baseURL+"/claw/topics/search?q="+url.QueryEscape(query), apiKey, nil)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}

		return mcp.NewToolResultText(string(respBody)), nil
	}
}

func mcpListTopicsHandler(baseURL, apiKey string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		respBody, err := mcpDoRequest(ctx, "GET", baseURL+"/claw/topics", apiKey, nil)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}

		return mcp.NewToolResultText(string(respBody)), nil
	}
}

func mcpCreateMemoryHandler(baseURL, apiKey string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		content, _ := args["content"].(string)
		if content == "" {
			return mcp.NewToolResultText("Error: content is required"), nil
		}

		body := map[string]string{"content": content}
		if memType, ok := args["type"].(string); ok && memType != "" {
			body["type"] = memType
		}

		respBody, err := mcpDoRequest(ctx, "POST", baseURL+"/claw/memories", apiKey, body)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
		}

		return mcp.NewToolResultText(string(respBody)), nil
	}
}
