package commands

// Purpose: The `nself security audit` checks covering certificate expiry,
// license/JWT key rotation age, admin port binding, and CSRF guard
// presence. Split out of security.go (CLI-R12) to keep each group of
// checks (this file, security_checks_system.go) in a file under the size
// cap; runChecks (security.go) calls every check function across both
// files to assemble the full finding list.
// Inputs: none beyond ambient project/host state (cert files, env vars,
// listening sockets).
// Outputs: a finding struct per check (name, pass/fail, detail message).
// Constraints: pure move — no behavior changes.

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ssl"
)

// checkCertExpiry reads the project cert (ssl/fullchain.pem in cwd) and warns
// if it expires within 30 days or is already expired.
func checkCertExpiry() finding {
	certPath := filepath.Join("ssl", "fullchain.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return finding{Name: "Cert expiry", OK: true, Detail: "no project cert found (skipped)"}
	}
	days, err := ssl.CheckCertExpiry(certPath)
	if err != nil {
		return finding{Name: "Cert expiry", OK: false, Detail: "cert expired or unreadable: " + err.Error()}
	}
	if days < 30 {
		return finding{Name: "Cert expiry", OK: false, Detail: fmt.Sprintf("expires in %d days — renew soon", days)}
	}
	return finding{Name: "Cert expiry", OK: true, Detail: fmt.Sprintf("%d days remaining", days)}
}

// checkKeyRotationAge reads a last-rotated timestamp from the .env.secrets
// header comment ("# last-rotated: YYYY-MM-DD") and warns if >180 days old.
func checkKeyRotationAge() finding {
	candidates := []string{".env.secrets", ".env.prod", ".env"}
	for _, p := range candidates {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "#") {
				break // stop at first non-comment line
			}
			lower := strings.ToLower(line)
			for _, prefix := range []string{"# last-rotated:", "# rotated:", "# key-rotated:"} {
				if strings.HasPrefix(lower, prefix) {
					dateStr := strings.TrimSpace(line[len(prefix):])
					t, parseErr := time.Parse("2006-01-02", dateStr)
					f.Close()
					if parseErr != nil {
						return finding{Name: "Key rotation age", OK: true, Detail: "last-rotated header unparseable (skipped)"}
					}
					age := int(time.Since(t).Hours()) / 24
					if age > 180 {
						return finding{Name: "Key rotation age", OK: false,
							Detail: fmt.Sprintf("keys last rotated %d days ago (>180d) — consider rotating", age)}
					}
					return finding{Name: "Key rotation age", OK: true,
						Detail: fmt.Sprintf("last rotated %d days ago", age)}
				}
			}
		}
		f.Close()
	}
	return finding{Name: "Key rotation age", OK: true, Detail: "no last-rotated header found (skipped)"}
}

// checkAdminPortBinding checks whether the nself-admin port (3021) is bound
// to 0.0.0.0 (world-accessible) rather than 127.0.0.1 (local only).
func checkAdminPortBinding() finding {
	ln, err := net.Listen("tcp", "127.0.0.1:3021")
	if err == nil {
		// Port is free — admin is not running.
		ln.Close()
		return finding{Name: "Admin port (3021)", OK: true, Detail: "admin not running on port 3021"}
	}
	// Port is in use. Check whether it's world-accessible by trying 0.0.0.0.
	conn, connErr := net.DialTimeout("tcp", "0.0.0.0:3021", 500*time.Millisecond)
	if connErr == nil {
		conn.Close()
		return finding{Name: "Admin port (3021)", OK: false,
			Detail: "admin port bound on 0.0.0.0 — should be 127.0.0.1 only"}
	}
	return finding{Name: "Admin port (3021)", OK: true, Detail: "admin running, bound to localhost"}
}

// checkCSRFGuard checks for a CSRF_SECRET or CSRF_DISABLED env variable in
// discovered .env files. An explicit CSRF_DISABLED=true is a fail.
func checkCSRFGuard() finding {
	candidates := []string{".env", ".env.dev", ".env.local", ".env.prod"}
	for _, p := range candidates {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}
			upper := strings.ToUpper(line)
			if strings.HasPrefix(upper, "CSRF_DISABLED=TRUE") ||
				strings.HasPrefix(upper, "CSRF_DISABLED=1") ||
				strings.HasPrefix(upper, "CSRF_DISABLED=YES") {
				f.Close()
				return finding{Name: "CSRF guard", OK: false,
					Detail: fmt.Sprintf("CSRF_DISABLED is set in %s — remove to enable CSRF protection", p)}
			}
		}
		f.Close()
	}
	return finding{Name: "CSRF guard", OK: true, Detail: "no CSRF_DISABLED flag found"}
}
