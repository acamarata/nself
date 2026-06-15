package dr

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SystemdInstallOptions controls install of the monthly DR drill timer.
//
// The unit runs `nself dr drill --now` on the first of each month at 05:00 UTC,
// provisioning a throwaway Hetzner VM, restoring the latest backup, running a
// smoke suite, recording timings to the dr_drill_report PG table, and
// destroying the VM.
type SystemdInstallOptions struct {
	// Schedule is the desired cadence. Only "monthly" is supported today.
	// Empty means monthly (OnCalendar=*-*-01 05:00:00).
	Schedule string

	// HetznerProject identifies the Hetzner project that owns the drill VM.
	// Used to select the correct API token from the env file (for example
	// "camarata" selects HETZNER_CAMARATA_TOKEN).
	HetznerProject string

	// VMType is the Hetzner server type used for the drill VM. Defaults to
	// "cx22" (smallest shared-CPU tier in fsn1) to keep drill cost < €0.05.
	VMType string

	// SSHKey is the absolute path to the public SSH key injected into the
	// drill VM via cloud-init. Defaults to /root/.config/nself/dr-key.pub.
	SSHKey string

	// Region is the Hetzner location for provisioning. Defaults to fsn1.
	Region string

	UnitDir    string // default /etc/systemd/system
	EnvFile    string // default /etc/nself/dr.env
	BinaryPath string // default /usr/local/bin/nself
	ProjectDir string // default /opt/nself
	DryRun     bool
}

// SystemdUnitFiles maps filename -> unit contents.
type SystemdUnitFiles map[string]string

// RenderSystemdUnits produces the unit files for the monthly DR drill without
// writing to disk. Returns a map suitable for writing into /etc/systemd/system.
func RenderSystemdUnits(opts SystemdInstallOptions) (SystemdUnitFiles, error) {
	if opts.Schedule == "" {
		opts.Schedule = "monthly"
	}
	if opts.Schedule != "monthly" {
		return nil, fmt.Errorf("unsupported drill schedule %q (only \"monthly\" is supported)", opts.Schedule)
	}
	if opts.HetznerProject == "" {
		return nil, fmt.Errorf("hetzner project required (e.g. --hetzner-project camarata)")
	}
	if opts.VMType == "" {
		opts.VMType = "cx22"
	}
	if opts.SSHKey == "" {
		opts.SSHKey = "/root/.config/nself/dr-key.pub"
	}
	if opts.Region == "" {
		opts.Region = "fsn1"
	}
	if opts.EnvFile == "" {
		opts.EnvFile = "/etc/nself/dr.env"
	}
	if opts.BinaryPath == "" {
		opts.BinaryPath = "/usr/local/bin/nself"
	}
	if opts.ProjectDir == "" {
		opts.ProjectDir = "/opt/nself"
	}

	files := SystemdUnitFiles{}

	execStart := fmt.Sprintf(
		"%s dr drill --now --hetzner-project %s --vm-type %s --ssh-key %s --region %s",
		opts.BinaryPath, opts.HetznerProject, opts.VMType, opts.SSHKey, opts.Region,
	)

	files["nself-dr-drill.service"] = renderDrillService(drillServiceSpec{
		Description: "nSelf monthly disaster recovery drill",
		EnvFile:     opts.EnvFile,
		WorkingDir:  opts.ProjectDir,
		ExecStart:   execStart,
	})

	files["nself-dr-drill.timer"] = renderDrillTimer(drillTimerSpec{
		Description: "Monthly DR drill on 1st at 05:00 UTC",
		OnCalendar:  "*-*-01 05:00:00 UTC",
		Unit:        "nself-dr-drill.service",
		Persistent:  true,
	})

	return files, nil
}

// InstallSystemdUnits renders and writes unit files, then runs
// `systemctl daemon-reload` and enables the drill timer. Requires root.
func InstallSystemdUnits(opts SystemdInstallOptions) error {
	files, err := RenderSystemdUnits(opts)
	if err != nil {
		return err
	}

	unitDir := opts.UnitDir
	if unitDir == "" {
		unitDir = "/etc/systemd/system"
	}

	if opts.DryRun {
		for name, content := range files {
			slog.Info("dry run: systemd unit", "path", filepath.Join(unitDir, name), "content", content)
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

	if err := runCmd("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := runCmd("systemctl", "enable", "--now", "nself-dr-drill.timer"); err != nil {
		return fmt.Errorf("enable nself-dr-drill.timer: %w", err)
	}

	return nil
}

type drillServiceSpec struct {
	Description string
	EnvFile     string
	WorkingDir  string
	ExecStart   string
}

func renderDrillService(s drillServiceSpec) string {
	return strings.Join([]string{
		"[Unit]",
		"Description=" + s.Description,
		"After=network-online.target",
		"Wants=network-online.target",
		"",
		"[Service]",
		"Type=oneshot",
		"EnvironmentFile=-" + s.EnvFile,
		"Environment=PATH=/usr/local/bin:/usr/bin:/bin",
		"WorkingDirectory=" + s.WorkingDir,
		"ExecStart=" + s.ExecStart,
		"Nice=10",
		"TimeoutStartSec=2h",
		"",
		"[Install]",
		"WantedBy=multi-user.target",
		"",
	}, "\n")
}

type drillTimerSpec struct {
	Description string
	OnCalendar  string
	Unit        string
	Persistent  bool
}

func renderDrillTimer(t drillTimerSpec) string {
	lines := []string{
		"[Unit]",
		"Description=" + t.Description,
		"",
		"[Timer]",
		"Unit=" + t.Unit,
		"OnCalendar=" + t.OnCalendar,
	}
	if t.Persistent {
		lines = append(lines, "Persistent=true")
	}
	lines = append(lines,
		"AccuracySec=1min",
		"",
		"[Install]",
		"WantedBy=timers.target",
		"",
	)
	return strings.Join(lines, "\n")
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
