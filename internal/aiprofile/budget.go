package aiprofile

import (
	"fmt"
	"os"
	"strconv"
)

// BudgetConfig holds the daily and monthly budget caps from .env.ai.
type BudgetConfig struct {
	DailyUSD   float64 // AI_DAILY_BUDGET_USD (0 = disabled / free tier only)
	MonthlyUSD float64 // AI_MONTHLY_BUDGET_USD (0 = disabled / free tier only)
}

// LoadBudget reads budget env vars. Zero means disabled (free tier only,
// paid tiers blocked). Positive value is a hard cap in USD.
func LoadBudget() BudgetConfig {
	return BudgetConfig{
		DailyUSD:   envFloat("AI_DAILY_BUDGET_USD", 0),
		MonthlyUSD: envFloat("AI_MONTHLY_BUDGET_USD", 0),
	}
}

// CheckBudget evaluates whether a request should be allowed given current spend.
// Returns nil if allowed, or an error with a user-facing message if blocked.
// spentDaily and spentMonthly are in USD (converted from cost_micros / 1e6).
func (b BudgetConfig) CheckBudget(spentDaily, spentMonthly float64) error {
	// Budget 0 = free tier only. Any positive spend means paid tier was used,
	// but we check at the router level. Here we enforce hard caps when budget > 0.
	if b.DailyUSD > 0 && spentDaily >= b.DailyUSD {
		return fmt.Errorf("daily paid spend hit $%.2f of $%.2f budget", spentDaily, b.DailyUSD)
	}
	if b.MonthlyUSD > 0 && spentMonthly >= b.MonthlyUSD {
		return fmt.Errorf("monthly paid spend hit $%.2f of $%.2f budget", spentMonthly, b.MonthlyUSD)
	}
	return nil
}

// IsPaidTierBlocked returns true when budget is zero (free tier only mode).
func (b BudgetConfig) IsPaidTierBlocked() bool {
	return b.DailyUSD == 0 && b.MonthlyUSD == 0
}

// ShouldAlert returns true when monthly spend has reached the budget threshold.
func (b BudgetConfig) ShouldAlert(spentMonthly float64) bool {
	if b.MonthlyUSD <= 0 {
		return false
	}
	return spentMonthly >= b.MonthlyUSD
}

func envFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
