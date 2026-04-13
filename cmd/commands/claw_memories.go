package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var clawMemoriesCmd = &cobra.Command{
	Use:   "memories",
	Short: "List or search memories",
	Long: `Browse and search your nClaw memories.

Without arguments, lists recent memories in a table.
With a search argument, searches memories by content.

Examples:
  nself claw memories                    # list recent memories
  nself claw memories search "Virginia"  # search memories`,
	RunE: runClawMemoriesList,
}

var clawMemoriesSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search memories by content",
	Args:  cobra.ExactArgs(1),
	RunE:  runClawMemoriesSearch,
}

func init() {
	clawMemoriesCmd.AddCommand(clawMemoriesSearchCmd)
}

type clawMemory struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

func runClawMemoriesList(cmd *cobra.Command, args []string) error {
	client, baseURL, err := clawClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	req, err := http.NewRequestWithContext(cmd.Context(), "GET", baseURL+"/claw/memories", nil)
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
		Memories []clawMemory `json:"memories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Memories) == 0 {
		fmt.Println("No memories found.")
		return nil
	}

	printMemoriesTable(result.Memories)
	return nil
}

func runClawMemoriesSearch(cmd *cobra.Command, args []string) error {
	client, baseURL, err := clawClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	query := url.QueryEscape(args[0])
	req, err := http.NewRequestWithContext(cmd.Context(), "GET", baseURL+"/claw/memories/search?q="+query, nil)
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
		Memories []clawMemory `json:"memories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Memories) == 0 {
		fmt.Printf("No memories matching %q found.\n", args[0])
		return nil
	}

	printMemoriesTable(result.Memories)
	return nil
}

func printMemoriesTable(memories []clawMemory) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CONTENT\tTYPE\tCREATED")
	fmt.Fprintln(w, "-------\t----\t-------")
	for _, m := range memories {
		content := truncate(trimLines(m.Content), 60)
		memType := m.Type
		if memType == "" {
			memType = "-"
		}
		created := m.CreatedAt
		if len(created) > 19 {
			created = created[:19]
		}
		if created == "" {
			created = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", content, memType, created)
	}
	w.Flush()

	fmt.Printf("\n%d memory(ies)\n", len(memories))
}
