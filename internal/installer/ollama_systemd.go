package installer

// Purpose: systemctl/systemd/firewall/probe helpers plus local-state persistence for the Ollama installer, split out of ollama.go's Install entrypoint.
// Inputs: an *InstallResult and context for systemctl/probe operations.
// Outputs: systemd unit state changes, a written/read local-state JSON file, and readiness probe results.
// Constraints: split out of ollama.go as a pure move (CLI-R12); no behavior change.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
)

// ---- Helpers ----

func systemctl(args ...string) error { //nolint:unused // kept: Ollama service management is unreachable, no build tag; see qa/bugs/declared-but-never-wired-symbols.md
	return systemctlContext(context.Background(), args...)
}

func systemctlContext(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func systemctlActive(unit string) bool { //nolint:unused // kept: Ollama service management is unreachable, no build tag; see qa/bugs/declared-but-never-wired-symbols.md
	return systemctlActiveContext(context.Background(), unit)
}

func systemctlActiveContext(ctx context.Context, unit string) bool {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", unit)
	return cmd.Run() == nil
}

func downloadAndRunInstaller(ctx context.Context, log func(string, string, map[string]any)) error {
	const installerURL = "https://ollama.com/install.sh"

	// Download and verify checksum unconditionally. The pinned SHA is always
	// non-empty; any mismatch aborts installation before execution.
	//
	// DownloadAndVerify returns a path inside a private 0700 temporary
	// directory. We must clean up the directory (not just the file) on exit.
	tmpPath, err := DownloadAndVerify(ctx, installerURL, ExpectedOllamaInstallChecksum())
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(tmpPath)) }()

	log("info", "install.sh checksum verified", map[string]any{"sha256": ExpectedOllamaInstallChecksum()})

	// Execute the verified script via sh. The script lives in a private 0700
	// directory; the path cannot be replaced by a same-uid process between the
	// checksum verification above and this exec call.
	//
	// install.sh itself shells out further — it curls the ollama release
	// tarball and unpacks it through zstd. Two things must hold or a
	// cancelled/short-lived ctx (as in any test that runs this via `nself
	// start`'s AI auto-install) leaves curl/zstd running as orphans that hang
	// the whole test binary:
	//
	//  1. Process-group isolation (setProcGroupAttr + the Cancel hook below):
	//     without it, ctx cancellation only kills the immediate `sh` process
	//     via SIGKILL — curl/zstd are reparented and keep running.
	//  2. Pipe-based I/O forwarding instead of direct `cmd.Stdout = os.Stdout`
	//     fd inheritance: if curl/zstd inherit the raw fd of this process's
	//     real stdout, any surviving orphan keeps that fd open even after
	//     this function returns, and `go test` reports "Test I/O incomplete
	//     Ns after exiting" for the entire package. Piping means only our own
	//     io.Copy goroutines below ever touch os.Stdout/os.Stderr directly.
	//
	// Same failure mode, same fix, as internal/docker/compose_exec.go's Run.
	cmd := exec.CommandContext(ctx, "sh", tmpPath)
	cmd.WaitDelay = 5 * time.Second
	setProcGroupAttr(cmd)
	cmd.Cancel = func() error {
		killProcessGroup(cmd)
		return cmd.Process.Kill()
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return errf(ErrOllamaInstallFailed, "install.sh stdout pipe", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return errf(ErrOllamaInstallFailed, "install.sh stderr pipe", err)
	}

	if err := cmd.Start(); err != nil {
		return errf(ErrOllamaInstallFailed, "install.sh start", err)
	}

	done := make(chan struct{}, 2)
	go func() { io.Copy(os.Stdout, stdoutPipe); done <- struct{}{} }() //nolint:errcheck
	go func() { io.Copy(os.Stderr, stderrPipe); done <- struct{}{} }() //nolint:errcheck

	waitErr := cmd.Wait()
	<-done
	<-done

	if waitErr != nil {
		return errf(ErrOllamaInstallFailed, "install.sh exec", waitErr)
	}
	return nil
}

const systemdOverridePath = "/etc/systemd/system/ollama.service.d/override.conf"

func writeSystemdOverride(bind string) error {
	if err := os.MkdirAll(filepath.Dir(systemdOverridePath), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(`[Service]
Environment="OLLAMA_HOST=%s"
Environment="OLLAMA_MODELS=/var/lib/ollama/models"
Environment="OLLAMA_NUM_PARALLEL=2"
Environment="OLLAMA_MAX_LOADED_MODELS=2"
Environment="OLLAMA_KEEP_ALIVE=5m"
`, bind)
	return os.WriteFile(systemdOverridePath, []byte(content), 0o644)
}

func configureFirewall(ctx context.Context) error {
	// Prefer ufw if active.
	if exec.CommandContext(ctx, "ufw", "status").Run() == nil {
		// ufw exists; try allow rule.
		cmd := exec.CommandContext(ctx, "ufw", "allow", "from", "172.16.0.0/12", "to", "any", "port", "11434")
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	// Fall back to iptables.
	rules := [][]string{
		{"-I", "INPUT", "-i", "docker0", "-p", "tcp", "--dport", "11434", "-j", "ACCEPT"},
		{"-I", "INPUT", "-s", "172.16.0.0/12", "-p", "tcp", "--dport", "11434", "-j", "ACCEPT"},
	}
	for _, r := range rules {
		if err := exec.CommandContext(ctx, "iptables", r...).Run(); err != nil {
			return errf(ErrIptablesNoPermission, "iptables rule", err)
		}
	}
	// Persist via iptables-save (best-effort).
	_ = exec.CommandContext(ctx, "sh", "-c", "iptables-save > /etc/iptables/rules.v4 2>/dev/null || true").Run()
	return nil
}

func probeUntilReady(ctx context.Context, url string, total time.Duration) error {
	deadline := time.Now().Add(total)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := httptimeout.Installer.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("probe timeout after %s", total)
}

func probeOllamaVersion(ctx context.Context) string {
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:11434/api/version", nil)
	resp, err := httptimeout.Installer.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	var body struct {
		Version string `json:"version"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return body.Version
}

func pullOllamaModel(ctx context.Context, name string) error {
	payload := map[string]any{"name": name, "stream": false}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST",
		"http://127.0.0.1:11434/api/pull", strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("pull %s HTTP %d: %s", name, resp.StatusCode, string(body))
	}
	// Drain remaining stream.
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func writeLocalState(res *InstallResult) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, LocalStateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// ReadLocalState returns the persisted install state if present.
func ReadLocalState() (*InstallResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, LocalStateFile)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r InstallResult
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
