package ui

import (
	"os"
)

// ANSI color and style constants.
const (
	// Styles
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Underline = "\033[4m"

	// Foreground
	Black   = "\033[0;30m"
	Red     = "\033[0;31m"
	Green   = "\033[0;32m"
	Yellow  = "\033[0;33m"
	Blue    = "\033[0;34m"
	Magenta = "\033[0;35m"
	Cyan    = "\033[0;36m"
	White   = "\033[0;37m"

	// Bright (for high-visibility)
	BrightRed    = "\033[1;31m"
	BrightGreen  = "\033[1;32m"
	BrightYellow = "\033[1;33m"

	// Background (rare, for critical alerts)
	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgBlue   = "\033[44m"
)

// colorsEnabled is false when NO_COLOR env var is set or stdout is not a TTY.
var colorsEnabled = os.Getenv("NO_COLOR") == "" && stdoutIsTerminal()

// C wraps text in the given ANSI color code if colors are enabled.
// Returns plain text otherwise.
func C(color, text string) string {
	if !colorsEnabled {
		return text
	}
	return color + text + Reset
}
