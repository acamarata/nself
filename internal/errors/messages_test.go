package errors

import (
	"strings"
	"testing"
)

// TestHelpFooter verifies the help footer contains all three support links.
func TestHelpFooter(t *testing.T) {
	footer := HelpFooter()
	for _, want := range []string{HelpURL, DiscordURL, DiscussionsURL} {
		if !strings.Contains(footer, want) {
			t.Errorf("HelpFooter() missing %q; got: %s", want, footer)
		}
	}
}

// TestMessageError verifies that every error code produces an Error() string
// that starts with "Error:" and contains all required fields.
func TestMessageError(t *testing.T) {
	msgs := []*Message{
		ErrInstallDockerNotFound,
		ErrInstallDockerNotRunning,
		ErrInstallDependencyMissing,
		ErrInstallPermissionDenied,
		ErrInstallDiskFull,
		ErrLicenseInvalid,
		ErrLicenseExpired,
		ErrLicenseNetworkUnavailable,
		ErrPluginNotFound,
		ErrPluginBuildStale,
		ErrNetworkDown,
		ErrPermissionDenied,
		ErrPortConflict,
		ErrEnvMalformed,
	}
	for _, m := range msgs {
		t.Run(m.Code, func(t *testing.T) {
			s := m.Error()
			if !strings.HasPrefix(s, "Error:") {
				t.Errorf("[%s] Error() must start with 'Error:'; got: %s", m.Code, s)
			}
			if !strings.Contains(s, "Need help?") {
				t.Errorf("[%s] Error() must contain 'Need help?'; got: %s", m.Code, s)
			}
			if !strings.Contains(s, DiscordURL) {
				t.Errorf("[%s] Error() must contain Discord URL; got: %s", m.Code, s)
			}
			if !strings.Contains(s, WikiErrorsURL) {
				t.Errorf("[%s] Error() must contain Errors wiki URL; got: %s", m.Code, s)
			}
		})
	}
}

// TestPortConflictWithProcess verifies the process placeholder substitution.
func TestPortConflictWithProcess(t *testing.T) {
	msg := ErrPortConflict.WithProcess("postgres")
	if !strings.Contains(msg.What, "postgres") {
		t.Errorf("WithProcess did not substitute in What; got: %s", msg.What)
	}
	if !strings.Contains(msg.Fix, "postgres") {
		t.Errorf("WithProcess did not substitute in Fix; got: %s", msg.Fix)
	}
}

// TestWithDetail verifies the detail is appended to Why.
func TestWithDetail(t *testing.T) {
	msg := ErrNetworkDown.WithDetail("(DNS resolution failure)")
	if !strings.Contains(msg.Why, "DNS resolution failure") {
		t.Errorf("WithDetail did not append to Why; got: %s", msg.Why)
	}
}

// TestMinimumErrorCodes verifies at least 10 error codes are defined.
func TestMinimumErrorCodes(t *testing.T) {
	all := []*Message{
		ErrInstallDockerNotFound,
		ErrInstallDockerNotRunning,
		ErrInstallDependencyMissing,
		ErrInstallPermissionDenied,
		ErrInstallDiskFull,
		ErrLicenseInvalid,
		ErrLicenseExpired,
		ErrLicenseNetworkUnavailable,
		ErrPluginNotFound,
		ErrPluginBuildStale,
		ErrNetworkDown,
		ErrPermissionDenied,
		ErrPortConflict,
		ErrEnvMalformed,
	}
	if len(all) < 10 {
		t.Errorf("expected at least 10 error codes; got %d", len(all))
	}
}

// TestErrorCodesUnique verifies no two error messages share the same Code.
func TestErrorCodesUnique(t *testing.T) {
	all := []*Message{
		ErrInstallDockerNotFound,
		ErrInstallDockerNotRunning,
		ErrInstallDependencyMissing,
		ErrInstallPermissionDenied,
		ErrInstallDiskFull,
		ErrLicenseInvalid,
		ErrLicenseExpired,
		ErrLicenseNetworkUnavailable,
		ErrPluginNotFound,
		ErrPluginBuildStale,
		ErrNetworkDown,
		ErrPermissionDenied,
		ErrPortConflict,
		ErrEnvMalformed,
	}
	seen := make(map[string]bool)
	for _, m := range all {
		if seen[m.Code] {
			t.Errorf("duplicate error code: %s", m.Code)
		}
		seen[m.Code] = true
	}
}
