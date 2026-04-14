package observability

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync/atomic"
)

// PII redaction patterns.
var (
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	phoneRegex = regexp.MustCompile(`\b(\+?1?[-.\s]?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4})\b`)
	ccRegex    = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)

	// API key prefixes that should be redacted.
	apiKeyPrefixes = []string{
		"nself_pro_",
		"sk-",
		"sk_live_",
		"sk_test_",
		"pk_live_",
		"pk_test_",
		"ghp_",
		"gho_",
		"Bearer ",
	}

	redactionPlaceholder = "[REDACTED]"

	// Global counter for redaction failures.
	redactionFailures atomic.Int64
)

// RedactionFailures returns the current count of redaction failures.
func RedactionFailures() int64 {
	return redactionFailures.Load()
}

// Redact applies PII pattern matching and replaces matches with [REDACTED].
func Redact(s string) string {
	result := s
	result = emailRegex.ReplaceAllString(result, redactionPlaceholder)
	result = phoneRegex.ReplaceAllString(result, redactionPlaceholder)
	result = ccRegex.ReplaceAllString(result, redactionPlaceholder)

	for _, prefix := range apiKeyPrefixes {
		for {
			idx := strings.Index(result, prefix)
			if idx < 0 {
				break
			}
			// Redact from prefix to next whitespace or end of string.
			end := idx + len(prefix)
			for end < len(result) && result[end] != ' ' && result[end] != '"' && result[end] != '\'' && result[end] != ',' && result[end] != '\n' {
				end++
			}
			result = result[:idx] + redactionPlaceholder + result[end:]
		}
	}

	return result
}

// RedactHandler wraps a slog.Handler and redacts PII from string attribute values.
type RedactHandler struct {
	inner slog.Handler
}

// NewRedactHandler creates a handler that redacts PII patterns from log attributes.
func NewRedactHandler(inner slog.Handler) *RedactHandler {
	return &RedactHandler{inner: inner}
}

func (h *RedactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *RedactHandler) Handle(ctx context.Context, r slog.Record) error {
	// Redact the message itself.
	r.Message = Redact(r.Message)

	// Redact all string attributes.
	var redacted []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		redacted = append(redacted, redactAttr(a))
		return true
	})

	// Create a new record with redacted attrs.
	newR := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	newR.AddAttrs(redacted...)

	return h.inner.Handle(ctx, newR)
}

func (h *RedactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	ra := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		ra[i] = redactAttr(a)
	}
	return &RedactHandler{inner: h.inner.WithAttrs(ra)}
}

func (h *RedactHandler) WithGroup(name string) slog.Handler {
	return &RedactHandler{inner: h.inner.WithGroup(name)}
}

func redactAttr(a slog.Attr) slog.Attr {
	// Skip known sensitive keys entirely.
	key := strings.ToLower(a.Key)
	if key == "password" || key == "token" || key == "secret" || key == "authorization" {
		return slog.String(a.Key, redactionPlaceholder)
	}

	switch a.Value.Kind() {
	case slog.KindString:
		return slog.String(a.Key, Redact(a.Value.String()))
	case slog.KindGroup:
		attrs := a.Value.Group()
		ra := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			ra[i] = redactAttr(ga)
		}
		return slog.Group(a.Key, attrsToAny(ra)...)
	}
	return a
}

func attrsToAny(attrs []slog.Attr) []any {
	out := make([]any, len(attrs))
	for i, a := range attrs {
		out[i] = a
	}
	return out
}
