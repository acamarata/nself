// S69-T05: doctor check for ai+moderation wiring on public-bound deployments.
//
// If the ai plugin is loaded AND the deployment binding is non-loopback AND
// the moderation plugin is NOT loaded, this emits a WARN.
// Severity is WARN (not FAIL) because self-hosted single-user setups are legitimate.
package doctor

import (
	"context"
	"os"
	"strings"
)

// CheckModerationWired implements S69-T05: warns when the ai plugin is loaded on a
// public-bound deployment without the moderation plugin also loaded.
//
// Detection logic:
//   - ai plugin loaded:         NSELF_AI_LOADED=1  OR  PLUGIN_AI_INTERNAL_URL non-empty
//   - moderation plugin loaded: NSELF_MODERATION_LOADED=1  OR  PLUGIN_MODERATION_INTERNAL_URL non-empty
//   - public bind:              NSELF_BIND_ADDRESS is NOT 127.x / localhost / ::1 / "" (empty = loopback default)
func CheckModerationWired(_ context.Context) CheckResult {
	const name = "ai+moderation wiring (S69-T05)"
	const section = "ai-safety"

	if !isAIPluginLoaded() {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "pass",
			Message: "ai plugin not loaded — check not applicable",
		}
	}

	bindAddr := os.Getenv("NSELF_BIND_ADDRESS")
	if isLoopbackBind(bindAddr) {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "pass",
			Message: "ai plugin on loopback binding — moderation optional for single-user setups",
		}
	}

	if isModerationPluginLoaded() {
		return CheckResult{
			Section: section,
			Name:    name,
			Status:  "pass",
			Message: "ai plugin loaded with moderation plugin on public binding",
		}
	}

	return CheckResult{
		Section: section,
		Name:    name,
		Status:  "warn",
		Message: "moderation not wired on public-bound deployment with ai loaded — " +
			"self-hosted single-user is fine; multi-user deployments should install the moderation plugin",
		FixCmd: "nself plugin install moderation",
	}
}

func isAIPluginLoaded() bool {
	return os.Getenv("NSELF_AI_LOADED") == "1" || os.Getenv("PLUGIN_AI_INTERNAL_URL") != ""
}

func isModerationPluginLoaded() bool {
	return os.Getenv("NSELF_MODERATION_LOADED") == "1" || os.Getenv("PLUGIN_MODERATION_INTERNAL_URL") != ""
}

// isLoopbackBind returns true when addr is empty (default loopback), "127.x.x.x", "localhost", or "::1".
func isLoopbackBind(addr string) bool {
	addr = strings.TrimSpace(strings.ToLower(addr))
	switch addr {
	case "", "127.0.0.1", "localhost", "::1":
		return true
	}
	return strings.HasPrefix(addr, "127.")
}
