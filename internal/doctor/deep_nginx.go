package doctor

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Purpose: nginx/TLS/reachability --deep checks — config test, certificate
// expiry, Let's Encrypt renewal cron, and the ping.nself.org health probe.
// Inputs: a context and verbose flag.
// Outputs: []CheckResult per category.
// Constraints: split out of deep.go (CLI-R12) as a pure move; no behavior
// changed.

// NginxChecks verifies config test and SSL cert expiry.
func NginxChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// nginx -t
	cmd := exec.CommandContext(ctx, "docker", "exec", "nself_nginx", "nginx", "-t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, CheckResult{Section: "nginx", Name: "Nginx config test", Status: "fail",
			Message: strings.TrimSpace(string(out))})
	} else {
		results = append(results, CheckResult{Section: "nginx", Name: "Nginx config test", Status: "pass", Message: "syntax ok"})
	}

	// SSL cert expiry >30d on all server_names
	cmd = exec.CommandContext(ctx, "docker", "exec", "nself_nginx", "find", "/etc/letsencrypt/live", "-name", "fullchain.pem")
	out, err = cmd.Output()
	if err == nil {
		for _, certPath := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if certPath == "" {
				continue
			}
			checkCmd := exec.CommandContext(ctx, "docker", "exec", "nself_nginx", "openssl", "x509",
				"-enddate", "-noout", "-in", certPath)
			certOut, err := checkCmd.Output()
			if err != nil {
				continue
			}
			// Parse "notAfter=..."
			dateStr := strings.TrimPrefix(strings.TrimSpace(string(certOut)), "notAfter=")
			expiry, err := time.Parse("Jan  2 15:04:05 2006 MST", dateStr)
			if err != nil {
				expiry, err = time.Parse("Jan 2 15:04:05 2006 MST", dateStr)
			}
			if err != nil {
				continue
			}
			domain := filepath.Base(filepath.Dir(certPath))
			daysLeft := int(time.Until(expiry).Hours() / 24)
			if daysLeft < 30 {
				results = append(results, CheckResult{Section: "nginx", Name: fmt.Sprintf("SSL expiry: %s", domain),
					Status: "warn", Message: fmt.Sprintf("%d days left (<30d)", daysLeft)})
			} else {
				results = append(results, CheckResult{Section: "nginx", Name: fmt.Sprintf("SSL expiry: %s", domain),
					Status: "pass", Message: fmt.Sprintf("%d days left", daysLeft)})
			}
		}
	}

	return results
}

// SSLChecks verifies LE renewal cron, last renewal, OCSP stapling.
func SSLChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	// LE renewal cron active
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "certbot.timer")
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "active" {
		results = append(results, CheckResult{Section: "ssl", Name: "Certbot timer", Status: "warn",
			Message: "certbot.timer not active", FixCmd: "sudo systemctl enable --now certbot.timer"})
	} else {
		results = append(results, CheckResult{Section: "ssl", Name: "Certbot timer", Status: "pass", Message: "active"})
	}

	// Last renewal <60d
	cmd = exec.CommandContext(ctx, "find", "/etc/letsencrypt/renewal", "-name", "*.conf", "-mtime", "-60")
	out, err = cmd.Output()
	if err == nil {
		files := strings.TrimSpace(string(out))
		if files == "" {
			results = append(results, CheckResult{Section: "ssl", Name: "Last renewal", Status: "warn",
				Message: "no renewals in past 60 days"})
		} else {
			count := len(strings.Split(files, "\n"))
			results = append(results, CheckResult{Section: "ssl", Name: "Last renewal", Status: "pass",
				Message: fmt.Sprintf("%d cert(s) renewed within 60d", count)})
		}
	}

	return results
}

// PingChecks verifies ping.nself.org reachable and license cache fresh.
func PingChecks(ctx context.Context, verbose bool) []CheckResult {
	var results []CheckResult

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://ping.nself.org/health")
	if err != nil {
		results = append(results, CheckResult{Section: "ping", Name: "ping.nself.org", Status: "warn",
			Message: fmt.Sprintf("unreachable: %v", err)})
	} else {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			results = append(results, CheckResult{Section: "ping", Name: "ping.nself.org", Status: "pass", Message: "reachable"})
		} else {
			results = append(results, CheckResult{Section: "ping", Name: "ping.nself.org", Status: "warn",
				Message: fmt.Sprintf("returned %d", resp.StatusCode)})
		}
	}

	return results
}
