//go:build darwin || linux

package maintenance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// systemdServiceName is the base name for the systemd unit files.
	systemdServiceName = "nself-disk-cleanup"
	// systemdUnitDir is where user systemd unit files live.
	systemdUnitDir = "/etc/systemd/system"
	// launchDaemonDir is where macOS LaunchDaemon plists live.
	launchDaemonDir = "/Library/LaunchDaemons"
	// launchDaemonLabel is the macOS plist label.
	launchDaemonLabel = "org.nself.disk-cleanup"
)

// ScheduleStatus reports the current state of the daily timer.
type ScheduleStatus struct {
	Enabled  bool
	Platform string
	Detail   string
}

// InstallDailyTimer installs a system-level timer that runs disk-cleanup every
// day at 03:00 local time. On Linux it writes systemd unit files; on macOS it
// writes a LaunchDaemon plist. Returns an error if the required privileges are
// not available.
func InstallDailyTimer() error {
	switch runtime.GOOS {
	case "linux":
		return installSystemdTimer()
	case "darwin":
		return installLaunchDaemon()
	default:
		return fmt.Errorf("daily timer is not supported on %s", runtime.GOOS)
	}
}

// RemoveDailyTimer uninstalls the daily timer installed by InstallDailyTimer.
func RemoveDailyTimer() error {
	switch runtime.GOOS {
	case "linux":
		return removeSystemdTimer()
	case "darwin":
		return removeLaunchDaemon()
	default:
		return fmt.Errorf("daily timer is not supported on %s", runtime.GOOS)
	}
}

// GetScheduleStatus returns the current status of the daily cleanup timer.
func GetScheduleStatus() ScheduleStatus {
	switch runtime.GOOS {
	case "linux":
		return systemdTimerStatus()
	case "darwin":
		return launchDaemonStatus()
	default:
		return ScheduleStatus{Platform: runtime.GOOS, Detail: "not supported"}
	}
}

// ── Linux / systemd ───────────────────────────────────────────────────────────

func installSystemdTimer() error {
	nself, err := resolveNselfBinary()
	if err != nil {
		return err
	}

	service := fmt.Sprintf(`[Unit]
Description=nSelf daily disk cleanup
After=docker.service
Wants=docker.service

[Service]
Type=oneshot
ExecStart=%s maintenance disk-cleanup
`, nself)

	timer := `[Unit]
Description=nSelf daily disk cleanup timer

[Timer]
OnCalendar=*-*-* 03:00:00
Persistent=true

[Install]
WantedBy=timers.target
`

	serviceFile := filepath.Join(systemdUnitDir, systemdServiceName+".service")
	timerFile := filepath.Join(systemdUnitDir, systemdServiceName+".timer")

	if err := writeRootFile(serviceFile, service, 0644); err != nil {
		return fmt.Errorf("write service unit: %w", err)
	}
	if err := writeRootFile(timerFile, timer, 0644); err != nil {
		return fmt.Errorf("write timer unit: %w", err)
	}

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w\n%s", err, out)
	}
	if out, err := exec.Command("systemctl", "enable", "--now", systemdServiceName+".timer").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable timer: %w\n%s", err, out)
	}
	return nil
}

func removeSystemdTimer() error {
	exec.Command("systemctl", "disable", "--now", systemdServiceName+".timer").Run() //nolint:errcheck
	serviceFile := filepath.Join(systemdUnitDir, systemdServiceName+".service")
	timerFile := filepath.Join(systemdUnitDir, systemdServiceName+".timer")
	os.Remove(serviceFile) //nolint:errcheck
	os.Remove(timerFile)   //nolint:errcheck
	exec.Command("systemctl", "daemon-reload").Run() //nolint:errcheck
	return nil
}

func systemdTimerStatus() ScheduleStatus {
	timerFile := filepath.Join(systemdUnitDir, systemdServiceName+".timer")
	if _, err := os.Stat(timerFile); os.IsNotExist(err) {
		return ScheduleStatus{Platform: "linux (systemd)", Enabled: false, Detail: "timer not installed"}
	}
	out, err := exec.Command("systemctl", "is-active", systemdServiceName+".timer").Output()
	active := err == nil && strings.TrimSpace(string(out)) == "active"
	detail := strings.TrimSpace(string(out))
	return ScheduleStatus{Platform: "linux (systemd)", Enabled: active, Detail: detail}
}

// ── macOS / LaunchDaemon ─────────────────────────────────────────────────────

func installLaunchDaemon() error {
	nself, err := resolveNselfBinary()
	if err != nil {
		return err
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>maintenance</string>
        <string>disk-cleanup</string>
    </array>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>3</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
    <key>RunAtLoad</key>
    <false/>
    <key>StandardOutPath</key>
    <string>/var/log/nself-disk-cleanup.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/nself-disk-cleanup.log</string>
</dict>
</plist>
`, launchDaemonLabel, nself)

	plistFile := filepath.Join(launchDaemonDir, launchDaemonLabel+".plist")
	if err := writeRootFile(plistFile, plist, 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	if out, err := exec.Command("launchctl", "load", "-w", plistFile).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load: %w\n%s", err, out)
	}
	return nil
}

func removeLaunchDaemon() error {
	plistFile := filepath.Join(launchDaemonDir, launchDaemonLabel+".plist")
	exec.Command("launchctl", "unload", "-w", plistFile).Run() //nolint:errcheck
	os.Remove(plistFile)                                        //nolint:errcheck
	return nil
}

func launchDaemonStatus() ScheduleStatus {
	plistFile := filepath.Join(launchDaemonDir, launchDaemonLabel+".plist")
	if _, err := os.Stat(plistFile); os.IsNotExist(err) {
		return ScheduleStatus{Platform: "darwin (launchd)", Enabled: false, Detail: "plist not installed"}
	}
	out, _ := exec.Command("launchctl", "list", launchDaemonLabel).Output()
	enabled := len(out) > 0 && !strings.Contains(string(out), "Could not find service")
	return ScheduleStatus{Platform: "darwin (launchd)", Enabled: enabled, Detail: strings.TrimSpace(string(out))}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// resolveNselfBinary returns the absolute path to the running nself binary.
func resolveNselfBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve nself binary: %w", err)
	}
	abs, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil // fallback: use the symlink path
	}
	return abs, nil
}

// writeRootFile writes content to a path that may require root. It first
// attempts a direct write; if that fails with permission denied it returns a
// clear error instructing the user to run with sudo.
func writeRootFile(path, content string, mode os.FileMode) error {
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("%w — run 'sudo nself maintenance schedule --daily'", err)
		}
		return err
	}
	return nil
}
