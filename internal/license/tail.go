package license

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// TailOptions holds parsed flags for license tail.
type TailOptions struct {
	// Filters is a list of field=value pairs (e.g. "key=nself_pro_xxx", "result=denied").
	Filters map[string]string
	// PingURL is the ping_api base URL (defaults to NSELF_PING_URL or https://ping.nself.org).
	PingURL string
	// Stdout for output.
	Stdout io.Writer
	// HTTPClient allows injection in tests.
	HTTPClient *http.Client
}

// LicenseEvent is a single validation event from ping_api.
type LicenseEvent struct {
	// Timestamp of the event.
	Timestamp time.Time `json:"ts"`
	// KeyPrefix is the first 12 chars of the license key (never full key).
	KeyPrefix string `json:"key_prefix"`
	// Plugin is the plugin being validated.
	Plugin string `json:"plugin"`
	// Result is "allow", "deny", or "rate_limit".
	Result string `json:"result"`
	// IP is the requester IP.
	IP string `json:"ip,omitempty"`
}

// TailStream streams ping_api /license/validate events until ctx is canceled.
//
// Color coding:
//   - green  → allow
//   - red    → deny
//   - yellow → rate_limit
//
// Ctrl-C (context cancel) exits cleanly with no goroutine leak.
// OAuth tokens and full key values are never emitted — only KeyPrefix.
// Connection loss reconnects with exponential backoff.
func TailStream(ctx context.Context, opts TailOptions) error {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	pingURL := opts.PingURL
	if pingURL == "" {
		pingURL = os.Getenv("NSELF_PING_URL")
	}
	if pingURL == "" {
		pingURL = "https://ping.nself.org"
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 0} // streaming — no timeout
	}

	fmt.Fprintln(stdout, "Streaming license validation events (Ctrl-C to stop)...")

	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return nil // clean exit on cancel
		}

		err := streamOnce(ctx, client, pingURL, opts.Filters, stdout)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			fmt.Fprintf(stdout, "\033[33mreconnecting in %s (lost connection: %v)\033[0m\n", backoff, err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second // reset on clean stream end
	}
}

// streamOnce opens the SSE stream and reads events until the connection closes.
func streamOnce(ctx context.Context, client *http.Client, pingURL string, filters map[string]string, stdout io.Writer) error {
	url := pingURL + "/license/events"
	if len(filters) > 0 {
		var params []string
		for k, v := range filters {
			params = append(params, k+"="+v)
		}
		url += "?" + strings.Join(params, "&")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stream endpoint returned HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}

		var ev LicenseEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}
		if !matchesFilters(ev, filters) {
			continue
		}
		printEvent(stdout, ev)
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	return nil
}

// matchesFilters returns true if the event matches all filter criteria.
func matchesFilters(ev LicenseEvent, filters map[string]string) bool {
	for k, v := range filters {
		switch k {
		case "key":
			if !strings.HasPrefix(ev.KeyPrefix, v[:min12(v)]) {
				return false
			}
		case "result":
			if ev.Result != v {
				return false
			}
		case "plugin":
			if ev.Plugin != v {
				return false
			}
		}
	}
	return true
}

// min12 returns min(len(s), 12) for key prefix matching.
func min12(s string) int {
	if len(s) < 12 {
		return len(s)
	}
	return 12
}

// printEvent writes a color-coded event line to w.
// Green = allow, Red = deny, Yellow = rate_limit.
func printEvent(w io.Writer, ev LicenseEvent) {
	ts := ev.Timestamp.UTC().Format("15:04:05")
	if ev.Timestamp.IsZero() {
		ts = time.Now().UTC().Format("15:04:05")
	}

	var color, reset string
	// Only use color when output is a terminal.
	if isTerminal(w) {
		reset = "\033[0m"
		switch ev.Result {
		case "allow":
			color = "\033[32m" // green
		case "deny":
			color = "\033[31m" // red
		case "rate_limit":
			color = "\033[33m" // yellow
		}
	}

	fmt.Fprintf(w, "%s%s  %-10s  key=%-12s  plugin=%-20s%s\n",
		color,
		ts,
		strings.ToUpper(ev.Result),
		ev.KeyPrefix,
		ev.Plugin,
		reset,
	)
}

// isTerminal reports whether w is a *os.File pointing to a terminal.
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		if err == nil {
			return (fi.Mode() & os.ModeCharDevice) != 0
		}
	}
	return false
}
