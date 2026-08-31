package license

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// buildEventStream returns a fake SSE server that sends n events then closes.
func buildEventStream(t *testing.T, events []LicenseEvent) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		for _, ev := range events {
			data := fmt.Sprintf(`{"ts":%q,"key_prefix":%q,"plugin":%q,"result":%q}`,
				ev.Timestamp.Format(time.RFC3339),
				ev.KeyPrefix,
				ev.Plugin,
				ev.Result,
			)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}))
}

func TestTail(t *testing.T) {
	events := []LicenseEvent{
		{Timestamp: time.Now(), KeyPrefix: "nself_pro_xx", Plugin: "ai", Result: "allow"},
		{Timestamp: time.Now(), KeyPrefix: "nself_pro_yy", Plugin: "claw", Result: "deny"},
	}
	srv := buildEventStream(t, events)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var buf bytes.Buffer
	err := TailStream(ctx, TailOptions{
		PingURL:    srv.URL,
		Stdout:     &buf,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("TailStream returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ALLOW") {
		t.Error("expected ALLOW in output")
	}
	if !strings.Contains(out, "DENY") {
		t.Error("expected DENY in output")
	}
}

func TestTailFilter(t *testing.T) {
	events := []LicenseEvent{
		{Timestamp: time.Now(), KeyPrefix: "nself_pro_xx", Plugin: "ai", Result: "allow"},
		{Timestamp: time.Now(), KeyPrefix: "nself_pro_yy", Plugin: "claw", Result: "deny"},
		{Timestamp: time.Now(), KeyPrefix: "nself_pro_zz", Plugin: "mux", Result: "rate_limit"},
	}
	srv := buildEventStream(t, events)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var buf bytes.Buffer
	err := TailStream(ctx, TailOptions{
		PingURL:    srv.URL,
		Filters:    map[string]string{"result": "deny"},
		Stdout:     &buf,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("TailStream returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "DENY") {
		t.Error("expected DENY in filtered output")
	}
	if strings.Contains(out, "ALLOW") {
		t.Error("ALLOW should be filtered out")
	}
	if strings.Contains(out, "RATE_LIMIT") {
		t.Error("RATE_LIMIT should be filtered out")
	}
}

func TestTailCtrlC(t *testing.T) {
	// Server that blocks until client disconnects.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Block until client disconnects.
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	var buf bytes.Buffer
	err := TailStream(ctx, TailOptions{
		PingURL:    srv.URL,
		Stdout:     &buf,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("clean Ctrl-C should not return error, got: %v", err)
	}
}
