package webhook

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	saferecover "github.com/nself-org/cli/internal/recover"
)

// retryDelays is the exponential backoff schedule per attempt (1-indexed).
// After attempt 8 the delivery is moved to the DLQ.
var retryDelays = []time.Duration{
	0,                // attempt 0 — unused sentinel
	1 * time.Second,  // attempt 1
	5 * time.Second,  // attempt 2
	30 * time.Second, // attempt 3
	5 * time.Minute,  // attempt 4
	30 * time.Minute, // attempt 5
	2 * time.Hour,    // attempt 6
	8 * time.Hour,    // attempt 7
	24 * time.Hour,   // attempt 8
}

// maxAttempts is the total number of delivery attempts before quarantine.
const maxAttempts = 8

// dispatchTimeout is the maximum time allowed per POST attempt.
const dispatchTimeout = 10 * time.Second

// Delivery represents a queued outbound webhook delivery.
type Delivery struct {
	ID            string
	EndpointID    string
	EndpointURL   string
	SigningSecret string // may be empty for unsigned endpoints
	EventName     string
	Payload       []byte
	AttemptCount  int
	// Delivered is set by the store when a prior attempt succeeded.
	// Dispatcher skips any delivery where Delivered is true (idempotency guard).
	Delivered bool
}

// DeliveryStore is the persistence contract used by the Dispatcher.
type DeliveryStore interface {
	// NextPending returns up to n deliveries whose next_retry_at <= now and delivered = false.
	NextPending(ctx context.Context, n int) ([]Delivery, error)
	// MarkDelivered marks a delivery as successfully delivered.
	MarkDelivered(ctx context.Context, id string, responseMS int) error
	// ScheduleRetry increments attempt_count and sets next_retry_at.
	ScheduleRetry(ctx context.Context, id string, nextAttempt int, at time.Time) error
	// UpdateHealth updates endpoint health_score and circuit_state.
	UpdateHealth(ctx context.Context, endpointID string, score int, state CircuitState) error
}

// DispatcherConfig controls Dispatcher concurrency and polling.
type DispatcherConfig struct {
	Workers   int           // goroutine pool size (default 8)
	BatchSize int           // max deliveries fetched per tick (default 32)
	PollTick  time.Duration // how often to poll for pending deliveries (default 5s)
}

func (c *DispatcherConfig) defaults() {
	if c.Workers <= 0 {
		c.Workers = 8
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 32
	}
	if c.PollTick <= 0 {
		c.PollTick = 5 * time.Second
	}
}

// Dispatcher is the goroutine pool that picks up pending deliveries and sends them.
type Dispatcher struct {
	cfg      DispatcherConfig
	store    DeliveryStore
	dlq      *DLQManager
	client   *http.Client
	circuits sync.Map // endpointID -> *Circuit
}

// NewDispatcher creates a Dispatcher. Call Run to start the worker pool.
func NewDispatcher(cfg DispatcherConfig, store DeliveryStore, dlq *DLQManager) *Dispatcher {
	cfg.defaults()
	return &Dispatcher{
		cfg:    cfg,
		store:  store,
		dlq:    dlq,
		client: &http.Client{Timeout: dispatchTimeout},
	}
}

// Run starts the dispatcher and blocks until ctx is cancelled.
func (d *Dispatcher) Run(ctx context.Context) {
	work := make(chan Delivery, d.cfg.BatchSize)

	// Start worker pool. Each worker is wrapped with SafeGo so a panic in
	// d.dispatch (e.g. malformed payload, transport bug) does not take down
	// the CLI process.  The wg.Done is registered inside SafeGo so the panic
	// recovery still releases the wait group.
	var wg sync.WaitGroup
	for i := 0; i < d.cfg.Workers; i++ {
		wg.Add(1)
		saferecover.SafeGo("webhook_dispatch_worker", func() {
			defer wg.Done()
			for delivery := range work {
				d.dispatch(ctx, delivery)
			}
		})
	}

	// Poll loop.
	ticker := time.NewTicker(d.cfg.PollTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			close(work)
			wg.Wait()
			return
		case <-ticker.C:
			deliveries, err := d.store.NextPending(ctx, d.cfg.BatchSize)
			if err != nil {
				continue
			}
			for _, del := range deliveries {
				select {
				case work <- del:
				case <-ctx.Done():
					close(work)
					wg.Wait()
					return
				}
			}
		}
	}
}

// NextRetryAt computes the next_retry_at timestamp for attempt n (1-indexed).
// Returns zero time if n exceeds maxAttempts (should move to DLQ instead).
func NextRetryAt(attempt int) time.Time {
	if attempt <= 0 || attempt > maxAttempts {
		return time.Time{}
	}
	return time.Now().Add(retryDelays[attempt])
}

// dispatch sends a single delivery, handling signing, circuit-breaking, and DLQ quarantine.
func (d *Dispatcher) dispatch(ctx context.Context, del Delivery) {
	// Idempotency: skip already-delivered deliveries.
	if del.Delivered {
		return
	}

	// Circuit breaker check.
	circ := d.circuitFor(del.EndpointID)
	if !circ.Allow() {
		// Endpoint is open — reschedule without incrementing attempt count.
		next := time.Now().Add(probeInterval)
		_ = d.store.ScheduleRetry(ctx, del.ID, del.AttemptCount, next)
		return
	}

	// Build request.
	now := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, del.EndpointURL, bytes.NewReader(del.Payload))
	if err != nil {
		d.handleFailure(ctx, del, fmt.Sprintf("build request: %v", err), circ)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	InjectHeaders(req, del.SigningSecret, del.ID, del.EventName, now, del.Payload)

	// Send.
	start := time.Now()
	resp, err := d.client.Do(req)
	elapsed := int(time.Since(start).Milliseconds())

	if err != nil {
		d.handleFailure(ctx, del, fmt.Sprintf("POST: %v", err), circ)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success.
		circ.RecordSuccess()
		_ = d.store.MarkDelivered(ctx, del.ID, elapsed)
		score := healthScore(circ)
		_ = d.store.UpdateHealth(ctx, del.EndpointID, score, circ.State())
		return
	}

	d.handleFailure(ctx, del, fmt.Sprintf("HTTP %d", resp.StatusCode), circ)
}

// handleFailure processes a failed delivery attempt: schedules retry or quarantines.
func (d *Dispatcher) handleFailure(ctx context.Context, del Delivery, reason string, circ *Circuit) {
	circ.RecordFailure()
	nextAttempt := del.AttemptCount + 1

	score := healthScore(circ)
	_ = d.store.UpdateHealth(ctx, del.EndpointID, score, circ.State())

	if nextAttempt > maxAttempts {
		// Move to DLQ.
		_ = d.dlq.Quarantine(ctx, del.EndpointID, del.ID, del.Payload, reason)
		return
	}

	// Schedule retry.
	nextAt := NextRetryAt(nextAttempt)
	_ = d.store.ScheduleRetry(ctx, del.ID, nextAttempt, nextAt)
}

// circuitFor returns (or creates) the Circuit for an endpoint.
func (d *Dispatcher) circuitFor(endpointID string) *Circuit {
	v, _ := d.circuits.LoadOrStore(endpointID, NewCircuit())
	return v.(*Circuit)
}

// healthScore derives a 0-100 health score from the circuit state and consecutive failures.
func healthScore(c *Circuit) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.state {
	case CircuitOpen:
		return 0
	case CircuitHalfOpen:
		return 30
	default:
		// Degrade 10 points per consecutive failure, floor 10.
		score := 100 - (c.consecutiveFail * 10)
		if score < 10 {
			return 10
		}
		return score
	}
}
