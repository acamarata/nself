package commands

// Purpose: Builds the trust.TrustConfig used by `nself trust` from resolved
// options, and derives the set of local-namespace hostname prefixes (e.g.
// per-app subdomains) that need trusted certs from the project's config and
// base domain. Split out of trust.go (CLI-R12) to separate config-building
// from the cobra wiring/entry point (trust.go), the plan/summary/status
// printers (trust_status.go), the undo path (trust_undo.go), and the
// detailed status report (trust_status_detailed.go).
// Inputs: a trustOpts value and, for extractNamespacePrefixes, the loaded
// *config.Config and the resolved base domain string.
// Outputs: a populated trust.TrustConfig and a []string of namespace
// prefixes.
// Constraints: pure move — no behavior changes.

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/trust"
)

// buildTrustConfig constructs a TrustConfig from CLI flags and the current
// project config (if found). Falls back to safe defaults when no project is
// present in the current directory.
func buildTrustConfig(opts trustOpts) trust.TrustConfig {
	cfg := trust.TrustConfig{
		NginxSSLPort:  8443,
		NginxHTTPPort: 8080,
		SkipDNS:       opts.SkipDNS,
		SkipSSL:       opts.SkipSSL,
		SkipPorts:     opts.SkipPorts,
	}

	var root string
	if opts.Project != "" {
		abs, err := filepath.Abs(opts.Project)
		if err != nil {
			return cfg
		}
		if _, err := config.FindNSelfRoot(abs); err != nil {
			return cfg
		}
		root = abs
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return cfg
		}
		r, err := config.FindNSelfRoot(cwd)
		if err != nil {
			// No project found — use defaults (machine-wide trust, no domain-specific certs).
			return cfg
		}
		root = r
	}

	cfg.WorkDir = root

	projectCfg, loadErr := config.Load(root)
	if loadErr != nil {
		return cfg
	}

	if projectCfg.BaseDomain != "" {
		cfg.BaseDomain = projectCfg.BaseDomain
	}

	if projectCfg.Nginx.SSLPort > 0 {
		cfg.NginxSSLPort = projectCfg.Nginx.SSLPort
	}

	if projectCfg.Nginx.HTTPPort > 0 {
		cfg.NginxHTTPPort = projectCfg.Nginx.HTTPPort
	}

	if projectCfg.ExtraSSLDomains != "" {
		for _, d := range strings.Split(projectCfg.ExtraSSLDomains, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				cfg.ExtraSSLDomains = append(cfg.ExtraSSLDomains, d)
			}
		}
	}

	cfg.NamespacePrefixes = extractNamespacePrefixes(projectCfg, cfg.BaseDomain)

	return cfg
}

// extractNamespacePrefixes parses FrontendApp and CustomService routes to find
// namespace prefixes — intermediate subdomain segments between the service name
// and BASE_DOMAIN. For example, a route of "www.pro.ummat.local" yields "pro".
// These are used to generate per-namespace wildcard SANs (*.pro.ummat.local)
// because mkcert does not support double-wildcard SANs (*.*.ummat.local).
func extractNamespacePrefixes(projectCfg *config.Config, baseDomain string) []string {
	seen := make(map[string]bool)
	var prefixes []string

	checkRoute := func(route string) {
		if route == "" {
			return
		}
		// Strip ".baseDomain" suffix to get the subdomain portion.
		sub := strings.TrimSuffix(route, "."+baseDomain)
		if sub == route {
			// Route doesn't contain baseDomain — treat the whole value as a subdomain path.
			sub = route
		}
		// If the subdomain has multiple dot-separated parts, the rightmost is the namespace.
		// Example: "www.pro" → namespace "pro"; "chat" → no namespace (single level).
		parts := strings.Split(sub, ".")
		if len(parts) < 2 {
			return
		}
		ns := parts[len(parts)-1]
		if ns == "" || seen[ns] {
			return
		}
		seen[ns] = true
		prefixes = append(prefixes, ns)
	}

	for _, fa := range projectCfg.FrontendApps {
		checkRoute(fa.Route)
	}
	for _, cs := range projectCfg.CustomServices {
		if cs.Public {
			checkRoute(cs.Route)
		}
	}

	return prefixes
}
