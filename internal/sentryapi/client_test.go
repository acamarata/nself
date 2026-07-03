package sentryapi

// client_test.go — httptest-backed unit tests for the sentryapi client.
//
// Purpose: Verify auth header wiring, envelope decoding, typed error mapping
//   (401 → ErrUnauthorized, 402/429 → ErrQuotaExceeded), and config resolution.
// Constraints: no network; every test runs against net/http/httptest.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// newTestServer returns a server that asserts the Bearer key and serves
// canned responses per path.
func newTestServer(t *testing.T, wantKey string, routes map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wantKey != "" {
			if got := r.Header.Get("Authorization"); got != "Bearer "+wantKey {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"bad key"}}`))
				return
			}
		}
		key := r.Method + " " + r.URL.Path
		if h, ok := routes[key]; ok {
			h(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"no route"}}`))
	}))
}

func jsonHandler(status int, body string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestListMonitors(t *testing.T) {
	srv := newTestServer(t, "nsk_testkey123", map[string]func(http.ResponseWriter, *http.Request){
		"GET /v1/monitors": jsonHandler(200, `{"monitors":[
			{"id":"m1","name":"api","url":"https://api.example.com","kind":"http","interval_seconds":60,"status":"up","paused":false},
			{"id":"m2","name":"web","url":"https://example.com","kind":"http","interval_seconds":300,"status":"paused","paused":true}
		]}`),
	})
	defer srv.Close()

	c := New(srv.URL, "nsk_testkey123")
	mons, err := c.ListMonitors(context.Background())
	if err != nil {
		t.Fatalf("ListMonitors: %v", err)
	}
	if len(mons) != 2 {
		t.Fatalf("want 2 monitors, got %d", len(mons))
	}
	if mons[0].ID != "m1" || mons[0].Kind != "http" || mons[0].IntervalSeconds != 60 {
		t.Errorf("monitor[0] decoded wrong: %+v", mons[0])
	}
	if !mons[1].Paused {
		t.Errorf("monitor[1] should be paused")
	}
}

func TestCreateMonitor(t *testing.T) {
	srv := newTestServer(t, "nsk_testkey123", map[string]func(http.ResponseWriter, *http.Request){
		"POST /v1/monitors": func(w http.ResponseWriter, r *http.Request) {
			var req CreateMonitorRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode create body: %v", err)
			}
			if req.Name != "api" || req.URL != "https://api.example.com" || req.IntervalSeconds != 60 {
				t.Errorf("unexpected create body: %+v", req)
			}
			jsonHandler(201, `{"monitor":{"id":"m9","name":"api","url":"https://api.example.com","kind":"http","interval_seconds":60,"status":"pending"}}`)(w, r)
		},
	})
	defer srv.Close()

	c := New(srv.URL, "nsk_testkey123")
	m, err := c.CreateMonitor(context.Background(), CreateMonitorRequest{
		Name: "api", URL: "https://api.example.com", Kind: "http", IntervalSeconds: 60,
	})
	if err != nil {
		t.Fatalf("CreateMonitor: %v", err)
	}
	if m.ID != "m9" || m.Status != "pending" {
		t.Errorf("created monitor decoded wrong: %+v", m)
	}
}

func TestDeleteAndPauseMonitor(t *testing.T) {
	srv := newTestServer(t, "nsk_testkey123", map[string]func(http.ResponseWriter, *http.Request){
		"DELETE /v1/monitors/m1":      jsonHandler(204, ""),
		"POST /v1/monitors/m1/pause":  jsonHandler(200, `{"monitor":{"id":"m1","status":"paused","paused":true}}`),
		"POST /v1/monitors/m1/resume": jsonHandler(200, `{"monitor":{"id":"m1","status":"pending","paused":false}}`),
	})
	defer srv.Close()

	c := New(srv.URL, "nsk_testkey123")
	if err := c.DeleteMonitor(context.Background(), "m1"); err != nil {
		t.Fatalf("DeleteMonitor: %v", err)
	}
	m, err := c.PauseMonitor(context.Background(), "m1", true)
	if err != nil {
		t.Fatalf("PauseMonitor: %v", err)
	}
	if !m.Paused {
		t.Errorf("expected paused=true, got %+v", m)
	}
	m, err = c.PauseMonitor(context.Background(), "m1", false)
	if err != nil {
		t.Fatalf("ResumeMonitor: %v", err)
	}
	if m.Paused {
		t.Errorf("expected paused=false after resume, got %+v", m)
	}
}

func TestIncidentsLifecycle(t *testing.T) {
	srv := newTestServer(t, "nsk_testkey123", map[string]func(http.ResponseWriter, *http.Request){
		"GET /v1/incidents": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("status") != "open" {
				t.Errorf("want status=open filter, got %q", r.URL.RawQuery)
			}
			jsonHandler(200, `{"incidents":[{"id":"i1","monitor_id":"m1","title":"api down","status":"open","severity":"critical","started_at":"2026-07-01T00:00:00Z"}]}`)(w, r)
		},
		"POST /v1/incidents/i1/ack":     jsonHandler(200, `{"incident":{"id":"i1","status":"acknowledged"}}`),
		"POST /v1/incidents/i1/resolve": jsonHandler(200, `{"incident":{"id":"i1","status":"resolved"}}`),
	})
	defer srv.Close()

	c := New(srv.URL, "nsk_testkey123")
	incs, err := c.ListIncidents(context.Background(), "open")
	if err != nil {
		t.Fatalf("ListIncidents: %v", err)
	}
	if len(incs) != 1 || incs[0].Title != "api down" {
		t.Fatalf("incidents decoded wrong: %+v", incs)
	}
	if inc, err := c.AckIncident(context.Background(), "i1"); err != nil || inc.Status != "acknowledged" {
		t.Errorf("AckIncident: inc=%+v err=%v", inc, err)
	}
	if inc, err := c.ResolveIncident(context.Background(), "i1"); err != nil || inc.Status != "resolved" {
		t.Errorf("ResolveIncident: inc=%+v err=%v", inc, err)
	}
}

func TestStatusPagesAndChannels(t *testing.T) {
	srv := newTestServer(t, "nsk_testkey123", map[string]func(http.ResponseWriter, *http.Request){
		"GET /v1/status-pages":             jsonHandler(200, `{"status_pages":[{"id":"s1","name":"Public","slug":"public","url":"https://sentry.nself.org/s/public","public":true}]}`),
		"POST /v1/status-pages":            jsonHandler(201, `{"status_page":{"id":"s2","name":"API","slug":"api","public":true}}`),
		"GET /v1/alerts/channels":          jsonHandler(200, `{"channels":[{"id":"c1","kind":"email","target":"ops@example.com","enabled":true}]}`),
		"POST /v1/alerts/channels/c1/test": jsonHandler(200, `{"delivered":true,"detail":"test email sent"}`),
	})
	defer srv.Close()

	c := New(srv.URL, "nsk_testkey123")
	pages, err := c.ListStatusPages(context.Background())
	if err != nil || len(pages) != 1 || pages[0].Slug != "public" {
		t.Fatalf("ListStatusPages: pages=%+v err=%v", pages, err)
	}
	sp, err := c.CreateStatusPage(context.Background(), CreateStatusPageRequest{Name: "API", Slug: "api"})
	if err != nil || sp.ID != "s2" {
		t.Fatalf("CreateStatusPage: sp=%+v err=%v", sp, err)
	}
	chans, err := c.ListAlertChannels(context.Background())
	if err != nil || len(chans) != 1 || chans[0].Kind != "email" {
		t.Fatalf("ListAlertChannels: chans=%+v err=%v", chans, err)
	}
	res, err := c.TestAlertChannel(context.Background(), "c1")
	if err != nil || !res.Delivered {
		t.Fatalf("TestAlertChannel: res=%+v err=%v", res, err)
	}
}

func TestWhoAmI(t *testing.T) {
	srv := newTestServer(t, "nsk_testkey123", map[string]func(http.ResponseWriter, *http.Request){
		"GET /v1/me": jsonHandler(200, `{"tenant_id":"t1","email":"dev@example.com","tier":"bundle","quotas":{"monitors":{"used":3,"limit":50}}}`),
	})
	defer srv.Close()

	c := New(srv.URL, "nsk_testkey123")
	acct, err := c.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("WhoAmI: %v", err)
	}
	if acct.Tier != "bundle" || acct.Quotas["monitors"].Limit != 50 {
		t.Errorf("account decoded wrong: %+v", acct)
	}
}

func TestErrorMapping(t *testing.T) {
	srv := newTestServer(t, "nsk_rightkey12", map[string]func(http.ResponseWriter, *http.Request){
		"GET /v1/monitors": jsonHandler(429, `{"error":{"code":"quota_exceeded","message":"monitor limit reached (10/10)"}}`),
	})
	defer srv.Close()

	// Wrong key → 401 → ErrUnauthorized.
	c := New(srv.URL, "nsk_wrongkey12")
	if _, err := c.ListMonitors(context.Background()); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}

	// Right key but 429 → ErrQuotaExceeded with server message.
	c = New(srv.URL, "nsk_rightkey12")
	_, err := c.ListMonitors(context.Background())
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Errorf("want ErrQuotaExceeded, got %v", err)
	}
}

func TestResolvePrecedence(t *testing.T) {
	// Isolate HOME so no real credentials file leaks in.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv(EnvAPIURL, "")
	t.Setenv(EnvAPIKey, "")

	// Defaults.
	u, k := Resolve("", "")
	if u != DefaultAPIURL || k != "" {
		t.Errorf("defaults: got url=%q key=%q", u, k)
	}

	// Credentials file.
	if err := WriteCredentials(&Credentials{APIURL: "http://localhost:3848", APIKey: "nsk_filekey123"}); err != nil {
		t.Fatalf("WriteCredentials: %v", err)
	}
	u, k = Resolve("", "")
	if u != "http://localhost:3848" || k != "nsk_filekey123" {
		t.Errorf("file: got url=%q key=%q", u, k)
	}

	// File permissions must be 0600.
	info, err := os.Stat(filepath.Join(tmp, ".nself", "sentry.json"))
	if err != nil {
		t.Fatalf("stat credentials: %v", err)
	}
	// Unix permission bits don't map onto NTFS ACLs — Windows reports
	// 0666/0444, so the 0600 assertion only holds on POSIX systems.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("credentials mode = %v, want 0600", info.Mode().Perm())
	}

	// Env beats file.
	t.Setenv(EnvAPIURL, "http://envhost:9999")
	t.Setenv(EnvAPIKey, "nsk_envkey1234")
	u, k = Resolve("", "")
	if u != "http://envhost:9999" || k != "nsk_envkey1234" {
		t.Errorf("env: got url=%q key=%q", u, k)
	}

	// Flag beats env.
	u, k = Resolve("http://flaghost:1111", "nsk_flagkey123")
	if u != "http://flaghost:1111" || k != "nsk_flagkey123" {
		t.Errorf("flag: got url=%q key=%q", u, k)
	}

	// Logout removes the file.
	if err := DeleteCredentials(); err != nil {
		t.Fatalf("DeleteCredentials: %v", err)
	}
	t.Setenv(EnvAPIURL, "")
	t.Setenv(EnvAPIKey, "")
	if _, err := ReadCredentials(); !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("want ErrNotLoggedIn after delete, got %v", err)
	}
}

func TestValidateKeyFormat(t *testing.T) {
	if err := ValidateKeyFormat("nsk_devlocal0000"); err != nil {
		t.Errorf("valid key rejected: %v", err)
	}
	if err := ValidateKeyFormat("sk_wrongprefix"); err == nil {
		t.Errorf("wrong prefix accepted")
	}
	if err := ValidateKeyFormat("nsk_a"); err == nil {
		t.Errorf("short key accepted")
	}
}
