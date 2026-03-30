package trust

import (
	"fmt"
	"runtime"
)

// TrustConfig holds runtime options for the trust setup.
type TrustConfig struct {
	WorkDir            string   // project root directory
	BaseDomain         string   // e.g. "ummat.local"
	NginxSSLPort       int      // e.g. 8443
	NginxHTTPPort      int      // e.g. 8080
	ExtraSSLDomains    []string // from EXTRA_SSL_DOMAINS
	NamespacePrefixes  []string // subdomain namespaces extracted from ROUTES (e.g. "pro", "app", "dev")
	SkipDNS            bool
	SkipSSL            bool
	SkipPorts          bool
}

// TrustResult holds the outcome of each setup step.
type TrustResult struct {
	DNSConfigured       bool
	DNSAlreadyDone      bool
	ResolverConfigured  bool
	ResolverAlreadyDone bool
	CertsGenerated      bool
	CertsAlreadyDone    bool
	PortsConfigured     bool
	PortsAlreadyDone    bool
	Errors              []error
}

// Setup orchestrates the full local dev trust setup for the current OS.
// Steps run in order: DNS → SSL → Ports. Each step is skipped if the
// corresponding SkipXxx flag is set in cfg. Steps are idempotent —
// already-configured state is recorded in TrustResult but not treated as
// an error.
func Setup(cfg TrustConfig) (*TrustResult, error) {
	result := &TrustResult{}

	switch runtime.GOOS {
	case "darwin":
		return setupDarwin(cfg, result)
	case "linux":
		return setupLinux(cfg, result)
	default:
		return nil, fmt.Errorf("unsupported OS: %s — nself trust supports macOS and Linux", runtime.GOOS)
	}
}

// setupDarwin runs the macOS-specific trust setup pipeline.
func setupDarwin(cfg TrustConfig, result *TrustResult) (*TrustResult, error) {
	// Step 1: DNS
	if !cfg.SkipDNS {
		dnsAlready, resolverAlready, err := SetupDNSDarwin(cfg)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("dns setup: %w", err))
			// Continue — DNS failure is non-fatal for SSL + port steps.
		} else {
			result.DNSConfigured = true
			result.DNSAlreadyDone = dnsAlready
			result.ResolverConfigured = true
			result.ResolverAlreadyDone = resolverAlready
		}
	}

	// Step 2: SSL / mkcert
	if !cfg.SkipSSL {
		certsAlready, err := SetupMkcert(cfg)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("ssl setup: %w", err))
		} else {
			result.CertsGenerated = true
			result.CertsAlreadyDone = certsAlready
		}
	}

	// Step 3: Port forwarding
	if !cfg.SkipPorts {
		portsAlready, err := SetupPortsDarwin(cfg)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("port forwarding setup: %w", err))
		} else {
			result.PortsConfigured = true
			result.PortsAlreadyDone = portsAlready
		}
	}

	return result, nil
}

// setupLinux runs the Linux trust setup pipeline.
func setupLinux(cfg TrustConfig, result *TrustResult) (*TrustResult, error) {
	// Step 1: DNS
	if !cfg.SkipDNS {
		dnsAlready, err := SetupDNSLinux(cfg)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("dns setup: %w", err))
		} else {
			result.DNSConfigured = true
			result.DNSAlreadyDone = dnsAlready
			result.ResolverConfigured = true
		}
	}

	// Step 2: SSL / mkcert
	if !cfg.SkipSSL {
		certsAlready, err := SetupMkcert(cfg)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("ssl setup: %w", err))
		} else {
			result.CertsGenerated = true
			result.CertsAlreadyDone = certsAlready
		}
	}

	// Step 3: Port forwarding
	if !cfg.SkipPorts {
		portsAlready, err := SetupPortsLinux(cfg)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("port forwarding setup: %w", err))
		} else {
			result.PortsConfigured = true
			result.PortsAlreadyDone = portsAlready
		}
	}

	return result, nil
}
