package trust

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const pfAnchorPath = "/etc/pf.anchors/nself.local"
const launchDaemonPlistPath = "/Library/LaunchDaemons/com.nself.portforward.plist"
const pfTimeout = 30 * time.Second

// pfAnchorContent returns the pf anchor rules for the given config.
func pfAnchorContent(cfg TrustConfig) string {
	return fmt.Sprintf(
		"rdr pass on lo0 inet proto tcp from any to any port 80 -> 127.0.0.1 port %d\n"+
			"rdr pass on lo0 inet proto tcp from any to any port 443 -> 127.0.0.1 port %d\n",
		cfg.NginxHTTPPort,
		cfg.NginxSSLPort,
	)
}

const launchDaemonPlistContent = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.nself.portforward</string>
    <key>ProgramArguments</key>
    <array>
        <string>/sbin/pfctl</string>
        <string>-ef</string>
        <string>/etc/pf.anchors/nself.local</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`

// CheckPortsDarwin returns true if the pf anchor rules are already loaded
// with the expected port forwarding configuration.
func CheckPortsDarwin(cfg TrustConfig) bool {
	ctx, cancel := context.WithTimeout(context.Background(), pfTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "pfctl", "-a", "nself.local", "-sr").CombinedOutput()
	if err != nil {
		return false
	}

	output := string(out)
	httpRule := fmt.Sprintf("port 80 -> 127.0.0.1 port %d", cfg.NginxHTTPPort)
	sslRule := fmt.Sprintf("port 443 -> 127.0.0.1 port %d", cfg.NginxSSLPort)

	return strings.Contains(output, httpRule) && strings.Contains(output, sslRule)
}

// SetupPortsDarwin configures pfctl port forwarding on macOS so that
// ports 80 and 443 are redirected to the Nginx HTTP and SSL ports.
// Returns alreadyDone=true when the rules were already active.
func SetupPortsDarwin(cfg TrustConfig) (alreadyDone bool, err error) {
	if CheckPortsDarwin(cfg) {
		return true, nil
	}

	anchor := pfAnchorContent(cfg)

	// Build the osascript shell commands to write anchor file, load it, and install LaunchDaemon.
	// All three steps run under administrator privileges via a single osascript call.
	shellCmd := fmt.Sprintf(
		"mkdir -p /etc/pf.anchors && "+
			"printf '%%s' '%s' > %s && "+
			"{ pfctl -ef %s; _pfrc=$?; [ \"$_pfrc\" -eq 0 ] || [ \"$_pfrc\" -eq 1 ]; } && "+
			"mkdir -p /Library/LaunchDaemons && "+
			"printf '%%s' '%s' > %s && "+
			"launchctl load -w %s",
		escapeForOsascript(anchor),
		pfAnchorPath,
		pfAnchorPath,
		escapeForOsascript(launchDaemonPlistContent),
		launchDaemonPlistPath,
		launchDaemonPlistPath,
	)

	script := fmt.Sprintf(`do shell script "%s" with administrator privileges`, escapeForOsascript(shellCmd))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, execErr := exec.CommandContext(ctx, "osascript", "-e", script).CombinedOutput()
	if execErr != nil {
		return false, fmt.Errorf("configuring pfctl port forwarding: %w — %s", execErr, strings.TrimSpace(string(out)))
	}

	return false, nil
}

// escapeForOsascript escapes a string for safe embedding inside an osascript
// double-quoted shell string. Single quotes inside the shell command are the
// main concern; we replace them with '\\'' to break out of single-quoting
// used by printf.
func escapeForOsascript(s string) string {
	// Escape backslashes first, then double-quotes (osascript outer quotes),
	// then percent signs (printf format specifiers handled by %% in callers).
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// writePfAnchorDirect writes the pf anchor file directly (for use when
// already running as root). Exposed for testing/Linux paths.
func writePfAnchorDirect(cfg TrustConfig) error {
	if err := os.MkdirAll("/etc/pf.anchors", 0755); err != nil {
		return fmt.Errorf("creating /etc/pf.anchors: %w", err)
	}
	content := pfAnchorContent(cfg)
	if err := os.WriteFile(pfAnchorPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing pf anchor file: %w", err)
	}
	return nil
}
