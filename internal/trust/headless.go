package trust

import "os"

// isHeadless reports whether the CLI is running in a headless context where
// macOS admin prompts (osascript ... with administrator privileges) must be
// suppressed entirely. Set NSELF_HEADLESS=1 in CI, automation, container
// builds, and any scripted workflow that cannot satisfy a graphical password
// dialog.
//
// When true:
//   - The trust package skips every osascript admin invocation.
//   - Calls return without error if the requested state is already configured.
//   - Calls log a clear message instructing the operator to perform the
//     admin-only step manually before re-running in headless mode.
//
// Accepted truthy values: "1", "true", "yes", "on" (case-insensitive).
func isHeadless() bool {
	v := os.Getenv("NSELF_HEADLESS")
	switch v {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes", "on", "ON", "On":
		return true
	}
	return false
}
