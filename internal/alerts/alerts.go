// Package alerts manages Prometheus alert rules and Alertmanager routing for nSelf.
package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Severity levels for alert rules.
const (
	SeverityP1 = "P1"
	SeverityP2 = "P2"
	SeverityP3 = "P3"
)

// AlertRule describes a single Prometheus alert rule.
type AlertRule struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Expr     string `json:"expr"`
	For      string `json:"for"`
	Summary  string `json:"summary"`
}

// Silence represents an active alert silence.
type Silence struct {
	ID        string    `json:"id"`
	AlertName string    `json:"alert_name"`
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// DefaultRules returns the 16 default nSelf alert rules.
func DefaultRules() []AlertRule {
	return []AlertRule{
		{Name: "ServiceDown", Severity: SeverityP1, For: "1m", Expr: `up == 0`, Summary: "Service {{ $labels.job }} is down"},
		{Name: "DiskHigh", Severity: SeverityP2, For: "5m", Expr: `(node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100 < 15`, Summary: "Disk usage above 85%"},
		{Name: "DiskCritical", Severity: SeverityP1, For: "1m", Expr: `(node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100 < 5`, Summary: "Disk usage above 95%"},
		{Name: "MemoryHigh", Severity: SeverityP2, For: "5m", Expr: `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 85`, Summary: "Memory usage above 85%"},
		{Name: "CPUHigh", Severity: SeverityP3, For: "15m", Expr: `100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 85`, Summary: "CPU usage above 85% for 15m"},
		{Name: "ErrorRateSpike", Severity: SeverityP2, For: "5m", Expr: `rate(nginx_http_requests_total{status=~"5.."}[5m]) / rate(nginx_http_requests_total[5m]) > 0.05`, Summary: "Error rate above 5%"},
		{Name: "SlowQuery", Severity: SeverityP3, For: "10m", Expr: `pg_stat_activity_max_tx_duration{datname!~"template.*"} > 300`, Summary: "Postgres query running longer than 5 minutes"},
		{Name: "BackupFailed", Severity: SeverityP1, For: "0m", Expr: `nself_backup_last_success_timestamp_seconds == 0 or time() - nself_backup_last_success_timestamp_seconds > 93600`, Summary: "Last backup older than 26 hours or never ran"},
		{Name: "BackupVerifyFailed", Severity: SeverityP1, For: "0m", Expr: `nself_backup_verify_last_success_timestamp_seconds == 0 or time() - nself_backup_verify_last_success_timestamp_seconds > 691200`, Summary: "Last backup verify older than 8 days"},
		{Name: "CertExpiring", Severity: SeverityP2, For: "0m", Expr: `(probe_ssl_earliest_cert_expiry - time()) / 86400 < 30`, Summary: "SSL certificate expires in less than 30 days"},
		{Name: "CertExpiringCritical", Severity: SeverityP1, For: "0m", Expr: `(probe_ssl_earliest_cert_expiry - time()) / 86400 < 7`, Summary: "SSL certificate expires in less than 7 days"},
		{Name: "LicenseExpiring", Severity: SeverityP2, For: "0m", Expr: `nself_license_days_remaining < 30`, Summary: "License expires in less than 30 days"},
		{Name: "LicenseGraceHard", Severity: SeverityP1, For: "0m", Expr: `nself_license_days_remaining < 0`, Summary: "License has expired"},
		{Name: "HasuraEventLag", Severity: SeverityP2, For: "5m", Expr: `nself_hasura_event_queue_length > 1000`, Summary: "Hasura event queue lag above 1000"},
		{Name: "PgReplicationLag", Severity: SeverityP2, For: "5m", Expr: `pg_replication_lag > 300`, Summary: "Postgres replication lag above 5 minutes"},
		{Name: "ContainerRestartLoop", Severity: SeverityP2, For: "10m", Expr: `increase(container_restart_count[10m]) > 3`, Summary: "Container {{ $labels.name }} restarting repeatedly"},
	}
}

// List returns alert rules, optionally filtered by severity.
func List(severity string) []AlertRule {
	rules := DefaultRules()
	if severity == "" {
		return rules
	}
	sev := strings.ToUpper(severity)
	var filtered []AlertRule
	for _, r := range rules {
		if r.Severity == sev {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// CreateSilence creates a silence for the given alert. It stores it locally
// and attempts to create it in Alertmanager if available.
func CreateSilence(ctx context.Context, alertName, reason string, duration time.Duration, workdir string) (*Silence, error) {
	s := &Silence{
		ID:        fmt.Sprintf("silence-%d", time.Now().UnixNano()),
		AlertName: alertName,
		Reason:    reason,
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
	}

	// Store locally
	silenceDir := filepath.Join(workdir, ".nself", "silences")
	if err := os.MkdirAll(silenceDir, 0o755); err != nil {
		return nil, fmt.Errorf("create silence dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal silence: %w", err)
	}

	path := filepath.Join(silenceDir, s.ID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("write silence: %w", err)
	}

	// Best-effort: post to Alertmanager via amtool if available
	amtool, lookErr := exec.LookPath("amtool")
	if lookErr == nil {
		cmd := exec.CommandContext(ctx, amtool, "silence", "add",
			"--alertname="+alertName,
			"--comment="+reason,
			"--duration="+duration.String(),
		)
		_ = cmd.Run() // best-effort
	}

	return s, nil
}

// TestAlert fires a synthetic test alert by sending to Alertmanager's API.
func TestAlert(ctx context.Context, alertName string, alertmanagerURL string) error {
	if alertmanagerURL == "" {
		alertmanagerURL = "http://127.0.0.1:9093"
	}

	payload := fmt.Sprintf(`[{"labels":{"alertname":"%s","severity":"test","source":"nself-cli"},"annotations":{"summary":"Test alert from nself alerts test"}}]`, alertName)

	cmd := exec.CommandContext(ctx, "curl", "-sf", "-X", "POST",
		alertmanagerURL+"/api/v2/alerts",
		"-H", "Content-Type: application/json",
		"-d", payload,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("post test alert: %w: %s", err, string(out))
	}
	return nil
}
