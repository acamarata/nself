package ui

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// spinnerFrames contains the braille animation characters (0.1s per frame).
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner provides a goroutine-based terminal spinner with braille animation.
type Spinner struct {
	message string
	mu      sync.Mutex
	running bool
	done    chan struct{}
}

// NewSpinner creates a new Spinner with the given message.
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
	}
}

// Start begins the spinner animation in a background goroutine.
// In non-TTY environments, it prints "... message" once with no animation.
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	// Non-TTY: print static message and return without goroutine.
	if !stdoutIsTerminal() {
		fmt.Printf("... %s\n", s.message)
		return
	}

	s.running = true
	s.done = make(chan struct{})

	go func() {
		frame := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.mu.Lock()
				fmt.Printf("\r%s %s", C(Blue, spinnerFrames[frame]), s.message)
				s.mu.Unlock()
				frame = (frame + 1) % len(spinnerFrames)
			}
		}
	}()
}

// Stop halts the spinner goroutine and clears the current line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.done)
	// Clear the spinner line.
	fmt.Print("\r\033[K")
}

// Success stops the spinner and prints a green checkmark with the given message.
func (s *Spinner) Success(msg string) {
	s.Stop()
	fmt.Printf("%s %s\n", C(Green, IconSuccess), msg)
}

// Fail stops the spinner and prints a red cross with the given message to stderr.
func (s *Spinner) Fail(msg string) {
	s.Stop()
	fmt.Fprintf(os.Stderr, "%s %s\n", C(Red, IconFailure), msg)
}
