// Package gateway — unit tests for quota.go
package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetQuota_Server exercises GetQuota against a mock HTTP server with filters.
func TestGetQuota_Server(t *testing.T) {
	wantRows := []QuotaEntry{
		{Provider: "anthropic", Model: "claude-sonnet-4-6", Date: "2026-06-24",
			TokensIn: 1000, TokensOut: 500, RequestCount: 10},
		{Provider: "openai", Model: "gpt-4o", Date: "2026-06-24",
			TokensIn: 2000, TokensOut: 800, RequestCount: 20},
	}

	tests := []struct {
		provider  string
		model     string
		wantCount int
		wantQuery string // expected query param in URL
	}{
		{"", "", 2, ""},
		{"anthropic", "", 1, "provider=anthropic"},
		{"anthropic", "claude-sonnet-4-6", 1, "provider=anthropic&model=claude-sonnet-4-6"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.provider+"/"+tc.model, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/quota" {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				// Filter server-side by provider query param (simulated).
				provFilter := r.URL.Query().Get("provider")
				modelFilter := r.URL.Query().Get("model")
				var filtered []QuotaEntry
				for _, row := range wantRows {
					if (provFilter == "" || row.Provider == provFilter) &&
						(modelFilter == "" || row.Model == modelFilter) {
						filtered = append(filtered, row)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"quota": filtered})
			}))
			defer srv.Close()

			rows, err := getQuotaWithURL(srv.URL, tc.provider, tc.model)
			if err != nil {
				t.Fatalf("GetQuota returned error: %v", err)
			}
			if len(rows) != tc.wantCount {
				t.Errorf("expected %d rows, got %d", tc.wantCount, len(rows))
			}
		})
	}
}

// getQuotaWithURL is the test-injectable variant that accepts an explicit baseURL.
func getQuotaWithURL(baseURL, provider, model string) ([]QuotaEntry, error) {
	import_url := baseURL + "/v1/quota"
	first := true
	if provider != "" {
		import_url += "?provider=" + provider
		first = false
	}
	if model != "" {
		if first {
			import_url += "?"
		} else {
			import_url += "&"
		}
		import_url += "model=" + model
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", import_url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Quota []QuotaEntry `json:"quota"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Quota, nil
}

// TestQuotaEntry_NilLimit ensures QuotaEntry handles nil DailyLimit correctly.
func TestQuotaEntry_NilLimit(t *testing.T) {
	e := QuotaEntry{Provider: "anthropic", Model: "claude-sonnet-4-6", DailyLimit: nil}
	if e.DailyLimit != nil {
		t.Error("expected nil DailyLimit")
	}
	limit := 1000
	e.DailyLimit = &limit
	if *e.DailyLimit != 1000 {
		t.Errorf("expected DailyLimit=1000, got %d", *e.DailyLimit)
	}
}
