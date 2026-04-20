package doctor

import (
	"context"
	"strings"
	"testing"
)

// TestPoolCapacityCheck_Pass verifies that a correctly-configured pool passes.
func TestPoolCapacityCheck_Pass(t *testing.T) {
	// 4 services × 10 conns = 40 < 100 max_connections.
	pools := []PoolConfig{
		{ServiceName: "postgres", MaxConns: 10},
		{ServiceName: "hasura", MaxConns: 10},
		{ServiceName: "auth", MaxConns: 10},
		{ServiceName: "storage", MaxConns: 10},
	}

	result := PoolCapacityCheck(context.Background(), pools, 100)

	if result.Status != "pass" {
		t.Errorf("expected pass, got %s: %s", result.Status, result.Message)
	}
	if result.Section != "performance" {
		t.Errorf("expected section 'performance', got %q", result.Section)
	}
}

// TestPoolCapacityCheck_Warn verifies that an over-configured pool triggers WARN.
// 23 services × 10 conns = 230 > 100 max_connections.
func TestPoolCapacityCheck_Warn(t *testing.T) {
	pools := make([]PoolConfig, 23)
	for i := range pools {
		pools[i] = PoolConfig{ServiceName: "svc", MaxConns: 10}
	}

	result := PoolCapacityCheck(context.Background(), pools, 100)

	if result.Status != "warn" {
		t.Errorf("expected warn for 23-service over-configured pool, got %s: %s",
			result.Status, result.Message)
	}
	if !strings.Contains(result.Message, "exceeds postgres_max_connections") {
		t.Errorf("expected message to mention exceeds, got: %s", result.Message)
	}
	if result.FixCmd == "" {
		t.Error("expected FixCmd to be non-empty on warn")
	}
}

// TestPoolCapacityCheck_InvalidMaxConnections verifies invalid max_connections fails.
func TestPoolCapacityCheck_InvalidMaxConnections(t *testing.T) {
	result := PoolCapacityCheck(context.Background(), []PoolConfig{{ServiceName: "svc", MaxConns: 5}}, 0)

	if result.Status != "fail" {
		t.Errorf("expected fail for max_connections=0, got %s", result.Status)
	}
}

// TestPoolCapacityCheck_NoServices verifies that zero services passes.
func TestPoolCapacityCheck_NoServices(t *testing.T) {
	result := PoolCapacityCheck(context.Background(), nil, 100)

	if result.Status != "pass" {
		t.Errorf("expected pass for no services, got %s: %s", result.Status, result.Message)
	}
}

// TestPoolCapacityCheck_DefaultMaxConns verifies that MaxConns=0 defaults to pgxpoolDefaultMaxConns.
func TestPoolCapacityCheck_DefaultMaxConns(t *testing.T) {
	// 12 services with explicit MaxConns=0 → defaults to 10 each = 120 > 100.
	pools := make([]PoolConfig, 12)
	for i := range pools {
		pools[i] = PoolConfig{ServiceName: "svc", MaxConns: 0}
	}

	result := PoolCapacityCheck(context.Background(), pools, 100)

	// 12 × 10 = 120 > 100 → warn.
	if result.Status != "warn" {
		t.Errorf("expected warn when zero MaxConns defaults to 10, got %s: %s",
			result.Status, result.Message)
	}
}

// TestRecommendedPoolSize_Standard verifies the standard case (100 max / 23 services).
func TestRecommendedPoolSize_Standard(t *testing.T) {
	got := RecommendedPoolSize(100, 23)
	// floor(100 / 23) = floor(4.34) = 4
	if got != 4 {
		t.Errorf("expected 4, got %d", got)
	}
}

// TestRecommendedPoolSize_LargeMax verifies cap at pgxpoolDefaultMaxConns=10.
func TestRecommendedPoolSize_LargeMax(t *testing.T) {
	got := RecommendedPoolSize(10000, 10)
	// floor(10000 / 10) = 1000 → capped at 10.
	if got != 10 {
		t.Errorf("expected cap at 10, got %d", got)
	}
}

// TestRecommendedPoolSize_ZeroServices verifies that zero services returns default.
func TestRecommendedPoolSize_ZeroServices(t *testing.T) {
	got := RecommendedPoolSize(100, 0)
	if got != pgxpoolDefaultMaxConns {
		t.Errorf("expected %d for zero services, got %d", pgxpoolDefaultMaxConns, got)
	}
}

// TestRecommendedPoolSize_ExceedsMaxConnections verifies minimum=1 when services > max.
func TestRecommendedPoolSize_ExceedsMaxConnections(t *testing.T) {
	// 5 max_connections / 10 services → floor = 0 → returns 1.
	got := RecommendedPoolSize(5, 10)
	if got != 1 {
		t.Errorf("expected 1 when services exceed max_connections, got %d", got)
	}
}

// TestPoolCapCheckID verifies the check ID constant matches docs reference.
func TestPoolCapCheckID(t *testing.T) {
	if PoolCapCheckID != "PERF-POOL-01" {
		t.Errorf("expected PERF-POOL-01, got %s", PoolCapCheckID)
	}
}
