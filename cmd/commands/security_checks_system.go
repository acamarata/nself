package commands

// Purpose: The `nself security audit` checks covering host-level hardening:
// iptables persistence, firewall status, fail2ban, SSH config hardening,
// root login, unattended upgrades, exposed Docker ports, and env file
// permissions. Split out of security.go (CLI-R12) to keep each group of
// checks (this file, security_checks_certs_keys.go) in a file under the
// size cap; runChecks (security.go) calls every check function across
// both files to assemble the full finding list.
// Inputs: none beyond ambient host state (running processes, config files,
// exposed ports).
// Outputs: a finding struct per check (name, pass/fail, detail message).
// Constraints: pure move — no behavior changes.

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

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
