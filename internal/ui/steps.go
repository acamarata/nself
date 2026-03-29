package ui

import "fmt"

// Step prints a step counter line: ⚙ Step N/M - message.
func Step(current, total int, message string) {
	fmt.Printf("\n%s %s ─ %s\n",
		C(Blue, iconGear),
		C(Bold, fmt.Sprintf("Step %d/%d", current, total)),
		message)
}
