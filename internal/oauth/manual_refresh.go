// Package oauth provides manual OAuth token refresh helpers.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// RefreshOptions holds parsed flags for oauth refresh.
type RefreshOptions struct {
	// Provider is the OAuth provider name (e.g. "google", "github").
	// Empty when --all is set.
	Provider string
	// AccountID scopes the refresh to a single account.
	// Empty means refresh all accounts for the provider.
	AccountID string
	// All refreshes all providers.
	All bool
	// BaseURL is the nSelf API base URL.
	BaseURL string
	// Stdout/Stderr for output.
	Stdout io.Writer
	Stderr io.Writer
	// HTTPClient allows injection in tests.
	HTTPClient *http.Client
}

// ProviderResult holds per-provider/account refresh outcome.
type ProviderResult struct {
	Provider  string
	AccountID string
	Success   bool
	Err       error
}

// RefreshResult aggregates all per-provider results.
type RefreshResult struct {
	Results  []ProviderResult
	FailCount int
}

// Refresh triggers immediate OAuth token refresh outside the cron schedule.
//
// --all refreshes every account on every provider in parallel.
// Success/failure is visible in the OAuth refresh Prometheus metrics within 5s
// (requires S41-T03 exporter).
// OAuth tokens are never exposed in output (only prefix match in logs).
// Exit non-zero if any account fails.
func Refresh(ctx context.Context, opts RefreshOptions) (*RefreshResult, error) {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("NSELF_API_URL")
	}
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	var providers []string
	if opts.All {
		listed, err := listProviders(ctx, client, baseURL)
		if err != nil {
			return nil, fmt.Errorf("listing OAuth providers: %w", err)
		}
		providers = listed
	} else {
		if opts.Provider == "" {
			return nil, fmt.Errorf("provider name is required (or use --all)")
		}
		providers = []string{opts.Provider}
	}

	fmt.Fprintf(stdout, "Triggering OAuth refresh for provider(s): %s\n", strings.Join(providers, ", "))

	// Run providers in parallel.
	resultCh := make(chan ProviderResult, len(providers)*4)
	var wg sync.WaitGroup

	for _, p := range providers {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			results := refreshProvider(ctx, client, baseURL, p, opts.AccountID)
			for _, r := range results {
				resultCh <- r
			}
		}()
	}

	wg.Wait()
	close(resultCh)

	var allResults []ProviderResult
	var failCount int
	for r := range resultCh {
		allResults = append(allResults, r)
		if r.Err != nil {
			fmt.Fprintf(stderr, "  FAIL provider=%s account=%s: %v\n",
				r.Provider, r.AccountID, r.Err)
			failCount++
		} else {
			fmt.Fprintf(stdout, "  OK   provider=%s account=%s\n",
				r.Provider, r.AccountID)
		}
	}

	result := &RefreshResult{Results: allResults, FailCount: failCount}

	fmt.Fprintf(stdout, "\nRefresh complete: %d/%d succeeded\n",
		len(allResults)-failCount, len(allResults))

	if failCount > 0 {
		return result, fmt.Errorf("%d account(s) failed to refresh", failCount)
	}
	return result, nil
}

// listProviders calls the nSelf API to enumerate configured OAuth providers.
func listProviders(ctx context.Context, client *http.Client, baseURL string) ([]string, error) {
	url := fmt.Sprintf("%s/api/oauth/providers", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list providers returned HTTP %d", resp.StatusCode)
	}

	var providers []string
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		return nil, fmt.Errorf("decoding providers: %w", err)
	}
	return providers, nil
}

// refreshProvider calls the provider's refresh-now endpoint for all accounts
// (or a specific account).
func refreshProvider(ctx context.Context, client *http.Client, baseURL, provider, accountID string) []ProviderResult {
	url := fmt.Sprintf("%s/api/plugins/%s/oauth/refresh-now", baseURL, provider)
	if accountID != "" {
		url += "?account_id=" + accountID
	}

	body := strings.NewReader(`{}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return []ProviderResult{{
			Provider:  provider,
			AccountID: accountID,
			Success:   false,
			Err:       err,
		}}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return []ProviderResult{{
			Provider:  provider,
			AccountID: accountID,
			Success:   false,
			Err:       fmt.Errorf("POST %s: %w", url, err),
		}}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return []ProviderResult{{
			Provider:  provider,
			AccountID: accountID,
			Success:   false,
			Err:       fmt.Errorf("refresh returned HTTP %d", resp.StatusCode),
		}}
	}

	// Decode per-account results from response body.
	var accounts []struct {
		AccountID string `json:"account_id"`
		Success   bool   `json:"success"`
		Error     string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil || len(accounts) == 0 {
		// Single-account response.
		return []ProviderResult{{
			Provider:  provider,
			AccountID: accountID,
			Success:   true,
		}}
	}

	results := make([]ProviderResult, 0, len(accounts))
	for _, a := range accounts {
		r := ProviderResult{
			Provider:  provider,
			AccountID: a.AccountID,
			Success:   a.Success,
		}
		if !a.Success && a.Error != "" {
			r.Err = fmt.Errorf("%s", a.Error)
		}
		results = append(results, r)
	}
	return results
}
