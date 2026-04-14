// Package tenant provides multi-tenancy management: RLS policy generation,
// tenant lifecycle (create/upgrade/suspend/destroy), usage metering, billing
// integration, and audit logging.
package tenant

import "time"

// Plan represents a tenant billing plan tier.
type Plan string

const (
	PlanBasic        Plan = "basic"
	PlanPro          Plan = "pro"
	PlanElite        Plan = "elite"
	PlanBusiness     Plan = "business"
	PlanBusinessPlus Plan = "business-plus"
	PlanEnterprise   Plan = "enterprise"
)

// ValidPlans lists all accepted plan identifiers.
var ValidPlans = []Plan{
	PlanBasic, PlanPro, PlanElite, PlanBusiness, PlanBusinessPlus, PlanEnterprise,
}

// IsValidPlan reports whether p is a recognized plan name.
func IsValidPlan(p string) bool {
	for _, v := range ValidPlans {
		if string(v) == p {
			return true
		}
	}
	return false
}

// Tenant represents a row in the tenants table.
type Tenant struct {
	ID               string    `json:"id"`
	Slug             string    `json:"slug"`
	Plan             Plan      `json:"plan"`
	Status           string    `json:"status"` // active, suspended, destroyed
	StripeCustomerID string    `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	SuspendedAt      *time.Time `json:"suspended_at,omitempty"`
	SuspendReason    string    `json:"suspend_reason,omitempty"`
	DestroyedAt      *time.Time `json:"destroyed_at,omitempty"`
}

// AuditEntry represents a row in nself_ops.audit_log.
type AuditEntry struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	UserID        string    `json:"user_id"`
	Action        string    `json:"action"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    string    `json:"resource_id"`
	IP            string    `json:"ip"`
	UserAgent     string    `json:"user_agent"`
	PayloadSHA256 string    `json:"payload_sha256"`
	Reason        string    `json:"reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// UsageRecord represents a row in nself_ops.usage_daily.
type UsageRecord struct {
	TenantID string  `json:"tenant_id"`
	Day      string  `json:"day"` // YYYY-MM-DD
	Metric   string  `json:"metric"`
	Value    float64 `json:"value"`
}

// StripeOutboxEntry represents a row in nself_ops.stripe_outbox.
type StripeOutboxEntry struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Payload     string     `json:"payload"` // JSON
	CreatedAt   time.Time  `json:"created_at"`
	Attempts    int        `json:"attempts"`
	LastError   string     `json:"last_error,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// Metric constants for usage_daily.
const (
	MetricRowsStored     = "rows_stored"
	MetricStorageBytes   = "storage_bytes"
	MetricAITokensInput  = "ai_tokens_input"
	MetricAITokensOutput = "ai_tokens_output"
	MetricAICostUSD      = "ai_cost_usd"
	MetricBandwidthBytes = "bandwidth_bytes"
	MetricRequestsCount  = "requests_count"
	MetricActiveUsers    = "active_users"
)

// HasuraRole constants for multi-tenancy role model.
const (
	RoleAnonymous    = "anonymous"
	RoleUser         = "user"
	RoleTenantMember = "tenant_member"
	RoleTenantAdmin  = "tenant_admin"
	RoleAdmin        = "admin"
	RoleService      = "service"
)

// AllRoles lists all Hasura roles in permission inheritance order.
var AllRoles = []string{
	RoleAnonymous, RoleUser, RoleTenantMember, RoleTenantAdmin, RoleAdmin, RoleService,
}
