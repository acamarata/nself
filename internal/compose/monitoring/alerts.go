// Package monitoring — alert rule and Alertmanager config generators.
//
// This file owns the generation of:
//   - otel-alerts.yml  — Prometheus alert rules for OTEL Collector + Tempo
//   - alertmanager.yml — Alertmanager routing and receiver configuration
//
// Both files are written into the target project's monitoring/ directory by
// nself build when MONITORING_ENABLED=true and MONITORING_TRACING_ENABLED=true.
//
// On-call integration is controlled by NSELF_ONCALL_PROVIDER (pagerduty,
// opsgenie, or email). Set the corresponding key env var:
//
//	NSELF_ONCALL_PROVIDER=pagerduty  ALERTMANAGER_PD_ROUTING_KEY=<key>
//	NSELF_ONCALL_PROVIDER=opsgenie   ALERTMANAGER_OG_API_KEY=<key>
//	NSELF_ONCALL_PROVIDER=email      ALERTMANAGER_ONCALL_EMAIL=<addr>
//
// When NSELF_ONCALL_PROVIDER is unset or empty the receiver falls back to
// email using ALERTMANAGER_ONCALL_EMAIL (default: support@nself.org).
package monitoring

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
)

// OncallProvider identifies which on-call integration backend to use.
type OncallProvider string

const (
	// OncallPagerDuty routes critical alerts via PagerDuty Events API v2.
	OncallPagerDuty OncallProvider = "pagerduty"
	// OncallOpsgenie routes critical alerts via Opsgenie Alerts API.
	OncallOpsgenie OncallProvider = "opsgenie"
	// OncallEmail routes critical alerts via SMTP. This is the default when
	// NSELF_ONCALL_PROVIDER is unset or not recognised.
	OncallEmail OncallProvider = "email"
)

// AlertReceiver defines an Alertmanager notification receiver.
type AlertReceiver struct {
	// Name is the receiver identifier referenced by routing rules.
	Name string
	// Provider selects the integration backend for this receiver.
	// One of OncallPagerDuty, OncallOpsgenie, OncallEmail.
	Provider OncallProvider
	// EmailTo is the address used when Provider is OncallEmail.
	// Reads ALERTMANAGER_ONCALL_EMAIL at runtime when empty.
	EmailTo string
	// PDRoutingKey is the PagerDuty Events API v2 routing key used when
	// Provider is OncallPagerDuty. Reads ALERTMANAGER_PD_ROUTING_KEY.
	PDRoutingKey string
	// OGAPIKey is the Opsgenie API key used when Provider is OncallOpsgenie.
	// Reads ALERTMANAGER_OG_API_KEY.
	OGAPIKey string
}

// AlertmanagerConfig captures everything needed to render alertmanager.yml.
type AlertmanagerConfig struct {
	// SMTPHost is the SMTP relay host:port for email notifications.
	SMTPHost string
	// SMTPFrom is the envelope-from address.
	SMTPFrom string
	// Receivers is the list of notification receivers.
	Receivers []AlertReceiver
	// GroupWait is how long Alertmanager buffers alerts before sending the
	// first notification. Default: 30s.
	GroupWait string
	// GroupInterval is the interval between notifications for the same group.
	// Default: 5m.
	GroupInterval string
	// RepeatInterval is the interval before re-sending a resolved or ongoing
	// alert. Default: 4h.
	RepeatInterval string
	// MaintenanceWindowStart is the start time of the weekly maintenance window
	// in HH:MM format (24h, UTC). Used to silence non-critical alerts during
	// planned maintenance. Default: "02:00" (2 AM UTC Saturday).
	MaintenanceWindowStart string
	// MaintenanceWindowEnd is the end time of the maintenance window.
	// Default: "04:00" (4 AM UTC Saturday).
	MaintenanceWindowEnd string
	// MaintenanceWindowDay is the day of week for the maintenance window.
	// Default: "saturday".
	MaintenanceWindowDay string
}

// DefaultAlertmanagerConfig returns nSelf's out-of-the-box Alertmanager
// settings. The on-call receiver is selected from NSELF_ONCALL_PROVIDER:
//
//   - "pagerduty" — PagerDuty Events API v2 via ALERTMANAGER_PD_ROUTING_KEY
//   - "opsgenie"  — Opsgenie Alerts API via ALERTMANAGER_OG_API_KEY
//   - "email" / "" (default) — SMTP via ALERTMANAGER_ONCALL_EMAIL
//
// Warning and info severity routes go to the null receiver (logged, no
// notification). Maintenance window: Saturday 02:00–04:00 UTC.
func DefaultAlertmanagerConfig() *AlertmanagerConfig {
	provider := OncallProvider(os.Getenv("NSELF_ONCALL_PROVIDER"))
	switch provider {
	case OncallPagerDuty, OncallOpsgenie, OncallEmail:
		// valid
	default:
		provider = OncallEmail
	}

	oncall := AlertReceiver{
		Name:     "oncall",
		Provider: provider,
	}
	switch provider {
	case OncallPagerDuty:
		oncall.PDRoutingKey = "${ALERTMANAGER_PD_ROUTING_KEY}"
	case OncallOpsgenie:
		oncall.OGAPIKey = "${ALERTMANAGER_OG_API_KEY}"
	default:
		oncall.EmailTo = "${ALERTMANAGER_ONCALL_EMAIL:-support@nself.org}"
	}

	return &AlertmanagerConfig{
		SMTPHost:               "${ALERTMANAGER_SMTP_HOST:-localhost:25}",
		SMTPFrom:               "${ALERTMANAGER_SMTP_FROM:-alertmanager@nself.org}",
		GroupWait:              "30s",
		GroupInterval:          "5m",
		RepeatInterval:         "4h",
		MaintenanceWindowStart: "02:00",
		MaintenanceWindowEnd:   "04:00",
		MaintenanceWindowDay:   "saturday",
		Receivers: []AlertReceiver{
			oncall,
			{Name: "null-receiver"},
		},
	}
}

const alertmanagerTmpl = `# Generated by nself build — DO NOT HAND EDIT
# Alertmanager configuration for the nSelf monitoring stack.
#
# Routing tree (severity-based):
#   critical → oncall (provider: {{(index .Receivers 0).Provider}})
#   warning  → null-receiver (logged, no notification)
#   info     → null-receiver (logged, no notification)
#   default  → null-receiver (catch-all)
#
# Inhibit rules:
#   1. Critical suppresses warnings for the same job (prevents alert storms).
#   2. Service-level critical suppresses instance-level warnings (parent-down inhibit).
#
# Maintenance window: non-critical alerts are silenced on {{.MaintenanceWindowDay}}s
# {{.MaintenanceWindowStart}}-{{.MaintenanceWindowEnd}} UTC (weekly planned maintenance).
#
# On-call integration: set NSELF_ONCALL_PROVIDER to one of:
#   pagerduty — requires ALERTMANAGER_PD_ROUTING_KEY
#   opsgenie  — requires ALERTMANAGER_OG_API_KEY
#   email     — requires ALERTMANAGER_ONCALL_EMAIL (default)
# See cli/monitoring/runbooks/ for per-alert runbooks.
#
# Environment variables:
#   NSELF_ONCALL_PROVIDER       — on-call backend (pagerduty|opsgenie|email)
#   ALERTMANAGER_SMTP_HOST      — SMTP relay host:port (default: localhost:25)
#   ALERTMANAGER_SMTP_FROM      — envelope-from address (default: alertmanager@nself.org)
#   ALERTMANAGER_ONCALL_EMAIL   — email destination when provider=email (default: support@nself.org)
#   ALERTMANAGER_PD_ROUTING_KEY — PagerDuty Events API v2 routing key
#   ALERTMANAGER_OG_API_KEY     — Opsgenie API key

global:
  smtp_smarthost: {{.SMTPHost}}
  smtp_from: {{.SMTPFrom}}
  smtp_require_tls: true

# Time intervals define windows when non-critical alerts are suppressed.
# The maintenance window silences warning/info alerts during planned downtime.
time_intervals:
  - name: maintenance-window
    time_intervals:
      - weekdays: ['{{.MaintenanceWindowDay}}']
        times:
          - start_time: '{{.MaintenanceWindowStart}}'
            end_time: '{{.MaintenanceWindowEnd}}'

route:
  receiver: null-receiver
  group_by: [alertname, job]
  group_wait: {{.GroupWait}}
  group_interval: {{.GroupInterval}}
  repeat_interval: {{.RepeatInterval}}
  routes:
    # Critical: page the on-call receiver immediately; never muted.
    - matchers:
        - severity = critical
      receiver: oncall
      continue: false

    # Warning: log only; silence during the maintenance window.
    - matchers:
        - severity = warning
      receiver: null-receiver
      active_time_intervals: []
      mute_time_intervals:
        - maintenance-window
      continue: false

    # Info: always ignore — informational events are not actionable.
    - matchers:
        - severity = info
      receiver: null-receiver
      continue: false

receivers:
{{- range .Receivers}}
  - name: {{.Name}}
{{- if eq .Provider "pagerduty"}}
    pagerduty_configs:
      - routing_key: {{.PDRoutingKey}}
        severity: '{{ "{{" }} .Labels.severity {{ "}}" }}'
        send_resolved: true
        description: >-
          [{{ "{{" }} .Labels.alertname {{ "}}" }}] {{ "{{" }} .Annotations.summary {{ "}}" }}
        details:
          job: '{{ "{{" }} .Labels.job {{ "}}" }}'
          nself_project: '{{ "{{" }} .Labels.nself_project {{ "}}" }}'
{{- else if eq .Provider "opsgenie"}}
    opsgenie_configs:
      - api_key: {{.OGAPIKey}}
        message: >-
          [{{ "{{" }} .Labels.alertname {{ "}}" }}] {{ "{{" }} .Annotations.summary {{ "}}" }}
        tags: 'nself,{{ "{{" }} .Labels.severity {{ "}}" }},{{ "{{" }} .Labels.job {{ "}}" }}'
        priority: >-
          {{ "{{" }} if eq .Labels.severity "critical" {{ "}}" }}P1{{ "{{" }} else {{ "}}" }}P3{{ "{{" }} end {{ "}}" }}
        send_resolved: true
{{- else if .EmailTo}}
    email_configs:
      - to: {{.EmailTo}}
        send_resolved: true
{{- end}}
{{- end}}

inhibit_rules:
  # Rule 1: Silence all warnings when a critical alert is already firing for the
  # same job. Prevents alert storms when a service is completely down.
  - source_matchers:
      - severity = critical
    target_matchers:
      - severity = warning
    equal: [job]

  # Rule 2: Parent-service-down inhibit. When a core service (Postgres, Hasura,
  # Auth) fires a critical alert, suppress downstream plugin/app warnings that
  # are consequences of the same outage rather than independent faults.
  # The 'service_group' label must be set in alert rules for this to match.
  # Populate service_group labels in plugin alert rules to activate this rule.
  - source_matchers:
      - severity = critical
      - service_group = core
    target_matchers:
      - severity =~ "warning|info"
      - service_group = plugin
    equal: [nself_project]
`

// RenderAlertmanagerYAML returns the rendered alertmanager.yml bytes for cfg.
func RenderAlertmanagerYAML(cfg *AlertmanagerConfig) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("monitoring: nil AlertmanagerConfig")
	}
	tmpl, err := template.New("alertmanager").Parse(alertmanagerTmpl)
	if err != nil {
		return nil, fmt.Errorf("monitoring: parse alertmanager template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("monitoring: execute alertmanager template: %w", err)
	}
	return buf.Bytes(), nil
}
