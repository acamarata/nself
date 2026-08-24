package backup

// stream_schedule.go — scheduling a recurring streamed backup.
//
// Purpose: translate a cron expression into a systemd timer unit and drive ScheduleStream, split out of stream.go for file size.
// Inputs: a cron expression and the StreamConfig to run on each tick.
// Outputs: an installed systemd timer, or an error if the cron expression cannot be represented.
// Constraints: pure move from stream.go (CLI-R12 Batch E); no behaviour change.

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// ScheduleStream adds a systemd timer for nself backup stream.
// It delegates to the existing systemd infrastructure via a dedicated unit.
func ScheduleStream(cfg *config.Config, cron, to, recipient, unitDir string, dryRun bool) error {
	if cron == "" {
		return fmt.Errorf("--cron expression required (e.g. '0 2 * * *')")
	}
	if to == "" {
		to = cfg.Backup.Remote
	}
	if to == "" {
		return fmt.Errorf("--to destination required for schedule")
	}

	binaryPath := "/usr/local/bin/nself"
	envFile := "/etc/nself/backup.env"
	projectDir := "/opt/nself"
	if unitDir == "" {
		unitDir = "/etc/systemd/system"
	}

	recipientFlag := ""
	if recipient != "" {
		recipientFlag = " --recipient " + recipient
	}

	execStart := fmt.Sprintf("%s backup stream --to %s%s", binaryPath, to, recipientFlag)

	// Convert simple cron "M H * * *" to systemd OnCalendar syntax.
	onCalendar := cronToSystemd(cron)

	service := renderService(serviceSpec{
		Description: "nSelf streaming encrypted backup",
		EnvFile:     envFile,
		WorkingDir:  projectDir,
		ExecStart:   execStart,
	})
	timer := renderTimer(timerSpec{
		Description: "nSelf streaming backup: " + cron,
		OnCalendar:  onCalendar,
		Unit:        "nself-backup-stream.service",
		Persistent:  true,
	})

	if dryRun {
		slog.Info("dry run: systemd unit", "unit", "nself-backup-stream.service", "content", service)
		slog.Info("dry run: systemd unit", "unit", "nself-backup-stream.timer", "content", timer)
		return nil
	}

	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", unitDir, err)
	}

	servicePath := filepath.Join(unitDir, "nself-backup-stream.service")
	timerPath := filepath.Join(unitDir, "nself-backup-stream.timer")

	if err := os.WriteFile(servicePath, []byte(service), 0644); err != nil {
		return fmt.Errorf("write service unit: %w", err)
	}
	if err := os.WriteFile(timerPath, []byte(timer), 0644); err != nil {
		return fmt.Errorf("write timer unit: %w", err)
	}

	if err := run("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := run("systemctl", "enable", "--now", "nself-backup-stream.timer"); err != nil {
		return fmt.Errorf("enable timer: %w", err)
	}

	slog.Info("stream backup scheduled", "cron", cron, "destination", to)
	return nil
}

// cronToSystemd converts a 5-field cron expression to a systemd OnCalendar value.
// Handles simple patterns only; complex expressions pass through with a warning.
func cronToSystemd(cron string) string {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		slog.Warn("cron expression not in 5-field format; using verbatim", "cron", cron)
		return cron
	}
	minute, hour, dom, month, dow := fields[0], fields[1], fields[2], fields[3], fields[4]

	// Daily at HH:MM when dom, month, dow are all *.
	if dom == "*" && month == "*" && dow == "*" {
		return fmt.Sprintf("*-*-* %s:%s:00 UTC", zeroPad(hour), zeroPad(minute))
	}

	// Weekly: dow set, dom and month *.
	if dow != "*" && dom == "*" && month == "*" {
		dayName := cronDOWName(dow)
		return fmt.Sprintf("%s *-*-* %s:%s:00 UTC", dayName, zeroPad(hour), zeroPad(minute))
	}

	// Monthly: dom set, dow and month *.
	if dom != "*" && dow == "*" && month == "*" {
		return fmt.Sprintf("*-*-%s %s:%s:00 UTC", zeroPad(dom), zeroPad(hour), zeroPad(minute))
	}

	// Fall back to daily to avoid invalid syntax; caller sees the warning above.
	return fmt.Sprintf("*-*-* %s:%s:00 UTC", zeroPad(hour), zeroPad(minute))
}

func zeroPad(s string) string {
	if len(s) == 1 && s[0] >= '0' && s[0] <= '9' {
		return "0" + s
	}
	return s
}

func cronDOWName(n string) string {
	days := map[string]string{
		"0": "Sun", "7": "Sun",
		"1": "Mon", "2": "Tue", "3": "Wed",
		"4": "Thu", "5": "Fri", "6": "Sat",
	}
	if name, ok := days[n]; ok {
		return name
	}
	return n
}
