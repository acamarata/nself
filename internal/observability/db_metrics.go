package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

// DBMetrics holds the seven database metrics per spec §1.4.
type DBMetrics struct {
	QueryDuration  *prometheus.HistogramVec
	QueriesTotal   *prometheus.CounterVec
	PoolInUse      *prometheus.GaugeVec
	PoolIdle       *prometheus.GaugeVec
	PoolMax        *prometheus.GaugeVec
	ConnWait       *prometheus.HistogramVec
	DeadlocksTotal *prometheus.CounterVec
}

// NewDBMetrics creates and registers database metrics on the given registry.
func NewDBMetrics(reg *prometheus.Registry) *DBMetrics {
	m := &DBMetrics{
		QueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds.",
				Buckets: DurationBuckets,
			},
			[]string{"service", "op", "table", "status"},
		),
		QueriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_queries_total",
				Help: "Total database queries.",
			},
			[]string{"service", "op", "table", "status"},
		),
		PoolInUse: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_connection_pool_in_use",
				Help: "Current in-use DB connections.",
			},
			[]string{"service"},
		),
		PoolIdle: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_connection_pool_idle",
				Help: "Current idle DB connections.",
			},
			[]string{"service"},
		),
		PoolMax: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_connection_pool_max",
				Help: "Maximum DB connections.",
			},
			[]string{"service"},
		),
		ConnWait: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_connection_wait_seconds",
				Help:    "Time waiting for a DB connection.",
				Buckets: DurationBuckets,
			},
			[]string{"service"},
		),
		DeadlocksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_deadlocks_total",
				Help: "Total database deadlocks.",
			},
			[]string{"service"},
		),
	}
	reg.MustRegister(m.QueryDuration, m.QueriesTotal, m.PoolInUse, m.PoolIdle,
		m.PoolMax, m.ConnWait, m.DeadlocksTotal)
	return m
}

// CacheMetrics holds cache metrics per spec §1.5.
type CacheMetrics struct {
	HitsTotal      *prometheus.CounterVec
	MissesTotal    *prometheus.CounterVec
	OpDuration     *prometheus.HistogramVec
	EvictionsTotal *prometheus.CounterVec
}

// NewCacheMetrics creates and registers cache metrics on the given registry.
func NewCacheMetrics(reg *prometheus.Registry) *CacheMetrics {
	m := &CacheMetrics{
		HitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total cache hits.",
			},
			[]string{"service", "cache", "op"},
		),
		MissesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total cache misses.",
			},
			[]string{"service", "cache", "op"},
		),
		OpDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "cache_operation_duration_seconds",
				Help:    "Cache operation duration in seconds.",
				Buckets: DurationBuckets,
			},
			[]string{"service", "cache", "op"},
		),
		EvictionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_evictions_total",
				Help: "Total cache evictions.",
			},
			[]string{"service", "cache", "reason"},
		),
	}
	reg.MustRegister(m.HitsTotal, m.MissesTotal, m.OpDuration, m.EvictionsTotal)
	return m
}

// QueueMetrics holds queue metrics per spec §1.6.
type QueueMetrics struct {
	Depth          *prometheus.GaugeVec
	EnqueuedTotal  *prometheus.CounterVec
	CompletedTotal *prometheus.CounterVec
	JobDuration    *prometheus.HistogramVec
	JobWait        *prometheus.HistogramVec
	WorkersActive  *prometheus.GaugeVec
	RetriedTotal   *prometheus.CounterVec
	DLQDepth       *prometheus.GaugeVec
}

// NewQueueMetrics creates and registers queue metrics on the given registry.
func NewQueueMetrics(reg *prometheus.Registry) *QueueMetrics {
	m := &QueueMetrics{
		Depth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_depth",
				Help: "Current queue depth.",
			},
			[]string{"queue"},
		),
		EnqueuedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_jobs_enqueued_total",
				Help: "Total jobs enqueued.",
			},
			[]string{"queue"},
		),
		CompletedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_jobs_completed_total",
				Help: "Total jobs completed.",
			},
			[]string{"queue", "status"},
		),
		JobDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_job_duration_seconds",
				Help:    "Job execution duration.",
				Buckets: DurationBuckets,
			},
			[]string{"queue"},
		),
		JobWait: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_job_wait_seconds",
				Help:    "Time job waited in queue before execution.",
				Buckets: DurationBuckets,
			},
			[]string{"queue"},
		),
		WorkersActive: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_workers_active",
				Help: "Active queue workers.",
			},
			[]string{"queue"},
		),
		RetriedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_jobs_retried_total",
				Help: "Total job retries.",
			},
			[]string{"queue", "reason"},
		),
		DLQDepth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_dlq_depth",
				Help: "Dead letter queue depth.",
			},
			[]string{"queue"},
		),
	}
	reg.MustRegister(m.Depth, m.EnqueuedTotal, m.CompletedTotal, m.JobDuration,
		m.JobWait, m.WorkersActive, m.RetriedTotal, m.DLQDepth)
	return m
}
