package webhook

import (
	"sync"
	"time"
)

// CircuitState represents the state of an endpoint's circuit breaker.
type CircuitState string

const (
	// CircuitClosed is the normal operating state — deliveries proceed.
	CircuitClosed CircuitState = "closed"
	// CircuitOpen means the endpoint is failing; deliveries are skipped.
	CircuitOpen CircuitState = "open"
	// CircuitHalfOpen means a single probe is allowed to test recovery.
	CircuitHalfOpen CircuitState = "half-open"
)

const (
	// openThreshold is the number of consecutive failures that trips the circuit.
	openThreshold = 5
	// probeInterval is how long an open circuit waits before sending a probe.
	probeInterval = 10 * time.Minute
	// closeThreshold is the number of consecutive successful probes needed to close.
	closeThreshold = 2
)

// Circuit is a per-endpoint circuit breaker.
// All methods are safe for concurrent use.
type Circuit struct {
	mu             sync.Mutex
	state          CircuitState
	consecutiveFail int
	successProbes  int
	openedAt       time.Time
}

// NewCircuit returns a circuit in the Closed state.
func NewCircuit() *Circuit {
	return &Circuit{state: CircuitClosed}
}

// State returns the current circuit state.
func (c *Circuit) State() CircuitState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.resolvedState()
}

// Allow reports whether a delivery attempt should proceed.
// Open circuits only allow attempts after the probe interval elapses (half-open transition).
func (c *Circuit) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.resolvedState() {
	case CircuitClosed:
		return true
	case CircuitHalfOpen:
		return true
	default: // open
		return false
	}
}

// RecordSuccess records a successful delivery.
func (c *Circuit) RecordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consecutiveFail = 0
	// Resolve half-open transition before switching on state.
	_ = c.resolvedState()
	switch c.state {
	case CircuitHalfOpen:
		c.successProbes++
		if c.successProbes >= closeThreshold {
			c.state = CircuitClosed
			c.successProbes = 0
		}
	default:
		c.state = CircuitClosed
	}
}

// RecordFailure records a failed delivery. After openThreshold consecutive failures
// the circuit trips to Open.
func (c *Circuit) RecordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.successProbes = 0
	c.consecutiveFail++
	if c.consecutiveFail >= openThreshold {
		c.state = CircuitOpen
		c.openedAt = time.Now()
	}
}

// resolvedState evaluates half-open transitions; must be called with mu held.
func (c *Circuit) resolvedState() CircuitState {
	if c.state == CircuitOpen && time.Since(c.openedAt) >= probeInterval {
		c.state = CircuitHalfOpen
	}
	return c.state
}
