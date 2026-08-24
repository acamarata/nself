package installer

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// downloadAndRunInstaller downloads the Ollama install script, verifies its
// SHA-256 against the pinned checksum, and executes it via sh.
//
// DownloadAndVerify places the script in a private 0700 owner-only directory
// to close the TOCTOU window between file close and exec. The entire directory
// is removed with os.RemoveAll after execution (not just the file).

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
