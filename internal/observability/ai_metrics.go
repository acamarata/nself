package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

// AIMetrics holds the 10 AI call metrics per spec §1.3.
type AIMetrics struct {
	CallsTotal        *prometheus.CounterVec
	CallDuration      *prometheus.HistogramVec
	InputTokensTotal  *prometheus.CounterVec
	OutputTokensTotal *prometheus.CounterVec
	CostUSDTotal      *prometheus.CounterVec
	RetriesTotal      *prometheus.CounterVec
	RateLimitHits     *prometheus.CounterVec
	QualityScore      *prometheus.HistogramVec
	ContextTokens     *prometheus.HistogramVec
	ConcurrentCalls   *prometheus.GaugeVec
}

// NewAIMetrics creates and registers the 10 AI metrics on the given registry.
func NewAIMetrics(reg *prometheus.Registry) *AIMetrics {
	m := &AIMetrics{
		CallsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_calls_total",
				Help: "Total AI calls.",
			},
			[]string{"provider", "model", "purpose", "status"},
		),
		CallDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ai_call_duration_seconds",
				Help:    "AI call duration in seconds.",
				Buckets: DurationBuckets,
			},
			[]string{"provider", "model", "purpose"},
		),
		InputTokensTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_input_tokens_total",
				Help: "Total AI input tokens.",
			},
			[]string{"provider", "model"},
		),
		OutputTokensTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_output_tokens_total",
				Help: "Total AI output tokens.",
			},
			[]string{"provider", "model"},
		),
		CostUSDTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_cost_usd_total",
				Help: "Total AI cost in USD.",
			},
			[]string{"provider", "model"},
		),
		RetriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_retries_total",
				Help: "Total AI call retries.",
			},
			[]string{"provider", "model", "reason"},
		),
		RateLimitHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ai_rate_limit_hits_total",
				Help: "Total AI provider rate limit hits.",
			},
			[]string{"provider"},
		),
		QualityScore: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ai_quality_score",
				Help:    "AI response quality score distribution.",
				Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
			},
			[]string{"provider", "model", "purpose"},
		),
		ContextTokens: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ai_context_tokens",
				Help:    "AI context window token count distribution.",
				Buckets: []float64{256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072},
			},
			[]string{"provider", "model"},
		),
		ConcurrentCalls: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ai_concurrent_calls",
				Help: "Current concurrent AI calls.",
			},
			[]string{"provider"},
		),
	}
	reg.MustRegister(m.CallsTotal, m.CallDuration, m.InputTokensTotal, m.OutputTokensTotal,
		m.CostUSDTotal, m.RetriesTotal, m.RateLimitHits, m.QualityScore,
		m.ContextTokens, m.ConcurrentCalls)
	return m
}
