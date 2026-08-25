package commands

// Purpose: Implements `nself trust undo` — removing the mkcert CA, pf
// anchor, and launch daemon artifacts that `nself trust` installed — plus
// the small path-resolution helpers it needs. Split out of trust.go
// (CLI-R12) to separate the undo path from the entry point/cobra wiring
// (trust.go), config building (trust_config.go), plan/summary/status
// printers (trust_status.go), and the detailed status report
// (trust_status_detailed.go).
// Inputs: a trust.TrustConfig describing what was installed.
// Outputs: removed CA/anchor/daemon files and printed confirmation;
// returns an error if any removal fails.
// Constraints: pure move — no behavior changes.

import (
	"fmt"
	"runtime"

	"github.com/nself-org/cli/internal/trust"
	"github.com/nself-org/cli/internal/ui"
)

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
