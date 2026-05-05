package watchdog

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/health"
)

// mockDocker implements health.DockerClient for testing.
type mockDocker struct {
	containers []health.RestartContainer
	healths    map[string]string // id -> health status
	restarts   int
	restartErr error
}

func (m *mockDocker) ContainerList(_ context.Context, _ map[string]string) ([]health.RestartContainer, error) {
	return m.containers, nil
}

func (m *mockDocker) ContainerInspect(_ context.Context, id string) (health.RestartContainerInfo, error) {
	return health.RestartContainerInfo{ID: id, Health: m.healths[id]}, nil
}

func (m *mockDocker) ContainerRestart(_ context.Context, _ string, _ int) error {
	m.restarts++
	return m.restartErr
}

func TestWatchdogCircuitBreaker(t *testing.T) {
	mock := &mockDocker{
		containers: []health.RestartContainer{
			{ID: "abc", Name: "test_svc", Service: "test"},
		},
		healths: map[string]string{"abc": "unhealthy"},
	}

	cfg := Config{
		Enabled:                true,
		CircuitBreakerAttempts: 3,
		CircuitBreakerWindow:   10 * time.Minute,
		PollInterval:           50 * time.Millisecond,
	}
	wd := New(cfg, mock)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go wd.Start(ctx)

	time.Sleep(400 * time.Millisecond)
	cancel()

	status := wd.GetStatus()
	// Should have tripped the circuit breaker after 3 attempts
	found := false
	for _, c := range status.Circuits {
		if c.Service == "test" && c.State == CircuitOpen {
			found = true
		}
	}
	if !found {
		t.Error("expected circuit breaker to be open for 'test' service")
	}
}

func TestMetricsPrometheusOutput(t *testing.T) {
	m := NewMetrics()
	m.IncRestart("claw", "success")
	m.IncRestart("claw", "success")
	m.IncRestart("claw", "failed")
	m.SetCircuit("claw", true)

	text := m.PrometheusText()
	if !strings.Contains(text, "nself_watchdog_restart_total") {
		t.Error("expected restart total metric")
	}
	if !strings.Contains(text, "nself_watchdog_circuit_open") {
		t.Error("expected circuit open metric")
	}
	if !strings.Contains(text, `service="claw"`) {
		t.Error("expected claw service label")
	}
}

func TestTestAlertNoConfig(t *testing.T) {
	// With no env vars, TestAlert should return errors for both channels
	delivered, errs := TestAlert("fake-service", "critical")
	if len(delivered) != 0 {
		t.Errorf("expected 0 deliveries with no config, got %d", len(delivered))
	}
	if len(errs) != 2 {
		t.Errorf("expected 2 errors (TG + email), got %d", len(errs))
	}
}
