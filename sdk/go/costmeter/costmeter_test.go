package costmeter

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordKnownModel(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New("ai", reg)
	m.RegisterModel("gpt-4o", ModelPrice{InputPer1K: 2.50, OutputPer1K: 10.00})

	m.Record("gpt-4o", 1000, 500)

	// Expected cost: 1000 * 2.50 / 1000 + 500 * 10.00 / 1000 = 2.50 + 5.00 = 7.50
	want := float64(7.5)
	got := testutil.ToFloat64(m.costTotal.WithLabelValues("gpt-4o"))
	if got != want {
		t.Fatalf("expected cost %.2f USD, got %.2f", want, got)
	}

	if tokens := testutil.ToFloat64(m.tokensTotal.WithLabelValues("gpt-4o", "input")); tokens != 1000 {
		t.Fatalf("expected 1000 input tokens, got %.0f", tokens)
	}
	if tokens := testutil.ToFloat64(m.tokensTotal.WithLabelValues("gpt-4o", "output")); tokens != 500 {
		t.Fatalf("expected 500 output tokens, got %.0f", tokens)
	}
	if reqs := testutil.ToFloat64(m.requestsTotal.WithLabelValues("gpt-4o")); reqs != 1 {
		t.Fatalf("expected 1 request, got %.0f", reqs)
	}
}

func TestRecordUnknownModelSkipsCost(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New("ai", reg)

	m.Record("unknown-model", 100, 50)

	if cost := testutil.ToFloat64(m.costTotal.WithLabelValues("unknown-model")); cost != 0 {
		t.Fatalf("expected zero cost for unknown model, got %.2f", cost)
	}
	if reqs := testutil.ToFloat64(m.requestsTotal.WithLabelValues("unknown-model")); reqs != 1 {
		t.Fatalf("expected request still counted, got %.0f", reqs)
	}
}

func TestMetricsRegistered(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New("my-plugin", reg)
	// Touch every counter so Prometheus surfaces them during Gather.
	m.RegisterModel("probe", ModelPrice{InputPer1K: 1, OutputPer1K: 1})
	m.Record("probe", 1, 1)

	gathered, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	seen := map[string]bool{}
	for _, mf := range gathered {
		seen[mf.GetName()] = true
	}
	for _, want := range []string{
		"nself_plugin_tokens_total",
		"nself_plugin_cost_requests_total",
		"nself_plugin_cost_usd_total",
	} {
		if !seen[want] {
			t.Fatalf("metric %q not registered; gathered: %v", want, seen)
		}
	}
}

func TestRegisterModelThenUpdate(t *testing.T) {
	m := New("ai", prometheus.NewRegistry())
	m.RegisterModel("gpt-4o", ModelPrice{InputPer1K: 1.0, OutputPer1K: 2.0})
	m.RegisterModel("gpt-4o", ModelPrice{InputPer1K: 5.0, OutputPer1K: 10.0})
	m.Record("gpt-4o", 1000, 0)
	if cost := testutil.ToFloat64(m.costTotal.WithLabelValues("gpt-4o")); cost != 5.0 {
		t.Fatalf("expected updated price applied, got %.2f", cost)
	}
}

func TestPlugin(t *testing.T) {
	m := New("my-plugin", prometheus.NewRegistry())
	if !strings.EqualFold(m.Plugin(), "my-plugin") {
		t.Fatalf("unexpected plugin name: %q", m.Plugin())
	}
}
