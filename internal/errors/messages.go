// Package errors provides the canonical error catalog for the nSelf CLI.
//
// Each error has:
//   - A stable code (ERR-INSTALL-001, ERR-LICENSE-001, etc.)
//   - A human-readable "what happened" message
//   - A root-cause explanation
//   - Actionable remediation steps
//   - A help link pointing to the Errors wiki
//
// Every user-visible error must end with HelpFooter() appended to the Fix field
// so users always know where to get more help.
//
// Wiki: https://github.com/nself-org/cli/wiki/Errors
package errors

import (
	"fmt"
	"strings"
)

const (
	// HelpURL is the base URL for the nSelf support portal.
	HelpURL = "https://nself.org/support"
	// DiscordURL is the nSelf community Discord invite.
	DiscordURL = "https://discord.gg/nself"
	// DiscussionsURL is the GitHub Discussions link.
	DiscussionsURL = "https://github.com/orgs/nself-org/discussions"
	// WikiErrorsURL is the CLI errors wiki page.
	WikiErrorsURL = "https://github.com/nself-org/cli/wiki/Errors"
)

// HelpFooter returns the standard "Need help?" line appended to every error Fix.
// Format: "Need help? <url1> · <url2> · <url3>"
func HelpFooter() string {
	return fmt.Sprintf("Need help? %s · Discord: %s · GitHub: %s",
		HelpURL, DiscordURL, DiscussionsURL)
}

// Message represents a structured user-facing error for the nSelf CLI.
type Message struct {
	// Code is the stable error code (e.g., "ERR-INSTALL-001").
	Code string
	// What describes what happened.
	What string
	// Why explains the root cause.
	Why string
	// Fix provides actionable remediation steps.
	Fix string
	// WikiAnchor is the fragment on the Errors wiki page (e.g., "err-install-001").
	WikiAnchor string
}

// Error implements the error interface.
func (m *Message) Error() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Error: [%s] %s", m.Code, m.What))
	if m.Why != "" {
		b.WriteString(fmt.Sprintf("\n  Why:  %s", m.Why))
	}
	if m.Fix != "" {
		b.WriteString(fmt.Sprintf("\n  Fix:  %s", m.Fix))
	}
	if m.WikiAnchor != "" {
		b.WriteString(fmt.Sprintf("\n  Docs: %s#%s", WikiErrorsURL, m.WikiAnchor))
	}
	b.WriteString(fmt.Sprintf("\n  %s", HelpFooter()))
	return b.String()
}

// WithProcess returns a copy of ERRPortConflict with the process name filled in.
func (m *Message) WithProcess(process string) *Message {
	cp := *m
	if process != "" {
		cp.What = strings.Replace(cp.What, "<process>", process, 1)
		cp.Fix = strings.Replace(cp.Fix, "<process>", process, 1)
	}
	return &cp
}

// WithDetail returns a copy of the message with additional context appended to Why.
func (m *Message) WithDetail(detail string) *Message {
	cp := *m
	if detail != "" {
		cp.Why = cp.Why + " " + detail
	}
	return &cp
}

// ── Install errors (ERR-INSTALL-*) ──────────────────────────────────────────

// ErrInstallDockerNotFound is returned when the docker binary is absent from PATH.
var ErrInstallDockerNotFound = &Message{
	Code:       "ERR-INSTALL-001",
	What:       "Docker not found in PATH",
	Why:        "The docker binary is not installed or is not on your system PATH.",
	Fix:        "Install Docker Desktop from https://docs.docker.com/get-docker/ then restart your terminal. " + HelpFooter(),
	WikiAnchor: "err-install-001",
}

// ErrInstallDockerNotRunning is returned when Docker is installed but the daemon is not responding.
var ErrInstallDockerNotRunning = &Message{
	Code:       "ERR-INSTALL-002",
	What:       "Docker daemon is not running",
	Why:        "The Docker daemon did not respond within the check timeout.",
	Fix:        "macOS: open Docker Desktop. Linux: run `sudo systemctl start docker`. " + HelpFooter(),
	WikiAnchor: "err-install-002",
}

// ErrInstallDependencyMissing is returned when a required dependency is absent.
var ErrInstallDependencyMissing = &Message{
	Code:       "ERR-INSTALL-003",
	What:       "Required dependency missing",
	Why:        "A binary required by nself was not found.",
	Fix:        "macOS: `brew install <dep>`. Ubuntu/Debian: `sudo apt install <dep>`. Arch: `sudo pacman -S <dep>`. " + HelpFooter(),
	WikiAnchor: "err-install-003",
}

// ErrInstallPermissionDenied is returned when the installer cannot write to the target directory.
var ErrInstallPermissionDenied = &Message{
	Code:       "ERR-INSTALL-004",
	What:       "Permission denied during install",
	Why:        "The install target directory is not writable by the current user.",
	Fix:        "Run the install command with sudo, or change the install prefix to a user-writable path. " + HelpFooter(),
	WikiAnchor: "err-install-004",
}

// ErrInstallDiskFull is returned when there is insufficient disk space.
var ErrInstallDiskFull = &Message{
	Code:       "ERR-INSTALL-005",
	What:       "Insufficient disk space",
	Why:        "The target filesystem does not have enough free space to complete the operation.",
	Fix:        "Free at least 2 GB of disk space, then retry. Run `df -h` to check free space. " + HelpFooter(),
	WikiAnchor: "err-install-005",
}

// ── License errors (ERR-LICENSE-*) ──────────────────────────────────────────

// ErrLicenseInvalid is returned when the license key format is wrong or the key is rejected by ping_api.
var ErrLicenseInvalid = &Message{
	Code:       "ERR-LICENSE-001",
	What:       "License key is invalid",
	Why:        "The key format was not recognised or ping.nself.org rejected it.",
	Fix:        "Check the key at https://nself.org/account/license. Keys start with `nself_pro_`. " + HelpFooter(),
	WikiAnchor: "err-license-001",
}

// ErrLicenseExpired is returned when the license key has expired.
var ErrLicenseExpired = &Message{
	Code:       "ERR-LICENSE-002",
	What:       "License key has expired",
	Why:        "The subscription associated with this key is no longer active.",
	Fix:        "Renew at https://nself.org/account/license or run `nself license info` to check status. " + HelpFooter(),
	WikiAnchor: "err-license-002",
}

// ErrLicenseNetworkUnavailable is returned when the license server cannot be reached and there is no valid cache.
var ErrLicenseNetworkUnavailable = &Message{
	Code:       "ERR-LICENSE-003",
	What:       "Cannot reach license server",
	Why:        "ping.nself.org is not reachable and no valid cached license was found.",
	Fix:        "Check your internet connection. If offline mode is needed, set NSELF_LICENSE_OFFLINE=1. " + HelpFooter(),
	WikiAnchor: "err-license-003",
}

// ── Plugin errors (ERR-PLUGIN-*) ─────────────────────────────────────────────

// ErrPluginNotFound is returned when the requested plugin does not exist in the registry.
var ErrPluginNotFound = &Message{
	Code:       "ERR-PLUGIN-001",
	What:       "Plugin not found in registry",
	Why:        "No plugin with that name is registered in the nSelf plugin registry.",
	Fix:        "Run `nself plugin list` to see available plugins. Check spelling. " + HelpFooter(),
	WikiAnchor: "err-plugin-001",
}

// ErrPluginBuildStale is returned when the installed plugin version does not match the built compose file.
var ErrPluginBuildStale = &Message{
	Code:       "ERR-PLUGIN-002",
	What:       "Plugin config is stale",
	Why:        "The installed plugin version differs from what was used to generate docker-compose.yml.",
	Fix:        "Run `nself build` to regenerate the compose file, then `nself start`. " + HelpFooter(),
	WikiAnchor: "err-plugin-002",
}

// ── Network errors (ERR-NETWORK-*) ──────────────────────────────────────────

// ErrNetworkDown is returned when a network operation fails due to connectivity.
var ErrNetworkDown = &Message{
	Code:       "ERR-NETWORK-001",
	What:       "Network unavailable",
	Why:        "A required outbound connection could not be established.",
	Fix:        "Check your internet connection. If behind a proxy, set HTTP_PROXY / HTTPS_PROXY. " + HelpFooter(),
	WikiAnchor: "err-network-001",
}

// ── Permission errors (ERR-PERMISSION-*) ────────────────────────────────────

// ErrPermissionDenied is returned when a file or directory operation is rejected by the OS.
var ErrPermissionDenied = &Message{
	Code:       "ERR-PERMISSION-001",
	What:       "Permission denied",
	Why:        "The current user does not have the required access rights.",
	Fix:        "Check file/directory ownership with `ls -la`. Change permissions or run as the correct user. " + HelpFooter(),
	WikiAnchor: "err-permission-001",
}

// ── Port conflict errors (ERR-PORT-*) ────────────────────────────────────────

// ErrPortConflict is returned when a required port is already in use.
// Call .WithProcess("<name>") to fill in the process placeholder.
var ErrPortConflict = &Message{
	Code:       "ERR-PORT-001",
	What:       "Port already in use by <process>",
	Why:        "The port required by nself is occupied by another process.",
	Fix:        "Stop <process> or set the port override in .env.local (e.g. POSTGRES_PORT=5433). " + HelpFooter(),
	WikiAnchor: "err-port-001",
}

// ErrEnvMalformed is returned when an .env file cannot be parsed.
var ErrEnvMalformed = &Message{
	Code:       "ERR-CONFIG-001",
	What:       ".env file is malformed",
	Why:        "The environment file contains a syntax error or an invalid key.",
	Fix:        "Each line must be KEY=value. Remove blank lines, quotes around keys, or spaces around =. " + HelpFooter(),
	WikiAnchor: "err-config-001",
}

// New constructs a formatted error string directly (when the caller does not need
// the struct, e.g. for fmt.Errorf wrapping).
func New(code, what, why, fix string) error {
	return &Message{Code: code, What: what, Why: why, Fix: fix}
}
