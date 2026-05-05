package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ClawMetrics holds the 10 claw-specific metrics per spec §1.7.
type ClawMetrics struct {
	MessagesTotal                 *prometheus.CounterVec
	ExtractionsTotal              *prometheus.CounterVec
	ExtractionDuration            *prometheus.HistogramVec
	DedupRatio                    prometheus.Histogram
	TopicClassificationConfidence prometheus.Histogram
	TopicsActive                  prometheus.Gauge
	MemoryBytes                   prometheus.Gauge
	GraphEdgesTotal               prometheus.Counter
	SearchLatency                 *prometheus.HistogramVec
	EmbeddingDim                  prometheus.Gauge
}

// NewClawMetrics creates and registers claw domain metrics.
func NewClawMetrics(reg *prometheus.Registry) *ClawMetrics {
	m := &ClawMetrics{
		MessagesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "claw_messages_total",
				Help: "Total claw messages processed.",
			},
			[]string{"type"},
		),
		ExtractionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "claw_extractions_total",
				Help: "Total knowledge extractions.",
			},
			[]string{"type", "status"},
		),
		ExtractionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "claw_extraction_duration_seconds",
				Help:    "Knowledge extraction duration.",
				Buckets: DurationBuckets,
			},
			[]string{"type"},
		),
		DedupRatio: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "claw_dedup_ratio",
				Help:    "Deduplication ratio of extracted facts.",
				Buckets: []float64{0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
			},
		),
		TopicClassificationConfidence: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "claw_topic_classification_confidence",
				Help:    "Topic classification confidence score.",
				Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
			},
		),
		TopicsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "claw_topics_active",
				Help: "Number of active topics in the knowledge graph.",
			},
		),
		MemoryBytes: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "claw_memory_bytes",
				Help: "Total memory usage of the claw knowledge store.",
			},
		),
		GraphEdgesTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "claw_graph_edges_total",
				Help: "Total edges in the knowledge graph.",
			},
		),
		SearchLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "claw_search_latency_seconds",
				Help:    "Knowledge search latency.",
				Buckets: DurationBuckets,
			},
			[]string{"type"},
		),
		EmbeddingDim: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "claw_embedding_dim",
				Help: "Dimensionality of embedding vectors.",
			},
		),
	}
	reg.MustRegister(m.MessagesTotal, m.ExtractionsTotal, m.ExtractionDuration,
		m.DedupRatio, m.TopicClassificationConfidence, m.TopicsActive,
		m.MemoryBytes, m.GraphEdgesTotal, m.SearchLatency, m.EmbeddingDim)
	return m
}

// MuxMetrics holds the 6 mux-specific metrics per spec §1.8.
type MuxMetrics struct {
	MessagesReceived       *prometheus.CounterVec
	ClassificationTotal    *prometheus.CounterVec
	ClassificationDuration *prometheus.HistogramVec
	ActionSuccess          *prometheus.CounterVec
	FalsePositiveTotal     *prometheus.CounterVec
	ProcessingDuration     *prometheus.HistogramVec
}

// NewMuxMetrics creates and registers mux domain metrics.
func NewMuxMetrics(reg *prometheus.Registry) *MuxMetrics {
	m := &MuxMetrics{
		MessagesReceived: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mux_messages_received_total",
				Help: "Total messages received by mux.",
			},
			[]string{"source"},
		),
		ClassificationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mux_classification_total",
				Help: "Total mux classifications.",
			},
			[]string{"category", "status"},
		),
		ClassificationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "mux_classification_duration_seconds",
				Help:    "Mux classification duration.",
				Buckets: DurationBuckets,
			},
			[]string{"category"},
		),
		ActionSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mux_action_success_total",
				Help: "Total successful mux actions.",
			},
			[]string{"action"},
		),
		FalsePositiveTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mux_false_positive_total",
				Help: "Total mux false positive classifications.",
			},
			[]string{"category"},
		),
		ProcessingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "mux_processing_duration_seconds",
				Help:    "Total mux message processing duration.",
				Buckets: DurationBuckets,
			},
			[]string{"source"},
		),
	}
	reg.MustRegister(m.MessagesReceived, m.ClassificationTotal, m.ClassificationDuration,
		m.ActionSuccess, m.FalsePositiveTotal, m.ProcessingDuration)
	return m
}

// PingMetrics holds the 5 ping_api/license metrics per spec §1.9.
type PingMetrics struct {
	LicenseChecksTotal   *prometheus.CounterVec
	LicenseCheckDuration *prometheus.HistogramVec
	PluginInstallsTotal  *prometheus.CounterVec
	PluginUptimeSeconds  *prometheus.GaugeVec
	PluginVersionInfo    *prometheus.GaugeVec
}

// NewPingMetrics creates and registers ping_api metrics.
func NewPingMetrics(reg *prometheus.Registry) *PingMetrics {
	m := &PingMetrics{
		LicenseChecksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "license_checks_total",
				Help: "Total license validation checks.",
			},
			[]string{"status", "tier"},
		),
		LicenseCheckDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "license_check_duration_seconds",
				Help:    "License check duration.",
				Buckets: DurationBuckets,
			},
			[]string{"status"},
		),
		PluginInstallsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "plugin_installations_total",
				Help: "Total plugin installations.",
			},
			[]string{"plugin", "tier"},
		),
		PluginUptimeSeconds: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "plugin_uptime_seconds",
				Help: "Plugin uptime in seconds.",
			},
			[]string{"plugin"},
		),
		PluginVersionInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "plugin_version_info",
				Help: "Plugin version info (value always 1, labels carry version).",
			},
			[]string{"plugin", "version"},
		),
	}
	reg.MustRegister(m.LicenseChecksTotal, m.LicenseCheckDuration,
		m.PluginInstallsTotal, m.PluginUptimeSeconds, m.PluginVersionInfo)
	return m
}
