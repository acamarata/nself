package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Signer tests ---

func TestSign_RoundTrip(t *testing.T) {
	secret := "test-secret-0123456789abcdef"
	ts := int64(1700000000)
	body := []byte(`{"event":"user.created"}`)

	sig := Sign(secret, ts, body)
	if sig == "" {
		t.Fatal("Sign returned empty string")
	}
	if sig[:7] != "sha256=" {
		t.Fatalf("expected sha256= prefix, got %q", sig[:7])
	}
	if !Verify(secret, ts, body, sig) {
		t.Fatal("Verify returned false for a valid signature")
	}
}

func TestSign_DifferentSecretFails(t *testing.T) {
	body := []byte(`{"event":"test"}`)
	ts := int64(1700000001)
	sig := Sign("correct-secret", ts, body)
	if Verify("wrong-secret", ts, body, sig) {
		t.Fatal("Verify should fail with wrong secret")
	}
}

func TestSign_TimestampIsBound(t *testing.T) {
	body := []byte(`{}`)
	secret := "s3cr3t"
	sig := Sign(secret, 100, body)
	if Verify(secret, 101, body, sig) {
		t.Fatal("Verify should fail with different timestamp")
	}
}

// --- Retry schedule tests ---

func TestNextRetryAt_Schedule(t *testing.T) {
	delays := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		30 * time.Second,
		5 * time.Minute,
		30 * time.Minute,
		2 * time.Hour,
		8 * time.Hour,
		24 * time.Hour,
	}
	tolerance := 0.10 // ±10%

	for i, want := range delays {
		attempt := i + 1
		before := time.Now()
		got := NextRetryAt(attempt)
		after := time.Now()

		elapsed := got.Sub(before)
		lo := time.Duration(float64(want) * (1 - tolerance))
		hi := time.Duration(float64(want)*(1+tolerance)) + (after.Sub(before))

		if elapsed < lo || elapsed > hi {
			t.Errorf("attempt %d: NextRetryAt delay %v not within ±10%% of %v", attempt, elapsed, want)
		}
	}
}

func TestNextRetryAt_BeyondMax(t *testing.T) {
	got := NextRetryAt(9)
	if !got.IsZero() {
		t.Errorf("attempt 9 should return zero time, got %v", got)
	}
}

// --- Circuit breaker tests ---

func TestCircuit_ClosedByDefault(t *testing.T) {
	c := NewCircuit()
	if c.State() != CircuitClosed {
		t.Errorf("expected closed, got %s", c.State())
	}
	if !c.Allow() {
		t.Error("expected Allow=true on closed circuit")
	}
}

func TestCircuit_OpensAfterFiveFailures(t *testing.T) {
	c := NewCircuit()
	for i := 0; i < 5; i++ {
		c.RecordFailure()
	}
	if c.State() != CircuitOpen {
		t.Errorf("expected open after 5 failures, got %s", c.State())
	}
	if c.Allow() {
		t.Error("expected Allow=false on open circuit")
	}
}

func TestCircuit_HalfOpenAfterInterval(t *testing.T) {
	c := NewCircuit()
	for i := 0; i < 5; i++ {
		c.RecordFailure()
	}
	// Simulate probe interval passing.
	c.mu.Lock()
	c.openedAt = time.Now().Add(-probeInterval - time.Second)
	c.mu.Unlock()

	if c.State() != CircuitHalfOpen {
		t.Errorf("expected half-open after interval, got %s", c.State())
	}
	if !c.Allow() {
		t.Error("expected Allow=true on half-open circuit")
	}
}

func TestCircuit_ClosesAfterTwoSuccessfulProbes(t *testing.T) {
	c := NewCircuit()
	for i := 0; i < 5; i++ {
		c.RecordFailure()
	}
	c.mu.Lock()
	c.openedAt = time.Now().Add(-probeInterval - time.Second)
	c.mu.Unlock()

	c.RecordSuccess() // first probe
	if c.State() != CircuitHalfOpen {
		t.Errorf("expected half-open after first probe, got %s", c.State())
	}
	c.RecordSuccess() // second probe — should close
	if c.State() != CircuitClosed {
		t.Errorf("expected closed after two probes, got %s", c.State())
	}
}

// --- Dispatcher integration: mock endpoint 500 → DLQ after 8 attempts ---

type mockDeliveryStore struct {
	deliveries map[string]*Delivery
	health     map[string]int
	retries    map[string]int
}

func newMockStore(del Delivery) *mockDeliveryStore {
	s := &mockDeliveryStore{
		deliveries: map[string]*Delivery{del.ID: &del},
		health:     map[string]int{},
		retries:    map[string]int{},
	}
	return s
}

func (s *mockDeliveryStore) NextPending(_ context.Context, n int) ([]Delivery, error) {
	var out []Delivery
	for _, d := range s.deliveries {
		if !d.Delivered {
			out = append(out, *d)
		}
		if len(out) >= n {
			break
		}
	}
	return out, nil
}

func (s *mockDeliveryStore) MarkDelivered(_ context.Context, id string, _ int) error {
	if d, ok := s.deliveries[id]; ok {
		d.Delivered = true
	}
	return nil
}

func (s *mockDeliveryStore) ScheduleRetry(_ context.Context, id string, nextAttempt int, _ time.Time) error {
	if d, ok := s.deliveries[id]; ok {
		d.AttemptCount = nextAttempt
	}
	s.retries[id]++
	return nil
}

func (s *mockDeliveryStore) UpdateHealth(_ context.Context, endpointID string, score int, _ CircuitState) error {
	s.health[endpointID] = score
	return nil
}

type mockDLQStore struct {
	entries []DLQEntry
}

func (s *mockDLQStore) Save(_ context.Context, e DLQEntry) error {
	s.entries = append(s.entries, e)
	return nil
}

func (s *mockDLQStore) List(_ context.Context, _, _ int) ([]DLQEntry, error) {
	return s.entries, nil
}

func (s *mockDLQStore) ReEnqueue(_ context.Context, id string) error {
	for i, e := range s.entries {
		if e.ID == id {
			now := time.Now()
			s.entries[i].ReEnqueuedAt = &now
			return nil
		}
	}
	return ErrDLQEntryNotFound
}

func TestDispatcher_SuccessfulDelivery(t *testing.T) {
	// Allow loopback so httptest.NewServer (127.0.0.1) passes the SSRF guard.
	t.Setenv("SSRF_ALLOWED_HOSTS", "127.0.0.1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify required headers are present.
		if r.Header.Get("X-Nself-Delivery") == "" {
			t.Error("missing X-Nself-Delivery header")
		}
		if r.Header.Get("X-Nself-Event") == "" {
			t.Error("missing X-Nself-Event header")
		}
		if r.Header.Get("X-Nself-Timestamp") == "" {
			t.Error("missing X-Nself-Timestamp header")
		}
		if r.Header.Get("X-Nself-Signature-256") == "" {
			t.Error("missing X-Nself-Signature-256 header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	del := Delivery{
		ID:            "del-001",
		EndpointID:    "ep-001",
		EndpointURL:   srv.URL,
		SigningSecret: "my-secret",
		EventName:     "user.created",
		Payload:       []byte(`{"id":"1"}`),
	}
	dlqStore := &mockDLQStore{}
	store := newMockStore(del)
	dlq := NewDLQManager(dlqStore)
	d := NewDispatcher(DispatcherConfig{Workers: 1, BatchSize: 1, PollTick: 50 * time.Millisecond}, store, dlq)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	d.Run(ctx)

	if !store.deliveries["del-001"].Delivered {
		t.Error("delivery should be marked delivered after 200 response")
	}
	if len(dlqStore.entries) != 0 {
		t.Errorf("DLQ should be empty after success, got %d entries", len(dlqStore.entries))
	}
}

func TestDispatcher_MovesToDLQAfterMaxAttempts(t *testing.T) {
	// Allow loopback so httptest.NewServer (127.0.0.1) passes the SSRF guard.
	t.Setenv("SSRF_ALLOWED_HOSTS", "127.0.0.1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	del := Delivery{
		ID:           "del-002",
		EndpointID:   "ep-002",
		EndpointURL:  srv.URL,
		EventName:    "plugin.failed",
		Payload:      []byte(`{}`),
		AttemptCount: maxAttempts, // already at max — next dispatch should quarantine
	}
	dlqStore := &mockDLQStore{}
	store := newMockStore(del)
	dlq := NewDLQManager(dlqStore)
	d := NewDispatcher(DispatcherConfig{Workers: 1, BatchSize: 1, PollTick: 50 * time.Millisecond}, store, dlq)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	d.Run(ctx)

	if len(dlqStore.entries) == 0 {
		t.Error("expected DLQ entry after maxAttempts exhausted")
	}
}
