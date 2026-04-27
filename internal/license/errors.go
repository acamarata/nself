// Package license — errors.go defines typed license errors with friendly
// user-facing messages. Each error carries a stable code (for programmatic
// matching), a human message (printed verbatim by the CLI), and an optional
// wrapped underlying error.
//
// Use IsLicenseError to extract a *Error from any error chain.
package license

import (
	"errors"
	"fmt"
)

// Error codes. Stable identifiers safe for scripts to match against.
const (
	CodeNotFound         = "LICENSE_NOT_FOUND"
	CodeExpired          = "LICENSE_EXPIRED"
	CodeRevoked          = "LICENSE_REVOKED"
	CodeInvalidSignature = "LICENSE_INVALID_SIGNATURE"
	CodeFailClosed       = "LICENSE_FAIL_CLOSED"
	CodeInsufficientTier = "LICENSE_INSUFFICIENT_TIER"
	CodeSlotExhausted    = "LICENSE_SLOT_EXHAUSTED"
)

// Error is the typed license error returned by the validation flow.
// Message is printed to the user verbatim. UserAction is a short next-step
// suggestion (already embedded in Message for the predefined errors below;
// kept separate so callers may format their own UI).
type Error struct {
	Code       string
	Message    string
	UserAction string
	Wrapped    error
}

// Error returns the user-facing message.
func (e *Error) Error() string {
	return e.Message
}

// Unwrap returns the wrapped underlying error, if any.
func (e *Error) Unwrap() error {
	return e.Wrapped
}

// IsLicenseError reports whether err is or wraps a *Error and returns it.
func IsLicenseError(err error) (*Error, bool) {
	if err == nil {
		return nil, false
	}
	var le *Error
	if errors.As(err, &le) {
		return le, true
	}
	return nil, false
}

// ErrLicenseNotFound is returned when no license key is configured.
var ErrLicenseNotFound = &Error{
	Code:       CodeNotFound,
	Message:    "No license found. Set one with: nself license set <key>",
	UserAction: "Run: nself license set <key>",
}

// NewExpiredError builds a license-expired error. expiredOn should be a
// formatted date string such as "2026-01-15".
func NewExpiredError(expiredOn string) *Error {
	return &Error{
		Code:       CodeExpired,
		Message:    fmt.Sprintf("Your license expired on %s. Renew at https://nself.org/account/billing", expiredOn),
		UserAction: "Renew at https://nself.org/account/billing",
	}
}

// NewRevokedError builds a license-revoked error with a human reason.
func NewRevokedError(reason string) *Error {
	if reason == "" {
		reason = "unspecified"
	}
	return &Error{
		Code:       CodeRevoked,
		Message:    fmt.Sprintf("Your license has been revoked. Reason: %s. Contact support@nself.org", reason),
		UserAction: "Contact support@nself.org",
	}
}

// ErrLicenseInvalidSignature is returned when the cached license signature
// does not validate against the public keys (likely tampered).
var ErrLicenseInvalidSignature = &Error{
	Code:       CodeInvalidSignature,
	Message:    "License signature invalid, likely tampered. Re-fetch with: nself license refresh",
	UserAction: "Run: nself license refresh",
}

// ErrLicenseFailClosed is returned when offline validation cannot proceed
// because the cached entry is older than the fail-closed TTL.
var ErrLicenseFailClosed = &Error{
	Code:       CodeFailClosed,
	Message:    "Cannot verify license offline (cache too stale). Reconnect to the internet, then run: nself license refresh",
	UserAction: "Reconnect to the internet and run: nself license refresh",
}

// NewInsufficientTierError builds an insufficient-tier error for a plugin
// install attempt. plugin is the plugin name, required is the required tier,
// current is the user's current tier.
func NewInsufficientTierError(plugin, required, current string) *Error {
	return &Error{
		Code:       CodeInsufficientTier,
		Message:    fmt.Sprintf("Plugin %s requires %s. You have %s. Upgrade at https://nself.org/pricing", plugin, required, current),
		UserAction: "Upgrade at https://nself.org/pricing",
	}
}

// NewSlotExhaustedError builds a slot-exhausted error. used and limit are
// the current and maximum activation counts.
func NewSlotExhaustedError(used, limit int) *Error {
	return &Error{
		Code:       CodeSlotExhausted,
		Message:    fmt.Sprintf("Activation limit reached: %d/%d machines. Deactivate one at https://nself.org/account/licenses", used, limit),
		UserAction: "Deactivate a machine at https://nself.org/account/licenses",
	}
}
