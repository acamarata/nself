package ci

// Purpose: regression test for siege-security-http-defaultclient-no-timeout
// (Finding #15, reopened 2026-08-31) — proves emitEvent's HTTP POST to
// NSELF_CI_EVENT_SINK is bounded by a real deadline and does not hang
// indefinitely against a stalled remote, now that the call site was migrated
// off http.DefaultClient.Do onto httptimeout.Default.Do (which still respects
// the request's own 5s context deadline set in emitEvent).
// Inputs:  a deliberately-hanging httptest.Server (blocks until test cleanup).
// Outputs: assertion that emitEvent returns within a bounded wall-clock window.
// Constraints: no network access; server-local only.

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestEmitEventHangingServerTimesOut points NSELF_CI_EVENT_SINK at a server
// that never responds and asserts emitEvent returns (does not block forever)
// within a bounded window close to its configured 5s context deadline —
// proving the http.DefaultClient.Do -> httptimeout.Default.Do migration
// preserved the existing timeout behavior instead of silently disabling it.
func TestEmitEventHangingServerTimesOut(t *testing.T) {
	block := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // hang until the test closes this channel
	}))
	// Close block BEFORE srv.Close(): Close() waits for in-flight handlers to
	// return, so closing it first (LIFO defer order) unblocks the handler and
	// lets Close() proceed instead of deadlocking on the still-blocked conn.
	defer srv.Close()
	defer close(block)

	t.Setenv("NSELF_CI_EVENT_SINK", srv.URL)

	start := time.Now()
	emitEvent(completionEvent{Repo: "nself-org/cli", Status: "success"})
	elapsed := time.Since(start)

	// emitEvent's own ctx has a 5s deadline. It must return in well under the
	// Go test binary's default timeout, and not near-instantly (which would
	// mean the request never really hung the round trip).
	if elapsed < 4*time.Second {
		t.Fatalf("emitEvent returned too fast (%v) — hang simulation may not have engaged the request", elapsed)
	}
	if elapsed > 15*time.Second {
		t.Fatalf("emitEvent took %v — did not respect its 5s context deadline (http.DefaultClient regression?)", elapsed)
	}
}
