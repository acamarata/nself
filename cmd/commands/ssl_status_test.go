package commands

// ssl_status_test.go — Unit tests for the pure helpers and precondition
// gate in ssl.go (certIssuer, runSSLStatus's project-detection guard).
// P6-E11-W2-S3-T18: security command test floor. The ssl family's
// acceptance criterion covers ssl.go alongside ssl_add/install/renewal/
// setup; runSSLStatus's live-network body (checkDomainTLS dialing a real
// domain on :443) is out of reach without live infrastructure — see this
// ticket's completion note — but certIssuer is a pure function and the
// no-project guard needs no network at all.

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"strings"
	"testing"
)

// TestCertIssuer_PrefersOrganization verifies the display-name property:
// when a certificate's issuer has an Organization set, that (not the raw
// CommonName) is what operators see in `nself ssl status` output — getting
// this backwards would show confusing CA internals instead of "Let's
// Encrypt".
func TestCertIssuer_PrefersOrganization(t *testing.T) {
	cert := &x509.Certificate{
		Issuer: pkix.Name{
			Organization: []string{"Let's Encrypt"},
			CommonName:   "R3",
		},
	}
	if got := certIssuer(cert); got != "Let's Encrypt" {
		t.Errorf("certIssuer = %q, want %q", got, "Let's Encrypt")
	}
}

// TestCertIssuer_FallsBackToCommonName verifies the fallback when no
// Organization is set.
func TestCertIssuer_FallsBackToCommonName(t *testing.T) {
	cert := &x509.Certificate{
		Issuer: pkix.Name{CommonName: "Self-Signed Test CA"},
	}
	if got := certIssuer(cert); got != "Self-Signed Test CA" {
		t.Errorf("certIssuer = %q, want %q", got, "Self-Signed Test CA")
	}
}

// TestRunSSLRenew_NoProject_Errors verifies `nself ssl renew` refuses to run
// outside an nself project — same rationale as the other ssl_*.go guards:
// proceeding would attempt to reload an nginx container for an undefined
// workdir.
func TestRunSSLRenew_NoProject_Errors(t *testing.T) {
	withProjectRoot(t, func(root string) {
		_ = root
		err := runSSLRenew(sslRenewCmd, nil)
		if err == nil {
			t.Fatal("expected error outside an nself project, got nil")
		}
		if !strings.Contains(err.Error(), "no nself project found") {
			t.Errorf("error = %q, want it to mention 'no nself project found'", err.Error())
		}
	})
}

// TestRunSSLStatus_NoProject_Errors verifies the same fail-fast guard as
// ssl add/setup: outside an nself project, status must refuse rather than
// dialing TLS against an undefined base domain (which would be "").
func TestRunSSLStatus_NoProject_Errors(t *testing.T) {
	withProjectRoot(t, func(root string) {
		_ = root
		err := runSSLStatus(sslStatusCmd, nil)
		if err == nil {
			t.Fatal("expected error outside an nself project, got nil")
		}
		if !strings.Contains(err.Error(), "no nself project found") {
			t.Errorf("error = %q, want it to mention 'no nself project found'", err.Error())
		}
	})
}
