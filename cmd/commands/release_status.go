package commands

// Purpose: Implements `nself release status`: checks each deployment
// artifact (CLI GitHub release, admin Docker image, Homebrew formula, ping
// API, web/org, Vercel) against the latest tagged version and reports which
// surfaces are current. Split out of release.go (CLI-R12); this file is
// pure artifact-status checking, distinct from the release/rollback
// cascades that live in release.go, release_deploy_steps.go, and
// release_rollback.go.
// Inputs: the releaseStatusCmd cobra.Command (registered onto releaseCmd in
// release.go's init — untouched by this split) and a context for the
// outbound HTTP/exec calls each checker makes.
// Outputs: artifactStatus values and the printed/JSON status report.
// Constraints: pure move — no behavior changes. releaseStatusCmd itself
// stays a var in release.go; only its RunE target's implementation and its
// checker helpers move here.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

type artifactStatus struct {
	Artifact string `json:"artifact"`
	Running  string `json:"running"`
	Latest   string `json:"latest"`
	Status   string `json:"status"` // "fresh", "stale", "unknown"
}

func runReleaseStatus(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	// Fetch latest GitHub release
	latest := fetchLatestGitHubRelease(ctx, "nself-org/cli")

	statuses := []artifactStatus{
		checkArtifactCLI(ctx, latest),
		checkArtifactAdmin(ctx, latest),
		checkArtifactHomebrew(ctx, latest),
		checkArtifactPingAPI(ctx, latest),
		checkArtifactWebOrg(ctx, latest),
		checkArtifactVercel(ctx, latest),
	}

	out := cmd.OutOrStdout()

	if jsonOut {
		data, err := json.Marshal(map[string]interface{}{
			"latest":    latest,
			"artifacts": statuses,
			"checked":   time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}
		fmt.Fprintln(out, string(data))
		return nil
	}

	fmt.Fprintf(out, "%s Release Status  (latest: %s)\n\n", ui.C(ui.Bold, "nSelf"), ui.C(ui.Cyan, latest))
	fmt.Fprintf(out, "  %-16s %-12s %-12s %s\n", "Artifact", "Running", "Latest", "Status")
	fmt.Fprintf(out, "  %s\n", strings.Repeat("─", 56))
	for _, s := range statuses {
		statusStr := s.Status
		color := ui.Green
		switch s.Status {
		case "stale":
			color = ui.Yellow
		case "unknown":
			color = ui.Dim
		}
		fmt.Fprintf(out, "  %-16s %-12s %-12s %s\n", s.Artifact, s.Running, s.Latest, ui.C(color, statusStr))
	}
	return nil
}

func fetchLatestGitHubRelease(ctx context.Context, repo string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "unknown"
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "unknown"
	}
	defer resp.Body.Close()
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "unknown"
	}
	return strings.TrimPrefix(release.TagName, "v")
}

func checkArtifactCLI(_ context.Context, latest string) artifactStatus {
	running := strings.TrimPrefix(strings.TrimPrefix(os.Getenv("NSELF_VERSION"), "v"), "")
	if running == "" {
		// Try reading version.go indirectly via go binary if available
		out, err := exec.Command("nself", "version", "--short").Output()
		if err == nil {
			running = strings.TrimSpace(string(out))
		}
	}
	return makeArtifactStatus("cli", running, latest)
}

func checkArtifactAdmin(ctx context.Context, latest string) artifactStatus {
	// curl Docker Hub manifest API
	url := "https://registry.hub.docker.com/v2/repositories/nself/nself-admin/tags/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return artifactStatus{Artifact: "admin", Status: "unknown"}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return artifactStatus{Artifact: "admin", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	defer resp.Body.Close()
	var tag struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tag); err != nil {
		return makeArtifactStatus("admin", "unknown", latest)
	}
	running := tag.Name
	if running == "latest" || running == "" {
		running = "unknown"
	}
	return makeArtifactStatus("admin", running, latest)
}

func checkArtifactHomebrew(ctx context.Context, latest string) artifactStatus {
	out, err := exec.CommandContext(ctx, "brew", "info", "--json=v1", "nself-org/nself/nself").Output()
	if err != nil {
		return artifactStatus{Artifact: "homebrew", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	var info []struct {
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(out, &info); err != nil || len(info) == 0 {
		return artifactStatus{Artifact: "homebrew", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	return makeArtifactStatus("homebrew", info[0].Versions.Stable, latest)
}

func checkArtifactPingAPI(ctx context.Context, latest string) artifactStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ping.nself.org/version", nil)
	if err != nil {
		return artifactStatus{Artifact: "ping_api", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return artifactStatus{Artifact: "ping_api", Running: "unreachable", Latest: latest, Status: "unknown"}
	}
	defer resp.Body.Close()
	var v struct {
		LatestCLIVersion string `json:"latestCliVersion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return artifactStatus{Artifact: "ping_api", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	return makeArtifactStatus("ping_api", strings.TrimPrefix(v.LatestCLIVersion, "v"), latest)
}

func checkArtifactWebOrg(ctx context.Context, latest string) artifactStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://nself.org", nil)
	if err != nil {
		return artifactStatus{Artifact: "web/org", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	req.Header.Set("User-Agent", "nself-release-check/1.0")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return artifactStatus{Artifact: "web/org", Running: "unreachable", Latest: latest, Status: "unknown"}
	}
	defer resp.Body.Close()
	running := strings.TrimPrefix(resp.Header.Get("X-Nself-Version"), "v")
	if running == "" {
		running = "unknown"
	}
	return makeArtifactStatus("web/org", running, latest)
}

func checkArtifactVercel(ctx context.Context, latest string) artifactStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://nself.org", nil)
	if err != nil {
		return artifactStatus{Artifact: "vercel", Running: "unknown", Latest: latest, Status: "unknown"}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return artifactStatus{Artifact: "vercel", Running: "unreachable", Latest: latest, Status: "unknown"}
	}
	defer resp.Body.Close()
	running := strings.TrimPrefix(resp.Header.Get("X-Nself-Version"), "v")
	if running == "" {
		running = "unknown"
	}
	return makeArtifactStatus("vercel", running, latest)
}

func makeArtifactStatus(name, running, latest string) artifactStatus {
	s := artifactStatus{Artifact: name, Running: running, Latest: latest}
	if running == "" || running == "unknown" || running == "unreachable" {
		s.Status = "unknown"
	} else if running == latest {
		s.Status = "fresh"
	} else {
		s.Status = "stale"
	}
	return s
}

// ── release-rollback ──────────────────────────────────────────────────────────
