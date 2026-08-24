package watchdog

// Purpose: status/history/reset/poll/unhealthy-handling and event-persistence methods on Watchdog, split out of watchdog.go's construction + Start path.
// Inputs: a *Watchdog constructed by New in watchdog.go.
// Outputs: status snapshots, circuit-breaker resets, event history, and persisted event files.
// Constraints: split out of watchdog.go as a pure move (CLI-R12); no behavior change.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nself-org/cli/internal/health"
)

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

// ResetBreakers resets all circuit breakers (including PERMANENT_OPEN) to closed state.
func (w *Watchdog) ResetBreakers() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	count := 0
	for _, c := range w.circuits {
		if c.State == CircuitOpen || c.State == CircuitPermanentOpen {
			c.State = CircuitClosed
			c.Attempts = 0
			c.ConsecutiveOpenWindows = 0
			c.TrippedAt = time.Time{}
			c.PermanentOpenAt = time.Time{}
			count++
			w.events = append(w.events, Event{
				Timestamp: time.Now(),
				Service:   c.Service,
				Action:    "circuit_reset",
				Detail:    "manually reset (all breakers)",
			})
		}
	}
	return count
}

// ResetService resets a single named service's circuit breaker to closed state.
// It clears PERMANENT_OPEN state and the consecutive-window counter.
// Returns false if the service has no tracked circuit.
func (w *Watchdog) ResetService(service, operator string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	c, ok := w.circuits[service]
	if !ok {
		return false
	}
	c.State = CircuitClosed
	c.Attempts = 0
	c.ConsecutiveOpenWindows = 0
	c.TrippedAt = time.Time{}
	c.PermanentOpenAt = time.Time{}
	w.events = append(w.events, Event{
		Timestamp: time.Now(),
		Service:   service,
		Action:    "circuit_reset",
		Detail:    fmt.Sprintf("manual reset by %s", operator),
	})
	return true
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
			// Service recovered: reset attempt counter and consecutive-open-window counter.
			// PERMANENT_OPEN is NOT cleared automatically — it requires an explicit operator reset.
			w.mu.Lock()
			if circuit, ok := w.circuits[c.Service]; ok && circuit.State == CircuitClosed {
				circuit.Attempts = 0
				circuit.ConsecutiveOpenWindows = 0
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

	// PERMANENT_OPEN: block all automated restarts; no automatic resets.
	if circuit.State == CircuitPermanentOpen {
		w.mu.Unlock()
		return
	}

	// Check if window has expired — roll the window.
	if time.Since(circuit.WindowStart) > w.cfg.CircuitBreakerWindow {
		circuit.Attempts = 0
		circuit.WindowStart = time.Now()
		// If the circuit was OPEN, count this as another consecutive open window.
		// Do NOT automatically reset to CLOSED; check the permanent-open threshold.
		if circuit.State == CircuitOpen {
			circuit.ConsecutiveOpenWindows++
			if circuit.ConsecutiveOpenWindows >= w.cfg.PermanentOpenThreshold {
				circuit.State = CircuitPermanentOpen
				circuit.PermanentOpenAt = time.Now()
				permDetail := fmt.Sprintf("permanently open after %d consecutive open windows (threshold %d)",
					circuit.ConsecutiveOpenWindows, w.cfg.PermanentOpenThreshold)
				w.events = append(w.events, Event{
					Timestamp: time.Now(),
					Service:   c.Service,
					Action:    "circuit_permanent_open",
					Detail:    permDetail,
				})
				w.mu.Unlock()
				// Self-healing is abandoned for this service; page a human.
				w.escalate(c.Service, "critical", permDetail)
				return
			}
			// Below threshold: stay OPEN but roll the window counter.
		} else {
			// Was CLOSED (rare: window expired while healthy counter reset hadn't triggered).
			circuit.ConsecutiveOpenWindows = 0
			circuit.State = CircuitClosed
		}
	}

	if circuit.State == CircuitOpen {
		w.mu.Unlock()
		return
	}

	if circuit.Attempts >= w.cfg.CircuitBreakerAttempts {
		circuit.State = CircuitOpen
		circuit.TrippedAt = time.Now()
		openDetail := fmt.Sprintf("tripped after %d attempts in %s", w.cfg.CircuitBreakerAttempts, w.cfg.CircuitBreakerWindow)
		w.events = append(w.events, Event{
			Timestamp: time.Now(),
			Service:   c.Service,
			Action:    "circuit_open",
			Detail:    openDetail,
		})
		w.mu.Unlock()
		// Automated restarts exhausted; the breaker is open. Alert the operator.
		w.escalate(c.Service, "warning", openDetail)
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
