package sentryapi

// types.go — typed request/response shapes for the ɳSentry cloud API.
//
// Purpose: Single source of the CLI↔API wire contract (v1) shared by the
//   `nself sentry` cloud subcommands and the MCP sentry tools.
// Inputs:  JSON bodies from api.sentry.nself.org (or a self-hosted bundle).
// Outputs: Go structs used for table + --json output.
// Constraints: Field names are the contract — W1 (tenantization/API) builds
//   the server to THIS shape. Change only with a matching plugins-pro change.
// SPORT: CLI-PKG-SENTRYAPI-001

// Monitor is an uptime monitor owned by the authenticated tenant.
type Monitor struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	URL             string `json:"url"`
	Kind            string `json:"kind"` // http | tcp | ping
	IntervalSeconds int    `json:"interval_seconds"`
	Status          string `json:"status"` // up | down | paused | pending
	Paused          bool   `json:"paused"`
	CreatedAt       string `json:"created_at"`
}

// Incident is a monitoring incident (open → acknowledged → resolved).
type Incident struct {
	ID             string `json:"id"`
	MonitorID      string `json:"monitor_id"`
	Title          string `json:"title"`
	Status         string `json:"status"` // open | acknowledged | resolved
	Severity       string `json:"severity"`
	StartedAt      string `json:"started_at"`
	AcknowledgedAt string `json:"acknowledged_at,omitempty"`
	ResolvedAt     string `json:"resolved_at,omitempty"`
}

// StatusPage is a public status page owned by the tenant.
type StatusPage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	URL       string `json:"url"`
	Public    bool   `json:"public"`
	CreatedAt string `json:"created_at"`
}

// AlertChannel is a notification target (email/webhook/slack/telegram).
type AlertChannel struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"` // email | webhook | slack | telegram
	Target  string `json:"target"`
	Enabled bool   `json:"enabled"`
}

// QuotaUsage is used/limit for a single quota dimension.
type QuotaUsage struct {
	Used  int64 `json:"used"`
	Limit int64 `json:"limit"`
}

// Account is the response of GET /v1/me — identity, tier, and quota usage.
type Account struct {
	TenantID string                `json:"tenant_id"`
	Email    string                `json:"email"`
	Tier     string                `json:"tier"` // free | bundle | nself-plus
	Quotas   map[string]QuotaUsage `json:"quotas"`
}

// CreateMonitorRequest is the body of POST /v1/monitors.
type CreateMonitorRequest struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	Kind            string `json:"kind"`
	IntervalSeconds int    `json:"interval_seconds"`
}

// CreateStatusPageRequest is the body of POST /v1/status-pages.
type CreateStatusPageRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// TestChannelResult is the response of POST /v1/alerts/channels/{id}/test.
type TestChannelResult struct {
	Delivered bool   `json:"delivered"`
	Detail    string `json:"detail"`
}

// apiError is the wire error envelope: {"error":{"code":"...","message":"..."}}.
type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Wrapper envelopes — every list/detail response is a named-key object so the
// contract can grow (pagination, meta) without breaking clients.
type monitorsEnvelope struct {
	Monitors []Monitor `json:"monitors"`
}
type monitorEnvelope struct {
	Monitor Monitor `json:"monitor"`
}
type incidentsEnvelope struct {
	Incidents []Incident `json:"incidents"`
}
type incidentEnvelope struct {
	Incident Incident `json:"incident"`
}
type statusPagesEnvelope struct {
	StatusPages []StatusPage `json:"status_pages"`
}
type statusPageEnvelope struct {
	StatusPage StatusPage `json:"status_page"`
}
type channelsEnvelope struct {
	Channels []AlertChannel `json:"channels"`
}
