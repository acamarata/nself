package tracing

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStartSpanCreatesNewTrace(t *testing.T) {
	ctx, sc, err := StartSpan(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	if err != nil {
		t.Fatalf("StartSpan: %v", err)
	}
	if !sc.IsValid() {
		t.Fatalf("span not valid: %+v", sc)
	}
	if got := FromContext(ctx); got != sc {
		t.Fatalf("context span mismatch: %+v vs %+v", got, sc)
	}
}

func TestStartSpanJoinsExistingTrace(t *testing.T) {
	parent := SpanContext{TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef", Flags: "01"}
	ctx := WithSpanContext(httptest.NewRequest(http.MethodGet, "/", nil).Context(), parent)

	_, sc, err := StartSpan(ctx)
	if err != nil {
		t.Fatalf("StartSpan: %v", err)
	}
	if sc.TraceID != parent.TraceID {
		t.Fatalf("expected trace id %q, got %q", parent.TraceID, sc.TraceID)
	}
	if sc.SpanID == parent.SpanID {
		t.Fatalf("expected new span id, got parent id")
	}
}

func TestExtractParsesHeader(t *testing.T) {
	h := http.Header{}
	h.Set(TraceParentHeader, "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
	sc, err := Extract(h)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if !sc.IsValid() {
		t.Fatalf("extracted span invalid: %+v", sc)
	}
	if sc.TraceID != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("trace id mismatch: %q", sc.TraceID)
	}
}

func TestExtractRejectsMalformed(t *testing.T) {
	h := http.Header{}
	h.Set(TraceParentHeader, "not-a-valid-traceparent")
	if _, err := Extract(h); err == nil {
		t.Fatal("expected error on malformed header")
	}
}

func TestExtractMissingHeader(t *testing.T) {
	h := http.Header{}
	if _, err := Extract(h); err == nil {
		t.Fatal("expected error when header absent")
	}
}

func TestInjectHeadersPropagates(t *testing.T) {
	sc := SpanContext{TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef", Flags: "01"}
	ctx := WithSpanContext(httptest.NewRequest(http.MethodGet, "/", nil).Context(), sc)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://upstream/", nil)
	InjectHeaders(ctx, req)
	if got := req.Header.Get(TraceParentHeader); got != sc.String() {
		t.Fatalf("expected %q, got %q", sc.String(), got)
	}
}

func TestMiddlewareStartsSpan(t *testing.T) {
	var observed SpanContext
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !observed.IsValid() {
		t.Fatal("expected middleware to populate span context")
	}
	if rec.Header().Get(TraceParentHeader) == "" {
		t.Fatal("expected traceparent echoed on response")
	}
}

func TestMiddlewareInheritsIncomingTrace(t *testing.T) {
	incoming := "00-aaaabbbbccccddddeeeeffff00001111-2222333344445555-01"
	var observed SpanContext
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TraceParentHeader, incoming)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if observed.TraceID != "aaaabbbbccccddddeeeeffff00001111" {
		t.Fatalf("expected inherited trace id, got %q", observed.TraceID)
	}
}
