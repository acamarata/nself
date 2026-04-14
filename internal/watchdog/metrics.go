package watchdog

import (
	"fmt"
	"strings"
	"sync"
)

// Metrics tracks Prometheus-compatible counters for the watchdog.
type Metrics struct {
	mu       sync.Mutex
	restarts map[string]map[string]int // service -> result -> count
	circuits map[string]int            // service -> 0 (closed) or 1 (open)
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		restarts: make(map[string]map[string]int),
		circuits: make(map[string]int),
	}
}

// IncRestart increments the restart counter for a service/result pair.
func (m *Metrics) IncRestart(service, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.restarts[service] == nil {
		m.restarts[service] = make(map[string]int)
	}
	m.restarts[service][result]++
}

// SetCircuit sets the circuit state for a service (0=closed, 1=open).
func (m *Metrics) SetCircuit(service string, open bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if open {
		m.circuits[service] = 1
	} else {
		m.circuits[service] = 0
	}
}

// PrometheusText returns all metrics in Prometheus exposition format.
func (m *Metrics) PrometheusText() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var b strings.Builder

	b.WriteString("# HELP nself_watchdog_restart_total Total container restarts by service and result.\n")
	b.WriteString("# TYPE nself_watchdog_restart_total counter\n")
	for svc, results := range m.restarts {
		for result, count := range results {
			fmt.Fprintf(&b, "nself_watchdog_restart_total{service=%q,result=%q} %d\n", svc, result, count)
		}
	}

	b.WriteString("# HELP nself_watchdog_circuit_open Whether the circuit breaker is open (1) or closed (0).\n")
	b.WriteString("# TYPE nself_watchdog_circuit_open gauge\n")
	for svc, val := range m.circuits {
		fmt.Fprintf(&b, "nself_watchdog_circuit_open{service=%q} %d\n", svc, val)
	}

	return b.String()
}
