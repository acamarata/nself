package eval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunSuite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/eval/run" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(RunQueued{RunID: "run-abc123", Status: "queued"})
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}
	result, err := c.RunSuite(context.Background(), "recall-quality-v1")
	if err != nil {
		t.Fatalf("RunSuite error: %v", err)
	}
	if result.RunID != "run-abc123" {
		t.Errorf("expected RunID run-abc123, got %q", result.RunID)
	}
	if result.Status != "queued" {
		t.Errorf("expected Status queued, got %q", result.Status)
	}
}

func TestGetRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/eval/runs/run-xyz" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(EvalRunResult{
			SchemaVer:  expectedSchemaVer,
			ID:         "run-xyz",
			SuiteSlug:  "recall-quality-v1",
			Status:     "passed",
			PassRate:   0.9,
			SuiteScore: 0.88,
			Passed:     true,
		})
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}
	result, err := c.GetRun(context.Background(), "run-xyz")
	if err != nil {
		t.Fatalf("GetRun error: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed=true")
	}
	if result.Status != "passed" {
		t.Errorf("expected Status=passed, got %q", result.Status)
	}
}

func TestGetRunSchemaVersionMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(EvalRunResult{
			SchemaVer: 99,
			ID:        "run-abc",
			Status:    "passed",
		})
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}
	_, err := c.GetRun(context.Background(), "run-abc")
	if err == nil {
		t.Fatal("expected schema version error, got nil")
	}
}

func TestWaitForRunPollsUntilTerminal(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := "running"
		if calls >= 3 {
			status = "passed"
		}
		_ = json.NewEncoder(w).Encode(EvalRunResult{
			SchemaVer: expectedSchemaVer,
			ID:        "run-poll",
			Status:    status,
			Passed:    calls >= 3,
		})
	}))
	defer srv.Close()

	// Override poll interval for test speed.
	origInterval := pollInterval
	_ = origInterval // pollInterval is const; test verifies polling via call count

	c := &Client{BaseURL: srv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}

	// Use a short-lived context to avoid waiting the full 2s poll interval.
	// We test that calls >= 1 (at least one poll attempt).
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := c.WaitForRun(ctx, "run-poll")
	if err != nil {
		t.Fatalf("WaitForRun error: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed=true after polling")
	}
	if calls < 3 {
		t.Errorf("expected at least 3 poll calls, got %d", calls)
	}
}

func TestWaitForRunPreconditionFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(EvalRunResult{
			SchemaVer:          expectedSchemaVer,
			ID:                 "run-pre",
			Status:             "failed",
			PreconditionFailed: true,
		})
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := c.WaitForRun(ctx, "run-pre")
	if err != ErrPreconditionNotMet {
		t.Errorf("expected ErrPreconditionNotMet, got %v", err)
	}
}

func TestValidateYAMLValid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ValidationResult{Valid: true})
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}
	result, err := c.ValidateYAML(context.Background(), []byte("suite: test"))
	if err != nil {
		t.Fatalf("ValidateYAML error: %v", err)
	}
	if !result.Valid {
		t.Error("expected Valid=true")
	}
}

func TestValidateYAMLInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ValidationResult{
			Valid:  false,
			Errors: []ValidationError{{Field: "suite", Message: "suite is required"}},
		})
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}
	result, err := c.ValidateYAML(context.Background(), []byte("{}"))
	if err != nil {
		t.Fatalf("ValidateYAML error: %v", err)
	}
	if result.Valid {
		t.Error("expected Valid=false")
	}
	if len(result.Errors) == 0 {
		t.Error("expected validation errors")
	}
}

func TestGetGateStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/eval/gate/semi-auto" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(GateStatus{
			Tier:    "semi-auto",
			Cleared: true,
		})
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: &http.Client{Timeout: 5 * time.Second}}
	gs, err := c.GetGateStatus(context.Background(), "semi-auto")
	if err != nil {
		t.Fatalf("GetGateStatus error: %v", err)
	}
	if !gs.Cleared {
		t.Error("expected Cleared=true")
	}
	if gs.Tier != "semi-auto" {
		t.Errorf("expected Tier=semi-auto, got %q", gs.Tier)
	}
}
