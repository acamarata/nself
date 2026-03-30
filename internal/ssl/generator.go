package ssl

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/errs"
)

// Generator creates SSL certificates for all domains in the project configuration.
type Generator struct {
	cfg *config.Config
}

// NewGenerator creates an SSL Generator from the given config.
func NewGenerator(cfg *config.Config) *Generator {
	return &Generator{cfg: cfg}
}

// GenerateResult holds the output of a Generate call including trust/hosts status.
type GenerateResult struct {
	// Count is the number of certificate sets generated.
	Count int
	// CAInstalled is true when the mkcert CA was already trusted or was
	// successfully installed during this call.
	CAInstalled bool
	// CAManualCmd is non-empty when the CA could not be installed automatically.
	// It holds the command the user should run manually.
	CAManualCmd string
	// HostsAdded is the number of new /etc/hosts entries written.
	HostsAdded int
	// HostsManualNote is non-empty when the hosts file could not be written.
	HostsManualNote string
}

// Generate creates SSL certificates for all collected domains.
// Certificates are written to outputDir/certificates/{dir}/fullchain.pem and privkey.pem.
// Returns the count of certificate sets generated.
func (g *Generator) Generate(outputDir string) (int, error) {
	result, err := g.GenerateWithResult(outputDir)
	if err != nil {
		return 0, err
	}
	return result.Count, nil
}

// GenerateWithResult creates SSL certificates and returns detailed results
// including the CA trust and /etc/hosts steps.
//
// When SSL_MODE is "letsencrypt", "custom", or "none", local certificate
// generation is skipped. In letsencrypt mode, certbot provisions the real
// certs after the stack starts. In custom mode, the user supplies their own
// certs. In none mode, SSL is disabled entirely. Nginx must be configured to
// start without SSL cert references in these modes — see nginx/generator.go.
func (g *Generator) GenerateWithResult(outputDir string) (*GenerateResult, error) {
	// Skip local certificate generation for non-local SSL modes.
	// letsencrypt: certbot provisions certs after first start via ACME.
	// custom:      user provides their own certificate files.
	// none:        SSL is disabled; nginx runs HTTP-only.
	switch g.cfg.SSLMode {
	case "letsencrypt", "custom", "none":
		return &GenerateResult{Count: 0}, nil
	}

	domains := g.collectDomains()
	if len(domains) == 0 {
		return nil, fmt.Errorf("no domains collected for SSL generation")
	}

	// Build the certificate directory name from BASE_DOMAIN.
	// Dots become dashes: nself.org -> nself-org, localhost stays localhost.
	certDirName := domainToDirName(g.cfg.BaseDomain)
	if certDirName == "" {
		certDirName = "localhost"
	}

	certDir := filepath.Join(outputDir, "certificates", certDirName)
	if err := os.MkdirAll(certDir, 0750); err != nil {
		return nil, fmt.Errorf("creating certificate directory %s: %w", certDir, err)
	}

	fullchainPath := filepath.Join(certDir, "fullchain.pem")
	privkeyPath := filepath.Join(certDir, "privkey.pem")

	result := &GenerateResult{}

	// Check for pre-existing mkcert certs from `nself trust`.
	// If ssl/fullchain.pem + ssl/privkey.pem exist at the top level of outputDir
	// (put there by `nself trust`), use them instead of generating new certs.
	topFullchain := filepath.Join(outputDir, "fullchain.pem")
	topPrivkey := filepath.Join(outputDir, "privkey.pem")
	if fileExists(topFullchain) && fileExists(topPrivkey) {
		if days, err := CheckCertExpiry(topFullchain); err == nil && days >= 30 {
			if copyErr := copyMkcertCerts(topFullchain, topPrivkey, certDir); copyErr == nil {
				result.Count = 1
				result.CAInstalled = true // mkcert certs are inherently trusted
				g.applyTrustAndHosts(domains, result)
				return result, nil
			}
			// Copy failed — fall through to normal generation.
		}
	}

	// Skip generation if certs already exist, are valid, and have >30 days remaining.
	if fileExists(fullchainPath) && fileExists(privkeyPath) {
		days, err := CheckCertExpiry(fullchainPath)
		if err == nil && days >= 30 {
			result.Count = 1
			// Still attempt CA trust and hosts even when certs are fresh.
			g.applyTrustAndHosts(domains, result)
			return result, nil
		}
		// Expired, near-expiry, or unreadable — regenerate.
	}

	// Try mkcert first for trusted local development certificates.
	var certCount int
	var mkcertUnavailable bool
	if mkcertAvailable() {
		count, err := generateWithMkcert(certDir, domains)
		if err == nil {
			certCount = count
		}
		// mkcert failed — fall through to OpenSSL.
	} else {
		mkcertUnavailable = true
	}

	if certCount == 0 {
		// Fallback to OpenSSL self-signed certificates.
		// Record the mkcert absence so callers can inspect via errors.Is.
		count, err := generateWithOpenSSL(certDir, domains)
		if err != nil {
			if mkcertUnavailable {
				return nil, fmt.Errorf("%w; OpenSSL fallback also failed: %v", errs.ErrMkcertNotFound, err)
			}
			return nil, fmt.Errorf("%w: %v", errs.ErrSSLGenerationFailed, err)
		}
		if mkcertUnavailable {
			// Succeeded with OpenSSL fallback — record the mkcert absence
			// as informational context without failing.
			result.CAManualCmd = errs.ErrMkcertNotFound.Error()
		}
		certCount = count
	}

	result.Count = certCount

	// Attempt CA trust installation and /etc/hosts management.
	// These are best-effort: failures produce instructions, not errors.
	g.applyTrustAndHosts(domains, result)

	return result, nil
}

// applyTrustAndHosts performs the mkcert CA trust and /etc/hosts steps.
// Results are written into result. Errors are never returned — only
// manual instructions are set.
func (g *Generator) applyTrustAndHosts(domains []string, result *GenerateResult) {
	// ── CA trust ────────────────────────────────────────────────────
	if mkcertAvailable() {
		caPath, err := MkcertCACertPath()
		if err == nil {
			installed, err := IsCAInstalled(caPath)
			if err == nil && installed {
				result.CAInstalled = true
			} else {
				// Not installed — attempt installation.
				installErr := installCA(caPath)
				if installErr == nil {
					result.CAInstalled = true
				} else if errors.Is(installErr, ErrSudoRequired) {
					result.CAManualCmd = ManualInstallCommand(caPath)
				}
				// Other errors: set manual cmd as well.
				if installErr != nil && result.CAManualCmd == "" {
					result.CAManualCmd = ManualInstallCommand(caPath)
				}
			}
		}
	}

	// ── /etc/hosts ──────────────────────────────────────────────────
	filtered := filterHostsEntries(domains)
	if len(filtered) == 0 {
		return
	}

	// Count how many are not yet in the file before adding.
	existing, err := readHostsFile(hostsFile)
	if err != nil {
		// Can't read — record note and skip.
		result.HostsManualNote = hostsManualNote(filtered)
		return
	}
	present := buildPresentSet(existing)
	var newEntries []string
	for _, h := range filtered {
		if !present[h] {
			newEntries = append(newEntries, h)
		}
	}

	if len(newEntries) == 0 {
		result.HostsAdded = 0
		return
	}

	addErr := addHosts(newEntries)
	if addErr == nil {
		result.HostsAdded = len(newEntries)
	} else {
		result.HostsManualNote = hostsManualNote(newEntries)
	}
}

// hostsManualNote builds a short note about what hosts entries need to be added manually.
func hostsManualNote(hostnames []string) string {
	var sb strings.Builder
	sb.WriteString("Run once to set up DNS:\n")
	sb.WriteString("  sudo nself dns-setup\n\n")
	sb.WriteString("Or add manually to /etc/hosts:\n")
	for _, h := range hostnames {
		sb.WriteString(fmt.Sprintf("  127.0.0.1\t%s\n", h))
	}
	return sb.String()
}

// domainToDirName converts a domain to a directory-safe name by replacing dots with dashes.
func domainToDirName(domain string) string {
	return strings.ReplaceAll(domain, ".", "-")
}

// fileExists returns true if the path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// CheckCertExpiry reads the PEM-encoded x509 certificate at certPath, computes the
// number of days until it expires, and returns that count.
// If the certificate is already expired, it returns 0 and an error.
// If fewer than 30 days remain, a WARN is written to stderr and the remaining days
// are returned with a nil error.
func CheckCertExpiry(certPath string) (int, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return 0, fmt.Errorf("reading certificate %s: %w", certPath, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return 0, fmt.Errorf("no PEM block found in %s", certPath)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return 0, fmt.Errorf("parsing certificate %s: %w", certPath, err)
	}

	now := time.Now()
	remaining := cert.NotAfter.Sub(now)

	// Convert to whole days using integer arithmetic (truncate toward zero).
	daysRemaining := int(remaining.Hours()) / 24

	if now.After(cert.NotAfter) {
		expired := int(-remaining.Hours()) / 24
		return 0, fmt.Errorf("certificate expired %d days ago", expired)
	}

	if daysRemaining < 30 {
		fmt.Fprintf(os.Stderr, "WARN: certificate at %s expires in %d days\n", certPath, daysRemaining)
	}

	return daysRemaining, nil
}

// copyMkcertCerts copies a fullchain.pem and privkey.pem pair into destDir.
// destDir is created if it does not exist. Both files are written with 0640
// permissions so nginx can read them without world-readable exposure.
func copyMkcertCerts(srcFullchain, srcPrivkey, destDir string) error {
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return fmt.Errorf("creating certificate directory %s: %w", destDir, err)
	}

	chainData, err := os.ReadFile(srcFullchain)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcFullchain, err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "fullchain.pem"), chainData, 0640); err != nil {
		return fmt.Errorf("writing fullchain.pem to %s: %w", destDir, err)
	}

	keyData, err := os.ReadFile(srcPrivkey)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcPrivkey, err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "privkey.pem"), keyData, 0640); err != nil {
		return fmt.Errorf("writing privkey.pem to %s: %w", destDir, err)
	}

	return nil
}
