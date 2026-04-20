package doctor

// pool_capacity.go — PERF-POOL-01: pgxpool connection cap check.
//
// S51-T04 + S51-T-PERF-01 (Yellow Dim amendment):
//   MaxConns = min(10, floor(postgres_max_connections / active_service_count))
//
// Default max_connections=100, 23 services × 10 conns = 230 > 100 → exhaustion.
// This check warns when the total configured pool size exceeds the Postgres hard limit.

import (
	"context"
	"fmt"
	"math"
)

const (
	// DefaultPostgresMaxConnections is the Postgres default max_connections setting.
	// Operators should raise this in POSTGRES_MAX_CONNECTIONS when running many services.
	DefaultPostgresMaxConnections = 100

	// PoolCapCheckID is the canonical check ID referenced in docs and Alertmanager.
	PoolCapCheckID = "PERF-POOL-01"

	// pgxpoolDefaultMaxConns is the default pgxpool.Config MaxConns before nself adjustment.
	// Go pgxpool defaults to 4 on low-CPU machines but many plugins set 10.
	pgxpoolDefaultMaxConns = 10
)

// PoolConfig describes the pool sizing input for a single service.
// Each plugin or required service exposes one pool config.
type PoolConfig struct {
	ServiceName string
	MaxConns    int // per-service pgxpool MaxConns (0 = use pgxpoolDefaultMaxConns)
}

// PoolCapacityCheck verifies that the total configured pgxpool connections
// across all active services do not exceed postgres_max_connections.
//
// Returns a CheckResult with:
//   - Status "pass" when total_conns ≤ postgres_max_connections
//   - Status "warn" when total_conns > postgres_max_connections (not a hard block)
//   - Status "fail" when inputs are invalid (zero services, negative max)
//
// This check fires in nself doctor --deep as PERF-POOL-01.
func PoolCapacityCheck(_ context.Context, pools []PoolConfig, postgresMaxConnections int) CheckResult {
	result := CheckResult{
		Section: "performance",
		Name:    fmt.Sprintf("%s: pool sizing vs max_connections", PoolCapCheckID),
	}

	if postgresMaxConnections <= 0 {
		result.Status = "fail"
		result.Message = "postgres_max_connections must be > 0"
		return result
	}

	if len(pools) == 0 {
		result.Status = "pass"
		result.Message = "no active services — pool sizing OK"
		return result
	}

	total := 0
	for _, p := range pools {
		mc := p.MaxConns
		if mc <= 0 {
			mc = pgxpoolDefaultMaxConns
		}
		total += mc
	}

	recommended := RecommendedPoolSize(postgresMaxConnections, len(pools))

	if total > postgresMaxConnections {
		result.Status = "warn"
		result.Message = fmt.Sprintf(
			"total pool capacity %d exceeds postgres_max_connections=%d (%d services × ~%d conns avg). "+
				"Recommended per-service cap: %d. Raise POSTGRES_MAX_CONNECTIONS or reduce per-service pool size.",
			total, postgresMaxConnections, len(pools),
			total/len(pools), recommended,
		)
		result.FixCmd = fmt.Sprintf(
			"# In .env: set POSTGRES_MAX_CONNECTIONS=%d or reduce per-plugin pool size to %d",
			total+50, recommended,
		)
		return result
	}

	result.Status = "pass"
	result.Message = fmt.Sprintf(
		"pool capacity OK: total=%d, postgres_max_connections=%d, services=%d, per-service cap=%d",
		total, postgresMaxConnections, len(pools), recommended,
	)
	return result
}

// RecommendedPoolSize computes the recommended per-service MaxConns cap.
//
//	MaxConns = min(10, floor(postgres_max_connections / num_services))
//
// Edge cases:
//   - numServices=0 → returns 10 (no services, no constraint)
//   - very large max_connections → capped at 10 (prevents runaway allocations)
//   - numServices > max_connections → returns 1 (absolute minimum)
func RecommendedPoolSize(postgresMaxConnections, numServices int) int {
	if numServices <= 0 {
		return pgxpoolDefaultMaxConns
	}
	if postgresMaxConnections <= 0 {
		return 1
	}

	floor := int(math.Floor(float64(postgresMaxConnections) / float64(numServices)))
	if floor <= 0 {
		return 1
	}
	if floor > pgxpoolDefaultMaxConns {
		return pgxpoolDefaultMaxConns
	}
	return floor
}

// DefaultPoolConfigs returns a pool config slice for the standard nSelf
// service set (4 required + up to 7 optional). Pass only the enabled services.
// Used by nself doctor --deep to compute the PERF-POOL-01 check.
func DefaultPoolConfigs(services []string) []PoolConfig {
	configs := make([]PoolConfig, 0, len(services))
	for _, svc := range services {
		configs = append(configs, PoolConfig{
			ServiceName: svc,
			MaxConns:    pgxpoolDefaultMaxConns, // default; each plugin may override
		})
	}
	return configs
}
