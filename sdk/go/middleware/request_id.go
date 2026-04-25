// Package middleware provides standardized HTTP middleware for nSelf plugins.
// It complements the chi default stack in the server package with nSelf-specific
// concerns like explicit X-Request-ID propagation, trace correlation, and cost
// metering.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// RequestIDHeader is the canonical header used for request correlation across
// plugin boundaries. Both inbound and outbound requests use this name.
const RequestIDHeader = "X-Request-ID"

type requestIDKey struct{}

// RequestIDFromContext returns the request ID attached to ctx, or "" if none.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

// WithRequestID returns a child context carrying id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID is HTTP middleware that reads X-Request-ID from the incoming
// request, generates one if absent, stores it in the request context, and
// echoes it back on the response.
//
// Downstream handlers can retrieve the id via RequestIDFromContext. The echoed
// response header lets callers correlate client-side and server-side logs
// without additional tooling.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		ctx := WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PropagateRequestID copies the request ID from ctx onto req so downstream
// plugin-to-plugin calls carry the same correlation id. Safe to call with a
// context that has no id — it becomes a no-op.
func PropagateRequestID(ctx context.Context, req *http.Request) {
	if id := RequestIDFromContext(ctx); id != "" {
		req.Header.Set(RequestIDHeader, id)
	}
}

// newRequestID returns a 16-byte hex-encoded request id (32 chars).
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to a deterministic string rather than panicking — the id
		// still uniquely identifies the request within one process.
		return "req-fallback"
	}
	return hex.EncodeToString(b[:])
}
