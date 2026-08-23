// Package errs (exit codes) defines the canonical exit-code contract for the
// nSelf CLI. Every command path that can fail must map its failure to one of
// these codes so wrappers (CI runners, schedulers, ops scripts) can branch on
// outcome class without parsing stderr.
//
// The codes are intentionally small and stable. Adding a new class requires a
// release-plan note. Re-using an existing class is preferred over inventing a
// new one.
package errs

import "errors"

// Canonical exit codes. Must match the contract documented in
// .claude/docs/standards/exit-codes.md and surfaced via 'nself --help' in
// the EXIT CODES section.
const (
	// ExitOK indicates successful completion.
	ExitOK = 0

	// ExitUserError indicates the caller supplied invalid input: bad flag
	// value, missing required argument, malformed env, unknown subcommand.
	// The fix is on the user side.
	ExitUserError = 1

	// ExitInfraError indicates a downstream infrastructure failure: docker
	// daemon down, port collision, network unreachable, disk full, plugin
	// runtime crash. Retrying after fixing the host condition may succeed.
	ExitInfraError = 2

	// ExitAuthError indicates an authentication or authorization failure:
	// invalid license key, expired key, tier insufficient for the requested
	// plugin, missing OAuth credentials.
	ExitAuthError = 3

	// ExitDestructiveBlocked indicates a destructive action was refused by
	// the safety guards: missing --force on prod, force_local mismatch,
	// confirmation prompt declined.
	ExitDestructiveBlocked = 4
)

// ExitCodeFor classifies a top-level error into one of the canonical exit
// codes. Used by main() to map RunE returns into process exit status.
//
// Classification order:
//  1. Auth-class sentinels in this package (license errors).
//  2. Destructive-blocked sentinels (force_local guard, confirmation cancel).
//     Callers in internal/confirm import errs implicitly via errors.Is.
//  3. Infra-class sentinels (docker, db, ports).
//  4. Everything else defaults to user-error (1).
//
// Returning ExitOK is reserved for nil errors and never inferred from
// classification — the caller must check err == nil first.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}

	// An explicit ExitError (or anything else exposing ExitCode) wins over
	// classification: the command already decided its exit status.
	var coder interface{ ExitCode() int }
	if errors.As(err, &coder) {
		return coder.ExitCode()
	}

	// Classify by sentinel error matching. Order matters: most specific
	// first.
	switch {
	case isAuthError(err):
		return ExitAuthError
	case isDestructiveBlocked(err):
		return ExitDestructiveBlocked
	case isInfraError(err):
		return ExitInfraError
	}

	return ExitUserError
}

// isAuthError reports whether err matches an auth-class sentinel.
func isAuthError(err error) bool {
	for _, target := range []error{
		ErrInvalidLicenseKey,
		ErrLicenseTierTooLow,
		ErrLicenseExpired,
		ErrLicenseNetworkUnavailable,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// isInfraError reports whether err matches an infra-class sentinel.
func isInfraError(err error) bool {
	for _, target := range []error{
		ErrDockerNotRunning,
		ErrDockerNotInstalled,
		ErrComposeNotFound,
		ErrPortConflict,
		ErrDatabaseNotRunning,
		ErrServiceUnhealthy,
		ErrHealthTimeout,
		ErrServiceNotFound,
		ErrSSLGenerationFailed,
		ErrBackupFailed,
		ErrBackupNotFound,
		ErrBackupVerifyFailed,
		ErrBackupRestoreFailed,
		ErrWALArchiveFailed,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// isDestructiveBlocked reports whether err signals that a destructive action
// was refused by a safety gate.
//
// We do not import internal/confirm here to avoid an import cycle (confirm
// imports errs implicitly through codes.go). Instead, callers in command
// glue map confirm.ErrForceLocalRequired and confirm.ErrDestructionCanceled
// onto a sentinel in this package via wrapping with %w against
// ErrDestructiveBlocked.
func isDestructiveBlocked(err error) bool {
	return errors.Is(err, ErrDestructiveBlocked)
}

// ErrDestructiveBlocked is the package-level sentinel that command glue can
// wrap with fmt.Errorf("...: %w", errs.ErrDestructiveBlocked) so that
// ExitCodeFor classifies the failure as exit code 4 without an import cycle
// through internal/confirm.
var ErrDestructiveBlocked = sentinel("destructive action blocked by safety gate")

// sentinel returns a plain error usable as an errors.Is target.
type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

func sentinel(s string) error { return sentinelErr(s) }
