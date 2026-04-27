package license

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrLicenseNotFound(t *testing.T) {
	if ErrLicenseNotFound.Code != CodeNotFound {
		t.Errorf("code mismatch: got %q want %q", ErrLicenseNotFound.Code, CodeNotFound)
	}
	if !strings.Contains(ErrLicenseNotFound.Error(), "nself license set") {
		t.Errorf("message missing user action: %q", ErrLicenseNotFound.Error())
	}
}

func TestNewExpiredError(t *testing.T) {
	err := NewExpiredError("2026-01-15")
	if err.Code != CodeExpired {
		t.Errorf("code mismatch: got %q", err.Code)
	}
	if !strings.Contains(err.Error(), "2026-01-15") {
		t.Errorf("message missing date: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "https://nself.org/account/billing") {
		t.Errorf("message missing renewal URL: %q", err.Error())
	}
}

func TestNewRevokedError(t *testing.T) {
	err := NewRevokedError("payment failure")
	if err.Code != CodeRevoked {
		t.Errorf("code mismatch: got %q", err.Code)
	}
	if !strings.Contains(err.Error(), "payment failure") {
		t.Errorf("message missing reason: %q", err.Error())
	}
	// Empty reason gets a default.
	empty := NewRevokedError("")
	if !strings.Contains(empty.Error(), "unspecified") {
		t.Errorf("empty reason should yield 'unspecified': %q", empty.Error())
	}
}

func TestErrLicenseInvalidSignature(t *testing.T) {
	if ErrLicenseInvalidSignature.Code != CodeInvalidSignature {
		t.Errorf("code mismatch: got %q", ErrLicenseInvalidSignature.Code)
	}
	if !strings.Contains(ErrLicenseInvalidSignature.Error(), "nself license refresh") {
		t.Errorf("message missing refresh action: %q", ErrLicenseInvalidSignature.Error())
	}
}

func TestErrLicenseFailClosed(t *testing.T) {
	if ErrLicenseFailClosed.Code != CodeFailClosed {
		t.Errorf("code mismatch: got %q", ErrLicenseFailClosed.Code)
	}
	if !strings.Contains(ErrLicenseFailClosed.Error(), "Reconnect") {
		t.Errorf("message missing reconnect action: %q", ErrLicenseFailClosed.Error())
	}
}

func TestNewInsufficientTierError(t *testing.T) {
	err := NewInsufficientTierError("ai", "Pro", "Free")
	if err.Code != CodeInsufficientTier {
		t.Errorf("code mismatch: got %q", err.Code)
	}
	for _, want := range []string{"ai", "Pro", "Free", "https://nself.org/pricing"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message missing %q: %q", want, err.Error())
		}
	}
}

func TestNewSlotExhaustedError(t *testing.T) {
	err := NewSlotExhaustedError(3, 3)
	if err.Code != CodeSlotExhausted {
		t.Errorf("code mismatch: got %q", err.Code)
	}
	if !strings.Contains(err.Error(), "3/3") {
		t.Errorf("message missing slot ratio: %q", err.Error())
	}
}

func TestIsLicenseError(t *testing.T) {
	// Direct.
	if le, ok := IsLicenseError(ErrLicenseNotFound); !ok || le.Code != CodeNotFound {
		t.Errorf("direct match failed")
	}
	// Wrapped.
	wrapped := fmt.Errorf("validation failed: %w", ErrLicenseFailClosed)
	le, ok := IsLicenseError(wrapped)
	if !ok || le.Code != CodeFailClosed {
		t.Errorf("wrapped match failed: ok=%v code=%v", ok, le)
	}
	// Non-license error.
	if _, ok := IsLicenseError(errors.New("random")); ok {
		t.Errorf("non-license error matched")
	}
	// Nil.
	if _, ok := IsLicenseError(nil); ok {
		t.Errorf("nil matched")
	}
}

func TestUnwrap(t *testing.T) {
	inner := errors.New("inner cause")
	e := &Error{Code: CodeFailClosed, Message: "wrap test", Wrapped: inner}
	if !errors.Is(e, inner) {
		t.Errorf("errors.Is failed to traverse Wrapped")
	}
}
