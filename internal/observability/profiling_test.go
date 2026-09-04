package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenAuth_ValidToken(t *testing.T) {
	handler := PprofHandler("test-token-123", "127.0.0.1:6060")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.Header.Set("X-Profile-Token", "test-token-123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code == http.StatusForbidden {
		t.Error("expected access with valid token, got 403")
	}
}

func TestTokenAuth_InvalidToken(t *testing.T) {
	handler := PprofHandler("test-token-123", "127.0.0.1:6060")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	req.Header.Set("X-Profile-Token", "wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// TestTokenAuth_NoToken_DefaultsClosed is the row-20 contract: an empty
// token means pprof is disabled (403) by default, with no escape hatch
// env var set. This replaces the old TestTokenAuth_NoToken_DisablesAuth,
// which asserted the opposite (empty token = open) — that behavior was
// the default-deny gap this fix closes.
func TestTokenAuth_NoToken_DefaultsClosed(t *testing.T) {
	handler := PprofHandler("", "127.0.0.1:6060")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 with empty token and no NSELF_PPROF_DEV, got %d", w.Code)
	}
}

func TestTokenAuth_NoToken_DevOptIn_Loopback_Allows(t *testing.T) {
	t.Setenv("NSELF_PPROF_DEV", "1")
	handler := PprofHandler("", "127.0.0.1:6060")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code == http.StatusForbidden {
		t.Error("expected access with NSELF_PPROF_DEV=1 on a loopback bind, got 403")
	}
}

func TestTokenAuth_NoToken_DevOptIn_NonLoopback_StillClosed(t *testing.T) {
	t.Setenv("NSELF_PPROF_DEV", "1")
	handler := PprofHandler("", "0.0.0.0:6060")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403: NSELF_PPROF_DEV=1 must not open pprof on a non-loopback bind, got %d", w.Code)
	}
}

func TestTokenAuth_MissingHeader(t *testing.T) {
	handler := PprofHandler("test-token-123", "127.0.0.1:6060")
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when header missing, got %d", w.Code)
	}
}

func TestIsLoopbackBind(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:6060", true},
		{"localhost:6060", true},
		{"[::1]:6060", true},
		{"0.0.0.0:6060", false},
		{":6060", false},
		{"192.168.1.5:6060", false},
		{"", false},
		{"not-a-valid-addr", false},
	}
	for _, c := range cases {
		if got := isLoopbackBind(c.addr); got != c.want {
			t.Errorf("isLoopbackBind(%q) = %v, want %v", c.addr, got, c.want)
		}
	}
}
