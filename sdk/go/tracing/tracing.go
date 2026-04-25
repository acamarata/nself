// Package tracing provides W3C Trace Context propagation for nSelf plugins.
//
// The SDK deliberately keeps OpenTelemetry out of its direct dependency set so
// plugins that do not need distributed tracing stay lean. Plugins that want
// full OTel export swap in the OTel SDK at their own module boundary; this
// package supplies the wire-format propagation and a noop tracer interface so
// every plugin can emit traceparent headers consistently.
//
// Wire format follows the W3C Trace Context spec:
//
//	traceparent: 00-<32 hex trace-id>-<16 hex span-id>-<2 hex flags>
//
// Plugins call StartSpan to obtain a span id, then propagate via
// InjectHeaders on outbound requests.
package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

// TraceParentHeader is the canonical W3C header name.
const TraceParentHeader = "traceparent"

// TraceStateHeader carries vendor-specific trace context. Propagated verbatim.
const TraceStateHeader = "tracestate"

// SpanContext identifies one span within a distributed trace.
type SpanContext struct {
	TraceID string // 32 hex chars
	SpanID  string // 16 hex chars
	Flags   string // 2 hex chars (01 = sampled)
}

// IsValid reports whether sc holds well-formed trace + span ids.
func (sc SpanContext) IsValid() bool {
	return len(sc.TraceID) == 32 && len(sc.SpanID) == 16 && len(sc.Flags) == 2
}

// String returns the W3C traceparent wire format.
func (sc SpanContext) String() string {
	if !sc.IsValid() {
		return ""
	}
	return "00-" + sc.TraceID + "-" + sc.SpanID + "-" + sc.Flags
}

type spanContextKey struct{}

// FromContext returns the active SpanContext, or the zero value if none.
func FromContext(ctx context.Context) SpanContext {
	if v, ok := ctx.Value(spanContextKey{}).(SpanContext); ok {
		return v
	}
	return SpanContext{}
}

// WithSpanContext stores sc on ctx for downstream propagation.
func WithSpanContext(ctx context.Context, sc SpanContext) context.Context {
	return context.WithValue(ctx, spanContextKey{}, sc)
}

// StartSpan creates a new span. If parent has a valid trace id, the new span
// joins that trace; otherwise a new trace id is minted. The returned context
// carries the new SpanContext, so subsequent propagation calls pick it up.
func StartSpan(ctx context.Context) (context.Context, SpanContext, error) {
	parent := FromContext(ctx)
	sc := SpanContext{Flags: "01"}

	if parent.IsValid() {
		sc.TraceID = parent.TraceID
	} else {
		tid, err := randomHex(16)
		if err != nil {
			return ctx, SpanContext{}, err
		}
		sc.TraceID = tid
	}

	sid, err := randomHex(8)
	if err != nil {
		return ctx, SpanContext{}, err
	}
	sc.SpanID = sid

	return WithSpanContext(ctx, sc), sc, nil
}

// Extract parses a W3C traceparent header into a SpanContext. Returns an
// error if the value is missing or malformed.
func Extract(h http.Header) (SpanContext, error) {
	raw := h.Get(TraceParentHeader)
	if raw == "" {
		return SpanContext{}, errors.New("tracing: no traceparent header")
	}
	parts := strings.Split(raw, "-")
	if len(parts) != 4 || parts[0] != "00" {
		return SpanContext{}, errors.New("tracing: malformed traceparent")
	}
	sc := SpanContext{TraceID: parts[1], SpanID: parts[2], Flags: parts[3]}
	if !sc.IsValid() {
		return SpanContext{}, errors.New("tracing: invalid trace/span id lengths")
	}
	return sc, nil
}

// Inject writes the SpanContext onto h as a traceparent header. No-op for
// empty SpanContexts so callers can invoke it unconditionally.
func (sc SpanContext) Inject(h http.Header) {
	if !sc.IsValid() {
		return
	}
	h.Set(TraceParentHeader, sc.String())
}

// InjectHeaders copies the active SpanContext from ctx onto req.Header.
func InjectHeaders(ctx context.Context, req *http.Request) {
	FromContext(ctx).Inject(req.Header)
}

// Middleware returns an http.Handler wrapper that extracts an incoming
// traceparent (or starts a new trace if absent) and stores the SpanContext
// on the request context. Pair with the plugin SDK's request-id middleware.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if parent, err := Extract(r.Header); err == nil {
			ctx = WithSpanContext(ctx, parent)
		}
		ctx, sc, err := StartSpan(ctx)
		if err == nil {
			w.Header().Set(TraceParentHeader, sc.String())
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
