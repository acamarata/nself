package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var clawTopicsCmd = &cobra.Command{
	Use:   "topics",
	Short: "List or search topics",
	Long: `Browse and search your nClaw conversation topics.

Without arguments, lists all topics in a table.
With a search argument, searches topics by name.

Examples:
  nself claw topics                  # list all topics
  nself claw topics search "travel"  # search for topics`,
	RunE: runClawTopicsList,
}

var clawTopicsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search topics by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runClawTopicsSearch,
}

func init() {
	clawTopicsCmd.AddCommand(clawTopicsSearchCmd)
}

type clawTopic struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	MessageCount int    `json:"message_count"`
	LastActiveAt string `json:"last_active_at"`
}

func runClawTopicsList(cmd *cobra.Command, args []string) error {
	client, baseURL, err := clawClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	req, err := http.NewRequestWithContext(cmd.Context(), "GET", baseURL+"/claw/topics", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Topics []clawTopic `json:"topics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Topics) == 0 {
		fmt.Println("No topics found.")
		return nil
	}

	printTopicsTable(result.Topics)
	return nil
}

func runClawTopicsSearch(cmd *cobra.Command, args []string) error {
	client, baseURL, err := clawClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	query := url.QueryEscape(args[0])
	req, err := http.NewRequestWithContext(cmd.Context(), "GET", baseURL+"/claw/topics/search?q="+query, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Topics []clawTopic `json:"topics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Topics) == 0 {
		fmt.Printf("No topics matching %q found.\n", args[0])
		return nil
	}

	printTopicsTable(result.Topics)
	return nil
}

func printTopicsTable(topics []clawTopic) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPATH\tMESSAGES\tLAST ACTIVE")
	fmt.Fprintln(w, "----\t----\t--------\t-----------")
	for _, t := range topics {
		name := t.Name
		if len(name) > 40 {
			name = name[:37] + "..."
		}
		path := t.Path
		if path == "" {
			path = "-"
		}
		lastActive := t.LastActiveAt
		if len(lastActive) > 19 {
			lastActive = lastActive[:19]
		}
		if lastActive == "" {
			lastActive = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", name, path, t.MessageCount, lastActive)
	}
	w.Flush()

	fmt.Printf("\n%d topic(s)\n", len(topics))
}

// truncate returns s truncated to max characters with ellipsis.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// trimLines removes leading/trailing whitespace and collapses newlines to spaces.
func trimLines(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
