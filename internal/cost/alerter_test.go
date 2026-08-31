package cost

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestEnabled_Default(t *testing.T) {
	_ = os.Unsetenv("COST_ALERT_ENABLED")
	if Enabled() {
		t.Error("expected Enabled() to return false by default")
	}
}

func TestEnabled_True(t *testing.T) {
	_ = os.Setenv("COST_ALERT_ENABLED", "true")
	defer func() { _ = os.Unsetenv("COST_ALERT_ENABLED") }()
	if !Enabled() {
		t.Error("expected Enabled() to return true when env is 'true'")
	}
}

func TestLoadBudgetFromEnv(t *testing.T) {
	_ = os.Setenv("COST_BUDGET_AI_DAILY_USD", "5.00")
	_ = os.Setenv("COST_BUDGET_MEDIA_DAILY_USD", "2.50")
	_ = os.Setenv("COST_BUDGET_TOTAL_DAILY_USD", "20.00")
	defer func() {
		_ = os.Unsetenv("COST_BUDGET_AI_DAILY_USD")
		_ = os.Unsetenv("COST_BUDGET_MEDIA_DAILY_USD")
		_ = os.Unsetenv("COST_BUDGET_TOTAL_DAILY_USD")
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

// TestPostSlack_RoutesToConfiguredChannel asserts the WIRE FORMAT rather than
// the message prose. SLACK_ALERT_CHANNEL names a destination, so putting the
// value in the payload's "channel" field is what can actually route the alert;
// printing it as a line of body text would satisfy a string assertion while
// still sending every alert wherever the webhook points, which is the bug this
// replaces. See the caveat in postSlack about app-scoped webhooks.
func TestPostSlack_RoutesToConfiguredChannel(t *testing.T) {
	for _, c := range []struct{ name, env, want string }{
		{"default when unset", "", "#nself-alerts"},
		{"custom when set", "#custom-alerts", "#custom-alerts"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.env == "" {
				t.Setenv("SLACK_ALERT_CHANNEL", "")
			} else {
				t.Setenv("SLACK_ALERT_CHANNEL", c.env)
			}

			var body string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				body = string(b)
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()
			t.Setenv("SLACK_WEBHOOK_URL", ts.URL)

			a := New(nil, Budget{})
			if err := a.postSlack(context.Background(), "budget exceeded"); err != nil {
				t.Fatalf("postSlack: %v", err)
			}
			if !strings.Contains(body, `"channel": "`+c.want+`"`) {
				t.Errorf("payload did not route to %s; got %s", c.want, body)
			}
		})
	}
}

func TestPostSlack_NoURL(t *testing.T) {
	_ = os.Unsetenv("SLACK_WEBHOOK_URL")
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

	_ = os.Setenv("SLACK_WEBHOOK_URL", ts.URL)
	defer func() { _ = os.Unsetenv("SLACK_WEBHOOK_URL") }()

	a := &Alerter{HTTPClient: ts.Client()}
	if err := a.postSlack(context.Background(), "budget exceeded"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected Slack webhook to be called")
	}
}

func TestCheckAndAlert_Disabled(t *testing.T) {
	_ = os.Unsetenv("COST_ALERT_ENABLED")
	a := &Alerter{}
	// Should return nil without touching DB
	if err := a.CheckAndAlert(context.Background()); err != nil {
		t.Errorf("expected nil when disabled, got: %v", err)
	}
}
