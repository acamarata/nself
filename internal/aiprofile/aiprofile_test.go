package aiprofile

import (
	"testing"
	"time"
)

// TestParseProfile_Valid verifies all documented profile strings parse correctly.
func TestParseProfile_Valid(t *testing.T) {
	cases := []struct {
		input string
		want  Profile
	}{
		{"auto", ProfileAuto},
		{"", ProfileAuto},
		{"local_only", ProfileLocalOnly},
		{"pool_only", ProfilePoolOnly},
		{"oauth_only", ProfileOAuthOnly},
		{"custom", ProfileCustom},
		// Case insensitive
		{"AUTO", ProfileAuto},
		{"LOCAL_ONLY", ProfileLocalOnly},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseProfile(tc.input)
			if err != nil {
				t.Fatalf("ParseProfile(%q): %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseProfile(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestParseProfile_Invalid verifies that unknown profile strings return an error.
func TestParseProfile_Invalid(t *testing.T) {
	invalid := []string{"unknown", "free", "pro", "1", "all"}
	for _, s := range invalid {
		t.Run(s, func(t *testing.T) {
			_, err := ParseProfile(s)
			if err == nil {
				t.Errorf("ParseProfile(%q): expected error, got nil", s)
			}
		})
	}
}

// TestBudgetConfig_CheckBudget verifies daily and monthly cap enforcement.
func TestBudgetConfig_CheckBudget(t *testing.T) {
	cases := []struct {
		name         string
		cfg          BudgetConfig
		spentDaily   float64
		spentMonthly float64
		wantErr      bool
	}{
		{
			name:         "within budget",
			cfg:          BudgetConfig{DailyUSD: 5.0, MonthlyUSD: 50.0},
			spentDaily:   4.99,
			spentMonthly: 49.99,
			wantErr:      false,
		},
		{
			name:         "daily cap hit",
			cfg:          BudgetConfig{DailyUSD: 5.0, MonthlyUSD: 50.0},
			spentDaily:   5.00,
			spentMonthly: 10.00,
			wantErr:      true,
		},
		{
			name:         "monthly cap hit",
			cfg:          BudgetConfig{DailyUSD: 5.0, MonthlyUSD: 50.0},
			spentDaily:   1.00,
			spentMonthly: 50.00,
			wantErr:      true,
		},
		{
			name:         "zero budget (free tier only) — no cap enforced here",
			cfg:          BudgetConfig{DailyUSD: 0, MonthlyUSD: 0},
			spentDaily:   999,
			spentMonthly: 999,
			wantErr:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.CheckBudget(tc.spentDaily, tc.spentMonthly)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestBudgetConfig_IsPaidTierBlocked verifies zero-budget implies free-tier-only.
func TestBudgetConfig_IsPaidTierBlocked(t *testing.T) {
	blocked := BudgetConfig{DailyUSD: 0, MonthlyUSD: 0}
	if !blocked.IsPaidTierBlocked() {
		t.Error("zero budget should block paid tier")
	}
	notBlocked := BudgetConfig{DailyUSD: 5.0, MonthlyUSD: 50.0}
	if notBlocked.IsPaidTierBlocked() {
		t.Error("positive budget should not block paid tier")
	}
}

// TestLoadFeatureFlags_Defaults verifies all feature flags default to true when
// env vars are unset.
func TestLoadFeatureFlags_Defaults(t *testing.T) {
	// Ensure vars are unset.
	for _, k := range []string{
		"AI_FEATURE_AUTO_PROVISION",
		"AI_FEATURE_LOCAL_AUTO_INSTALL",
		"AI_FEATURE_ROUTING_UI",
		"AI_FEATURE_HEALTH_DAEMON",
	} {
		t.Setenv(k, "")
	}
	flags := LoadFeatureFlags()
	if !flags.AutoProvision {
		t.Error("AutoProvision should default to true")
	}
	if !flags.LocalAutoInstall {
		t.Error("LocalAutoInstall should default to true")
	}
	if !flags.RoutingUI {
		t.Error("RoutingUI should default to true")
	}
	if !flags.HealthDaemon {
		t.Error("HealthDaemon should default to true")
	}
}

// TestLoadFeatureFlags_ExplicitFalse verifies feature flags can be disabled.
func TestLoadFeatureFlags_ExplicitFalse(t *testing.T) {
	t.Setenv("AI_FEATURE_AUTO_PROVISION", "false")
	t.Setenv("AI_FEATURE_LOCAL_AUTO_INSTALL", "0")
	flags := LoadFeatureFlags()
	if flags.AutoProvision {
		t.Error("AutoProvision should be false when env=false")
	}
	if flags.LocalAutoInstall {
		t.Error("LocalAutoInstall should be false when env=0")
	}
}

// TestTokenBucket_Allow verifies rate limiting behaviour.
func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket()

	// First 15 requests (burst) should be allowed.
	for i := 0; i < defaultBurst; i++ {
		if !tb.Allow(1) {
			t.Errorf("request %d should be allowed within burst", i+1)
		}
	}
	// After burst exhaustion, next request should be denied.
	if tb.Allow(1) {
		t.Error("request after burst exhaustion should be denied")
	}
}

// TestTierTimeouts_TimeoutForTier verifies zero-duration for unset tiers.
func TestTierTimeouts_TimeoutForTier(t *testing.T) {
	for _, k := range []string{
		"AI_TIMEOUT_LOCAL_MS",
		"AI_TIMEOUT_OAUTH_MS",
		"AI_TIMEOUT_POOL_MS",
		"AI_TIMEOUT_PAID_MS",
	} {
		t.Setenv(k, "")
	}
	tt := LoadTierTimeouts()
	if tt.Local != 0 {
		t.Errorf("Local timeout should be 0 when unset, got %v", tt.Local)
	}
	if tt.TimeoutForTier(TierLocal) != 0 {
		t.Errorf("TimeoutForTier(local) should be 0, got %v", tt.TimeoutForTier(TierLocal))
	}
}

// TestTierTimeouts_ExplicitValues verifies that env vars are read correctly.
func TestTierTimeouts_ExplicitValues(t *testing.T) {
	t.Setenv("AI_TIMEOUT_LOCAL_MS", "5000")
	t.Setenv("AI_TIMEOUT_PAID_MS", "30000")
	tt := LoadTierTimeouts()
	if tt.Local != 5*time.Second {
		t.Errorf("Local timeout: got %v, want 5s", tt.Local)
	}
	if tt.Paid != 30*time.Second {
		t.Errorf("Paid timeout: got %v, want 30s", tt.Paid)
	}
}
