// Package watchdog implements self-healing container monitoring with circuit breaker.
package watchdog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nself-org/cli/internal/health"
)

// Config holds watchdog configuration.
type Config struct {
	Enabled                bool
	CircuitBreakerAttempts int           // default 3
	CircuitBreakerWindow   time.Duration // default 10m
	EscalationWebhook      string
	PollInterval           time.Duration // default 30s
}

// DefaultConfig returns watchdog configuration with defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:                true,
		CircuitBreakerAttempts: 3,
		CircuitBreakerWindow:   10 * time.Minute,
		PollInterval:           30 * time.Second,
	}
}

// CircuitState represents the state of a circuit breaker for a service.
type CircuitState string

const (
	CircuitClosed CircuitState = "closed" // healthy, restarts allowed
	CircuitOpen   CircuitState = "open"   // tripped, restarts blocked
)

// ServiceCircuit tracks circuit breaker state for one service.
type ServiceCircuit struct {
	Service     string       `json:"service"`
	State       CircuitState `json:"state"`
	Attempts    int          `json:"attempts"`
	LastRestart time.Time    `json:"last_restart"`
	WindowStart time.Time    `json:"window_start"`
	TrippedAt   time.Time    `json:"tripped_at,omitempty"`
}

// Event records a watchdog action.
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Action    string    `json:"action"` // restart, circuit_open, circuit_reset, escalate
	Detail    string    `json:"detail"`
}

// Status holds the current watchdog status.
type Status struct {
	Running    bool             `json:"running"`
	Circuits   []ServiceCircuit `json:"circuits"`
	EventCount int              `json:"event_count"`
	Since      time.Time        `json:"since"`
}

// Watchdog monitors container health and restarts unhealthy services.
type Watchdog struct {
	cfg      Config
	docker   health.DockerClient
	mu       sync.Mutex
	circuits map[string]*ServiceCircuit
	events   []Event
	started  time.Time
	running  bool
}

// New creates a new Watchdog instance.
func New(cfg Config, docker health.DockerClient) *Watchdog {
	return &Watchdog{
		cfg:      cfg,
		docker:   docker,
		circuits: make(map[string]*ServiceCircuit),
	}
}

// Start begins the watchdog monitoring loop.
func (w *Watchdog) Start(ctx context.Context) {
	w.mu.Lock()
	w.running = true
	w.started = time.Now()
	w.mu.Unlock()

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.mu.Lock()
			w.running = false
			w.mu.Unlock()
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

// GetStatus returns the current watchdog status.
func (w *Watchdog) GetStatus() Status {
	w.mu.Lock()
	defer w.mu.Unlock()

	circuits := make([]ServiceCircuit, 0, len(w.circuits))
	for _, c := range w.circuits {
		circuits = append(circuits, *c)
	}

	return Status{
		Running:    w.running,
		Circuits:   circuits,
		EventCount: len(w.events),
		Since:      w.started,
	}
}

// ResetBreakers resets all circuit breakers to closed state.
func (w *Watchdog) ResetBreakers() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	count := 0
	for _, c := range w.circuits {
		if c.State == CircuitOpen {
			c.State = CircuitClosed
			c.Attempts = 0
			c.TrippedAt = time.Time{}
			count++
			w.events = append(w.events, Event{
				Timestamp: time.Now(),
				Service:   c.Service,
				Action:    "circuit_reset",
				Detail:    "manually reset",
			})
		}
	}
	return count
}

// GetHistory returns watchdog events, optionally filtered by duration.
func (w *Watchdog) GetHistory(since time.Duration) []Event {
	w.mu.Lock()
	defer w.mu.Unlock()

	if since <= 0 {
		result := make([]Event, len(w.events))
		copy(result, w.events)
		return result
	}

	cutoff := time.Now().Add(-since)
	var filtered []Event
	for _, e := range w.events {
		if e.Timestamp.After(cutoff) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (w *Watchdog) poll(ctx context.Context) {
	containers, err := w.docker.ContainerList(ctx, nil)
	if err != nil {
		return
	}

	for _, c := range containers {
		info, err := w.docker.ContainerInspect(ctx, c.ID)
		if err != nil {
			continue
		}

		if info.Health != "unhealthy" {
			// Service recovered, reset window if circuit was closed
			w.mu.Lock()
			if circuit, ok := w.circuits[c.Service]; ok && circuit.State == CircuitClosed {
				circuit.Attempts = 0
			}
			w.mu.Unlock()
			continue
		}

		w.handleUnhealthy(ctx, c)
	}
}

func (w *Watchdog) handleUnhealthy(ctx context.Context, c health.RestartContainer) {
	w.mu.Lock()
	circuit, ok := w.circuits[c.Service]
	if !ok {
		circuit = &ServiceCircuit{
			Service:     c.Service,
			State:       CircuitClosed,
			WindowStart: time.Now(),
		}
		w.circuits[c.Service] = circuit
	}

	// Check if window has expired and reset
	if time.Since(circuit.WindowStart) > w.cfg.CircuitBreakerWindow {
		circuit.Attempts = 0
		circuit.WindowStart = time.Now()
		circuit.State = CircuitClosed
	}

	if circuit.State == CircuitOpen {
		w.mu.Unlock()
		return
	}

	if circuit.Attempts >= w.cfg.CircuitBreakerAttempts {
		circuit.State = CircuitOpen
		circuit.TrippedAt = time.Now()
		w.events = append(w.events, Event{
			Timestamp: time.Now(),
			Service:   c.Service,
			Action:    "circuit_open",
			Detail:    fmt.Sprintf("tripped after %d attempts in %s", w.cfg.CircuitBreakerAttempts, w.cfg.CircuitBreakerWindow),
		})
		w.mu.Unlock()
		return
	}

	circuit.Attempts++
	attempt := circuit.Attempts
	w.mu.Unlock()

	// Restart the container
	err := w.docker.ContainerRestart(ctx, c.ID, 0)

	w.mu.Lock()
	circuit.LastRestart = time.Now()
	action := "restart"
	detail := fmt.Sprintf("attempt %d/%d", attempt, w.cfg.CircuitBreakerAttempts)
	if err != nil {
		detail += fmt.Sprintf(" (failed: %v)", err)
	}
	w.events = append(w.events, Event{
		Timestamp: time.Now(),
		Service:   c.Service,
		Action:    action,
		Detail:    detail,
	})
	w.mu.Unlock()
}

// SaveEvents persists watchdog events to a JSONL file.
func (w *Watchdog) SaveEvents(dir string) error {
	w.mu.Lock()
	events := make([]Event, len(w.events))
	copy(events, w.events)
	w.mu.Unlock()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(dir, "watchdog-events.jsonl"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, e := range events {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return nil
}
