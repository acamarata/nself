package cost

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestEnabled_Default(t *testing.T) {
	os.Unsetenv("COST_ALERT_ENABLED")
	if Enabled() {
		t.Error("expected Enabled() to return false by default")
	}
}

func TestEnabled_True(t *testing.T) {
	os.Setenv("COST_ALERT_ENABLED", "true")
	defer os.Unsetenv("COST_ALERT_ENABLED")
	if !Enabled() {
		t.Error("expected Enabled() to return true when env is 'true'")
	}
}

func TestLoadBudgetFromEnv(t *testing.T) {
	os.Setenv("COST_BUDGET_AI_DAILY_USD", "5.00")
	os.Setenv("COST_BUDGET_MEDIA_DAILY_USD", "2.50")
	os.Setenv("COST_BUDGET_TOTAL_DAILY_USD", "20.00")
	defer func() {
		os.Unsetenv("COST_BUDGET_AI_DAILY_USD")
		os.Unsetenv("COST_BUDGET_MEDIA_DAILY_USD")
		os.Unsetenv("COST_BUDGET_TOTAL_DAILY_USD")
	}()

	b := LoadBudgetFromEnv()
	if b.AIDailyUSD != 5.00 {
		t.Errorf("expected AI budget 5.00, got %f", b.AIDailyUSD)
	}
	if b.MediaDailyUSD != 2.50 {
		t.Errorf("expected media budget 2.50, got %f", b.MediaDailyUSD)
	}
	if b.TotalDailyUSD != 20.00 {
		t.Errorf("expected total budget 20.00, got %f", b.TotalDailyUSD)
	}
}

func TestFormatAlert(t *testing.T) {
	a := &Alerter{}
	msg := a.formatAlert("ai plugin", 5.00, 7.23)
	if !strings.Contains(msg, "$5.00/day") {
		t.Errorf("expected budget in message, got: %s", msg)
	}
	if !strings.Contains(msg, "$7.23") {
		t.Errorf("expected actual cost in message, got: %s", msg)
	}
	if !strings.Contains(msg, "ai plugin") {
		t.Errorf("expected service name in message, got: %s", msg)
	}
}

func TestPostSlack_NoURL(t *testing.T) {
	os.Unsetenv("SLACK_WEBHOOK_URL")
	a := New(nil, Budget{})
	// Should not error when URL is unset
	if err := a.postSlack(context.Background(), "test message"); err != nil {
		t.Errorf("expected nil error when SLACK_WEBHOOK_URL unset, got: %v", err)
	}
}

func TestPostSlack_Success(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	os.Setenv("SLACK_WEBHOOK_URL", ts.URL)
	defer os.Unsetenv("SLACK_WEBHOOK_URL")

	a := &Alerter{HTTPClient: ts.Client()}
	if err := a.postSlack(context.Background(), "budget exceeded"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected Slack webhook to be called")
	}
}

func TestCheckAndAlert_Disabled(t *testing.T) {
	os.Unsetenv("COST_ALERT_ENABLED")
	a := &Alerter{}
	// Should return nil without touching DB
	if err := a.CheckAndAlert(context.Background()); err != nil {
		t.Errorf("expected nil when disabled, got: %v", err)
	}
}
