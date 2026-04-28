// G1-T10: claw_keys --bootstrap headless-mode tests.
//
// Tests cover the validation gate (missing flags exit 2, bad tier exits 2,
// bad email exits 2) and the happy-path POST against a fake server.
//
// We avoid invoking the cobra command directly because runClawKeysCreateBootstrap
// calls os.Exit(2) on validation failure — instead we test in subprocess mode
// (see TestBootstrapValidation_*) and exercise the network path via a
// httptest.Server with the validation bypassed by setting the flags first.

package commands

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestBootstrap_ValidTiers(t *testing.T) {
	wantOK := []string{"owner", "plus", "claw", "chat", "media", "family", "pro", "enterprise"}
	for _, tier := range wantOK {
		if !validBootstrapTiers[tier] {
			t.Errorf("expected tier %q to be valid", tier)
		}
	}
	if validBootstrapTiers["bogus"] {
		t.Error("unexpected tier 'bogus' marked valid")
	}
	if validBootstrapTiers[""] {
		t.Error("empty tier should not be valid")
	}
}

// TestBootstrapValidation_MissingFlags re-execs the test binary with a special
// env var (BOOTSTRAP_TEST_MODE) so we can observe the os.Exit(2) without
// crashing the parent test process.
func TestBootstrapValidation_MissingFlags(t *testing.T) {
	if os.Getenv("BOOTSTRAP_TEST_MODE") == "missing-flags" {
		// Child process: run the bootstrap path with empty flags.
		clawKeysCreateBootstrap = true
		clawKeysCreateName = "ci-key"
		clawKeysCreateOwner = ""
		clawKeysCreateTier = ""
		clawKeysCreateMachineID = ""
		_ = runClawKeysCreateBootstrap(nil)
		// runClawKeysCreateBootstrap should have called os.Exit(2). If we
		// reach here, exit 0 — the parent will see "did not exit 2" and fail.
		os.Exit(0)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestBootstrapValidation_MissingFlags")
	cmd.Env = append(os.Environ(), "BOOTSTRAP_TEST_MODE=missing-flags")
	out, _ := cmd.CombinedOutput()
	exitErr, ok := getExitErr(cmd)
	if !ok || exitErr.ExitCode() != bootstrapExitCode {
		t.Fatalf("expected exit code %d, got output: %s", bootstrapExitCode, string(out))
	}
	if !strings.Contains(string(out), "--owner-email") || !strings.Contains(string(out), "--tier") || !strings.Contains(string(out), "--machine-id") {
		t.Errorf("expected stderr to list missing flags, got: %s", string(out))
	}
}

func TestBootstrapValidation_InvalidTier(t *testing.T) {
	if os.Getenv("BOOTSTRAP_TEST_MODE") == "bad-tier" {
		clawKeysCreateBootstrap = true
		clawKeysCreateName = "ci-key"
		clawKeysCreateOwner = "ci@example.com"
		clawKeysCreateTier = "bogus"
		clawKeysCreateMachineID = "m1"
		_ = runClawKeysCreateBootstrap(nil)
		os.Exit(0)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestBootstrapValidation_InvalidTier")
	cmd.Env = append(os.Environ(), "BOOTSTRAP_TEST_MODE=bad-tier")
	out, _ := cmd.CombinedOutput()
	exitErr, ok := getExitErr(cmd)
	if !ok || exitErr.ExitCode() != bootstrapExitCode {
		t.Fatalf("expected exit code %d, got output: %s", bootstrapExitCode, string(out))
	}
	if !strings.Contains(string(out), "invalid --tier") {
		t.Errorf("expected 'invalid --tier' message, got: %s", string(out))
	}
}

func TestBootstrapValidation_InvalidEmail(t *testing.T) {
	if os.Getenv("BOOTSTRAP_TEST_MODE") == "bad-email" {
		clawKeysCreateBootstrap = true
		clawKeysCreateName = "ci-key"
		clawKeysCreateOwner = "not-an-email"
		clawKeysCreateTier = "owner"
		clawKeysCreateMachineID = "m1"
		_ = runClawKeysCreateBootstrap(nil)
		os.Exit(0)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestBootstrapValidation_InvalidEmail")
	cmd.Env = append(os.Environ(), "BOOTSTRAP_TEST_MODE=bad-email")
	out, _ := cmd.CombinedOutput()
	exitErr, ok := getExitErr(cmd)
	if !ok || exitErr.ExitCode() != bootstrapExitCode {
		t.Fatalf("expected exit code %d, got output: %s", bootstrapExitCode, string(out))
	}
	if !strings.Contains(string(out), "valid email") {
		t.Errorf("expected 'valid email' message, got: %s", string(out))
	}
}

// TestBootstrap_HappyPath spins up a fake claw server, points the CLI's
// server-URL config at it, and confirms that the bootstrap path emits the
// raw key on stdout exactly as expected. We exercise the function directly
// (no subprocess) since the validation passes and no os.Exit is triggered.
func TestBootstrap_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/claw/v1/api-keys/bootstrap" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["owner_email"] != "ci@example.com" || body["tier"] != "owner" || body["machine_id"] != "host-1" {
			t.Errorf("unexpected request body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"key": "nself_pro_TESTKEY1234567890abcdef",
			"id":  "key-uuid-1",
		})
	}))
	defer srv.Close()

	t.Setenv("NSELF_CLAW_SERVER", srv.URL)

	clawKeysCreateBootstrap = true
	clawKeysCreateName = "ci-key"
	clawKeysCreateOwner = "ci@example.com"
	clawKeysCreateTier = "owner"
	clawKeysCreateMachineID = "host-1"
	defer func() {
		clawKeysCreateBootstrap = false
		clawKeysCreateName = ""
		clawKeysCreateOwner = ""
		clawKeysCreateTier = ""
		clawKeysCreateMachineID = ""
	}()

	stdout := captureLocalStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runClawKeysCreateBootstrap(cmd); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if strings.TrimSpace(stdout) != "nself_pro_TESTKEY1234567890abcdef" {
		t.Fatalf("expected raw key on stdout, got: %q", stdout)
	}
}

// captureLocalStdout swaps os.Stdout for a pipe, runs fn, and returns
// everything it wrote. Local helper so we don't collide with config_test.go's
// captureStdout signature.
func captureLocalStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

// getExitErr extracts an *exec.ExitError from a finished cmd.
func getExitErr(cmd *exec.Cmd) (*exec.ExitError, bool) {
	if cmd.ProcessState == nil {
		return nil, false
	}
	if cmd.ProcessState.Success() {
		return nil, false
	}
	// On Unix, ProcessState.Sys() is *syscall.WaitStatus. We just need ExitCode().
	// Use exec.ExitError which wraps ProcessState.
	ee := &exec.ExitError{ProcessState: cmd.ProcessState}
	return ee, true
}
