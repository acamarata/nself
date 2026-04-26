package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// noopExporter discards all spans. It is used to create a real TracerProvider
// in tests without requiring a network connection.
type noopExporter struct{}

func (noopExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error { return nil }
func (noopExporter) Shutdown(_ context.Context) error                                { return nil }

// captureHandler is a minimal slog.Handler that captures records for assertion.
type captureHandler struct {
	buf *bytes.Buffer
}

func newCaptureHandler() *captureHandler {
	return &captureHandler{buf: &bytes.Buffer{}}
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler         { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler              { return h }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	// Serialise the record to JSON for easy field lookup in tests.
	obj := map[string]any{
		"msg":   r.Message,
		"level": r.Level.String(),
	}
	r.Attrs(func(a slog.Attr) bool {
		obj[a.Key] = a.Value.String()
		return true
	})
	return json.NewEncoder(h.buf).Encode(obj)
}

func (h *captureHandler) decode(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(h.buf.Bytes(), &m); err != nil {
		t.Fatalf("captureHandler: decode failed: %v\nraw: %s", err, h.buf.String())
	}
	return m
}

// newTestTracer creates a real TracerProvider backed by a no-op exporter with
// AlwaysSample so every span has a valid, non-zero TraceID and SpanID.
func newTestTracer(t *testing.T) trace.Tracer {
	t.Helper()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(noopExporter{}),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return tp.Tracer("nself-test")
}

// TestTraceLogHandler_InjectsTraceIDWhenSpanActive verifies that when an active
// OTEL span is present in the context, the handler emits non-empty trace_id and
// span_id fields on every log record. This is the core behaviour enabling the
// Loki → Tempo click-through in the nSelf observability stack.
func TestTraceLogHandler_InjectsTraceIDWhenSpanActive(t *testing.T) {
	tracer := newTestTracer(t)

	cap := newCaptureHandler()
	logger := slog.New(NewTraceLogHandler(cap))

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	logger.ErrorContext(ctx, "something failed", "error", "disk full")

	fields := cap.decode(t)

	traceID, ok := fields["trace_id"]
	if !ok {
		t.Fatal("expected trace_id field in log record, not found")
	}
	if traceID == "" || traceID == "00000000000000000000000000000000" {
		t.Fatalf("trace_id must be a non-zero value when span is active, got %q", traceID)
	}

	spanID, ok := fields["span_id"]
	if !ok {
		t.Fatal("expected span_id field in log record, not found")
	}
	if spanID == "" || spanID == "0000000000000000" {
		t.Fatalf("span_id must be a non-zero value when span is active, got %q", spanID)
	}

	// Cross-check emitted IDs against the live span.
	sc := span.SpanContext()
	if want := sc.TraceID().String(); traceID != want {
		t.Errorf("trace_id mismatch: got %q, want %q", traceID, want)
	}
	if want := sc.SpanID().String(); spanID != want {
		t.Errorf("span_id mismatch: got %q, want %q", spanID, want)
	}
}

// TestTraceLogHandler_NoopWhenNoSpanActive verifies that the handler is safe
// to call when the context carries no active span. It must not panic, and
// must not emit trace_id or span_id fields (their absence is correct and
// expected — the Loki query can filter on the label being present).
func TestTraceLogHandler_NoopWhenNoSpanActive(t *testing.T) {
	cap := newCaptureHandler()
	logger := slog.New(NewTraceLogHandler(cap))

	// Plain background context — no span. Must not panic.
	logger.ErrorContext(context.Background(), "no active span")

	fields := cap.decode(t)

	if v, found := fields["trace_id"]; found {
		t.Errorf("expected no trace_id field when no span is active, got %q", v)
	}
	if v, found := fields["span_id"]; found {
		t.Errorf("expected no span_id field when no span is active, got %q", v)
	}
}

// TestTraceLogHandler_WithAttrsReturnsNewHandler verifies that WithAttrs
// returns a new *TraceLogHandler wrapping the inner handler. Trace injection
// must still work on the derived handler.
func TestTraceLogHandler_WithAttrsReturnsNewHandler(t *testing.T) {
	tracer := newTestTracer(t)

	cap := newCaptureHandler()
	base := NewTraceLogHandler(cap)
	derived := base.WithAttrs([]slog.Attr{slog.String("service", "test-svc")})

	logger := slog.New(derived)

	ctx, span := tracer.Start(context.Background(), "derived-span")
	defer span.End()

	logger.WarnContext(ctx, "derived handler check")

	fields := cap.decode(t)

	if _, ok := fields["trace_id"]; !ok {
		t.Error("derived handler (WithAttrs) must still inject trace_id")
	}
	if _, ok := fields["span_id"]; !ok {
		t.Error("derived handler (WithAttrs) must still inject span_id")
	}
}

// TestTraceLogHandler_WithGroupReturnsNewHandler verifies that WithGroup
// returns a new *TraceLogHandler and satisfies the slog.Handler interface
// without panicking.
func TestTraceLogHandler_WithGroupReturnsNewHandler(t *testing.T) {
	cap := newCaptureHandler()
	base := NewTraceLogHandler(cap)
	grouped := base.WithGroup("req")

	// Must implement slog.Handler without panicking.
	if !grouped.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("grouped handler must correctly delegate Enabled to inner handler")
	}
}

// TestTraceLogHandler_EnabledDelegatesToInner verifies that the Enabled method
// delegates correctly to the inner handler's Enabled result.
func TestTraceLogHandler_EnabledDelegatesToInner(t *testing.T) {
	cap := newCaptureHandler()
	h := NewTraceLogHandler(cap)

	// captureHandler always returns true for Enabled.
	if !h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Enabled should return true when inner handler returns true")
	}
}

// TestNewJSONLogger_HandlerChainIncludesTraceInjection is an integration-level
// test verifying that the handler stack wired by NewJSONLogger (which wraps
// TraceLogHandler → RedactHandler → JSONHandler) correctly injects trace fields
// when a span is active.
//
// The test rebuilds the chain using a captureHandler as the sink to avoid
// stdout redirection complexity.
func TestNewJSONLogger_HandlerChainIncludesTraceInjection(t *testing.T) {
	tracer := newTestTracer(t)

	// Replicate the NewJSONLogger stack: JSONHandler → RedactHandler → TraceLogHandler.
	// Substitute captureHandler for the JSON sink for testability.
	cap := newCaptureHandler()
	redacted := NewRedactHandler(cap)
	traced := NewTraceLogHandler(redacted)
	logger := slog.New(traced)

	ctx, span := tracer.Start(context.Background(), "integration-span")
	defer span.End()

	logger.ErrorContext(ctx, "integration handler chain test", "component", "logger")

	fields := cap.decode(t)

	if _, ok := fields["trace_id"]; !ok {
		t.Error("NewJSONLogger handler chain must emit trace_id on error logs with active span")
	}
	if _, ok := fields["span_id"]; !ok {
		t.Error("NewJSONLogger handler chain must emit span_id on error logs with active span")
	}
}
