package commands

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/ssl"
	"github.com/nself-org/cli/internal/trust"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// trustOpts holds resolved options for the trust command.
type trustOpts struct {
	SkipDNS   bool
	SkipSSL   bool
	SkipPorts bool
	Status    bool
	Undo      bool
	Project   string
}

var trustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Set up local dev trust (DNS, SSL, port forwarding)",
	Long: `Set up trusted local development: dnsmasq DNS resolution, mkcert SSL certificates,
and port forwarding so all *.local projects work with HTTPS.

Covers ALL .local projects with a single setup — no per-project configuration needed.

Run once per machine:
  nself trust

Check current status:
  nself trust --status

Undo all changes:
  nself trust --undo`,
	RunE: runTrust,
}

// trustStatusCmd is the `nself trust status` subcommand.
var trustStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show trusted cert CAs, last rotation date, and expiry warnings",
	Long: `Show the current state of all local dev trust components.

Lists trusted certificate authorities, SSL certificate expiry dates, and warns
when any certificate expires within 30 days.

Exit codes:
  0 — all trust components configured and no expiry warnings
  2 — one or more components missing or cert expiry <30 days`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, _ := cmd.Flags().GetString("project")
		opts := trustOpts{Project: project}
		cfg := buildTrustConfig(opts)
		return runTrustStatusDetailed(cfg)
	},
}

func init() {
	trustCmd.Flags().Bool("skip-dns", false, "Skip dnsmasq and /etc/resolver setup")
	trustCmd.Flags().Bool("skip-ssl", false, "Skip mkcert CA and certificate generation")
	trustCmd.Flags().Bool("skip-ports", false, "Skip port forwarding setup")
	trustCmd.Flags().Bool("status", false, "Show current trust status and exit")
	trustCmd.Flags().Bool("undo", false, "Print instructions to undo all trust changes")
	trustCmd.Flags().StringP("project", "p", "", "Path to nself project directory (allows running from any cwd)")

	trustStatusCmd.Flags().StringP("project", "p", "", "Path to nself project directory")

	trustCmd.AddCommand(trustStatusCmd)
	RootCmd.AddCommand(trustCmd)
}

// resolveTrustOpts reads flag values from the cobra command into a trustOpts struct.
func resolveTrustOpts(cmd *cobra.Command) trustOpts {
	skipDNS, _ := cmd.Flags().GetBool("skip-dns")
	skipSSL, _ := cmd.Flags().GetBool("skip-ssl")
	skipPorts, _ := cmd.Flags().GetBool("skip-ports")
	status, _ := cmd.Flags().GetBool("status")
	undo, _ := cmd.Flags().GetBool("undo")
	project, _ := cmd.Flags().GetString("project")
	return trustOpts{
		SkipDNS:   skipDNS,
		SkipSSL:   skipSSL,
		SkipPorts: skipPorts,
		Status:    status,
		Undo:      undo,
		Project:   project,
	}
}

func runTrust(cmd *cobra.Command, args []string) error {
	opts := resolveTrustOpts(cmd)

	ui.CommandHeader("nself trust", "Local dev trust setup")

	// Validate OS support.
	goos := runtime.GOOS
	if goos != "darwin" && goos != "linux" {
		ui.Warn(fmt.Sprintf("nself trust is not supported on %s (supports macOS and Linux)", goos))
		return nil
	}

	// Build TrustConfig — try to load project config but fall back to defaults.
	cfg := buildTrustConfig(opts)

	// --status: show status table and exit.
	if opts.Status {
		return runTrustStatus(cfg)
	}

	// --undo: print instructions and exit.
	if opts.Undo {
		return runTrustUndo(cfg)
	}

	// Print what will be done.
	printTrustPlan(opts, goos)

	// Execute trust setup.
	result, err := trust.Setup(cfg)
	if err != nil {
		ui.Error(fmt.Sprintf("Trust setup failed: %v", err))
		return err
	}

	// Print summary.
	printTrustSummary(result)

	// Surface any non-fatal errors.
	if len(result.Errors) > 0 {
		fmt.Println()
		ui.Warn("Some steps reported errors:")
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
	}

	return nil
}

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

// printTrustPlan prints a summary of what the trust command will do.
func printTrustPlan(opts trustOpts, goos string) {
	fmt.Println()
	ui.Section("Setup plan")

	if !opts.SkipDNS {
		switch goos {
		case "darwin":
			fmt.Println("  1. dnsmasq — install (if needed) + configure address=/.local/127.0.0.1")
			fmt.Println("     /etc/resolver/local — write nameserver 127.0.0.1 (requires admin)")
		case "linux":
			fmt.Println("  1. DNS — configure systemd-resolved or dnsmasq for .local wildcard resolution")
		}
	} else {
		fmt.Println("  1. DNS — SKIPPED (--skip-dns)")
	}

	if !opts.SkipSSL {
		fmt.Println("  2. mkcert — install CA (if needed) + generate wildcard certificates")
	} else {
		fmt.Println("  2. SSL — SKIPPED (--skip-ssl)")
	}

	if !opts.SkipPorts {
		switch goos {
		case "darwin":
			fmt.Println("  3. pfctl — configure 80→Nginx HTTP and 443→Nginx SSL port forwarding (requires admin)")
		case "linux":
			fmt.Println("  3. iptables — configure 80→Nginx HTTP and 443→Nginx SSL port forwarding (requires root)")
		}
	} else {
		fmt.Println("  3. Port forwarding — SKIPPED (--skip-ports)")
	}

	fmt.Println()
}

// printTrustSummary prints the result of the trust setup.
func printTrustSummary(result *trust.TrustResult) {
	fmt.Println()
	ui.Section("Setup complete")

	if result.DNSConfigured {
		if result.DNSAlreadyDone {
			fmt.Println("  \u2713 dnsmasq — already configured")
		} else {
			fmt.Println("  \u2713 dnsmasq — configured")
		}
	}

	if result.ResolverConfigured {
		if result.ResolverAlreadyDone {
			fmt.Println("  \u2713 /etc/resolver/local — already present")
		} else {
			fmt.Println("  \u2713 /etc/resolver/local — written")
		}
	}

	if result.CertsGenerated {
		if result.CertsAlreadyDone {
			fmt.Println("  \u2713 SSL certificates — already valid")
		} else {
			fmt.Println("  \u2713 SSL certificates — generated")
		}
	}

	if result.PortsConfigured {
		if result.PortsAlreadyDone {
			fmt.Println("  \u2713 Port forwarding — already active")
		} else {
			fmt.Println("  \u2713 Port forwarding — configured")
		}
	}

	if len(result.Errors) == 0 {
		fmt.Println()
		ui.Success("Local dev trust setup complete — your *.local projects are ready for HTTPS.")
	}
}

// runTrustStatus prints the current trust status as a table.
func runTrustStatus(cfg trust.TrustConfig) error {
	status := trust.CheckStatus(cfg)

	fmt.Println()
	ui.Section("Trust status")

	check := func(ok bool) string {
		if ok {
			return "\u2713"
		}
		return "\u2717"
	}

	fmt.Printf("  %s mkcert installed\n", check(status.MkcertInstalled))
	fmt.Printf("  %s mkcert CA trusted\n", check(status.CAInstalled))
	fmt.Printf("  %s SSL certificates exist\n", check(status.CertsExist))
	fmt.Printf("  %s SSL certificates valid (>30 days)\n", check(status.CertsValid))
	fmt.Printf("  %s dnsmasq installed\n", check(status.DNSInstalled))
	fmt.Printf("  %s dnsmasq .local configured\n", check(status.DNSRunning))
	fmt.Printf("  %s resolver configured\n", check(status.ResolverConfigured))
	fmt.Printf("  %s port forwarding active\n", check(status.PortsForwarding))
	fmt.Println()

	allGood := status.MkcertInstalled &&
		status.CAInstalled &&
		status.CertsValid &&
		status.DNSRunning &&
		status.ResolverConfigured &&
		status.PortsForwarding

	if allGood {
		ui.Success("All trust components are configured and active.")
	} else {
		ui.Warn("Some trust components are not configured. Run 'nself trust' to set them up.")
	}

	return nil
}

// runTrustUndo prints instructions to undo all trust changes.
func runTrustUndo(cfg trust.TrustConfig) error {
	goos := runtime.GOOS

	fmt.Println()
	ui.Section("Undo instructions")
	ui.Warn("The following commands will remove all nself trust configuration.")
	fmt.Println()

	switch goos {
	case "darwin":
		fmt.Println("  # 1. Remove dnsmasq .local entry")
		fmt.Println("  #    Edit /opt/homebrew/etc/dnsmasq.conf (or /usr/local/etc/dnsmasq.conf)")
		fmt.Println("  #    Remove the line: address=/.local/127.0.0.1")
		fmt.Println("  brew services restart dnsmasq")
		fmt.Println()
		fmt.Println("  # 2. Remove /etc/resolver/local")
		fmt.Println("  sudo rm /etc/resolver/local")
		fmt.Println("  dscacheutil -flushcache && killall -HUP mDNSResponder")
		fmt.Println()
		fmt.Println("  # 3. Remove mkcert CA from keychain")
		caPath, err := getMkcertCAPathForUndo()
		if err == nil && caPath != "" {
			fmt.Printf("  sudo security remove-trusted-cert -d %s\n", caPath)
		} else {
			fmt.Println("  # Run: mkcert -uninstall")
		}
		fmt.Println()
		fmt.Println("  # 4. Remove pfctl port forwarding rules")
		fmt.Printf("  sudo rm -f %s\n", pfAnchorPath())
		fmt.Printf("  sudo launchctl unload -w %s\n", launchDaemonPath())
		fmt.Printf("  sudo rm -f %s\n", launchDaemonPath())
		fmt.Println()
		fmt.Println("  # 5. Remove SSL certificates (optional)")
		if cfg.WorkDir != "" {
			fmt.Printf("  rm -rf %s/ssl/\n", cfg.WorkDir)
		}

	case "linux":
		fmt.Println("  # 1. Remove DNS configuration")
		fmt.Println("  sudo rm -f /etc/systemd/resolved.conf.d/nself.conf")
		fmt.Println("  sudo systemctl restart systemd-resolved")
		fmt.Println("  # OR for raw dnsmasq:")
		fmt.Println("  #   Edit /etc/dnsmasq.conf and remove: address=/.local/127.0.0.1")
		fmt.Println("  #   sudo systemctl restart dnsmasq")
		fmt.Println()
		fmt.Println("  # 2. Remove iptables port forwarding rules")
		fmt.Printf("  sudo iptables -t nat -D OUTPUT -p tcp --dport 443 -j REDIRECT --to-port %d\n", cfg.NginxSSLPort)
		fmt.Printf("  sudo iptables -t nat -D OUTPUT -p tcp --dport 80 -j REDIRECT --to-port %d\n", cfg.NginxHTTPPort)
		fmt.Println("  sudo iptables-save > /etc/iptables/rules.v4")
		fmt.Println()
		fmt.Println("  # 3. Remove mkcert CA")
		fmt.Println("  mkcert -uninstall")
		fmt.Println()
		fmt.Println("  # 4. Remove SSL certificates (optional)")
		if cfg.WorkDir != "" {
			fmt.Printf("  rm -rf %s/ssl/\n", cfg.WorkDir)
		}

	default:
		ui.Warn(fmt.Sprintf("nself trust is not supported on %s — nothing to undo.", goos))
	}

	fmt.Println()
	ui.Info("These instructions remove all changes made by 'nself trust'.")
	return nil
}

// pfAnchorPath returns the pf anchor file path constant.
func pfAnchorPath() string {
	return "/etc/pf.anchors/nself.local"
}

// launchDaemonPath returns the LaunchDaemon plist path constant.
func launchDaemonPath() string {
	return "/Library/LaunchDaemons/com.nself.portforward.plist"
}

// getMkcertCAPathForUndo returns the mkcert CA certificate path for use in
// undo instructions. Returns empty string on error (graceful degradation).
func getMkcertCAPathForUndo() (string, error) {
	return trust.MkcertCAPath()
}

// runTrustStatusDetailed shows trusted cert CAs with expiry dates and warns
// on certificates expiring within 30 days. Returns exit code 2 via error when
// any component is missing or a cert is near-expiry.
func runTrustStatusDetailed(cfg trust.TrustConfig) error {
	status := trust.CheckStatus(cfg)

	fmt.Println()
	ui.Section("Trust status — certificate authorities")

	hasWarning := false

	// mkcert CA info.
	if !status.MkcertInstalled {
		fmt.Println("  ✗ mkcert — not installed")
		hasWarning = true
	} else {
		caPath, caErr := ssl.MkcertCACertPath()
		if caErr != nil {
			fmt.Println("  ✗ mkcert CA — path unavailable")
			hasWarning = true
		} else {
			caInfo := parseCACertInfo(caPath)
			if caInfo == nil {
				if status.CAInstalled {
					fmt.Printf("  ✓ mkcert CA — trusted (expiry unreadable)\n")
				} else {
					fmt.Printf("  ✗ mkcert CA — not trusted  path: %s\n", caPath)
					hasWarning = true
				}
			} else {
				daysLeft := int(time.Until(caInfo.NotAfter).Hours()) / 24
				mark := "✓"
				if !status.CAInstalled {
					mark = "✗"
					hasWarning = true
				}
				if daysLeft < 30 {
					hasWarning = true
					fmt.Printf("  %s mkcert CA %-32s  expires: %s  WARN: %d days remaining\n",
						mark, caInfo.Subject.CommonName,
						caInfo.NotAfter.Format("2006-01-02"), daysLeft)
				} else {
					fmt.Printf("  %s mkcert CA %-32s  expires: %s  (%d days)\n",
						mark, caInfo.Subject.CommonName,
						caInfo.NotAfter.Format("2006-01-02"), daysLeft)
				}
			}
		}
	}

	// Project SSL cert (if WorkDir is set).
	fmt.Println()
	ui.Section("Trust status — project certificates")

	if cfg.WorkDir == "" {
		fmt.Println("  (no project directory — run from a project root or pass -p)")
	} else {
		certPath := filepath.Join(cfg.WorkDir, "ssl", "fullchain.pem")
		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			fmt.Println("  ✗ project cert — not found (run 'nself trust' to generate)")
			hasWarning = true
		} else {
			certInfo := parseCACertInfo(certPath)
			if certInfo == nil {
				fmt.Printf("  ? project cert — found but unreadable: %s\n", certPath)
				hasWarning = true
			} else {
				daysLeft := int(time.Until(certInfo.NotAfter).Hours()) / 24
				if daysLeft < 0 {
					fmt.Printf("  ✗ project cert %-30s  EXPIRED %d days ago\n",
						certInfo.Subject.CommonName, -daysLeft)
					hasWarning = true
				} else if daysLeft < 30 {
					fmt.Printf("  ⚠ project cert %-30s  expires: %s  WARN: %d days remaining\n",
						certInfo.Subject.CommonName,
						certInfo.NotAfter.Format("2006-01-02"), daysLeft)
					hasWarning = true
				} else {
					fmt.Printf("  ✓ project cert %-30s  expires: %s  (%d days)\n",
						certInfo.Subject.CommonName,
						certInfo.NotAfter.Format("2006-01-02"), daysLeft)
				}
			}
		}
	}

	// Other trust components.
	fmt.Println()
	ui.Section("Trust status — components")

	check := func(ok bool) string {
		if ok {
			return "✓"
		}
		return "✗"
	}

	fmt.Printf("  %s DNS (.local wildcard)\n", check(status.DNSRunning))
	fmt.Printf("  %s /etc/resolver configured\n", check(status.ResolverConfigured))
	fmt.Printf("  %s port forwarding (80/443)\n", check(status.PortsForwarding))
	fmt.Println()

	if !status.DNSRunning || !status.ResolverConfigured || !status.PortsForwarding {
		hasWarning = true
	}

	if hasWarning {
		ui.Warn("Some trust components need attention. Run 'nself trust' to fix.")
		// Return a sentinel that produces exit code 2 (warning, not hard error).
		return fmt.Errorf("trust warnings present (exit code 2)")
	}

	ui.Success("All trust components configured — no expiry warnings.")
	return nil
}

// parseCACertInfo reads and parses the first PEM certificate in path.
// Returns nil if the file cannot be read or parsed (graceful degradation).
func parseCACertInfo(path string) *x509.Certificate {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil
	}
	return cert
}
