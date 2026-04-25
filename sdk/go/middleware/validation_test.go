package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ValidateContentType
// ---------------------------------------------------------------------------

func TestValidateContentType_CorrectTypePasses(t *testing.T) {
	h := ValidateContentType("application/json")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := `{"key":"value"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestValidateContentType_WithCharsetPasses(t *testing.T) {
	h := ValidateContentType("application/json")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with charset param, got %d", rec.Code)
	}
}

func TestValidateContentType_WrongTypeReturns415(t *testing.T) {
	h := ValidateContentType("application/json")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`data=value`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rec.Code)
	}
}

func TestValidateContentType_MissingHeaderReturns415(t *testing.T) {
	h := ValidateContentType("application/json")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415 for missing Content-Type, got %d", rec.Code)
	}
}

func TestValidateContentType_GETPassesWithoutHeader(t *testing.T) {
	h := ValidateContentType("application/json")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected GET to bypass content-type check, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// MaxBodySize
// ---------------------------------------------------------------------------

func TestMaxBodySize_UnderLimitPasses(t *testing.T) {
	const limit = 64
	h := MaxBodySize(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.Repeat("x", 32)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for body under limit, got %d", rec.Code)
	}
}

func TestMaxBodySize_ExactLimitPasses(t *testing.T) {
	const limit = 16
	h := MaxBodySize(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.Repeat("a", int(limit))
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for body at exact limit, got %d", rec.Code)
	}
}

func TestMaxBodySize_OverLimitReturns413(t *testing.T) {
	const limit = 16
	h := MaxBodySize(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := strings.Repeat("z", int(limit)+1)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized Content-Length, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// RejectUnknownFields
// ---------------------------------------------------------------------------

type sampleRequest struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

func TestRejectUnknownFields_CleanJSONPasses(t *testing.T) {
	body, _ := json.Marshal(sampleRequest{Name: "plugin-ai", Score: 99})
	got, err := RejectUnknownFields[sampleRequest](body)
	if err != nil {
		t.Fatalf("unexpected error for valid JSON: %v", err)
	}
	if got.Name != "plugin-ai" || got.Score != 99 {
		t.Fatalf("unexpected decoded value: %+v", got)
	}
}

func TestRejectUnknownFields_ExtraFieldReturnsError(t *testing.T) {
	body := []byte(`{"name":"plugin-ai","score":99,"extra":"surprise"}`)
	_, err := RejectUnknownFields[sampleRequest](body)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

func TestRejectUnknownFields_MalformedJSONReturnsError(t *testing.T) {
	body := []byte(`{"name": }`)
	_, err := RejectUnknownFields[sampleRequest](body)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestRejectUnknownFields_EmptyBodyReturnsError(t *testing.T) {
	_, err := RejectUnknownFields[sampleRequest]([]byte{})
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

// ---------------------------------------------------------------------------
// RequireFields
// ---------------------------------------------------------------------------

func TestRequireFields_AllPresentPasses(t *testing.T) {
	payload := map[string]any{"user_id": "u-1", "action": "read"}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	ok := RequireFields("user_id", "action")(w, r, payload)
	if !ok {
		t.Fatal("expected RequireFields to return true for complete payload")
	}
}

func TestRequireFields_MissingFieldReturns400(t *testing.T) {
	payload := map[string]any{"user_id": "u-1"}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	ok := RequireFields("user_id", "action")(w, r, payload)
	if ok {
		t.Fatal("expected RequireFields to return false for missing field")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRequireFields_EmptyStringFieldReturns400(t *testing.T) {
	payload := map[string]any{"user_id": "u-1", "action": "   "}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	ok := RequireFields("user_id", "action")(w, r, payload)
	if ok {
		t.Fatal("expected RequireFields to return false for whitespace-only field")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRequireFields_NullFieldReturns400(t *testing.T) {
	payload := map[string]any{"user_id": "u-1", "action": nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	ok := RequireFields("user_id", "action")(w, r, payload)
	if ok {
		t.Fatal("expected RequireFields to return false for null field")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// ValidateJSON
// ---------------------------------------------------------------------------

type requiredStruct struct {
	Name    string `validate:"required"`
	Version string `validate:"required"`
	Tag     string // no validate tag — not checked
}

func TestValidateJSON_AllRequiredFieldsSet(t *testing.T) {
	v := &requiredStruct{Name: "ai", Version: "1.0.0"}
	if err := ValidateJSON(v); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateJSON_MissingRequiredField(t *testing.T) {
	v := &requiredStruct{Name: "ai"} // Version zero-value
	if err := ValidateJSON(v); err == nil {
		t.Fatal("expected error for missing required field Version")
	}
}

func TestValidateJSON_UntaggedFieldIgnored(t *testing.T) {
	v := &requiredStruct{Name: "ai", Version: "1.0.0"} // Tag empty, no validate tag
	if err := ValidateJSON(v); err != nil {
		t.Fatalf("untagged field should be ignored, got error: %v", err)
	}
}

func TestValidateJSON_NilPointerReturnsError(t *testing.T) {
	if err := ValidateJSON(nil); err == nil {
		t.Fatal("expected error for nil value")
	}
}

func TestValidateJSON_NonPointerReturnsError(t *testing.T) {
	v := requiredStruct{Name: "ai", Version: "1.0.0"}
	if err := ValidateJSON(v); err == nil {
		t.Fatal("expected error for non-pointer struct")
	}
}

// ---------------------------------------------------------------------------
// ValidateUUIDParam
// ---------------------------------------------------------------------------

func TestValidateUUIDParam_ValidLowercasePasses(t *testing.T) {
	w := httptest.NewRecorder()
	if !ValidateUUIDParam(w, "id", "550e8400-e29b-41d4-a716-446655440000") {
		t.Fatalf("expected valid UUID to pass, got status %d", w.Code)
	}
}

func TestValidateUUIDParam_ValidUppercasePasses(t *testing.T) {
	w := httptest.NewRecorder()
	if !ValidateUUIDParam(w, "id", "550E8400-E29B-41D4-A716-446655440000") {
		t.Fatalf("expected uppercase UUID to pass, got status %d", w.Code)
	}
}

func TestValidateUUIDParam_EmptyValueReturns400(t *testing.T) {
	w := httptest.NewRecorder()
	if ValidateUUIDParam(w, "id", "") {
		t.Fatal("expected empty value to fail")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestValidateUUIDParam_NonUUIDReturns400(t *testing.T) {
	w := httptest.NewRecorder()
	if ValidateUUIDParam(w, "id", "not-a-uuid") {
		t.Fatal("expected non-UUID to fail")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestValidateUUIDParam_ShortHexReturns400(t *testing.T) {
	w := httptest.NewRecorder()
	// Looks like a UUID but is too short.
	if ValidateUUIDParam(w, "id", "550e8400-e29b-41d4-a716-44665544") {
		t.Fatal("expected truncated UUID to fail")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestValidateUUIDParam_SQLInjectionAttemptReturns400(t *testing.T) {
	w := httptest.NewRecorder()
	if ValidateUUIDParam(w, "id", "1' OR '1'='1") {
		t.Fatal("expected SQL injection string to fail UUID validation")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Integration: stacked middleware
// ---------------------------------------------------------------------------

func TestStackedMiddleware_ContentTypeAndBodySize(t *testing.T) {
	handler := ValidateContentType("application/json")(
		MaxBodySize(128)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})),
	)

	// Good request: correct type, small body.
	body := bytes.NewBufferString(`{"ok":true}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for valid stacked request, got %d", rec.Code)
	}

	// Wrong content type should short-circuit before body limit check.
	body2 := bytes.NewBufferString(`form=data`)
	req2 := httptest.NewRequest(http.MethodPost, "/test", body2)
	req2.Header.Set("Content-Type", "text/plain")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415 for wrong content type, got %d", rec2.Code)
	}
}
