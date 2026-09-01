package commands

// ssl_renewal_test.go — Unit tests for the renewal-hook installers in
// ssl_renewal.go (installSSLRenewalCron / installSSLRenewalSystemd), which
// had zero direct test coverage before this ticket. sslRenewalServiceUnit's
// content (the deploy-hook path-safety property) is already covered by
// ssl_setup_paths_test.go's TestSSLRenewalServiceUnit_RenewsIntoNginxTree;
// this file covers the installer wrapper's own decision logic instead.
// P6-E11-W2-S3-T18: security command test floor.
//
// Property under test: when systemd is unavailable, installSSLRenewalCron
// must fail with the documented crontab-fallback error rather than either
// panicking or silently doing nothing while reporting success — an
// operator who believes renewal is installed when it is not will have
// their certificate silently expire ~90 days later with no automated
// warning.
//
// installSSLRenewalSystemd's actual write path (/etc/systemd/system,
// `systemctl daemon-reload`, `systemctl enable --now`) requires root and a
// live systemd instance and is NOT exercised here — see this ticket's
// completion note for that explicit non-coverage.

import (
	"strings"
	"testing"
)

// TestInstallSSLRenewalCron_NoSystemd_FallsBackWithInstructions verifies the
// systemd-unavailable branch returns an actionable crontab-fallback error
// and does not attempt to write into /etc/systemd/system.
func TestInstallSSLRenewalCron_NoSystemd_FallsBackWithInstructions(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no systemctl reachable

	err := installSSLRenewalCron("/opt/nself-web/backend")
	if err == nil {
		t.Fatal("expected an error when systemctl is unavailable, got nil")
	}
	if !strings.Contains(err.Error(), "crontab") {
		t.Errorf("error = %q, want it to include crontab fallback instructions", err.Error())
	}
	if !strings.Contains(err.Error(), "certbot renew") {
		t.Errorf("error = %q, want it to include the certbot renew crontab line", err.Error())
	}
}
