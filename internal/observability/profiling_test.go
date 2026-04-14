package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenAuth_ValidToken(t *testing.T) {
	handler := PprofHandler("test-token-123")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.Header.Set("X-Profile-Token", "test-token-123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code == http.StatusForbidden {
		t.Error("expected access with valid token, got 403")
	}
}

func TestTokenAuth_InvalidToken(t *testing.T) {
	handler := PprofHandler("test-token-123")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.Header.Set("X-Profile-Token", "wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestTokenAuth_NoToken_DisablesAuth(t *testing.T) {
	handler := PprofHandler("")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code == http.StatusForbidden {
		t.Error("expected access when token is empty (dev mode), got 403")
	}
}

func TestTokenAuth_MissingHeader(t *testing.T) {
	handler := PprofHandler("test-token-123")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when header missing, got %d", w.Code)
	}
}
