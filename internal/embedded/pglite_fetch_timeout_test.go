package embedded

// Purpose: regression test for siege-security-http-defaultclient-no-timeout
// (Finding #15, reopened 2026-08-31) — proves downloadAndVerify's fetch of the
// pglite WASM artifact is bounded by the caller's context deadline and does
// not hang indefinitely against a stalled CDN, now that the call site was
// migrated off http.DefaultClient.Do onto httptimeout.Installer.Do (which
// still honors context.WithTimeout(ctx, downloadTimeout) — a short parent
// deadline here proves the request actually respects cancellation rather than
// silently blocking on a client with no timeout at all).
// Inputs:  a deliberately-hanging httptest.Server, a 1s parent context.
// Outputs: assertion that downloadAndVerify returns an error close to 1s.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadAndVerifyHangingServerTimesOut(t *testing.T) {
	block := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // hang until the test closes this channel
	}))
	// Close block before srv.Close() (LIFO defer order) so Close() doesn't
	// deadlock waiting for the still-blocked in-flight handler.
	defer srv.Close()
	defer close(block)

	dir := t.TempDir()
	dst := filepath.Join(dir, "pglite.wasm")

	// Parent ctx deadline (1s) is far shorter than downloadTimeout (120s), so
	// dlCtx := context.WithTimeout(ctx, downloadTimeout) inherits this earlier
	// deadline — proving the request is genuinely cancellable, not just
	// theoretically bounded by a 120s wait this test can't afford.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	err := downloadAndVerify(ctx, srv.URL, dst, "irrelevant-digest")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from a hanging server, got nil")
	}
	if elapsed < 900*time.Millisecond {
		t.Fatalf("downloadAndVerify returned too fast (%v) — hang simulation may not have engaged the request", elapsed)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("downloadAndVerify took %v — did not respect the context deadline (http.DefaultClient regression?)", elapsed)
	}
}
