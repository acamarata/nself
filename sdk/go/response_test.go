// Package sdk_test covers the top-level Respond / Error helpers.
package sdk_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/nself-org/cli/sdk/go/v2"
)

func TestRespond_WithBody(t *testing.T) {
	rr := httptest.NewRecorder()
	sdk.Respond(rr, http.StatusOK, map[string]string{"key": "val"})
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("want Content-Type=application/json, got %q", ct)
	}
	var out map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["key"] != "val" {
		t.Errorf("want key=val, got %v", out["key"])
	}
}

func TestRespond_NilBody(t *testing.T) {
	rr := httptest.NewRecorder()
	sdk.Respond(rr, http.StatusNoContent, nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("want empty body, got %q", rr.Body.String())
	}
}

func TestError(t *testing.T) {
	rr := httptest.NewRecorder()
	sdk.Error(rr, http.StatusBadRequest, "bad input")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "bad input") {
		t.Errorf("want body to contain 'bad input', got %q", rr.Body.String())
	}
}
