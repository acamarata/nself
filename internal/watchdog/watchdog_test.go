package watchdog

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/health"
)

// TestCircuitPermanentOpen verifies that after PermanentOpenThreshold consecutive
// OPEN windows the circuit transitions to PERMANENT_OPEN with no further resets.
func TestCircuitPermanentOpen(t *testing.T) {
	const threshold = 3
	mock := &mockDocker{
		containers: []health.RestartContainer{
			{ID: "svc1", Name: "crash_svc", Service: "crash"},
		},
		healths: map[string]string{"svc1": "unhealthy"},
	}

	cfg := Config{
		Enabled:                true,
		CircuitBreakerAttempts: 1,                    // trip immediately on first failure
		CircuitBreakerWindow:   1 * time.Millisecond, // very short window to force rollovers
		PollInterval:           5 * time.Millisecond,
		PermanentOpenThreshold: threshold,
	}
	wd := New(cfg, mock)

	// Manually drive the circuit through threshold consecutive open windows.
	// Inject a pre-existing OPEN circuit and simulate window rollovers.
	wd.mu.Lock()
	circuit := &ServiceCircuit{
		Service:     "crash",
		State:       CircuitOpen,
		Attempts:    cfg.CircuitBreakerAttempts,
		TrippedAt:   time.Now(),
		WindowStart: time.Now().Add(-2 * cfg.CircuitBreakerWindow), // window already expired
	}
	wd.circuits["crash"] = circuit
	wd.mu.Unlock()

	// Simulate threshold window rollovers by calling handleUnhealthy repeatedly
	// with an expired window each time.
	for i := 0; i < threshold; i++ {
		wd.mu.Lock()
		circuit.WindowStart = time.Now().Add(-2 * cfg.CircuitBreakerWindow)
		wd.mu.Unlock()
		wd.handleUnhealthy(context.Background(), mock.containers[0])
	}

	wd.mu.Lock()
	state := circuit.State
	windows := circuit.ConsecutiveOpenWindows
	wd.mu.Unlock()

	if state != CircuitPermanentOpen {
		t.Errorf("expected PERMANENT_OPEN after %d windows, got %q (consecutiveOpenWindows=%d)", threshold, state, windows)
	}

	// Confirm no further restart attempts after PERMANENT_OPEN.
	restartsBefore := mock.restarts
	wd.handleUnhealthy(context.Background(), mock.containers[0])
	if mock.restarts != restartsBefore {
		t.Errorf("expected no restarts in PERMANENT_OPEN, got %d additional", mock.restarts-restartsBefore)
	}
}

// TestResetServiceClearsPermanentOpen verifies that ResetService transitions
// PERMANENT_OPEN back to CLOSED and clears the consecutive counter.
func TestResetServiceClearsPermanentOpen(t *testing.T) {
	mock := &mockDocker{}
	cfg := Config{
		Enabled:                true,
		CircuitBreakerAttempts: 3,
		CircuitBreakerWindow:   10 * time.Minute,
		PollInterval:           30 * time.Second,
		PermanentOpenThreshold: 3,
	}
	wd := New(cfg, mock)

	// Inject a PERMANENT_OPEN circuit.
	wd.mu.Lock()
	wd.circuits["svc"] = &ServiceCircuit{
		Service:                "svc",
		State:                  CircuitPermanentOpen,
		ConsecutiveOpenWindows: 3,
	}
	wd.mu.Unlock()

	ok := wd.ResetService("svc", "testoperator")
	if !ok {
		t.Fatal("ResetService returned false for a known service")
	}

	wd.mu.Lock()
	c := wd.circuits["svc"]
	state := c.State
	windows := c.ConsecutiveOpenWindows
	wd.mu.Unlock()

	if state != CircuitClosed {
		t.Errorf("expected CLOSED after reset, got %q", state)
	}
	if windows != 0 {
		t.Errorf("expected consecutiveOpenWindows=0 after reset, got %d", windows)
	}

	// Verify an event was logged with operator identity.
	events := wd.GetHistory(0)
	found := false
	for _, e := range events {
		if e.Service == "svc" && e.Action == "circuit_reset" && strings.Contains(e.Detail, "testoperator") {
			found = true
		}
	}
	if !found {
		t.Error("expected a circuit_reset event with operator identity 'testoperator'")
	}
}

// TestResetServiceUnknown verifies that ResetService returns false for an unknown service.
func TestResetServiceUnknown(t *testing.T) {
	mock := &mockDocker{}
	cfg := Config{
		Enabled:                true,
		CircuitBreakerAttempts: 3,
		CircuitBreakerWindow:   10 * time.Minute,
		PollInterval:           30 * time.Second,
		PermanentOpenThreshold: 3,
	}
	wd := New(cfg, mock)

	ok := wd.ResetService("nonexistent", "op")
	if ok {
		t.Error("expected false for unknown service, got true")
	}
}

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
