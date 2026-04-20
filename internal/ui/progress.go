package ui

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

// FirstRunProgress prints a first-run progress note to stdout.
// It outputs a static one-liner in non-TTY/quiet mode and a timed line in TTY mode.
// Returns a done function that must be called when the pull is complete.
func FirstRunProgress(quiet bool) func() {
	if quiet || !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Println("First run: pulling Docker images (this takes 1-3 minutes)...")
		return func() {}
	}

	done := make(chan struct{})
	start := time.Now()

	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		count := 0
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				count++
				elapsed := int(time.Since(start).Seconds())
				fmt.Printf("\r%s Pulling images [%ds elapsed] — first run takes 1-3 minutes...",
					C(Blue, iconGear), elapsed)
			}
		}
	}()

	return func() {
		close(done)
		// Clear the progress line.
		fmt.Print("\r\033[K")
		fmt.Printf("%s Docker images ready\n", C(Green, IconSuccess))
	}
}
