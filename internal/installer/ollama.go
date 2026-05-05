package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/httptimeout"
)

// Error codes (per spec §3.2.1).
const (
	ErrOllamaInstallFailed  = "OLLAMA_INSTALL_FAILED"
	ErrSystemdUnavailable   = "SYSTEMD_UNAVAILABLE"
	ErrIptablesNoPermission = "IPTABLES_NO_PERMISSION"
	ErrPortBindConflict     = "PORT_BIND_CONFLICT"
	ErrRAMInsufficient      = "RAM_INSUFFICIENT"
	ErrUnsupportedOS        = "UNSUPPORTED_OS"
)

// InstallerError wraps a coded installer error.
type InstallerError struct {
	Code string
	Msg  string
	Err  error
}

func (e *InstallerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Msg, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

func errf(code, msg string, err error) *InstallerError {
	return &InstallerError{Code: code, Msg: msg, Err: err}
}

// InstallOptions controls the installer flow.
type InstallOptions struct {
	SkipModels bool
	Model      string // optional single model to pull (overrides matrix)
	Bind       string // host:port, default 0.0.0.0:11434
	Yes        bool   // non-interactive
	JSON       bool
	LogFn      func(level, msg string, kv map[string]any)
}

// InstallResult summarises an install run.
type InstallResult struct {
	AlreadyInstalled bool      `json:"already_installed"`
	OllamaVersion    string    `json:"ollama_version,omitempty"`
	Bind             string    `json:"bind"`
	Tier             TierKey   `json:"ram_tier"`
	ModelsPulled     []string  `json:"models_pulled"`
	CompletedAt      time.Time `json:"completed_at"`
}

// LocalStateFile is where the installer records its run.
const LocalStateFile = ".nself/ai/local-state.json"

// Install performs the full install flow. Returns an *InstallerError on failure.
func Install(ctx context.Context, opts InstallOptions) (*InstallResult, error) {
	log := opts.LogFn
	if log == nil {
		log = func(level, msg string, kv map[string]any) {}
	}

	// Step 1: OS check.
	if runtime.GOOS != "linux" {
		return nil, errf(ErrUnsupportedOS,
			"macOS/Windows: install Ollama manually from https://ollama.com", nil)
	}

	bind := opts.Bind
	if bind == "" {
		bind = "0.0.0.0:11434"
	}
	res := &InstallResult{Bind: bind}

	// Step 2: systemd status.
	systemdActive := systemctlActiveContext(ctx, "ollama")
	if systemdActive {
		log("info", "ollama already running; skipping install", nil)
		res.AlreadyInstalled = true
	} else {
		// Step 3: download + verify + run install.sh.
		if err := downloadAndRunInstaller(ctx, log); err != nil {
			return nil, err
		}

		// Step 4: systemd override.
		if err := writeSystemdOverride(bind); err != nil {
			return nil, errf(ErrSystemdUnavailable, "write systemd override", err)
		}

		// Step 5: daemon-reload + enable --now.
		if err := systemctlContext(ctx, "daemon-reload"); err != nil {
			return nil, errf(ErrSystemdUnavailable, "systemctl daemon-reload", err)
		}
		if err := systemctlContext(ctx, "enable", "--now", "ollama"); err != nil {
			return nil, errf(ErrSystemdUnavailable, "systemctl enable --now ollama", err)
		}
	}

	// Step 6: iptables / ufw.
	if err := configureFirewall(ctx); err != nil {
		// Non-fatal by default — we surface the coded error but continue probe.
		log("warn", "firewall configuration failed", map[string]any{"err": err.Error()})
	}

	// Step 7: probe /api/tags within 30s.
	probeURL := "http://127.0.0.1:11434/api/tags"
	if err := probeUntilReady(ctx, probeURL, 30*time.Second); err != nil {
		return nil, errf(ErrPortBindConflict, "ollama did not become reachable", err)
	}
	res.OllamaVersion = probeOllamaVersion(ctx)

	// Step 8: pull recommended models.
	if !opts.SkipModels {
		tier, recs := RecommendForHost()
		res.Tier = tier
		if opts.Model != "" {
			recs = []ModelRec{{Name: opts.Model, Tasks: []string{"chat"}}}
		}
		if tier == TierNone && opts.Model == "" {
			return nil, errf(ErrRAMInsufficient,
				"host has <4GB RAM available; local LLM not recommended", nil)
		}
		for _, r := range recs {
			log("info", "pulling model", map[string]any{"model": r.Name})
			if err := pullOllamaModel(ctx, r.Name); err != nil {
				log("warn", "model pull failed (continuing)", map[string]any{
					"model": r.Name, "err": err.Error(),
				})
				continue
			}
			res.ModelsPulled = append(res.ModelsPulled, r.Name)
		}
	}

	// Step 9: write state file.
	res.CompletedAt = time.Now().UTC()
	_ = writeLocalState(res)

	return res, nil
}

// ---- Helpers ----

func systemctl(args ...string) error {
	return systemctlContext(context.Background(), args...)
}

func systemctlContext(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func systemctlActive(unit string) bool {
	return systemctlActiveContext(context.Background(), unit)
}

func systemctlActiveContext(ctx context.Context, unit string) bool {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", unit)
	return cmd.Run() == nil
}

func downloadAndRunInstaller(ctx context.Context, log func(string, string, map[string]any)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://ollama.com/install.sh", nil)
	if err != nil {
		return errf(ErrOllamaInstallFailed, "new request", err)
	}
	resp, err := httptimeout.Installer.Do(req)
	if err != nil {
		return errf(ErrOllamaInstallFailed, "download install.sh", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errf(ErrOllamaInstallFailed,
			fmt.Sprintf("install.sh download HTTP %d", resp.StatusCode), nil)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return errf(ErrOllamaInstallFailed, "read install.sh", err)
	}

	// Verify checksum if pinned.
	if expected := ExpectedOllamaInstallChecksum(); expected != "" {
		sum := sha256.Sum256(body)
		got := hex.EncodeToString(sum[:])
		if got != expected {
			return errf(ErrOllamaInstallFailed,
				fmt.Sprintf("install.sh checksum mismatch: got %s want %s", got, expected), nil)
		}
		log("info", "install.sh checksum verified", map[string]any{"sha256": got})
	}

	// Execute via sh -s.
	cmd := exec.CommandContext(ctx, "sh", "-s")
	cmd.Stdin = strings.NewReader(string(body))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errf(ErrOllamaInstallFailed, "install.sh exec", err)
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
			resp.Body.Close()
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
	defer resp.Body.Close()
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
	defer resp.Body.Close()
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
