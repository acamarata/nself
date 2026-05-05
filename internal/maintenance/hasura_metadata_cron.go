//go:build darwin || linux

// Package maintenance — daily Hasura metadata backup cron registration.
//
// Installs a system-level timer that calls `nself backup hasura-metadata` at
// 02:00 UTC every day. Uses systemd (Linux) or launchd (macOS).
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
	hasuraMetaServiceName = "nself-hasura-metadata-backup"
	hasuraMetaDaemonLabel = "org.nself.hasura-metadata-backup"
)

// HasuraMetadataCronStatus reports the state of the Hasura metadata backup timer.
type HasuraMetadataCronStatus struct {
	Enabled  bool
	Platform string
	Detail   string
}

// InstallHasuraMetadataCron installs a daily cron that runs Hasura metadata
// backup at 02:00 UTC. On Linux it writes systemd unit files; on macOS it
// writes a LaunchDaemon plist.
func InstallHasuraMetadataCron() error {
	switch runtime.GOOS {
	case "linux":
		return installHasuraMetadataSystemdTimer()
	case "darwin":
		return installHasuraMetadataLaunchDaemon()
	default:
		return fmt.Errorf("hasura metadata cron is not supported on %s", runtime.GOOS)
	}
}

// RemoveHasuraMetadataCron removes the timer installed by InstallHasuraMetadataCron.
func RemoveHasuraMetadataCron() error {
	switch runtime.GOOS {
	case "linux":
		return removeHasuraMetadataSystemdTimer()
	case "darwin":
		return removeHasuraMetadataLaunchDaemon()
	default:
		return fmt.Errorf("hasura metadata cron is not supported on %s", runtime.GOOS)
	}
}

// GetHasuraMetadataCronStatus returns the current status of the Hasura metadata backup timer.
func GetHasuraMetadataCronStatus() HasuraMetadataCronStatus {
	switch runtime.GOOS {
	case "linux":
		return hasuraMetadataSystemdTimerStatus()
	case "darwin":
		return hasuraMetadataLaunchDaemonStatus()
	default:
		return HasuraMetadataCronStatus{Platform: runtime.GOOS, Detail: "not supported"}
	}
}

// ── Linux / systemd ───────────────────────────────────────────────────────────

func installHasuraMetadataSystemdTimer() error {
	nself, err := resolveNselfBinary()
	if err != nil {
		return err
	}

	service := fmt.Sprintf(`[Unit]
Description=nSelf daily Hasura metadata backup
After=docker.service
Wants=docker.service

[Service]
Type=oneshot
ExecStart=%s backup hasura-metadata
`, nself)

	// Fire at 02:00 UTC every day.
	timer := `[Unit]
Description=nSelf daily Hasura metadata backup timer

[Timer]
OnCalendar=*-*-* 02:00:00 UTC
Persistent=true

[Install]
WantedBy=timers.target
`

	serviceFile := filepath.Join(systemdUnitDir, hasuraMetaServiceName+".service")
	timerFile := filepath.Join(systemdUnitDir, hasuraMetaServiceName+".timer")

	if err := writeRootFile(serviceFile, service, 0644); err != nil {
		return fmt.Errorf("write hasura metadata service unit: %w", err)
	}
	if err := writeRootFile(timerFile, timer, 0644); err != nil {
		return fmt.Errorf("write hasura metadata timer unit: %w", err)
	}

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w\n%s", err, out)
	}
	if out, err := exec.Command("systemctl", "enable", "--now", hasuraMetaServiceName+".timer").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable hasura metadata timer: %w\n%s", err, out)
	}
	return nil
}

func removeHasuraMetadataSystemdTimer() error {
	exec.Command("systemctl", "disable", "--now", hasuraMetaServiceName+".timer").Run() //nolint:errcheck
	serviceFile := filepath.Join(systemdUnitDir, hasuraMetaServiceName+".service")
	timerFile := filepath.Join(systemdUnitDir, hasuraMetaServiceName+".timer")
	os.Remove(serviceFile)                           //nolint:errcheck
	os.Remove(timerFile)                             //nolint:errcheck
	exec.Command("systemctl", "daemon-reload").Run() //nolint:errcheck
	return nil
}

func hasuraMetadataSystemdTimerStatus() HasuraMetadataCronStatus {
	timerFile := filepath.Join(systemdUnitDir, hasuraMetaServiceName+".timer")
	if _, err := os.Stat(timerFile); os.IsNotExist(err) {
		return HasuraMetadataCronStatus{
			Platform: "linux (systemd)",
			Enabled:  false,
			Detail:   "timer not installed",
		}
	}
	out, err := exec.Command("systemctl", "is-active", hasuraMetaServiceName+".timer").Output()
	active := err == nil && strings.TrimSpace(string(out)) == "active"
	return HasuraMetadataCronStatus{
		Platform: "linux (systemd)",
		Enabled:  active,
		Detail:   strings.TrimSpace(string(out)),
	}
}

// ── macOS / LaunchDaemon ─────────────────────────────────────────────────────

// buildHasuraMetadataPlist returns the plist XML for the hasura-metadata-backup
// LaunchDaemon. Extracted as a pure function so it can be unit-tested without
// requiring OS privileges or filesystem access.
func buildHasuraMetadataPlist(label, nselfBin string) string {
	// TZ=UTC ensures StartCalendarInterval 02:00 fires at 02:00 UTC on any
	// macOS host regardless of the local timezone, matching the systemd timer.
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>backup</string>
        <string>hasura-metadata</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>TZ</key>
        <string>UTC</string>
    </dict>
    <key>StartCalendarInterval</key>
    <dict>
        <key>Hour</key>
        <integer>2</integer>
        <key>Minute</key>
        <integer>0</integer>
    </dict>
    <key>RunAtLoad</key>
    <false/>
    <key>StandardOutPath</key>
    <string>/var/log/nself-hasura-metadata-backup.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/nself-hasura-metadata-backup.log</string>
</dict>
</plist>
`, label, nselfBin)
}

func installHasuraMetadataLaunchDaemon() error {
	nself, err := resolveNselfBinary()
	if err != nil {
		return err
	}

	plist := buildHasuraMetadataPlist(hasuraMetaDaemonLabel, nself)

	plistFile := filepath.Join(launchDaemonDir, hasuraMetaDaemonLabel+".plist")
	if err := writeRootFile(plistFile, plist, 0644); err != nil {
		return fmt.Errorf("write hasura metadata plist: %w", err)
	}
	if out, err := exec.Command("launchctl", "load", "-w", plistFile).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load hasura metadata plist: %w\n%s", err, out)
	}
	return nil
}

func removeHasuraMetadataLaunchDaemon() error {
	plistFile := filepath.Join(launchDaemonDir, hasuraMetaDaemonLabel+".plist")
	exec.Command("launchctl", "unload", "-w", plistFile).Run() //nolint:errcheck
	os.Remove(plistFile)                                       //nolint:errcheck
	return nil
}

func hasuraMetadataLaunchDaemonStatus() HasuraMetadataCronStatus {
	plistFile := filepath.Join(launchDaemonDir, hasuraMetaDaemonLabel+".plist")
	if _, err := os.Stat(plistFile); os.IsNotExist(err) {
		return HasuraMetadataCronStatus{
			Platform: "darwin (launchd)",
			Enabled:  false,
			Detail:   "plist not installed",
		}
	}
	out, _ := exec.Command("launchctl", "list", hasuraMetaDaemonLabel).Output()
	enabled := len(out) > 0 && !strings.Contains(string(out), "Could not find service")
	return HasuraMetadataCronStatus{
		Platform: "darwin (launchd)",
		Enabled:  enabled,
		Detail:   strings.TrimSpace(string(out)),
	}
}
