package ui

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ProgressBar renders a terminal progress bar with percentage and elapsed time.
// It is safe to call from a single goroutine; updates are synchronous.
type ProgressBar struct {
	label   string
	total   int
	current int
	width   int
	quiet   bool
	isTTY   bool
	started time.Time
}

// NewProgressBar creates a ProgressBar with the given label and total steps.
// Width defaults to 40 characters of fill area.
func NewProgressBar(label string, total int, quiet bool) *ProgressBar {
	return &ProgressBar{
		label:   label,
		total:   total,
		width:   40,
		quiet:   quiet,
		isTTY:   stdoutIsTerminal(),
		started: time.Now(),
	}
}

// Set advances the bar to an absolute position (0..total).
func (p *ProgressBar) Set(n int) {
	p.current = n
	p.render()
}

// Inc increments the bar by one step.
func (p *ProgressBar) Inc() {
	p.current++
	p.render()
}

// Done marks the bar at 100% and prints a newline.
func (p *ProgressBar) Done() {
	p.current = p.total
	p.render()
	if !p.quiet && p.isTTY {
		fmt.Println()
	}
}

func (p *ProgressBar) render() {
	if p.quiet || !p.isTTY || p.total <= 0 {
		return
	}

	pct := float64(p.current) / float64(p.total)
	filled := int(pct * float64(p.width))
	if filled > p.width {
		filled = p.width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)
	elapsed := time.Since(p.started).Round(time.Second)
	fmt.Printf("\r%s [%s] %3.0f%%  %s  %s",
		C(Blue, p.label),
		C(Cyan, bar),
		pct*100,
		C(Dim, elapsed.String()),
		strings.Repeat(" ", 4), // trailing pad to clear stale chars
	)
}

// DockerPullProgress prints an animated Docker pull progress indicator.
// It returns a done-func that, when called, clears the line and prints completion.
func DockerPullProgress(quiet bool) func(error) {
	if quiet || !stdoutIsTerminal() {
		fmt.Println("Pulling Docker images — first run takes 1-3 minutes...")
		return func(err error) {
			if err == nil {
				fmt.Println("Docker images ready.")
			}
		}
	}

	done := make(chan struct{})
	start := time.Now()

	layers := []string{"postgres", "hasura", "auth", "nginx"}
	layerIdx := 0

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				elapsed := time.Since(start).Round(time.Second)
				layer := layers[layerIdx%len(layers)]
				layerIdx++
				fmt.Printf("\r%s Pulling %-12s  %s elapsed  (1-3 min on first run)%s",
					C(Blue, iconGear),
					layer+"...",
					elapsed,
					strings.Repeat(" ", 4),
				)
			}
		}
	}()

	return func(err error) {
		close(done)
		fmt.Print("\r\033[K")
		if err == nil {
			fmt.Printf("%s Docker images ready\n", C(Green, IconSuccess))
		} else {
			fmt.Fprintf(os.Stderr, "%s Docker pull incomplete: %v\n", C(Yellow, IconWarning), err)
		}
	}
}

// InitSteps renders a multi-step progress display for nself init.
// Each call to Next() advances one step and prints the current step label.
type InitSteps struct {
	steps   []string
	current int
	quiet   bool
}

// NewInitSteps creates an InitSteps tracker for the given ordered step labels.
func NewInitSteps(quiet bool, steps ...string) *InitSteps {
	return &InitSteps{steps: steps, quiet: quiet}
}

// Next advances to the next step, printing its label.
// Returns false when all steps are consumed.
func (s *InitSteps) Next() bool {
	if s.current >= len(s.steps) {
		return false
	}
	if !s.quiet {
		label := s.steps[s.current]
		fmt.Printf("  %s %s\n",
			C(Blue, fmt.Sprintf("[%d/%d]", s.current+1, len(s.steps))),
			label,
		)
	}
	s.current++
	return true
}

// Done prints the final "all done" summary line.
func (s *InitSteps) Done() {
	if !s.quiet {
		fmt.Printf("  %s All steps complete\n", C(Green, IconSuccess))
	}
}
