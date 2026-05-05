package confirm

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrForceLocalRequired is returned by RequireLocal when a command is executing
// against a non-local target without an explicit local opt-in (--local flag or
// NSELF_LOCAL=true env var).
var ErrForceLocalRequired = errors.New("force_local guard: command targets non-local environment")

// LocalIntent encodes how a caller declared its intent for a destructive
// command. Exactly one of WantLocal / WantRemote should be set.
type LocalIntent struct {
	// LocalFlag is true when the caller passed --local (or equivalent).
	LocalFlag bool

	// Target is the resolved deploy target string ("local", "staging",
	// "prod", "production", or any non-local label). Empty means caller
	// could not determine the target — treated as non-local for safety.
	Target string

	// CommandName is the user-facing command label (e.g. "deploy",
	// "db migrate", "stack restart") used in error messages.
	CommandName string
}

// targetIsLocal returns true for the canonical local target labels.
// Anything that does not parse as a known local label is treated as non-local
// and therefore subject to the guard.
func targetIsLocal(t string) bool {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "", "local", "localhost", "dev", "development":
		return true
	}
	return false
}

// envOptInLocal returns true when NSELF_LOCAL is explicitly set to a truthy
// value. This is the second valid opt-in pathway alongside an explicit --local
// flag.
func envOptInLocal() bool {
	v := os.Getenv("NSELF_LOCAL")
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// RequireLocal enforces that destructive or production-affecting commands
// only run against a non-local target when the caller has explicitly opted in
// (via --local flag OR NSELF_LOCAL=true env var) AND the target itself is
// genuinely local.
//
// The guard fires (returns ErrForceLocalRequired) when:
//   - intent.Target resolves to a non-local label, AND
//   - intent.LocalFlag is false, AND
//   - NSELF_LOCAL is not truthy in the environment.
//
// Callers that legitimately operate on staging or prod must NOT route through
// this guard — it exists specifically to catch accidental cloud actions when
// the operator's intent was local. Production deploy paths use their own
// --force / --confirm gates higher up the call stack.
func RequireLocal(intent LocalIntent) error {
	if targetIsLocal(intent.Target) {
		// Target is local — guard does not apply, regardless of flags.
		return nil
	}

	if intent.LocalFlag {
		return fmt.Errorf("%w: --local was passed but resolved target is %q. Refusing to apply local-intent command to a non-local target. Re-run without --local if cloud action is intended", ErrForceLocalRequired, intent.Target)
	}

	if envOptInLocal() {
		return fmt.Errorf("%w: NSELF_LOCAL=true is set but resolved target is %q. Refusing to apply local-intent command to a non-local target. Unset NSELF_LOCAL if cloud action is intended", ErrForceLocalRequired, intent.Target)
	}

	// No local opt-in, target is non-local — outside the guard's scope.
	// Caller's own production-target gates apply.
	return nil
}
