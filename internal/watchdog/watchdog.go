// Package watchdog implements self-healing container monitoring with circuit breaker.
package watchdog

import (
	"context"
	"log/slog"
	"os"
	"strconv"
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
	// PermanentOpenThreshold is the number of consecutive OPEN windows after which
	// the circuit transitions to PERMANENT_OPEN and stops all automated resets.
	// Reads from WATCHDOG_PERMANENT_OPEN_THRESHOLD env var; default 3.
	PermanentOpenThreshold int
}

// DefaultConfig returns watchdog configuration with defaults.
func DefaultConfig() Config {
	threshold := 3
	if v := os.Getenv("WATCHDOG_PERMANENT_OPEN_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			threshold = n
		}
	}
	return Config{
		Enabled:                true,
		CircuitBreakerAttempts: 3,
		CircuitBreakerWindow:   10 * time.Minute,
		PollInterval:           30 * time.Second,
		PermanentOpenThreshold: threshold,
	}
}

// CircuitState represents the state of a circuit breaker for a service.
type CircuitState string

const (
	CircuitClosed        CircuitState = "closed"         // healthy, restarts allowed
	CircuitOpen          CircuitState = "open"           // tripped, restarts blocked
	CircuitPermanentOpen CircuitState = "PERMANENT_OPEN" // permanently open; requires manual reset
)

// ServiceCircuit tracks circuit breaker state for one service.
type ServiceCircuit struct {
	Service                string       `json:"service"`
	State                  CircuitState `json:"state"`
	Attempts               int          `json:"attempts"`
	ConsecutiveOpenWindows int          `json:"consecutive_open_windows"`
	LastRestart            time.Time    `json:"last_restart"`
	WindowStart            time.Time    `json:"window_start"`
	TrippedAt              time.Time    `json:"tripped_at,omitempty"`
	PermanentOpenAt        time.Time    `json:"permanent_open_at,omitempty"`
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

// escalate pages a human operator when self-healing trips or is abandoned.
// It is a no-op (logged) when no escalation channel is configured, so a stack
// without Telegram/SMTP set up still records the breaker state without erroring.
func (w *Watchdog) escalate(service, severity, detail string) {
	cfg := LoadEscalationConfig()
	configured := (cfg.TelegramBotToken != "" && cfg.TelegramChatID != "") ||
		(cfg.SMTPHost != "" && cfg.SMTPFrom != "")
	if !configured {
		slog.Warn("watchdog: circuit breaker tripped but no escalation channel configured",
			"service", service, "severity", severity, "detail", detail)
		return
	}
	for _, err := range Escalate(service, severity, detail) {
		slog.Warn("watchdog: escalation channel failed", "service", service, "err", err)
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
