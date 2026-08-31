// Package cost implements per-service cost alerting for nSelf AI and media
// plugins. It runs hourly (via the cron plugin or nself doctor), queries
// np_ai_usage.cost_usd for the last 24h, and fires a Slack webhook when any
// configured budget is exceeded.
//
// Configuration env vars:
//
//	COST_ALERT_ENABLED       true | false (default false)
//	COST_BUDGET_AI_DAILY_USD float, e.g. "5.00"
//	COST_BUDGET_MEDIA_DAILY_USD float
//	COST_BUDGET_TOTAL_DAILY_USD float
//	SLACK_WEBHOOK_URL        https://hooks.slack.com/...
//	SLACK_ALERT_CHANNEL      #nself-alerts (informational only — embedded in message)
package cost

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Budget holds per-service daily spend limits in USD.
// A zero value means "no limit for this service".
type Budget struct {
	AIDailyUSD    float64
	MediaDailyUSD float64
	TotalDailyUSD float64
}

// LoadBudgetFromEnv reads COST_BUDGET_* vars and returns the parsed Budget.
// Malformed values are ignored (budget treated as unlimited).
func LoadBudgetFromEnv() Budget {
	return Budget{
		AIDailyUSD:    parseFloat(os.Getenv("COST_BUDGET_AI_DAILY_USD")),
		MediaDailyUSD: parseFloat(os.Getenv("COST_BUDGET_MEDIA_DAILY_USD")),
		TotalDailyUSD: parseFloat(os.Getenv("COST_BUDGET_TOTAL_DAILY_USD")),
	}
}

// Alerter checks cost against budgets and sends Slack notifications on breach.
type Alerter struct {
	DB         *sql.DB
	Budget     Budget
	HTTPClient *http.Client
	Log        *slog.Logger
}

// New creates an Alerter with defaults.
func New(db *sql.DB, budget Budget) *Alerter {
	return &Alerter{
		DB:         db,
		Budget:     budget,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Log:        slog.Default(),
	}
}

// Enabled returns true when COST_ALERT_ENABLED=true.
func Enabled() bool {
	return strings.EqualFold(os.Getenv("COST_ALERT_ENABLED"), "true")
}

// CheckAndAlert queries the last 24h of cost_usd from np_ai_usage, compares
// against configured budgets, and posts to Slack on breach.
// It is safe to call from a cron handler or periodic health check.
func (a *Alerter) CheckAndAlert(ctx context.Context) error {
	if !Enabled() {
		return nil
	}

	totals, err := a.queryDailyCosts(ctx)
	if err != nil {
		return fmt.Errorf("querying daily costs: %w", err)
	}

	var alerts []string

	if a.Budget.AIDailyUSD > 0 {
		if actual := totals["ai"]; actual > a.Budget.AIDailyUSD {
			alerts = append(alerts, a.formatAlert("ai plugin", a.Budget.AIDailyUSD, actual))
		}
	}
	if a.Budget.MediaDailyUSD > 0 {
		if actual := totals["media"]; actual > a.Budget.MediaDailyUSD {
			alerts = append(alerts, a.formatAlert("media-processing plugin", a.Budget.MediaDailyUSD, actual))
		}
	}
	if a.Budget.TotalDailyUSD > 0 {
		total := 0.0
		for _, v := range totals {
			total += v
		}
		if total > a.Budget.TotalDailyUSD {
			alerts = append(alerts, a.formatAlert("all plugins (combined)", a.Budget.TotalDailyUSD, total))
		}
	}

	for _, msg := range alerts {
		if err := a.postSlack(ctx, msg); err != nil {
			a.Log.Error("failed to post cost alert to Slack", "err", err)
		} else {
			a.Log.Info("cost alert sent", "message", msg)
		}
	}

	return nil
}

// queryDailyCosts returns a map of service→total USD spent in the last 24h.
// It queries np_ai_usage which is populated by the ai plugin with cost_usd per call.
func (a *Alerter) queryDailyCosts(ctx context.Context) (map[string]float64, error) {
	const q = `
		SELECT COALESCE(source_account_id, 'ai') AS service, COALESCE(SUM(cost_usd), 0)
		FROM np_ai_usage
		WHERE created_at >= now() - interval '24 hours'
		  AND cost_usd IS NOT NULL
		GROUP BY service
	`
	rows, err := a.DB.QueryContext(ctx, q)
	if err != nil {
		// Table may not exist if ai plugin is not installed; treat as zero cost.
		if strings.Contains(err.Error(), "does not exist") {
			return map[string]float64{}, nil
		}
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	totals := map[string]float64{}
	for rows.Next() {
		var service string
		var cost float64
		if err := rows.Scan(&service, &cost); err != nil {
			return nil, err
		}
		totals[service] += cost
	}
	return totals, rows.Err()
}

// formatAlert returns a Slack-formatted alert message.
func (a *Alerter) formatAlert(service string, budget, actual float64) string {
	channel := os.Getenv("SLACK_ALERT_CHANNEL")
	if channel == "" {
		channel = "#nself-alerts"
	}
	return fmt.Sprintf(
		"[nSelf Alert] Daily AI cost exceeded budget\nService:  %s\nBudget:   $%.2f/day\nActual:   $%.2f (as of %s UTC)\nAction:   Check /ai/usage or reduce sampling rate",
		service, budget, actual,
		time.Now().UTC().Format("15:04"),
	)
}

// postSlack sends a plain-text message to the configured Slack webhook URL.
func (a *Alerter) postSlack(ctx context.Context, message string) error {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		a.Log.Info("SLACK_WEBHOOK_URL not set, skipping Slack notification")
		return nil
	}
	escaped := strings.ReplaceAll(message, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)
	payload := fmt.Sprintf(`{"text": "%s"}`, escaped)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Slack webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return f
}
