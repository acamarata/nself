package doctor

// nself_audit_scan.go — NSCAN-001..010 dispatch via the nself-audit plugin.
//
// `nself doctor --deep` calls the nself-audit plugin's POST /scan/run endpoint
// to execute every baseline misconfiguration scan rule. The plugin is part of
// the ɳSentry bundle but the scan endpoint is Security-Always-Free — no
// license check, no DB access required.
//
// Graceful degradation:
//   - If the plugin is not running on its configured port → emit a single
//     "skipped" CheckResult, never a "fail".
//   - If the plugin returns a parse error → emit "warn" with the error message.
//   - Each finding is mapped to one CheckResult so users can grep by NSCAN-ID.
//
// Wired into DeepChecks via CheckNSelfAuditScan(...). See deep.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	nselfAuditSection      = "security"
	nselfAuditCheckIDBase  = "NSCAN-SCAN"
	defaultNSelfAuditPort  = "3843"
	defaultNSelfAuditHTTPT = 5 * time.Second
)

// nselfAuditScanFinding mirrors the plugin's scan_api.go scanResponse shape.
// We redeclare it locally to avoid importing the plugin (which has its own
// go.mod under plugins-pro/).
type nselfAuditScanFinding struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
	Evidence    string `json:"evidence"`
}

type nselfAuditScanSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Total    int `json:"total"`
}

type nselfAuditScanResponse struct {
	Findings []nselfAuditScanFinding `json:"findings"`
	Summary  nselfAuditScanSummary   `json:"summary"`
}

type nselfAuditScanRequest struct {
	EnvFilePath       string `json:"env_file_path,omitempty"`
	NginxConfigDir    string `json:"nginx_config_dir,omitempty"`
	DockerComposePath string `json:"docker_compose_path,omitempty"`
}

// CheckNSelfAuditScan dispatches the NSCAN-001..010 rule set to the nself-audit
// plugin and returns the per-finding CheckResults.
//
// projectDir is used to compute the env file and docker-compose path. When the
// plugin is not reachable, returns a single "skipped" pass-result rather than
// a fail (Security-Always-Free contract — the scan is an enhancement, not a
// blocker for projects that don't have the plugin installed).
func CheckNSelfAuditScan(ctx context.Context, projectDir string) []CheckResult {
	port := os.Getenv("NSELF_AUDIT_PORT")
	if port == "" {
		port = defaultNSelfAuditPort
	}
	url := fmt.Sprintf("http://127.0.0.1:%s/scan/run", port)

	// Resolve probable file locations.
	req := nselfAuditScanRequest{
		EnvFilePath:       firstExistingFile(projectDir, ".env.prod", ".env.staging", ".env.dev", ".env.local", ".env"),
		NginxConfigDir:    firstExistingDir(projectDir, "nginx/conf.d", "nginx/sites", "nginx"),
		DockerComposePath: firstExistingFile(projectDir, "docker-compose.yml", "docker-compose.yaml"),
	}

	body, err := json.Marshal(req)
	if err != nil {
		// Marshalling a struct of strings shouldn't fail; if it does we treat
		// the whole step as warn (not fail) and continue doctor.
		return []CheckResult{{
			Section: nselfAuditSection,
			Name:    nselfAuditCheckIDBase,
			Status:  "warn",
			Message: fmt.Sprintf("NSCAN-SCAN: build request: %v", err),
		}}
	}

	reqCtx, cancel := context.WithTimeout(ctx, defaultNSelfAuditHTTPT)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return []CheckResult{nselfAuditSkipped(fmt.Sprintf("build HTTP request: %v", err))}
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: defaultNSelfAuditHTTPT}
	resp, err := client.Do(httpReq)
	if err != nil {
		// Plugin not running — graceful skip. Most users running doctor on a
		// fresh project will hit this; it's not an error.
		return []CheckResult{nselfAuditSkipped(fmt.Sprintf("nself-audit plugin not reachable on :%s (run `nself plugin install nself-audit`)", port))}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return []CheckResult{{
			Section: nselfAuditSection,
			Name:    nselfAuditCheckIDBase,
			Status:  "warn",
			Message: fmt.Sprintf("NSCAN-SCAN: plugin returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw))),
		}}
	}

	var decoded nselfAuditScanResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return []CheckResult{{
			Section: nselfAuditSection,
			Name:    nselfAuditCheckIDBase,
			Status:  "warn",
			Message: fmt.Sprintf("NSCAN-SCAN: decode response: %v", err),
		}}
	}

	return scanFindingsToCheckResults(decoded)
}

// nselfAuditSkipped builds a single informational CheckResult indicating the
// scan was skipped (plugin not installed / not reachable). Status is "pass"
// rather than "warn" so doctor's overall exit code is unaffected — the scan
// is opt-in via the bundle.
func nselfAuditSkipped(reason string) CheckResult {
	return CheckResult{
		Section: nselfAuditSection,
		Name:    nselfAuditCheckIDBase,
		Status:  "pass",
		Message: "NSCAN-SCAN: skipped — " + reason,
	}
}

// scanFindingsToCheckResults maps plugin findings into the doctor schema.
// One CheckResult per finding so users can identify by NSCAN-NNN. When there
// are zero findings we emit a single "pass" result so the section is visible.
func scanFindingsToCheckResults(resp nselfAuditScanResponse) []CheckResult {
	if len(resp.Findings) == 0 {
		return []CheckResult{{
			Section: nselfAuditSection,
			Name:    nselfAuditCheckIDBase,
			Status:  "pass",
			Message: "NSCAN-SCAN: 0 findings (NSCAN-001..010 clean)",
		}}
	}
	results := make([]CheckResult, 0, len(resp.Findings)+1)
	results = append(results, CheckResult{
		Section: nselfAuditSection,
		Name:    nselfAuditCheckIDBase,
		Status:  scanStatusFromSummary(resp.Summary),
		Message: fmt.Sprintf("NSCAN-SCAN: %d findings (critical=%d high=%d medium=%d low=%d)",
			resp.Summary.Total, resp.Summary.Critical, resp.Summary.High,
			resp.Summary.Medium, resp.Summary.Low),
	})
	for _, f := range resp.Findings {
		results = append(results, CheckResult{
			Section: nselfAuditSection,
			Name:    f.RuleID,
			Status:  scanStatusFromSeverity(f.Severity),
			Message: fmt.Sprintf("%s: %s — %s", f.RuleID, f.Title, f.Description),
			FixCmd:  f.Remediation,
		})
	}
	return results
}

// scanStatusFromSeverity maps NSCAN severity → CheckResult.Status.
//   - critical|high → fail (forces exit 1 per Security-Always-Free doctrine)
//   - medium        → warn
//   - low|unknown   → warn
func scanStatusFromSeverity(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical", "high":
		return "fail"
	case "medium", "low":
		return "warn"
	default:
		return "warn"
	}
}

// scanStatusFromSummary produces the header-row status: fail if any
// critical/high finding, warn if any medium/low, pass otherwise.
func scanStatusFromSummary(s nselfAuditScanSummary) string {
	if s.Critical > 0 || s.High > 0 {
		return "fail"
	}
	if s.Medium > 0 || s.Low > 0 {
		return "warn"
	}
	return "pass"
}

// firstExistingFile returns the first path under projectDir that exists, or "".
func firstExistingFile(projectDir string, names ...string) string {
	for _, n := range names {
		p := filepath.Join(projectDir, n)
		info, err := os.Stat(p)
		if err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// firstExistingDir returns the first directory under projectDir that exists, or "".
func firstExistingDir(projectDir string, names ...string) string {
	for _, n := range names {
		p := filepath.Join(projectDir, n)
		info, err := os.Stat(p)
		if err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}
