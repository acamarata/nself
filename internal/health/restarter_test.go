package health

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockDockerClient implements DockerClient for testing.
type mockDockerClient struct {
	mu sync.Mutex

	containers []RestartContainer
	health     map[string]string // containerID -> health string

	restartCalls  []string // IDs that were restarted
	inspectCalled int32    // atomic counter
	listError     error
	inspectError  error
	restartError  error
}

func (m *mockDockerClient) ContainerList(_ context.Context, _ map[string]string) ([]RestartContainer, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]RestartContainer, len(m.containers))
	copy(out, m.containers)
	return out, nil
}

func (m *mockDockerClient) ContainerInspect(_ context.Context, id string) (RestartContainerInfo, error) {
	atomic.AddInt32(&m.inspectCalled, 1)
	if m.inspectError != nil {
		return RestartContainerInfo{}, m.inspectError
	}
	m.mu.Lock()
	h := m.health[id]
	m.mu.Unlock()
	return RestartContainerInfo{ID: id, Health: h}, nil
}

func (m *mockDockerClient) ContainerRestart(_ context.Context, id string, _ int) error {
	if m.restartError != nil {
		return m.restartError
	}
	m.mu.Lock()
	m.restartCalls = append(m.restartCalls, id)
	m.mu.Unlock()
	return nil
}

func (m *mockDockerClient) restartCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.restartCalls)
}

// ---- tests ----

// TestRestarterContextCancellation verifies that cancelling the context stops the loop.
func TestRestarterContextCancellation(t *testing.T) {
	mock := &mockDockerClient{
		containers: []RestartContainer{
			{ID: "c1", Name: "postgres", Service: "postgres"},
		},
		health: map[string]string{"c1": "healthy"},
	}

	policy := RestartPolicy{
		MaxAttempts:  3,
		PollInterval: 20 * time.Millisecond,
		RestartDelay: 1 * time.Millisecond,
	}

	r := NewRestarter(mock, policy)
	ctx, cancel := context.WithCancel(context.Background())

	if err := r.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	// Let it run briefly.
	time.Sleep(60 * time.Millisecond)

	cancel()

	// After cancellation, no restarts should have happened (container is healthy).
	if mock.restartCount() != 0 {
		t.Errorf("expected 0 restarts for healthy container, got %d", mock.restartCount())
	}
}

// TestRestarterUnhealthyServiceIsRestarted verifies that an unhealthy service is
// restarted up to MaxAttempts.
func TestRestarterUnhealthyServiceIsRestarted(t *testing.T) {
	mock := &mockDockerClient{
		containers: []RestartContainer{
			{ID: "c1", Name: "hasura", Service: "hasura"},
		},
		health: map[string]string{"c1": "unhealthy"},
	}

	policy := RestartPolicy{
		MaxAttempts:  3,
		PollInterval: 20 * time.Millisecond,
		RestartDelay: 1 * time.Millisecond,
	}

	r := NewRestarter(mock, policy)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := r.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	// Wait enough time for MaxAttempts polls to fire.
	// Each poll: 20ms interval + 1ms delay = ~21ms. 3 attempts => ~63ms. Give 400ms headroom.
	time.Sleep(400 * time.Millisecond)

	cancel()
	time.Sleep(10 * time.Millisecond) // let goroutine settle

	states := r.Status()
	if len(states) == 0 {
		t.Fatal("expected at least one service state, got none")
	}

	var st *ServiceState
	for i := range states {
		if states[i].Service == "hasura" {
			st = &states[i]
			break
		}
	}
	if st == nil {
		t.Fatal("no state for 'hasura' service")
	}

	if st.Attempts != policy.MaxAttempts {
		t.Errorf("expected %d attempts, got %d", policy.MaxAttempts, st.Attempts)
	}
	if !st.PermanentFail {
		t.Error("expected PermanentFail=true after exhausting MaxAttempts")
	}
}

// TestRestarterPermanentFailNoMoreRestarts verifies that after PermanentFail is set,
// no further restarts are attempted.
func TestRestarterPermanentFailNoMoreRestarts(t *testing.T) {
	mock := &mockDockerClient{
		containers: []RestartContainer{
			{ID: "c1", Name: "auth", Service: "auth"},
		},
		health: map[string]string{"c1": "unhealthy"},
	}

	policy := RestartPolicy{
		MaxAttempts:  2,
		PollInterval: 15 * time.Millisecond,
		RestartDelay: 1 * time.Millisecond,
	}

	r := NewRestarter(mock, policy)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := r.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	// Wait long enough for permanent fail to be set and a few more polls to run.
	time.Sleep(500 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	restarts := mock.restartCount()
	if restarts > policy.MaxAttempts {
		t.Errorf("expected at most %d restarts, got %d (container should be permanently failed)",
			policy.MaxAttempts, restarts)
	}

	states := r.Status()
	var st *ServiceState
	for i := range states {
		if states[i].Service == "auth" {
			st = &states[i]
			break
		}
	}
	if st == nil {
		t.Fatal("no state for 'auth' service")
	}
	if !st.PermanentFail {
		t.Error("expected PermanentFail=true")
	}
}

// TestRestarterStartInvalidPolicy verifies that a zero PollInterval is rejected.
func TestRestarterStartInvalidPolicy(t *testing.T) {
	mock := &mockDockerClient{}
	policy := RestartPolicy{MaxAttempts: 3, PollInterval: 0, RestartDelay: 5 * time.Second}
	r := NewRestarter(mock, policy)
	if err := r.Start(context.Background()); err == nil {
		t.Error("expected error for zero PollInterval, got nil")
	}
}

// TestDefaultPolicy verifies defaults are applied when env vars are not set.
func TestDefaultPolicy(t *testing.T) {
	p := defaultPolicy()
	if p.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", p.MaxAttempts)
	}
	if p.PollInterval != 30*time.Second {
		t.Errorf("expected PollInterval=30s, got %v", p.PollInterval)
	}
	if p.RestartDelay != 5*time.Second {
		t.Errorf("expected RestartDelay=5s, got %v", p.RestartDelay)
	}
}
