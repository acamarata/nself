// Package telemetry — opt-in anonymous CLI telemetry client (S65.T01 / Q07).
//
// Two send paths exist for backward compatibility:
//   - Send(event, props) — legacy strict opt-in via NSELF_TELEMETRY_OPT_IN=1.
//   - SendEvent(eventType, metadata) — v1.1.0+ path, opt-out model via IsEnabled().
//
// Events POST to ping.nself.org/telemetry as JSON. Network failures are silently
// dropped (logged at DEBUG level). The CLI never blocks: each Send call dispatches
// a goroutine with a 3-second context deadline.
//
// Privacy invariant: payloads MUST NOT contain email addresses, file paths, project
// names, license keys, or any user-identifying string. Only anonymous IDs and
// categorical values are permitted.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/nself-org/cli/internal/version"
)

const (
	// pingAPIEndpoint is the telemetry ingestion endpoint.
	pingAPIEndpoint = "https://ping.nself.org/telemetry"

	// sendTimeout is the maximum time allowed for a single telemetry POST.
	// 3 seconds per Q07 spec (CLI must exit within 3.1 s when ping_api unreachable).
	sendTimeout = 3 * time.Second

	// installSourceFile is a one-shot file that carries the install source from
	// the install script into the first nself init invocation.
	installSourceFile = ".config/nself/install-source"
)

// payload is the JSON body sent to ping_api.
type payload struct {
	// Event is the event name (snake_case).
	Event string `json:"event"`

	// InstallSource is the ?ref= value from install.sh (nullable).
	InstallSource string `json:"install_source,omitempty"`

	// InstallMethod is how nself was installed (brew/curl/docker/other).
	InstallMethod string `json:"install_method,omitempty"`

	// AppContext is the Type-C app that initiated this install (nullable).
	AppContext string `json:"app_context,omitempty"`

	// Props holds event-specific key/value pairs. All values must be categorical
	// or numeric — never free-form user text or paths.
	Props map[string]any `json:"props,omitempty"`
}

// httpClient is a shared client with a conservative timeout for the goroutine dispatch.
// Initialized once; safe for concurrent use.
var httpClient = &http.Client{
	Timeout: sendTimeout * 2, // outer transport bound; context timeout is the real limit
}

// installSourceOnce guards the one-shot file read.
var installSourceOnce sync.Once
var cachedInstallSource string
var cachedInstallMethod string
var cachedAppContext string

// Send dispatches an anonymous telemetry event to ping.nself.org in a background
// goroutine. It returns immediately regardless of network conditions.
//
// Opt-in gate: if NSELF_TELEMETRY_OPT_IN != "1" this function is a complete no-op.
// Call site opt-in check: callers SHOULD check IsOptedIn() before building props to
// avoid unnecessary allocations, but Send itself is always safe to call without
// the guard — it will no-op internally.
func Send(event string, props map[string]any) {
	if !IsOptedIn() {
		return
	}

	// Capture install attribution fields once (reads one-shot file on first call).
	initInstallAttribution()

	p := payload{
		Event:         event,
		InstallSource: cachedInstallSource,
		InstallMethod: cachedInstallMethod,
		AppContext:    cachedAppContext,
		Props:         props,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "[telemetry] goroutine panic: %v\n", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		defer cancel()

		if err := sendPayload(ctx, p); err != nil {
			// Silent drop. Debug logging via stderr only in verbose builds.
			if os.Getenv("NSELF_TELEMETRY_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "[telemetry DEBUG] send failed: %v\n", err)
			}
		}
	}()
}

// IsOptedIn returns true only when NSELF_TELEMETRY_OPT_IN=1 is set.
// This is a strict opt-in: the absence of the variable means no telemetry.
func IsOptedIn() bool {
	return os.Getenv("NSELF_TELEMETRY_OPT_IN") == "1"
}

// sendPayload marshals p to JSON and POSTs it to pingAPIEndpoint.
func sendPayload(ctx context.Context, p payload) error {
	data, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pingAPIEndpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "nself-cli/telemetry")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server responded %d", resp.StatusCode)
	}
	return nil
}

// initInstallAttribution reads and caches install attribution from env vars and
// the one-shot install-source file. Called once via sync.Once.
func initInstallAttribution() {
	installSourceOnce.Do(func() {
		// AppContext: set by Type-C app .backend/init scripts.
		cachedAppContext = sanitizeInstallApp(os.Getenv("NSELF_INSTALL_APP"))

		// InstallMethod: set by install.sh or manually.
		cachedInstallMethod = sanitizeInstallMethod(os.Getenv("NSELF_INSTALL_METHOD"))

		// InstallSource: prefer env var, then one-shot file.
		src := os.Getenv("NSELF_INSTALL_SOURCE")
		if src == "" {
			src = readAndDeleteInstallSourceFile()
		}
		cachedInstallSource = sanitizeInstallSource(src)
	})
}

// readAndDeleteInstallSourceFile reads ~/.config/nself/install-source atomically:
// reads the value then deletes the file so it is only consumed once.
func readAndDeleteInstallSourceFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, installSourceFile)

	data, err := os.ReadFile(path)
	if err != nil {
		return "" // file absent is normal
	}

	// Delete the file immediately after read (one-shot semantics).
	_ = os.Remove(path)

	src := string(bytes.TrimSpace(data))
	return src
}

// sanitizeInstallSource allows only alphanumeric + hyphen + underscore, max 64 chars.
// Prevents injection of arbitrary strings into the telemetry payload.
func sanitizeInstallSource(s string) string {
	if s == "" {
		return ""
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s) && i < 64; i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			out = append(out, c)
		}
	}
	return string(out)
}

// sanitizeInstallApp allows only known app context values.
func sanitizeInstallApp(s string) string {
	allowed := map[string]bool{
		"nclaw": true, "nchat": true, "ntv": true,
		"ntask": true, "clawde": true, "nfamily": true,
	}
	if allowed[s] {
		return s
	}
	return ""
}

// sanitizeInstallMethod allows only known install method values.
func sanitizeInstallMethod(s string) string {
	allowed := map[string]bool{
		"brew": true, "curl": true, "docker": true, "other": true,
	}
	if allowed[s] {
		return s
	}
	return ""
}

// eventPayload is the JSON body for the v1.1.0+ telemetry endpoint (Q07).
// Fields map 1:1 to the np_telemetry_events table in ping_api.
type eventPayload struct {
	InstallID  string         `json:"install_id"`
	Event      string         `json:"event"`
	CLIVersion string         `json:"version"`
	OS         string         `json:"os"`
	Arch       string         `json:"arch"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// SendEvent dispatches one of the T01-T05 catalog events to ping.nself.org/telemetry
// using the v1.1.0+ opt-out model. It is a fire-and-forget goroutine with a 3-second
// hard deadline. Network failures are silently dropped.
//
// Gate: returns immediately without dispatching if IsEnabled() is false.
//
// Privacy: metadata values MUST be categorical or numeric only — never free-form
// user text, file paths, license keys, or any personally-identifying data.
func SendEvent(eventType string, metadata map[string]any) {
	if !IsEnabled() {
		return
	}

	installID := LoadOrCreateInstallID()
	p := eventPayload{
		InstallID:  installID,
		Event:      eventType,
		CLIVersion: version.GetVersion(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Metadata:   metadata,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "[telemetry] goroutine panic: %v\n", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		defer cancel()

		data, err := json.Marshal(p)
		if err != nil {
			return // never blocks
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, pingAPIEndpoint, bytes.NewReader(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "nself-cli/telemetry")

		resp, err := httpClient.Do(req)
		if err != nil {
			if os.Getenv("NSELF_TELEMETRY_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "[telemetry DEBUG] SendEvent %s failed: %v\n", eventType, err)
			}
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if os.Getenv("NSELF_TELEMETRY_DEBUG") == "1" && resp.StatusCode >= 400 {
			fmt.Fprintf(os.Stderr, "[telemetry DEBUG] SendEvent %s: server %d\n", eventType, resp.StatusCode)
		}
	}()
}
