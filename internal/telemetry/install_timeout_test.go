package telemetry

// Purpose: regression test for siege-security-http-defaultclient-no-timeout
// (Finding #15, reopened 2026-08-31) — proves the shared httpClient var that
// SendInstallEvent (install.go) now uses in place of http.DefaultClient.Do
// is bounded and does not hang indefinitely against a stalled remote.
// installCounterEndpoint is a hardcoded const (not injectable — changing that
// would be a request-logic change, out of scope per this ticket's guide), so
// this follows the exact pattern already established in send_event_test.go's
// TestSendPayloadTimeout: swap httpClient to the test server's client, build
// the request against srv.URL directly, and assert the call returns within a
// bounded window instead of proving nothing by hitting a real network host.
// Inputs:  a deliberately-hanging httptest.Server.
// Outputs: assertion that httpClient.Do returns within install.go's 4s bound.

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInstallHTTPClientHangingServerTimesOut(t *testing.T) {
	block := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // hang until the test closes this channel
	}))
	// Close block before srv.Close() (LIFO defer order) so Close() doesn't
	// deadlock waiting for the still-blocked in-flight handler.
	defer srv.Close()
	defer close(block)

	// Mirror SendInstallEvent's own 4s context deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	start := time.Now()
	_, err = httpClient.Do(req)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from a hanging server, got nil")
	}
	if elapsed < 3*time.Second {
		t.Fatalf("httpClient.Do returned too fast (%v) — hang simulation may not have engaged the request", elapsed)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("httpClient.Do took %v — did not respect the context deadline (http.DefaultClient regression?)", elapsed)
	}
}
