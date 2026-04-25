package ux

import (
	"errors"
	"strings"
	"testing"
)

// TestNew verifies that New constructs a UXError from the error registry.
func TestNew(t *testing.T) {
	t.Parallel()
	e := New("E001", "something went wrong")
	if e == nil {
		t.Fatal("New returned nil")
	}
	if e.Code != "E001" {
		t.Errorf("Code = %q; want E001", e.Code)
	}
	if e.What != "something went wrong" {
		t.Errorf("What = %q; want 'something went wrong'", e.What)
	}
}

// TestNewf verifies Newf applies format verbs correctly.
func TestNewf(t *testing.T) {
	t.Parallel()
	e := Newf("E002", "failed: %s", "disk full")
	if !strings.Contains(e.What, "disk full") {
		t.Errorf("What = %q; should contain 'disk full'", e.What)
	}
}

// TestWrap verifies that Wrap stores the underlying error.
func TestWrap(t *testing.T) {
	t.Parallel()
	inner := errors.New("root cause")
	e := Wrap("E003", "outer message", inner)
	if !errors.Is(e, inner) {
		t.Errorf("errors.Is should unwrap to inner: got %v", e.Wrapped)
	}
}

// TestUnwrap verifies that Unwrap returns the wrapped error for errors.As/Is.
func TestUnwrap(t *testing.T) {
	t.Parallel()
	inner := errors.New("inner")
	e := &UXError{Code: "E004", What: "test", Wrapped: inner}
	if e.Unwrap() != inner {
		t.Errorf("Unwrap() = %v; want %v", e.Unwrap(), inner)
	}
}

// TestError verifies that the Error() string includes code and what.
func TestError(t *testing.T) {
	t.Parallel()
	e := &UXError{Code: "E005", What: "something failed", Why: "disk full"}
	s := e.Error()
	if !strings.Contains(s, "E005") {
		t.Errorf("Error() = %q; should contain 'E005'", s)
	}
	if !strings.Contains(s, "something failed") {
		t.Errorf("Error() = %q; should contain 'something failed'", s)
	}
}

// TestErrorWithWhy verifies that Error() includes the Why when non-empty.
func TestErrorWithWhy(t *testing.T) {
	t.Parallel()
	e := &UXError{Code: "E006", What: "op failed", Why: "disk full"}
	s := e.Error()
	if !strings.Contains(s, "disk full") {
		t.Errorf("Error() = %q; should contain 'disk full'", s)
	}
}

// TestWithSuggestions verifies that WithSuggestions appends to an existing set.
func TestWithSuggestions(t *testing.T) {
	t.Parallel()
	e := New("E007", "bad config").WithSuggestions("run nself doctor")
	if len(e.Suggestions) == 0 {
		t.Fatal("Suggestions should be non-empty after WithSuggestions")
	}
	if !strings.Contains(e.Suggestions[len(e.Suggestions)-1], "nself doctor") {
		t.Errorf("last suggestion = %q; should contain 'nself doctor'", e.Suggestions[len(e.Suggestions)-1])
	}
}

// TestWithWhy verifies that WithWhy sets the Why field.
func TestWithWhy(t *testing.T) {
	t.Parallel()
	e := New("E008", "operation failed").WithWhy("no route to host")
	if e.Why != "no route to host" {
		t.Errorf("Why = %q; want 'no route to host'", e.Why)
	}
}

// TestPrintAndReturn verifies that PrintAndReturn returns the error itself.
func TestPrintAndReturn(t *testing.T) {
	t.Parallel()
	e := New("E009", "test print")
	returned := e.PrintAndReturn()
	if returned == nil {
		t.Fatal("PrintAndReturn returned nil")
	}
	// Should be able to unwrap back to UXError
	var ux *UXError
	if !errors.As(returned, &ux) {
		t.Errorf("returned error should be a *UXError")
	}
}

// TestWithSuggestionsImmutability verifies that WithSuggestions returns a copy
// and does not mutate the original.
func TestWithSuggestionsImmutability(t *testing.T) {
	t.Parallel()
	base := New("E010", "base error")
	before := len(base.Suggestions)
	derived := base.WithSuggestions("extra fix")
	if len(base.Suggestions) != before {
		t.Errorf("WithSuggestions mutated the original: before=%d after=%d", before, len(base.Suggestions))
	}
	if len(derived.Suggestions) <= before {
		t.Errorf("derived.Suggestions should have grown; got %d", len(derived.Suggestions))
	}
}

// TestWithWhyImmutability verifies that WithWhy returns a copy.
func TestWithWhyImmutability(t *testing.T) {
	t.Parallel()
	base := &UXError{Code: "E011", What: "base", Why: "original why"}
	derived := base.WithWhy("new why")
	if base.Why != "original why" {
		t.Errorf("WithWhy mutated the original: base.Why = %q", base.Why)
	}
	if derived.Why != "new why" {
		t.Errorf("derived.Why = %q; want 'new why'", derived.Why)
	}
}
