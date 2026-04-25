// Package middleware provides standardized HTTP middleware for nSelf plugins.
// This file contains input validation helpers that every plugin endpoint
// should apply to reject malformed requests before they reach handler logic.
package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

// uuidPattern matches canonical lowercase UUID v4 format
// (8-4-4-4-12 hex groups separated by hyphens).
var uuidPattern = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
)

// ValidateContentType returns middleware that rejects POST, PUT, and PATCH
// requests whose Content-Type does not match mediaType (e.g. "application/json").
// Non-matching requests receive a 415 Unsupported Media Type response. Other
// HTTP methods pass through without inspection.
func ValidateContentType(mediaType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				ct := r.Header.Get("Content-Type")
				if ct == "" {
					http.Error(w, "Content-Type header is required", http.StatusUnsupportedMediaType)
					return
				}
				mt, _, err := mime.ParseMediaType(ct)
				if err != nil || !strings.EqualFold(mt, mediaType) {
					http.Error(w,
						fmt.Sprintf("unsupported Content-Type %q; expected %q", ct, mediaType),
						http.StatusUnsupportedMediaType,
					)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySize returns middleware that limits the request body to bytes. If the
// body exceeds the limit the handler receives a 413 Request Entity Too Large
// response and the body is not forwarded. bytes must be > 0.
func MaxBodySize(bytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > bytes {
				http.Error(w,
					fmt.Sprintf("request body exceeds maximum allowed size of %d bytes", bytes),
					http.StatusRequestEntityTooLarge,
				)
				return
			}
			// Wrap body in a reader that hard-stops at the limit, then verify
			// the actual stream length did not exceed it.
			r.Body = http.MaxBytesReader(w, r.Body, bytes)
			next.ServeHTTP(w, r)
		})
	}
}

// RequireFields checks that every named field is present and non-empty in
// payload. It writes a 400 Bad Request response and returns false when any
// field is missing or blank so the caller can return early. Returns true when
// all fields are satisfied.
//
// Usage:
//
//	var payload map[string]any
//	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil { ... }
//	if !RequireFields(fields...)(w, r, payload) { return }
func RequireFields(fields ...string) func(w http.ResponseWriter, r *http.Request, payload map[string]any) bool {
	return func(w http.ResponseWriter, r *http.Request, payload map[string]any) bool {
		for _, f := range fields {
			v, ok := payload[f]
			if !ok {
				http.Error(w, fmt.Sprintf("missing required field: %q", f), http.StatusBadRequest)
				return false
			}
			switch val := v.(type) {
			case string:
				if strings.TrimSpace(val) == "" {
					http.Error(w, fmt.Sprintf("field %q must not be empty", f), http.StatusBadRequest)
					return false
				}
			case nil:
				http.Error(w, fmt.Sprintf("field %q must not be null", f), http.StatusBadRequest)
				return false
			}
		}
		return true
	}
}

// ValidateJSON inspects v (a non-nil pointer to a struct) for exported fields
// tagged with `validate:"required"`. It returns an error naming the first field
// whose value is the zero value for its type. Non-pointer or non-struct values
// return an error immediately.
func ValidateJSON(v any) error {
	if v == nil {
		return fmt.Errorf("ValidateJSON: value must not be nil")
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("ValidateJSON: value must be a non-nil pointer")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("ValidateJSON: value must point to a struct")
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}
		tag, ok := field.Tag.Lookup("validate")
		if !ok {
			continue
		}
		for _, part := range strings.Split(tag, ",") {
			if strings.TrimSpace(part) == "required" {
				fv := rv.Field(i)
				if fv.IsZero() {
					return fmt.Errorf("field %q is required", field.Name)
				}
				break
			}
		}
	}
	return nil
}

// RejectUnknownFields decodes body into a value of type T using a strict
// decoder that returns an error for any JSON key that does not map to a field
// in T. This prevents clients from sending silently-ignored extra fields that
// may indicate a malformed or malicious request.
//
// The body slice is consumed entirely; callers that read r.Body first must pass
// the already-read bytes here.
func RejectUnknownFields[T any](body []byte) (T, error) {
	var out T
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		var zero T
		return zero, fmt.Errorf("invalid request body: %w", err)
	}
	// Ensure no trailing non-whitespace content follows the first JSON value.
	if _, err := dec.Token(); err != io.EOF {
		var zero T
		return zero, fmt.Errorf("invalid request body: unexpected trailing content")
	}
	return out, nil
}

// ValidateUUIDParam checks that value is a well-formed UUID
// (8-4-4-4-12 hex digits, case-insensitive). It writes a 400 Bad Request
// response and returns false when the value is empty or does not match the
// UUID format. Returns true when the value is valid so handlers can use the
// short-circuit pattern:
//
//	id := chi.URLParam(r, "id")
//	if !middleware.ValidateUUIDParam(w, "id", id) { return }
func ValidateUUIDParam(w http.ResponseWriter, name, value string) bool {
	if value == "" {
		http.Error(w,
			fmt.Sprintf("path parameter %q is required", name),
			http.StatusBadRequest,
		)
		return false
	}
	if !uuidPattern.MatchString(value) {
		http.Error(w,
			fmt.Sprintf("path parameter %q must be a valid UUID, got %q", name, value),
			http.StatusBadRequest,
		)
		return false
	}
	return true
}
