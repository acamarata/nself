package soak

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestSoakAbortDryRun(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Abort(context.Background(), AbortOptions{
		Version: "v1.0.8",
		DryRun:  true,
		Env:     "staging",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("dry-run should not return error, got: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "DRY RUN") {
		t.Error("expected DRY RUN in output")
	}
	if !strings.Contains(out, "v1.0.8") {
		t.Error("expected target version in dry-run output")
	}
	if !strings.Contains(out, "No changes made") {
		t.Error("expected 'No changes made' in dry-run output")
	}
}

func TestSoakAbortConfirm(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Provide "yes" as confirmation input.
	stdin := strings.NewReader("yes\n")

	called := false
	err := Abort(context.Background(), AbortOptions{
		Version: "v1.0.8",
		Env:     "staging",
		Stdout:  &stdout,
		Stderr:  &stderr,
		Stdin:   stdin,
		RollbackFunc: func(_ context.Context, _, _ string, _, _ io.Writer) error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("abort with 'yes' confirmation should succeed, got: %v", err)
	}
	if !called {
		t.Error("expected rollback to be invoked")
	}
	if !strings.Contains(stdout.String(), "complete") {
		t.Error("expected completion message in output")
	}
}

func TestSoakAbortConfirmCancel(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Provide "no" to cancel.
	stdin := strings.NewReader("no\n")

	err := Abort(context.Background(), AbortOptions{
		Version: "v1.0.8",
		Env:     "staging",
		Stdout:  &stdout,
		Stderr:  &stderr,
		Stdin:   stdin,
	})
	if err != ErrAbortCanceled {
		t.Fatalf("expected ErrAbortCanceled, got: %v", err)
	}
}

func TestSoakAbortIdempotent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	t.Setenv("NSELF_CURRENT_VERSION", "v1.0.8")

	err := Abort(context.Background(), AbortOptions{
		Version:     "v1.0.8",
		SkipConfirm: true,
		Env:         "staging",
		Stdout:      &stdout,
		Stderr:      &stderr,
	})
	if err != ErrAlreadyRolledBack {
		t.Fatalf("expected ErrAlreadyRolledBack, got: %v", err)
	}
	if !strings.Contains(stdout.String(), "already at") {
		t.Error("expected idempotent no-op message")
	}
}

func TestSoakAbortProdGate(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Abort(context.Background(), AbortOptions{
		Version: "v1.0.8",
		Env:     "prod",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err == nil {
		t.Fatal("expected error for prod without --prod-i-mean-it")
	}
	if !strings.Contains(err.Error(), "--prod-i-mean-it") {
		t.Error("expected --prod-i-mean-it in error message")
	}
}

func TestSoakAbortProdWithFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("yes\n")

	err := Abort(context.Background(), AbortOptions{
		Version:   "v1.0.8",
		Env:       "prod",
		AllowProd: true,
		Stdout:    &stdout,
		Stderr:    &stderr,
		Stdin:     stdin,
		RollbackFunc: func(_ context.Context, _, _ string, _, _ io.Writer) error {
			return nil
		},
	})
	if err != nil {
		t.Fatalf("prod with --prod-i-mean-it and yes confirmation should succeed, got: %v", err)
	}
}
