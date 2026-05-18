package queue

import (
	"context"
	"testing"
	"time"
)

// TestIsKnownPGBossState_Known verifies all documented pg-boss states return true.
func TestIsKnownPGBossState_Known(t *testing.T) {
	known := []string{
		"created", "active", "completed", "failed", "expired", "cancelled", "retry",
	}
	for _, s := range known {
		if !isKnownPGBossState(s) {
			t.Errorf("isKnownPGBossState(%q): got false, want true", s)
		}
	}
}

// TestIsKnownPGBossState_Unknown verifies non-pg-boss states return false.
func TestIsKnownPGBossState_Unknown(t *testing.T) {
	unknown := []string{"pending", "dead", "running", "", "ACTIVE", "COMPLETED"}
	for _, s := range unknown {
		if isKnownPGBossState(s) {
			t.Errorf("isKnownPGBossState(%q): got true, want false", s)
		}
	}
}

// TestMapState_KnownMappings verifies the CLI state → pg-boss mapping table.
func TestMapState_KnownMappings(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"pending", "created"},
		{"PENDING", "created"}, // case-insensitive
		{"active", "active"},
		{"failed", "failed"},
		{"dead", "expired"},
		{"completed", "completed"},
		{"unknown_state", "unknown_state"}, // passthrough
		{"", ""},                           // passthrough empty
	}
	for _, tc := range tests {
		got := mapState(tc.input)
		if got != tc.want {
			t.Errorf("mapState(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestEscapeSQLString_QuoteSingleQuote verifies SQL injection prevention.
func TestEscapeSQLString_QuoteSingleQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with'quote", "with''quote"},
		{"it's", "it''s"},
		{"multi'ple'quotes", "multi''ple''quotes"},
		{"", ""},
		{"no-special-chars-123", "no-special-chars-123"},
	}
	for _, tc := range tests {
		got := escapeSQLString(tc.input)
		if got != tc.want {
			t.Errorf("escapeSQLString(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestPurgeQueue_InvalidQueueName verifies queue name validation fires before DB call.
func TestPurgeQueue_InvalidQueueName(t *testing.T) {
	ctx := context.Background()
	_, err := PurgeQueue(ctx, "postgres://no-such-host/db", "", time.Hour)
	if err == nil {
		t.Error("PurgeQueue(empty queue): expected error, got nil")
	}
	_, err = PurgeQueue(ctx, "postgres://no-such-host/db", "has space", time.Hour)
	if err == nil {
		t.Error("PurgeQueue(invalid name): expected error, got nil")
	}
}

// TestPurgeQueue_NegativeRetention verifies negative duration is rejected early.
func TestPurgeQueue_NegativeRetention(t *testing.T) {
	ctx := context.Background()
	_, err := PurgeQueue(ctx, "postgres://no-such-host/db", "my-queue", -time.Hour)
	if err == nil {
		t.Error("PurgeQueue(negative retention): expected error, got nil")
	}
}

// TestRFC3339Regex_Matches verifies the internal regex accepts canonical RFC3339.
func TestRFC3339Regex_Matches(t *testing.T) {
	valid := []string{
		"2026-01-15T12:00:00Z",
		"2026-05-17T08:30:00+05:30",
		"2000-01-01T00:00:00-07:00",
	}
	for _, ts := range valid {
		if !rfc3339Regex.MatchString(ts) {
			t.Errorf("rfc3339Regex.MatchString(%q) = false, want true", ts)
		}
	}
}

// TestRFC3339Regex_Rejects verifies the regex rejects non-RFC3339 strings.
func TestRFC3339Regex_Rejects(t *testing.T) {
	invalid := []string{
		"2026-01-15",
		"not-a-date",
		"2026-01-15T12:00:00",
		"",
	}
	for _, ts := range invalid {
		if rfc3339Regex.MatchString(ts) {
			t.Errorf("rfc3339Regex.MatchString(%q) = true, want false", ts)
		}
	}
}
