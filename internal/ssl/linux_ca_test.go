package ssl

import (
	"testing"
)

// TestIsCAInstalledLinux exercises isCAInstalledLinux which checks two hardcoded
// system paths (/usr/local/share/ca-certificates/mkcert-rootCA.crt and
// /etc/ssl/certs/mkcert-rootCA.pem).  On macOS CI neither path exists so both
// stat calls reach the err-!=nil branch and return false, nil.  On a Linux host
// where mkcert has been installed the function may return true; we accept either.
//
// The caPath parameter is accepted by the function signature but not used in its
// current implementation (it checks hardcoded system destinations, not the CA
// source). Passing various values must not panic or error.
func TestIsCAInstalledLinux_BothPathsMissing(t *testing.T) {
	// On any host where neither /usr/local/share/ca-certificates/mkcert-rootCA.crt
	// nor /etc/ssl/certs/mkcert-rootCA.pem exists, the function returns false, nil.
	// We don't mandate false because a Linux host with mkcert installed may return
	// true — both results are valid.  What we mandate: no error, no panic.
	installed, err := isCAInstalledLinux("/tmp/test-ca.pem")
	if err != nil {
		t.Fatalf("isCAInstalledLinux: unexpected error: %v", err)
	}
	// Log for observability; don't assert the boolean (env-dependent).
	t.Logf("isCAInstalledLinux returned installed=%v (path-dependent)", installed)
}

func TestIsCAInstalledLinux_EmptyCaPath(t *testing.T) {
	// caPath is unused by the current implementation; empty string must not error.
	installed, err := isCAInstalledLinux("")
	if err != nil {
		t.Fatalf("isCAInstalledLinux empty caPath: unexpected error: %v", err)
	}
	t.Logf("isCAInstalledLinux('') returned installed=%v", installed)
}

func TestIsCAInstalledLinux_NonexistentCaPath(t *testing.T) {
	// A path that definitely does not exist as caPath must not change behavior.
	installed, err := isCAInstalledLinux("/no/such/file/ca.pem")
	if err != nil {
		t.Fatalf("isCAInstalledLinux nonexistent caPath: unexpected error: %v", err)
	}
	t.Logf("isCAInstalledLinux('/no/such/file/ca.pem') returned installed=%v", installed)
}
