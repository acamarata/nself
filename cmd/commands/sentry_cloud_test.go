package commands

// sentry_cloud_test.go — happy-path tests for the 'nself sentry' cloud
// subcommands against an httptest ɳSentry API.
//
// Purpose: Verify flag→client wiring, --json output, login credential
//   persistence (0600), and the nsentry argv0 alias shim.
// Constraints: no network; HOME isolated per test via t.Setenv.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/sentryapi"
	"github.com/spf13/cobra"
)

// newSentryTestAPI serves a minimal ɳSentry API for command tests.
func newSentryTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer nsk_cmdtest12345" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"bad key"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"tenant_id":"t-cmd","email":"cmd@test.dev","tier":"bundle","quotas":{"monitors":{"used":1,"limit":50}}}`))
	})
	mux.HandleFunc("GET /v1/monitors", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"monitors":[{"id":"m1","name":"api","url":"https://api.test.dev","kind":"http","interval_seconds":60,"status":"up"}]}`))
	})
	mux.HandleFunc("POST /v1/monitors", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"monitor":{"id":"m2","name":"web","url":"https://test.dev","kind":"http","interval_seconds":300,"status":"pending"}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// isolateSentryEnv points the resolver at the test server with a clean HOME.
func isolateSentryEnv(t *testing.T, apiURL string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv(sentryapi.EnvAPIURL, apiURL)
	t.Setenv(sentryapi.EnvAPIKey, "nsk_cmdtest12345")
}

// runSentrySubcommand executes a RunE with the cloud flag trio + extra flags,
// capturing --json output written via cmd.Println.
func runSentrySubcommand(t *testing.T, target *cobra.Command, args []string) (string, error) {
	t.Helper()
	// Fresh command that shares the target's flag definitions (cloud trio +
	// subcommand-specific flags were registered on target in init()).
	c := &cobra.Command{Use: target.Use, RunE: target.RunE, Args: target.Args, SilenceUsage: true}
	c.Flags().AddFlagSet(target.Flags())
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), err
}

func TestSentryMonitorsList_JSON(t *testing.T) {
	srv := newSentryTestAPI(t)
	isolateSentryEnv(t, srv.URL)

	out, err := runSentrySubcommand(t, sentryMonitorsListCmd, []string{"--json"})
	if err != nil {
		t.Fatalf("monitors list: %v (out=%s)", err, out)
	}
	var monitors []sentryapi.Monitor
	if err := json.Unmarshal([]byte(out), &monitors); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if len(monitors) != 1 || monitors[0].ID != "m1" {
		t.Errorf("unexpected monitors: %+v", monitors)
	}
}

func TestSentryWhoami_JSON(t *testing.T) {
	srv := newSentryTestAPI(t)
	isolateSentryEnv(t, srv.URL)

	out, err := runSentrySubcommand(t, sentryWhoamiCmd, []string{"--json"})
	if err != nil {
		t.Fatalf("whoami: %v (out=%s)", err, out)
	}
	var acct sentryapi.Account
	if err := json.Unmarshal([]byte(out), &acct); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if acct.Tier != "bundle" || acct.Quotas["monitors"].Limit != 50 {
		t.Errorf("unexpected account: %+v", acct)
	}
}

func TestSentryLogin_PersistsCredentials0600(t *testing.T) {
	srv := newSentryTestAPI(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(sentryapi.EnvAPIURL, "")
	t.Setenv(sentryapi.EnvAPIKey, "")

	out, err := runSentrySubcommand(t, sentryLoginCmd,
		[]string{"--api-url", srv.URL, "--api-key", "nsk_cmdtest12345"})
	if err != nil {
		t.Fatalf("login: %v (out=%s)", err, out)
	}

	path := filepath.Join(home, ".nself", "sentry.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("credentials not written: %v", err)
	}
	// Unix permission bits don't map onto NTFS ACLs — Windows reports
	// 0666/0444 — so the 0600 assertion only holds on POSIX systems.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("credentials mode = %v, want 0600", info.Mode().Perm())
	}
	creds, err := sentryapi.ReadCredentials()
	if err != nil {
		t.Fatalf("ReadCredentials: %v", err)
	}
	if creds.APIKey != "nsk_cmdtest12345" || creds.APIURL != srv.URL {
		t.Errorf("unexpected credentials: %+v", creds)
	}
}

func TestSentryLogin_RejectsBadKey(t *testing.T) {
	srv := newSentryTestAPI(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv(sentryapi.EnvAPIURL, "")
	t.Setenv(sentryapi.EnvAPIKey, "")

	_, err := runSentrySubcommand(t, sentryLoginCmd,
		[]string{"--api-url", srv.URL, "--api-key", "nsk_wrongkey9999"})
	if err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Errorf("want rejection error, got %v", err)
	}
}

func TestNormalizeInvokedBinary_NSentryAlias(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()

	os.Args = []string{"/usr/local/bin/nsentry", "monitors", "list"}
	normalizeInvokedBinary()
	want := []string{"/usr/local/bin/nsentry", "sentry", "monitors", "list"}
	if strings.Join(os.Args, " ") != strings.Join(want, " ") {
		t.Errorf("nsentry shim: got %v, want %v", os.Args, want)
	}

	// Idempotent: running again must not double-insert.
	normalizeInvokedBinary()
	if strings.Join(os.Args, " ") != strings.Join(want, " ") {
		t.Errorf("nsentry shim not idempotent: got %v", os.Args)
	}

	// Plain nself invocation untouched.
	os.Args = []string{"/usr/local/bin/nself", "status"}
	normalizeInvokedBinary()
	if strings.Join(os.Args, " ") != "/usr/local/bin/nself status" {
		t.Errorf("nself args mutated: %v", os.Args)
	}
}

func TestSentryPresetRegistered(t *testing.T) {
	p, ok := initPresets["sentry"]
	if !ok {
		t.Fatal("sentry preset missing from initPresets")
	}
	if len(p.SuggestedPlugins) != 13 {
		t.Errorf("sentry preset should list the 13 bundle plugins, got %d", len(p.SuggestedPlugins))
	}
	joined := strings.Join(p.Notes, "\n")
	if !strings.Contains(joined, "nsk_dev_local") || !strings.Contains(joined, "3848") {
		t.Errorf("sentry preset notes must document the dev API key and local API port:\n%s", joined)
	}
}
