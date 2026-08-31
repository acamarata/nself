package telemetry

// install.go — fire-and-forget install-counter event to ping.nself.org.
//
// Called by the install command after a successful `nself install` run.
// Posts to POST /install-counter/increment with only: cli_version, os, arch.
// No install_id, no paths, no PII.
//
// Failure is silently dropped — the install succeeds regardless.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/nself-org/cli/internal/version"
)

const installCounterEndpoint = "https://ping.nself.org/install-counter/increment"

// installCountPayload is the JSON body posted to the install counter endpoint.
// No PII: version, os, arch only.
type installCountPayload struct {
	CLIVersion string `json:"cli_version"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
}

// SendInstallEvent posts an anonymous "install completed" event to ping_api.
// It dispatches in a goroutine with a 4-second deadline and returns immediately.
// If telemetry is disabled (IsEnabled() == false), this is a no-op.
func SendInstallEvent() {
	if !IsEnabled() {
		return
	}

	p := installCountPayload{
		CLIVersion: version.GetVersion(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		data, err := json.Marshal(p)
		if err != nil {
			return
		}

		req, err := http.NewRequestWithContext(
			ctx, http.MethodPost, installCounterEndpoint, bytes.NewReader(data),
		)
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "nself-cli/install-counter")

		resp, err := httpClient.Do(req)
		if err != nil {
			if os.Getenv("NSELF_TELEMETRY_DEBUG") == "1" {
				fmt.Fprintf(os.Stderr, "[telemetry DEBUG] SendInstallEvent failed: %v\n", err)
			}
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if os.Getenv("NSELF_TELEMETRY_DEBUG") == "1" && resp.StatusCode >= 400 {
			fmt.Fprintf(os.Stderr, "[telemetry DEBUG] SendInstallEvent: server %d\n", resp.StatusCode)
		}
	}()
}
