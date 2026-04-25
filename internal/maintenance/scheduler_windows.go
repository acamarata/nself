//go:build windows

package maintenance

import "fmt"

// ScheduleStatus reports the current state of the daily timer.
type ScheduleStatus struct {
	Enabled  bool
	Platform string
	Detail   string
}

// InstallDailyTimer is not supported on Windows.
func InstallDailyTimer() error {
	return fmt.Errorf("daily timer is not supported on Windows — use Task Scheduler manually")
}

// RemoveDailyTimer is not supported on Windows.
func RemoveDailyTimer() error {
	return fmt.Errorf("daily timer removal is not supported on Windows")
}

// GetScheduleStatus returns a stub status on Windows.
func GetScheduleStatus() ScheduleStatus {
	return ScheduleStatus{Platform: "windows", Enabled: false, Detail: "not supported"}
}
