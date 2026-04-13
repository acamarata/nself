package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"github.com/spf13/cobra"
)

var clawStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show nClaw server status and health",
	Long: `Check the health of your nClaw server connection.

Shows server connectivity, active model, and counts for
memories, topics, and sessions. Green checkmarks indicate
healthy checks, red X marks indicate failures.`,
	RunE: runClawStatus,
}

func runClawStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("nClaw Status")
	fmt.Println("------------")
	fmt.Println()

	// Check if configured
	apiKey := clawAPIKey()
	serverURL := clawServerURL()

	if serverURL == "" {
		fmt.Println("  Server:  not configured")
		fmt.Println()
		fmt.Println("  Run: nself claw config set server <url>")
		return nil
	}
	fmt.Printf("  Server:  %s\n", serverURL)

	if apiKey == "" {
		fmt.Println("  API Key: not configured")
		fmt.Println()
		fmt.Println("  Run: nself claw config set api-key <key>")
		return nil
	}
	masked := apiKey
	if len(masked) > 8 {
		masked = masked[:4] + "..." + masked[len(masked)-4:]
	}
	fmt.Printf("  API Key: %s\n", masked)
	fmt.Println()

	client, baseURL, err := clawClient()
	if err != nil {
		fmt.Printf("  Connection: \033[31m✗\033[0m %v\n", err)
		return nil
	}

	// Health check
	req, err := http.NewRequestWithContext(cmd.Context(), "GET", baseURL+"/claw/health", nil)
	if err != nil {
		fmt.Printf("  Health:     \033[31m✗\033[0m could not create request\n")
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  Connection: \033[31m✗\033[0m unreachable (%v)\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("  Health:     \033[31m✗\033[0m HTTP %d: %s\n", resp.StatusCode, string(body))
		return nil
	}

	var health struct {
		Status       string `json:"status"`
		Model        string `json:"model"`
		MemoryCount  int    `json:"memory_count"`
		TopicCount   int    `json:"topic_count"`
		SessionCount int    `json:"session_count"`
		Version      string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		fmt.Printf("  Health:     \033[31m✗\033[0m could not parse response\n")
		return nil
	}

	if health.Status == "ok" || health.Status == "healthy" {
		fmt.Printf("  Health:     \033[32m✓\033[0m %s\n", health.Status)
	} else {
		fmt.Printf("  Health:     \033[31m✗\033[0m %s\n", health.Status)
	}

	if health.Model != "" {
		fmt.Printf("  Model:      %s\n", health.Model)
	}
	if health.Version != "" {
		fmt.Printf("  Version:    %s\n", health.Version)
	}

	fmt.Println()
	fmt.Printf("  Memories:   %d\n", health.MemoryCount)
	fmt.Printf("  Topics:     %d\n", health.TopicCount)
	fmt.Printf("  Sessions:   %d\n", health.SessionCount)

	return nil
}
