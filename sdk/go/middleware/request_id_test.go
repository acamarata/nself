package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	var capturedID string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if capturedID == "" {
		t.Fatal("expected generated request id in context, got empty")
	}
	if got := rec.Header().Get(RequestIDHeader); got != capturedID {
		t.Fatalf("expected response header %q, got %q", capturedID, got)
	}
	if len(capturedID) != 32 {
		t.Fatalf("expected 32-char hex id, got %d chars: %q", len(capturedID), capturedID)
	}
}

func TestRequestIDPreservesIncoming(t *testing.T) {
	const incoming = "client-supplied-id-abc123"
	var capturedID string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, incoming)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if capturedID != incoming {
		t.Fatalf("expected context id %q, got %q", incoming, capturedID)
	}
	if got := rec.Header().Get(RequestIDHeader); got != incoming {
		t.Fatalf("expected response header %q, got %q", incoming, got)
	}
}

func TestPropagateRequestID(t *testing.T) {
	const id = "propagate-me"
	ctx := WithRequestID(httptest.NewRequest(http.MethodGet, "/", nil).Context(), id)

	out, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://upstream/foo", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	PropagateRequestID(ctx, out)
	if got := out.Header.Get(RequestIDHeader); got != id {
		t.Fatalf("expected propagated header %q, got %q", id, got)
	}
}

func TestPropagateRequestIDNoopOnEmpty(t *testing.T) {
	out, err := http.NewRequest(http.MethodGet, "http://upstream/foo", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	PropagateRequestID(out.Context(), out)
	if got := out.Header.Get(RequestIDHeader); got != "" {
		t.Fatalf("expected no header, got %q", got)
	}
}
