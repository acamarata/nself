package commands

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var clawExportFormat string

var clawExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all nClaw data",
	Long: `Export all your nClaw data (messages, memories, topics, knowledge, config).

Outputs to stdout so you can pipe to a file:
  nself claw export > my-data.json
  nself claw export --format csv > my-data.csv

Formats:
  json   Single JSON object with all sections (default)
  csv    Multiple CSV sections with headers`,
	RunE: runClawExport,
}

func init() {
	clawExportCmd.Flags().StringVar(&clawExportFormat, "format", "json", "Output format: json or csv")
}

type clawExportData struct {
	Messages  []map[string]interface{} `json:"messages"`
	Memories  []map[string]interface{} `json:"memories"`
	Topics    []map[string]interface{} `json:"topics"`
	Knowledge []map[string]interface{} `json:"knowledge"`
	Config    map[string]interface{}   `json:"config"`
}

func runClawExport(cmd *cobra.Command, args []string) error {
	if clawExportFormat != "json" && clawExportFormat != "csv" {
		return fmt.Errorf("unknown format %q (use json or csv)", clawExportFormat)
	}

	client, baseURL, err := clawClient()
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	data := &clawExportData{}

	// Fetch all data sections
	endpoints := []struct {
		path  string
		key   string
		dest  *[]map[string]interface{}
	}{
		{"/claw/v1/messages", "messages", &data.Messages},
		{"/claw/memories", "memories", &data.Memories},
		{"/claw/topics", "topics", &data.Topics},
		{"/claw/v1/knowledge", "knowledge", &data.Knowledge},
	}

	for _, ep := range endpoints {
		items, err := fetchExportSection(cmd, client, baseURL+ep.path, ep.key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not fetch %s: %v\n", ep.key, err)
			continue
		}
		*ep.dest = items
	}

	// Fetch config separately (not an array)
	configData, err := fetchExportRaw(cmd, client, baseURL+"/claw/v1/config")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch config: %v\n", err)
	} else {
		var cfg map[string]interface{}
		if json.Unmarshal(configData, &cfg) == nil {
			data.Config = cfg
		}
	}

	switch clawExportFormat {
	case "json":
		return exportJSON(data)
	case "csv":
		return exportCSV(data)
	}

	return nil
}

func fetchExportSection(cmd *cobra.Command, client *http.Client, urlStr, key string) ([]map[string]interface{}, error) {
	var allItems []map[string]interface{}
	limit := 100
	offset := 0

	for {
		sep := "?"
		if strings.Contains(urlStr, "?") {
			sep = "&"
		}
		pageURL := fmt.Sprintf("%s%slimit=%d&offset=%d", urlStr, sep, limit, offset)

		req, err := http.NewRequestWithContext(cmd.Context(), "GET", pageURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var pageItems []map[string]interface{}

		// Try to extract array from key
		var wrapper map[string]json.RawMessage
		if err := json.Unmarshal(body, &wrapper); err == nil {
			if raw, ok := wrapper[key]; ok {
				json.Unmarshal(raw, &pageItems)
			}
		}

		// Try as direct array if key extraction failed
		if pageItems == nil {
			json.Unmarshal(body, &pageItems)
		}

		if len(pageItems) == 0 {
			break
		}

		allItems = append(allItems, pageItems...)
		if len(pageItems) < limit {
			break
		}
		offset += limit
	}

	return allItems, nil
}

func fetchExportRaw(cmd *cobra.Command, client *http.Client, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(cmd.Context(), "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func exportJSON(data *clawExportData) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func exportCSV(data *clawExportData) error {
	w := csv.NewWriter(os.Stdout)

	sections := []struct {
		name string
		data []map[string]interface{}
	}{
		{"messages", data.Messages},
		{"memories", data.Memories},
		{"topics", data.Topics},
		{"knowledge", data.Knowledge},
	}

	for _, section := range sections {
		if len(section.data) == 0 {
			continue
		}

		// Section header
		w.Write([]string{"# " + section.name})

		// Collect all keys
		keySet := map[string]bool{}
		for _, item := range section.data {
			for k := range item {
				keySet[k] = true
			}
		}
		var keys []string
		for k := range keySet {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Header row
		w.Write(keys)

		// Data rows
		for _, item := range section.data {
			row := make([]string, len(keys))
			for i, k := range keys {
				if v, ok := item[k]; ok {
					row[i] = fmt.Sprintf("%v", v)
				}
			}
			w.Write(row)
		}

		// Blank line between sections
		w.Write([]string{})
	}

	w.Flush()
	return w.Error()
}
