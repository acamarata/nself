package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// SystemdInstallOptions controls install of nself-backup systemd units.
type SystemdInstallOptions struct {
	FullAt      string // "HH:MM" UTC, e.g. "03:00"
	WALEvery    string // e.g. "15m"
	PruneAt     string // "HH:MM" UTC, default "04:00"
	VerifyOnDay string // systemd OnCalendar day name (Sun..Sat), default "Sun"
	VerifyAt    string // "HH:MM", default "05:00"
	UnitDir     string // default /etc/systemd/system
	EnvFile     string // default /etc/nself/backup.env
	BinaryPath  string // default /usr/local/bin/nself
	ProjectDir  string // working directory for nself commands
	Remote      string // e.g. b2-nclaw
	DryRun      bool
}

// SystemdUnitFiles maps filename -> unit contents.
type SystemdUnitFiles map[string]string

// RenderSystemdUnits produces the unit files needed for unattended backup,
// verify, and prune without writing to disk. Returns a map suitable for
// writing into /etc/systemd/system.
func RenderSystemdUnits(cfg *config.Config, opts SystemdInstallOptions) (SystemdUnitFiles, error) {
	if opts.FullAt == "" {
		opts.FullAt = "03:00"
	}
	if opts.WALEvery == "" {
		opts.WALEvery = "15m"
	}
	if opts.PruneAt == "" {
		opts.PruneAt = "04:00"
	}
	if opts.VerifyOnDay == "" {
		opts.VerifyOnDay = "Sun"
	}
	if opts.VerifyAt == "" {
		opts.VerifyAt = "05:00"
	}
	if opts.EnvFile == "" {
		opts.EnvFile = "/etc/nself/backup.env"
	}
	if opts.BinaryPath == "" {
		opts.BinaryPath = "/usr/local/bin/nself"
	}
	if opts.ProjectDir == "" {
		opts.ProjectDir = "/opt/nself"
	}
	remote := opts.Remote
	if remote == "" {
		remote = cfg.Backup.Remote
	}

	remoteFlag := ""
	if remote != "" {
		remoteFlag = fmt.Sprintf(" --remote %s", remote)
	}

	files := SystemdUnitFiles{}

	// ── nself-backup-full ──
	files["nself-backup-full.service"] = renderService(serviceSpec{
		Description: "nSelf full PostgreSQL backup",
		EnvFile:     opts.EnvFile,
		WorkingDir:  opts.ProjectDir,
		ExecStart:   fmt.Sprintf("%s backup create --type all%s", opts.BinaryPath, remoteFlag),
	})
	files["nself-backup-full.timer"] = renderTimer(timerSpec{
		Description: "Nightly full backup at " + opts.FullAt + " UTC",
		OnCalendar:  fmt.Sprintf("*-*-* %s:00 UTC", opts.FullAt),
		Unit:        "nself-backup-full.service",
		Persistent:  true,
	})

	// ── nself-backup-wal ──
	files["nself-backup-wal.service"] = renderService(serviceSpec{
		Description: "nSelf WAL archive checkpoint",
		EnvFile:     opts.EnvFile,
		WorkingDir:  opts.ProjectDir,
		ExecStart:   fmt.Sprintf("%s backup create --type wal%s", opts.BinaryPath, remoteFlag),
	})
	files["nself-backup-wal.timer"] = renderTimer(timerSpec{
		Description: "WAL archive every " + opts.WALEvery,
		OnBootSec:   "2min",
		OnUnitSec:   opts.WALEvery,
		Unit:        "nself-backup-wal.service",
	})

	// ── nself-backup-prune ──
	files["nself-backup-prune.service"] = renderService(serviceSpec{
		Description: "nSelf backup retention pruning",
		EnvFile:     opts.EnvFile,
		WorkingDir:  opts.ProjectDir,
		ExecStart: fmt.Sprintf("%s backup prune --keep-daily %d --keep-weekly %d --keep-monthly %d",
			opts.BinaryPath,
			valueOr(cfg.Backup.RetentionDaily, 7),
			valueOr(cfg.Backup.RetentionWeekly, 4),
			valueOr(cfg.Backup.RetentionMonthly, 12),
		),
	})
	files["nself-backup-prune.timer"] = renderTimer(timerSpec{
		Description: "Nightly prune at " + opts.PruneAt + " UTC",
		OnCalendar:  fmt.Sprintf("*-*-* %s:00 UTC", opts.PruneAt),
		Unit:        "nself-backup-prune.service",
		Persistent:  true,
	})

	// ── nself-backup-verify ──
	files["nself-backup-verify.service"] = renderService(serviceSpec{
		Description: "nSelf weekly backup restore-test",
		EnvFile:     opts.EnvFile,
		WorkingDir:  opts.ProjectDir,
		ExecStart:   fmt.Sprintf("%s backup verify latest --restore-test", opts.BinaryPath),
	})
	files["nself-backup-verify.timer"] = renderTimer(timerSpec{
		Description: "Weekly restore-test on " + opts.VerifyOnDay + " " + opts.VerifyAt + " UTC",
		OnCalendar:  fmt.Sprintf("%s *-*-* %s:00 UTC", opts.VerifyOnDay, opts.VerifyAt),
		Unit:        "nself-backup-verify.service",
		Persistent:  true,
	})

	return files, nil
}

// InstallSystemdUnits renders and writes unit files, then runs
// `systemctl daemon-reload` and enables each timer. Requires root.
func InstallSystemdUnits(cfg *config.Config, opts SystemdInstallOptions) error {
	files, err := RenderSystemdUnits(cfg, opts)
	if err != nil {
		return err
	}

	unitDir := opts.UnitDir
	if unitDir == "" {
		unitDir = "/etc/systemd/system"
	}

	if opts.DryRun {
		for name, content := range files {
			fmt.Printf("# ── %s ──\n%s\n", filepath.Join(unitDir, name), content)
		}
		return nil
	}

	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", unitDir, err)
	}

	for name, content := range files {
		path := filepath.Join(unitDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	if err := run("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	timers := []string{
		"nself-backup-full.timer",
		"nself-backup-wal.timer",
		"nself-backup-prune.timer",
		"nself-backup-verify.timer",
	}
	for _, t := range timers {
		if err := run("systemctl", "enable", "--now", t); err != nil {
			return fmt.Errorf("enable %s: %w", t, err)
		}
	}

	return nil
}

type serviceSpec struct {
	Description string
	EnvFile     string
	WorkingDir  string
	ExecStart   string
}

func renderService(s serviceSpec) string {
	return strings.Join([]string{
		"[Unit]",
		"Description=" + s.Description,
		"After=network-online.target docker.service",
		"Wants=network-online.target",
		"",
		"[Service]",
		"Type=oneshot",
		"EnvironmentFile=-" + s.EnvFile,
		"Environment=PATH=/usr/local/bin:/usr/bin:/bin",
		"WorkingDirectory=" + s.WorkingDir,
		"ExecStart=" + s.ExecStart,
		"Nice=10",
		"IOSchedulingClass=best-effort",
		"IOSchedulingPriority=7",
		"TimeoutStartSec=4h",
		"",
		"[Install]",
		"WantedBy=multi-user.target",
		"",
	}, "\n")
}

type timerSpec struct {
	Description string
	OnCalendar  string
	OnBootSec   string
	OnUnitSec   string
	Unit        string
	Persistent  bool
}

func renderTimer(t timerSpec) string {
	lines := []string{
		"[Unit]",
		"Description=" + t.Description,
		"",
		"[Timer]",
		"Unit=" + t.Unit,
	}
	if t.OnCalendar != "" {
		lines = append(lines, "OnCalendar="+t.OnCalendar)
	}
	if t.OnBootSec != "" {
		lines = append(lines, "OnBootSec="+t.OnBootSec)
	}
	if t.OnUnitSec != "" {
		lines = append(lines, "OnUnitActiveSec="+t.OnUnitSec)
	}
	if t.Persistent {
		lines = append(lines, "Persistent=true")
	}
	lines = append(lines,
		"AccuracySec=30s",
		"",
		"[Install]",
		"WantedBy=timers.target",
		"",
	)
	return strings.Join(lines, "\n")
}

func valueOr(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
