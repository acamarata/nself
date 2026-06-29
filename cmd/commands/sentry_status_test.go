package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// operationalPayload returns a fully-operational status page JSON body.
func operationalPayload() string {
	return `{
		"site_name": "nSelf Status",
		"overall_status": "operational",
		"components": [
			{"id": "c1", "name": "API", "status": "operational", "updated_at": "2026-06-01T00:00:00Z"},
			{"id": "c2", "name": "Auth", "status": "operational", "updated_at": "2026-06-01T00:00:00Z"},
			{"id": "c3", "name": "DB",   "status": "operational", "updated_at": "2026-06-01T00:00:00Z"}
		]
	}`
}

// degradedPayload returns a status page where one component is degraded.
func degradedPayload() string {
	return `{
		"site_name": "nSelf Status",
		"overall_status": "degraded_performance",
		"components": [
			{"id": "c1", "name": "API",  "status": "operational",         "updated_at": "2026-06-01T00:00:00Z"},
			{"id": "c2", "name": "Auth", "status": "degraded_performance","updated_at": "2026-06-01T00:00:00Z"},
			{"id": "c3", "name": "DB",   "status": "partial_outage",      "updated_at": "2026-06-01T00:00:00Z"}
		]
	}`
}

// TestFetchStatusPage_Operational verifies that a fully-operational response
// is parsed correctly and no outage components are reported.
func TestFetchStatusPage_Operational(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(operationalPayload()))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	page, err := fetchStatusPage(context.Background(), client, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.OverallStatus != "operational" {
		t.Errorf("expected overall_status=operational, got %q", page.OverallStatus)
	}
	if len(page.Components) != 3 {
		t.Errorf("expected 3 components, got %d", len(page.Components))
	}

	// Exit-code logic: no outage → should be exit 0.
	for _, c := range page.Components {
		if outageStatuses[c.Status] {
			t.Errorf("unexpected outage component %q with status %q", c.Name, c.Status)
		}
	}
}

// TestFetchStatusPage_Degraded verifies that a degraded response is parsed and
// outage components are correctly identified for exit-1 logic.
func TestFetchStatusPage_Degraded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(degradedPayload()))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	page, err := fetchStatusPage(context.Background(), client, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.OverallStatus == "operational" {
		t.Errorf("expected non-operational overall_status, got %q", page.OverallStatus)
	}

	// DB component has partial_outage → exitBad = true.
	exitBad := page.OverallStatus != "operational"
	var downNames []string
	for _, c := range page.Components {
		if outageStatuses[c.Status] {
			downNames = append(downNames, c.Name)
		}
	}
	exitBad = exitBad || len(downNames) > 0
	if !exitBad {
		t.Error("expected exitBad=true for degraded/outage response")
	}
	if len(downNames) != 1 || downNames[0] != "DB" {
		t.Errorf("expected down=[DB], got %v", downNames)
	}
}

// TestFetchStatusPage_Non200 verifies that a non-200 HTTP response returns an error.
func TestFetchStatusPage_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	_, err := fetchStatusPage(context.Background(), client, srv.URL)
	if err == nil {
		t.Fatal("expected error for HTTP 503, got nil")
	}
}

// TestStatusJSONOutput verifies the normalized JSON output structure matches spec.
func TestStatusJSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(degradedPayload()))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	page, err := fetchStatusPage(context.Background(), client, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Build the same normalized output as runSentryStatus --json would.
	operationalCount := 0
	degradedCount := 0
	var downNames []string
	var compacts []compactComponent

	for _, c := range page.Components {
		compacts = append(compacts, compactComponent{Name: c.Name, Status: c.Status})
		if c.Status == "operational" {
			operationalCount++
		} else if degradedStatuses[c.Status] {
			degradedCount++
		} else if outageStatuses[c.Status] {
			degradedCount++
			downNames = append(downNames, c.Name)
		}
	}

	out := statusJSONOut{
		Overall:     page.OverallStatus,
		Total:       len(page.Components),
		Operational: operationalCount,
		Degraded:    degradedCount,
		Down:        downNames,
		Components:  compacts,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var roundTrip statusJSONOut
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if roundTrip.Total != 3 {
		t.Errorf("expected total=3, got %d", roundTrip.Total)
	}
	if roundTrip.Operational != 1 {
		t.Errorf("expected operational=1, got %d", roundTrip.Operational)
	}
	if roundTrip.Degraded != 2 {
		t.Errorf("expected degraded=2, got %d", roundTrip.Degraded)
	}
	if len(roundTrip.Down) != 1 || roundTrip.Down[0] != "DB" {
		t.Errorf("expected down=[DB], got %v", roundTrip.Down)
	}
	if roundTrip.Overall != "degraded_performance" {
		t.Errorf("expected overall=degraded_performance, got %q", roundTrip.Overall)
	}
}

// TestFetchStatusPage_TokenQueryParam verifies the token is appended to the URL.
func TestFetchStatusPage_TokenQueryParam(t *testing.T) {
	var receivedToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.URL.Query().Get("t")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(operationalPayload()))
	}))
	defer srv.Close()

	// Manually append token as runSentryStatus does.
	targetURL := srv.URL + "?t=mytoken"
	client := &http.Client{Timeout: 5 * time.Second}
	_, err := fetchStatusPage(context.Background(), client, targetURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedToken != "mytoken" {
		t.Errorf("expected token=mytoken, got %q", receivedToken)
	}
}
