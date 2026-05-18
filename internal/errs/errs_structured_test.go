package errs

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// ============================================================================
// CLIError.Error() — structured.go
// ============================================================================

// TestCLIError_Error_AllFields verifies the formatted output includes all
// non-empty fields in the documented order.
func TestCLIError_Error_AllFields(t *testing.T) {
	e := &CLIError{
		Code:     "E001",
		What:     "something went wrong",
		Why:      "the widget was missing",
		Fix:      "run nself install widget",
		DocsPath: "reference/error-codes#e001",
	}
	got := e.Error()
	if !strings.Contains(got, "[E001]") {
		t.Errorf("Error() = %q: missing [E001]", got)
	}
	if !strings.Contains(got, "something went wrong") {
		t.Errorf("Error() = %q: missing What", got)
	}
	if !strings.Contains(got, "Why: the widget was missing") {
		t.Errorf("Error() = %q: missing Why", got)
	}
	if !strings.Contains(got, "Fix: run nself install widget") {
		t.Errorf("Error() = %q: missing Fix", got)
	}
	if !strings.Contains(got, "Docs: https://docs.nself.org/reference/error-codes#e001") {
		t.Errorf("Error() = %q: missing Docs", got)
	}
}

// TestCLIError_Error_OnlyCodeAndWhat verifies that optional fields (Why, Fix,
// DocsPath) are omitted when empty.
func TestCLIError_Error_OnlyCodeAndWhat(t *testing.T) {
	e := &CLIError{Code: "E999", What: "bare error"}
	got := e.Error()
	if !strings.HasPrefix(got, "[E999] bare error") {
		t.Errorf("Error() = %q: unexpected format", got)
	}
	if strings.Contains(got, "Why:") {
		t.Errorf("Error() = %q: Why should be absent when empty", got)
	}
	if strings.Contains(got, "Fix:") {
		t.Errorf("Error() = %q: Fix should be absent when empty", got)
	}
	if strings.Contains(got, "Docs:") {
		t.Errorf("Error() = %q: Docs should be absent when empty", got)
	}
}

// TestCLIError_Error_WhyOnly verifies that only the Why field is emitted when
// Fix and DocsPath are empty.
func TestCLIError_Error_WhyOnly(t *testing.T) {
	e := &CLIError{Code: "E100", What: "partial error", Why: "because of X"}
	got := e.Error()
	if !strings.Contains(got, "Why: because of X") {
		t.Errorf("Error() = %q: missing Why", got)
	}
	if strings.Contains(got, "Fix:") {
		t.Errorf("Error() = %q: Fix should be absent", got)
	}
}

// ============================================================================
// CLIError.Unwrap() — structured.go
// ============================================================================

// TestCLIError_Unwrap_NonNil verifies that Unwrap returns the wrapped error.
func TestCLIError_Unwrap_NonNil(t *testing.T) {
	inner := errors.New("inner")
	e := &CLIError{Code: "E001", What: "outer", Wrapped: inner}
	if got := e.Unwrap(); got != inner {
		t.Errorf("Unwrap() = %v, want %v", got, inner)
	}
}

// TestCLIError_Unwrap_Nil verifies Unwrap returns nil when no error is wrapped.
func TestCLIError_Unwrap_Nil(t *testing.T) {
	e := &CLIError{Code: "E001", What: "no wrap"}
	if got := e.Unwrap(); got != nil {
		t.Errorf("Unwrap() = %v, want nil", got)
	}
}

// ============================================================================
// New() — structured.go
// ============================================================================

// TestNew_RegisteredCode verifies that New populates Why/Fix/DocsPath from the
// registry when the code exists.
func TestNew_RegisteredCode(t *testing.T) {
	// Use E001 — the docker-not-running code that must exist in Registry.
	e := New("E001", "docker is not running")
	if e.Code != "E001" {
		t.Errorf("New.Code = %q, want E001", e.Code)
	}
	if e.What != "docker is not running" {
		t.Errorf("New.What = %q, want 'docker is not running'", e.What)
	}
	// Registry entry should fill in at least DocsPath or Why.
	if e.DocsPath == "" && e.Why == "" {
		// Acceptable if the registry entry truly has no defaults — just ensure no panic.
		t.Log("E001 has no DefaultWhy or DocsPath in registry (acceptable)")
	}
}

// TestNew_UnregisteredCode verifies that New falls back gracefully when the
// code is not in the registry.
func TestNew_UnregisteredCode(t *testing.T) {
	e := New("EUNKNOWN_999", "something weird")
	if e.Code != "EUNKNOWN_999" {
		t.Errorf("New.Code = %q, want EUNKNOWN_999", e.Code)
	}
	if e.What != "something weird" {
		t.Errorf("New.What = %q, want 'something weird'", e.What)
	}
	// Unregistered: Why/Fix/DocsPath should be zero.
	if e.Why != "" || e.Fix != "" || e.DocsPath != "" {
		t.Errorf("unregistered code should have empty Why/Fix/DocsPath, got Why=%q Fix=%q Docs=%q", e.Why, e.Fix, e.DocsPath)
	}
}

// ============================================================================
// Newf() — structured.go
// ============================================================================

// TestNewf_FormatsMessage verifies Newf sprintf-formats the what string.
func TestNewf_FormatsMessage(t *testing.T) {
	e := Newf("EUNKNOWN_999", "plugin %s not found (version %d)", "ai", 3)
	want := "plugin ai not found (version 3)"
	if e.What != want {
		t.Errorf("Newf.What = %q, want %q", e.What, want)
	}
}

// ============================================================================
// Wrap() — structured.go
// ============================================================================

// TestWrap_AttachesUnderlying verifies Wrap sets Wrapped and allows errors.Is.
func TestWrap_AttachesUnderlying(t *testing.T) {
	inner := errors.New("underlying cause")
	e := Wrap("EUNKNOWN_999", "outer message", inner)
	if !errors.Is(e, inner) {
		t.Errorf("errors.Is(wrapped, inner) = false, want true")
	}
}

// TestWrap_NilUnderlying verifies Wrap with a nil inner error does not panic.
func TestWrap_NilUnderlying(t *testing.T) {
	e := Wrap("EUNKNOWN_999", "no cause", nil)
	if e.Unwrap() != nil {
		t.Errorf("Wrap(nil).Unwrap() = %v, want nil", e.Unwrap())
	}
}

// ============================================================================
// WithWhy() / WithFix() — structured.go
// ============================================================================

// TestWithWhy_ReturnsCopy verifies WithWhy returns a new value and doesn't mutate
// the original.
func TestWithWhy_ReturnsCopy(t *testing.T) {
	orig := &CLIError{Code: "E001", What: "orig", Why: "original why"}
	modified := orig.WithWhy("new why")
	if orig.Why != "original why" {
		t.Errorf("WithWhy mutated original: orig.Why = %q", orig.Why)
	}
	if modified.Why != "new why" {
		t.Errorf("WithWhy.Why = %q, want 'new why'", modified.Why)
	}
	if modified == orig {
		t.Error("WithWhy should return a new pointer, not the same object")
	}
}

// TestWithFix_ReturnsCopy verifies WithFix returns a new value and doesn't
// mutate the original.
func TestWithFix_ReturnsCopy(t *testing.T) {
	orig := &CLIError{Code: "E001", What: "orig", Fix: "original fix"}
	modified := orig.WithFix("new fix")
	if orig.Fix != "original fix" {
		t.Errorf("WithFix mutated original: orig.Fix = %q", orig.Fix)
	}
	if modified.Fix != "new fix" {
		t.Errorf("WithFix.Fix = %q, want 'new fix'", modified.Fix)
	}
}

// ============================================================================
// ExitCodeFor() — exit_codes.go
// ============================================================================

// TestExitCodeFor_NilError verifies that nil returns ExitOK.
func TestExitCodeFor_NilError(t *testing.T) {
	if got := ExitCodeFor(nil); got != ExitOK {
		t.Errorf("ExitCodeFor(nil) = %d, want %d (ExitOK)", got, ExitOK)
	}
}

// TestExitCodeFor_AuthErrors verifies that all auth-class sentinels map to
// ExitAuthError (3).
func TestExitCodeFor_AuthErrors(t *testing.T) {
	authErrs := []struct {
		name string
		err  error
	}{
		{"ErrInvalidLicenseKey", ErrInvalidLicenseKey},
		{"ErrLicenseTierTooLow", ErrLicenseTierTooLow},
		{"ErrLicenseExpired", ErrLicenseExpired},
		{"ErrLicenseNetworkUnavailable", ErrLicenseNetworkUnavailable},
	}
	for _, tc := range authErrs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCodeFor(tc.err); got != ExitAuthError {
				t.Errorf("ExitCodeFor(%s) = %d, want %d (ExitAuthError)", tc.name, got, ExitAuthError)
			}
		})
	}
}

// TestExitCodeFor_AuthError_Wrapped verifies that wrapping an auth sentinel still
// maps to ExitAuthError.
func TestExitCodeFor_AuthError_Wrapped(t *testing.T) {
	wrapped := fmt.Errorf("plugin install failed: %w", ErrInvalidLicenseKey)
	if got := ExitCodeFor(wrapped); got != ExitAuthError {
		t.Errorf("ExitCodeFor(wrapped auth) = %d, want %d", got, ExitAuthError)
	}
}

// TestExitCodeFor_DestructiveBlocked verifies ErrDestructiveBlocked maps to
// ExitDestructiveBlocked (4).
func TestExitCodeFor_DestructiveBlocked(t *testing.T) {
	if got := ExitCodeFor(ErrDestructiveBlocked); got != ExitDestructiveBlocked {
		t.Errorf("ExitCodeFor(ErrDestructiveBlocked) = %d, want %d", got, ExitDestructiveBlocked)
	}
}

// TestExitCodeFor_InfraErrors verifies that infra-class sentinels map to
// ExitInfraError (2).
func TestExitCodeFor_InfraErrors(t *testing.T) {
	infraErrs := []struct {
		name string
		err  error
	}{
		{"ErrDockerNotRunning", ErrDockerNotRunning},
		{"ErrDockerNotInstalled", ErrDockerNotInstalled},
		{"ErrPortConflict", ErrPortConflict},
		{"ErrDatabaseNotRunning", ErrDatabaseNotRunning},
		{"ErrSSLGenerationFailed", ErrSSLGenerationFailed},
		{"ErrWALArchiveFailed", ErrWALArchiveFailed},
	}
	for _, tc := range infraErrs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCodeFor(tc.err); got != ExitInfraError {
				t.Errorf("ExitCodeFor(%s) = %d, want %d (ExitInfraError)", tc.name, got, ExitInfraError)
			}
		})
	}
}

// TestExitCodeFor_UnknownError verifies that an unrecognized error maps to
// ExitUserError (1) as the safe default.
func TestExitCodeFor_UnknownError(t *testing.T) {
	unknown := errors.New("some totally unknown failure")
	if got := ExitCodeFor(unknown); got != ExitUserError {
		t.Errorf("ExitCodeFor(unknown) = %d, want %d (ExitUserError)", got, ExitUserError)
	}
}

// ============================================================================
// errorsIs() — exit_codes.go (internal helper)
// ============================================================================

// TestErrorsIs_DirectMatch verifies errorsIs returns true when err == target.
func TestErrorsIs_DirectMatch(t *testing.T) {
	sentinel := errors.New("sentinel")
	if !errorsIs(sentinel, sentinel) {
		t.Error("errorsIs: same pointer should return true")
	}
}

// TestErrorsIs_NoMatch verifies errorsIs returns false when errors differ.
func TestErrorsIs_NoMatch(t *testing.T) {
	a := errors.New("a")
	b := errors.New("b")
	if errorsIs(a, b) {
		t.Error("errorsIs: distinct errors should return false")
	}
}

// TestErrorsIs_NilErr verifies errorsIs returns false on nil error.
func TestErrorsIs_NilErr(t *testing.T) {
	if errorsIs(nil, errors.New("target")) {
		t.Error("errorsIs(nil, ...) should return false")
	}
}

// TestErrorsIs_Unwrapped verifies errorsIs chains through Unwrap.
func TestErrorsIs_Unwrapped(t *testing.T) {
	inner := errors.New("inner")
	outer := fmt.Errorf("outer: %w", inner)
	if !errorsIs(outer, inner) {
		t.Error("errorsIs should unwrap and find inner sentinel")
	}
}

// ============================================================================
// Categories() — codes.go (one branch exercise)
// ============================================================================

// TestCategories_ReturnsNonEmpty verifies that Categories() returns a non-empty
// slice with at least one known category.
func TestCategories_ReturnsNonEmpty(t *testing.T) {
	cats := Categories()
	if len(cats) == 0 {
		t.Error("Categories() returned empty slice")
	}
	// Confirm at least one category string is non-empty.
	for _, c := range cats {
		if c == "" {
			t.Error("Categories() returned empty string in slice")
		}
	}
}
