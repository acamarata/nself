// Package costmeter tracks per-plugin operational cost in a Prometheus-friendly
// shape. It records token counts, request counts, and a running cost estimate
// so operators can see "how much is this plugin spending per minute" on a
// Grafana dashboard.
//
// The meter is intentionally narrow: it does not implement a billing system.
// Cost in this package is an *estimate* derived from model prices supplied at
// registration time; actual provider billing is the source of truth.
package costmeter

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// ModelPrice describes the unit prices for a single model. Values are USD per
// 1,000 tokens (the conventional pricing unit for LLM providers). Either
// field may be zero if the provider does not distinguish input/output.
type ModelPrice struct {
	InputPer1K  float64
	OutputPer1K float64
}

// CostMeter aggregates token usage + estimated spend for a single plugin.
// All methods are safe for concurrent use.
type CostMeter struct {
	plugin string

	mu     sync.RWMutex
	prices map[string]ModelPrice

	tokensTotal   *prometheus.CounterVec // labels: model, direction (input|output)
	requestsTotal *prometheus.CounterVec // labels: model
	costTotal     *prometheus.CounterVec // labels: model — USD (float)
}

// New builds a CostMeter and registers its metrics on reg.
func New(pluginName string, reg prometheus.Registerer) *CostMeter {
	labels := prometheus.Labels{"plugin": pluginName}

	m := &CostMeter{
		plugin: pluginName,
		prices: make(map[string]ModelPrice),
		tokensTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "nself_plugin_tokens_total",
				Help:        "Total tokens processed by the plugin, by model and direction.",
				ConstLabels: labels,
			},
			[]string{"model", "direction"},
		),
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "nself_plugin_cost_requests_total",
				Help:        "Total metered requests by model.",
				ConstLabels: labels,
			},
			[]string{"model"},
		),
		costTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "nself_plugin_cost_usd_total",
				Help:        "Estimated operational cost in USD by model.",
				ConstLabels: labels,
			},
			[]string{"model"},
		),
	}
	if reg != nil {
		reg.MustRegister(m.tokensTotal, m.requestsTotal, m.costTotal)
	}
	return m
}

// RegisterModel teaches the meter how to price a model. Pricing can be
// updated at runtime if the provider changes their rates.
func (m *CostMeter) RegisterModel(model string, price ModelPrice) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prices[model] = price
}

// Record logs a single metered request. inputTokens and outputTokens are
// counted toward token totals; the cost counter increments by the product of
// token counts and the registered price. Unknown models record tokens + a
// request but contribute zero cost — the caller is expected to register
// prices before use.
func (m *CostMeter) Record(model string, inputTokens, outputTokens int) {
	m.requestsTotal.WithLabelValues(model).Inc()

	if inputTokens > 0 {
		m.tokensTotal.WithLabelValues(model, "input").Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		m.tokensTotal.WithLabelValues(model, "output").Add(float64(outputTokens))
	}

	m.mu.RLock()
	price, ok := m.prices[model]
	m.mu.RUnlock()
	if !ok {
		return
	}

	cost := float64(inputTokens)*price.InputPer1K/1000 +
		float64(outputTokens)*price.OutputPer1K/1000
	if cost > 0 {
		m.costTotal.WithLabelValues(model).Add(cost)
	}
}

// Plugin returns the plugin name this meter belongs to.
func (m *CostMeter) Plugin() string { return m.plugin }
