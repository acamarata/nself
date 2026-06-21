// Package commands — security.go
//
// `nself security` provides server-level security operations: audit, setup,
// and status. These commands inspect the host environment (firewall, fail2ban,
// SSH config) and the running stack for common hardening gaps. They are
// intentionally non-destructive by default — `setup` requires `--apply` to
// make changes, otherwise it prints what would be done.
package commands

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ssl"
	"github.com/nself-org/cli/internal/ui"
	"github.com/spf13/cobra"
)

// securityCmd is the parent command for `nself security ...`.
var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Server security: audit, setup, and status",
	Long: `Inspect and harden the host running your nself stack.

Subcommands:
  nself security audit    Run a read-only security audit of the host + stack
  nself security setup    Apply baseline hardening (firewall, fail2ban, sshd)
  nself security status   Show the current security posture as a summary

All commands are safe by default — 'setup' will not change anything unless
'--apply' is passed. Without '--apply' it prints the steps it would take.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var securityAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Run a read-only security audit",
	RunE:  runSecurityAudit,
}

var securityStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current security posture",
	RunE:  runSecurityStatus,
}

var securitySetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Apply baseline server hardening",
	Long: `Apply baseline server hardening:
  - Enable UFW firewall and allow 22/80/443
  - Install and enable fail2ban with sshd jail
  - Disable SSH password authentication and root login

Without --apply, prints the planned actions and exits without making changes.`,
	RunE: runSecuritySetup,
}

// securityScanCmd is the `nself security scan` subcommand.
// Scan is always FREE — no license check on this path (Security-Always-Free Doctrine).
// It re-uses runChecks() which is the same read-only probe set used by audit/status.
// Exit codes: 0 — all checks pass; 1 — one or more checks fail.
var securityScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan host security posture (free, no license required)",
	Long: `Run a read-only security scan of the host + nSelf stack.

All checks are free and require no license. The scan never modifies the system.

Exit codes:
  0 — all security checks pass
  1 — one or more checks failed (run 'nself security setup --apply' to remediate)`,
	RunE: runSecurityScan,
}

func init() {
	securitySetupCmd.Flags().Bool("apply", false, "Actually apply changes (default: dry-run)")
	securityCmd.AddCommand(securityAuditCmd)
	securityCmd.AddCommand(securityStatusCmd)
	securityCmd.AddCommand(securitySetupCmd)
	securityCmd.AddCommand(securityScanCmd)
	RootCmd.AddCommand(securityCmd)
}

// finding represents one security check result.
type finding struct {
	Name   string
	OK     bool
	Detail string
}

// runChecks performs all read-only security probes and returns findings.
func runChecks() []finding {
	var out []finding
	out = append(out, checkFirewall())
	out = append(out, checkIptablesPersistent())
	out = append(out, checkFail2ban())
	out = append(out, checkSSHConfig())
	out = append(out, checkRootLogin())
	out = append(out, checkUnattendedUpgrades())
	out = append(out, checkDockerExposedPorts())
	out = append(out, checkEnvFilePerms())
	out = append(out, checkCertExpiry())
	out = append(out, checkKeyRotationAge())
	out = append(out, checkAdminPortBinding())
	out = append(out, checkCSRFGuard())
	return out
}

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

// checkIptablesPersistent verifies that netfilter-persistent is installed and
// enabled so firewall rules survive reboots (S32-T11).
func checkIptablesPersistent() finding {
	if runtime.GOOS != "linux" {
		return finding{Name: "Iptables persistence", OK: true, Detail: "skipped (non-linux host)"}
	}
	if _, err := exec.LookPath("netfilter-persistent"); err != nil {
		return finding{Name: "Iptables persistence", OK: false, Detail: "netfilter-persistent not installed — rules will not survive reboot"}
	}
	if err := exec.Command("systemctl", "is-enabled", "--quiet", "netfilter-persistent").Run(); err != nil {
		return finding{Name: "Iptables persistence", OK: false, Detail: "netfilter-persistent installed but not enabled"}
	}
	if _, err := os.Stat("/etc/iptables/rules.v4"); err != nil {
		return finding{Name: "Iptables persistence", OK: false, Detail: "/etc/iptables/rules.v4 missing — rules not saved"}
	}
	return finding{Name: "Iptables persistence", OK: true, Detail: "enabled, rules saved"}
}

func checkFirewall() finding {
	if runtime.GOOS != "linux" {
		return finding{Name: "Firewall (ufw)", OK: true, Detail: "skipped (non-linux host)"}
	}
	if _, err := exec.LookPath("ufw"); err != nil {
		return finding{Name: "Firewall (ufw)", OK: false, Detail: "ufw not installed"}
	}
	out, err := exec.Command("ufw", "status").Output()
	if err != nil {
		return finding{Name: "Firewall (ufw)", OK: false, Detail: "could not query ufw: " + err.Error()}
	}
	if strings.Contains(strings.ToLower(string(out)), "status: active") {
		return finding{Name: "Firewall (ufw)", OK: true, Detail: "active"}
	}
	return finding{Name: "Firewall (ufw)", OK: false, Detail: "installed but inactive"}
}

func checkFail2ban() finding {
	if runtime.GOOS != "linux" {
		return finding{Name: "fail2ban", OK: true, Detail: "skipped (non-linux host)"}
	}
	if _, err := exec.LookPath("fail2ban-client"); err != nil {
		return finding{Name: "fail2ban", OK: false, Detail: "not installed"}
	}
	if err := exec.Command("fail2ban-client", "ping").Run(); err != nil {
		return finding{Name: "fail2ban", OK: false, Detail: "installed but not running"}
	}
	return finding{Name: "fail2ban", OK: true, Detail: "running"}
}

func checkSSHConfig() finding {
	path := "/etc/ssh/sshd_config"
	f, err := os.Open(path)
	if err != nil {
		return finding{Name: "SSH password auth", OK: true, Detail: "sshd_config not readable (skipped)"}
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "passwordauthentication") {
			if strings.Contains(strings.ToLower(line), "no") {
				return finding{Name: "SSH password auth", OK: true, Detail: "disabled"}
			}
			return finding{Name: "SSH password auth", OK: false, Detail: "enabled — keys-only is recommended"}
		}
	}
	return finding{Name: "SSH password auth", OK: false, Detail: "not explicitly disabled (default is yes)"}
}

func checkRootLogin() finding {
	path := "/etc/ssh/sshd_config"
	f, err := os.Open(path)
	if err != nil {
		return finding{Name: "SSH root login", OK: true, Detail: "sshd_config not readable (skipped)"}
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "permitrootlogin") {
			if strings.Contains(strings.ToLower(line), "no") {
				return finding{Name: "SSH root login", OK: true, Detail: "disabled"}
			}
			return finding{Name: "SSH root login", OK: false, Detail: "permitted"}
		}
	}
	return finding{Name: "SSH root login", OK: false, Detail: "not explicitly disabled"}
}

func checkUnattendedUpgrades() finding {
	if runtime.GOOS != "linux" {
		return finding{Name: "Unattended upgrades", OK: true, Detail: "skipped (non-linux host)"}
	}
	if _, err := os.Stat("/etc/apt/apt.conf.d/20auto-upgrades"); err == nil {
		return finding{Name: "Unattended upgrades", OK: true, Detail: "configured"}
	}
	return finding{Name: "Unattended upgrades", OK: false, Detail: "not configured"}
}

func checkDockerExposedPorts() finding {
	if _, err := exec.LookPath("docker"); err != nil {
		return finding{Name: "Docker port exposure", OK: true, Detail: "docker not installed (skipped)"}
	}
	out, err := exec.Command("docker", "ps", "--format", "{{.Ports}}").Output()
	if err != nil {
		return finding{Name: "Docker port exposure", OK: true, Detail: "could not query docker (skipped)"}
	}
	exposed := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		// "0.0.0.0:" indicates a port bound to all interfaces
		if strings.Contains(line, "0.0.0.0:") {
			exposed++
		}
	}
	if exposed == 0 {
		return finding{Name: "Docker port exposure", OK: true, Detail: "no containers bound to 0.0.0.0"}
	}
	return finding{Name: "Docker port exposure", OK: false, Detail: fmt.Sprintf("%d container(s) bound to 0.0.0.0 — front with nginx", exposed)}
}

func checkEnvFilePerms() finding {
	candidates := []string{".env", ".env.secrets", ".env.prod"}
	for _, p := range candidates {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		mode := info.Mode().Perm()
		if mode&0o077 != 0 {
			return finding{Name: "Env file permissions", OK: false, Detail: fmt.Sprintf("%s is %v (should be 0600)", p, mode)}
		}
	}
	return finding{Name: "Env file permissions", OK: true, Detail: "ok"}
}

func runSecurityAudit(cmd *cobra.Command, args []string) error {
	ui.CommandHeader("nself security audit", "Read-only security audit")
	findings := runChecks()
	pass, fail := 0, 0
	for _, f := range findings {
		mark := "FAIL"
		if f.OK {
			mark = "PASS"
			pass++
		} else {
			fail++
		}
		fmt.Printf("  [%s] %-25s  %s\n", mark, f.Name, f.Detail)
	}
	fmt.Printf("\n%d passed, %d failed\n", pass, fail)
	if fail > 0 {
		fmt.Println("\nRun 'nself security setup' to apply baseline hardening.")
	}
	return nil
}

func runSecurityStatus(cmd *cobra.Command, args []string) error {
	ui.CommandHeader("nself security status", "Current security posture")
	findings := runChecks()
	score := 0
	for _, f := range findings {
		if f.OK {
			score++
		}
	}
	pct := (score * 100) / len(findings)
	fmt.Printf("Posture score: %d/%d (%d%%)\n\n", score, len(findings), pct)
	for _, f := range findings {
		state := "needs attention"
		if f.OK {
			state = "ok"
		}
		fmt.Printf("  - %s: %s (%s)\n", f.Name, state, f.Detail)
	}
	return nil
}

// hardeningStep describes a single setup action.
type hardeningStep struct {
	Name string
	Cmd  []string
}

func plannedHardeningSteps() []hardeningStep {
	// S32-T11: install iptables-persistent / netfilter-persistent so firewall
	// rules (ufw + any custom iptables) survive reboots. On Debian/Ubuntu ufw
	// rules persist via the ufw systemd unit, but raw iptables rules do not
	// without netfilter-persistent. Install both packages + enable the service
	// and save current rules to /etc/iptables/rules.v4 and rules.v6.
	return []hardeningStep{
		{Name: "Install ufw + fail2ban + iptables-persistent", Cmd: []string{"apt-get", "install", "-y", "ufw", "fail2ban", "iptables-persistent", "netfilter-persistent"}},
		{Name: "Allow SSH (22)", Cmd: []string{"ufw", "allow", "22/tcp"}},
		{Name: "Allow HTTP (80)", Cmd: []string{"ufw", "allow", "80/tcp"}},
		{Name: "Allow HTTPS (443)", Cmd: []string{"ufw", "allow", "443/tcp"}},
		{Name: "Enable ufw", Cmd: []string{"ufw", "--force", "enable"}},
		{Name: "Enable fail2ban", Cmd: []string{"systemctl", "enable", "--now", "fail2ban"}},
		{Name: "Enable netfilter-persistent", Cmd: []string{"systemctl", "enable", "--now", "netfilter-persistent"}},
		{Name: "Save iptables rules (v4)", Cmd: []string{"sh", "-c", "iptables-save > /etc/iptables/rules.v4"}},
		{Name: "Save iptables rules (v6)", Cmd: []string{"sh", "-c", "ip6tables-save > /etc/iptables/rules.v6"}},
		{Name: "Disable SSH password auth", Cmd: []string{"sed", "-i", "s/^#*PasswordAuthentication.*/PasswordAuthentication no/", "/etc/ssh/sshd_config"}},
		{Name: "Disable SSH root login", Cmd: []string{"sed", "-i", "s/^#*PermitRootLogin.*/PermitRootLogin no/", "/etc/ssh/sshd_config"}},
		{Name: "Reload sshd", Cmd: []string{"systemctl", "reload", "sshd"}},
	}
}

func runSecuritySetup(cmd *cobra.Command, args []string) error {
	apply, _ := cmd.Flags().GetBool("apply")
	ui.CommandHeader("nself security setup", "Baseline server hardening")

	if runtime.GOOS != "linux" {
		fmt.Println("Server hardening is only supported on Linux hosts.")
		return nil
	}

	steps := plannedHardeningSteps()

	if !apply {
		fmt.Println("Dry-run — no changes will be made. Pass --apply to execute.")
		fmt.Println()
		for i, s := range steps {
			fmt.Printf("  %2d. %-32s  $ %s\n", i+1, s.Name, strings.Join(s.Cmd, " "))
		}
		fmt.Println("\nRe-run with --apply (as root) to perform these steps.")
		return nil
	}

	if os.Geteuid() != 0 {
		return fmt.Errorf("'nself security setup --apply' must be run as root (try sudo)")
	}

	for _, s := range steps {
		fmt.Printf("→ %s\n", s.Name)
		c := exec.Command(s.Cmd[0], s.Cmd[1:]...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			fmt.Printf("  step failed: %v (continuing)\n", err)
		}
	}
	fmt.Println("\nHardening complete. Run 'nself security audit' to verify.")
	return nil
}

// runSecurityScan performs the same read-only checks as audit but exits 1 when
// any check fails. No license check — this path is always free per the
// Security-Always-Free Doctrine.
func runSecurityScan(cmd *cobra.Command, args []string) error {
	ui.CommandHeader("nself security scan", "Security scan (free)")
	findings := runChecks()
	fail := 0
	for _, f := range findings {
		mark := "PASS"
		if !f.OK {
			mark = "FAIL"
			fail++
		}
		fmt.Printf("  [%s] %-25s  %s\n", mark, f.Name, f.Detail)
	}
	fmt.Println()
	if fail > 0 {
		return fmt.Errorf("%d finding(s) require attention — run 'nself security setup --apply' to remediate", fail)
	}
	fmt.Println("All checks passed.")
	return nil
}
