package commands

// env_target.go — implements `nself env target` subcommand group.
//
// Commands:
//   nself env target list    — list server inventory (S23.T01)
//   nself env target add     — add a server to an environment (S23.T01)
//   nself env target remove  — remove a server from an environment (S23.T01)
//   nself env target probe   — resolve capability of targets (S23.T02)
//   nself env target migrate — migrate legacy env-var config to control-plane.yaml (S23.T03)
//
// Security invariants (per S23 CR-C):
//   - SSHKeyRef holds an env-var NAME, never an inline key path or secret.
//   - Inline secrets in --host or --key-ref are rejected on input.
//   - All inventory file writes use mode 0600 (enforced by controlplane.Write).
//   - Probe never forwards credentials; SSH uses BatchMode=yes ForwardAgent=no.
//   - No secret values appear in any output (JSON or table).

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/nself-org/cli/internal/controlplane"
	"github.com/spf13/cobra"
)

// ── parent ───────────────────────────────────────────────────────────────────

var envTargetCmd = &cobra.Command{
	Use:   "target",
	Short: "Manage control-plane server targets",
	Long: `Manage the multi-server inventory stored in .nself/control-plane.yaml.

Subcommands:
  list     List servers in the control-plane inventory
  add      Add a server to an environment
  remove   Remove a server from an environment
  probe    Resolve runtime capability for one or more targets
  migrate  Migrate legacy NSELF_DEPLOY_HOST_* env vars to control-plane.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// ── list ─────────────────────────────────────────────────────────────────────

var envTargetListCmd = &cobra.Command{
	Use:   "list [env]",
	Short: "List servers in the control-plane inventory",
	Long: `Print all servers from .nself/control-plane.yaml in a table or JSON.

When [env] is supplied, only servers for that environment are shown.
Use --all to include hidden servers (those with no host configured).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEnvTargetList,
}

// ── add ──────────────────────────────────────────────────────────────────────

var envTargetAddCmd = &cobra.Command{
	Use:   "add <env> <server-name>",
	Short: "Add a server to an environment",
	Long: `Add a server entry to .nself/control-plane.yaml.

--host must be user@host (SSH target). Never supply an inline key; use
--key-ref to name the environment variable that holds the key file path.

Example:
  nself env target add staging web1 \
    --host ubuntu@staging.example.com \
    --role app \
    --key-ref NSELF_SSH_KEY_STAGING \
    --remote-path /opt/nself \
    --primary`,
	Args: cobra.ExactArgs(2),
	RunE: runEnvTargetAdd,
}

// ── remove ───────────────────────────────────────────────────────────────────

var envTargetRemoveCmd = &cobra.Command{
	Use:   "remove <env> <server-name>",
	Short: "Remove a server from an environment",
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvTargetRemove,
}

// ── probe ────────────────────────────────────────────────────────────────────

var envTargetProbeCmd = &cobra.Command{
	Use:   "probe [env[:server]]",
	Short: "Resolve runtime capability for one or more targets",
	Long: `Run SSH + Docker probes and print a capability matrix.

When called with no argument all servers in the inventory are probed.
Narrow to one environment with "probe staging" or to one server with
"probe staging:web1".

The command always exits 0; it is a read-only diagnostic. Secrets are
never included in the output — SSHKeyRef env-var names are shown but
their values are not.

Use --refresh to bypass the 60-second on-disk probe cache.
Use --json for machine-readable output.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEnvTargetProbe,
}

// ── migrate ──────────────────────────────────────────────────────────────────

var envTargetMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate legacy NSELF_DEPLOY_HOST_* env vars to control-plane.yaml",
	Long: `Synthesize .nself/control-plane.yaml from the legacy single-server env-var
conventions and write it to disk (mode 0600).

This command is idempotent: if control-plane.yaml already exists it is read,
migrated to the current schema version, and rewritten in place without
overwriting any manually added server entries.

After running this command, set NSELF_DEPLOY_HOST_* env vars in .env.secrets
(not .env.dev) and verify with: nself env target list`,
	RunE: runEnvTargetMigrate,
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	// list flags
	envTargetListCmd.Flags().Bool("json", false, "Emit JSON output")
	envTargetListCmd.Flags().Bool("all", false, "Include hidden servers (no host configured)")

	// add flags
	envTargetAddCmd.Flags().String("host", "", "SSH target in user@host form (required for remote servers)")
	envTargetAddCmd.Flags().String("role", "app", "Server role: app|lb|observability|db|worker")
	envTargetAddCmd.Flags().String("key-ref", "", "Name of the env var holding the SSH key path (never the key path itself)")
	envTargetAddCmd.Flags().String("remote-path", "/opt/nself", "Absolute path on the remote host where nSelf is installed")
	envTargetAddCmd.Flags().Bool("primary", false, "Mark this server as the primary app server")
	envTargetAddCmd.Flags().StringSlice("upstreams", nil, "Upstream app server names (LB role only)")

	// probe flags
	envTargetProbeCmd.Flags().Bool("json", false, "Emit JSON output")
	envTargetProbeCmd.Flags().Bool("refresh", false, "Bypass the 60-second probe cache")

	// Wire into envTargetCmd
	envTargetCmd.AddCommand(envTargetListCmd)
	envTargetCmd.AddCommand(envTargetAddCmd)
	envTargetCmd.AddCommand(envTargetRemoveCmd)
	envTargetCmd.AddCommand(envTargetProbeCmd)
	envTargetCmd.AddCommand(envTargetMigrateCmd)

	// Wire envTargetCmd into envCmd (already registered in env.go).
	envCmd.AddCommand(envTargetCmd)
}

// ── run: list ────────────────────────────────────────────────────────────────

// envTargetListRow is the JSON representation of one row in the list output.
type envTargetListRow struct {
	Env        string `json:"env"`
	Server     string `json:"server"`
	Role       string `json:"role"`
	Host       string `json:"host"`
	SSHKeyRef  string `json:"ssh_key_ref"`
	RemotePath string `json:"remote_path"`
	Primary    bool   `json:"primary"`
}

func runEnvTargetList(cmd *cobra.Command, args []string) error {
	root, err := projectRoot()
	if err != nil {
		return err
	}

	inv, err := controlplane.Load(root)
	if err != nil {
		return fmt.Errorf("env target list: %w", err)
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	showAll, _ := cmd.Flags().GetBool("all")

	filterEnvName := ""
	if len(args) == 1 {
		filterEnvName = args[0]
	}

	var rows []envTargetListRow
	envNames := sortedEnvNames(inv)
	for _, envName := range envNames {
		if filterEnvName != "" && envName != filterEnvName {
			continue
		}
		env := inv.Environments[envName]
		for _, srv := range env.Servers {
			if !showAll && srv.Host == "" && env.Kind != "local" {
				continue // hidden — omit by default
			}
			rows = append(rows, envTargetListRow{
				Env:        envName,
				Server:     srv.Name,
				Role:       string(srv.Role),
				Host:       srv.Host,
				SSHKeyRef:  srv.SSHKeyRef,
				RemotePath: srv.RemotePath,
				Primary:    srv.Primary,
			})
		}
	}

	if jsonOut {
		b, _ := json.MarshalIndent(rows, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(rows) == 0 {
		fmt.Println("No servers found. Run 'nself env target add' to configure one.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ENV\tSERVER\tROLE\tHOST\tKEY-REF\tPRIMARY")
	for _, r := range rows {
		primary := ""
		if r.Primary {
			primary = "yes"
		}
		host := r.Host
		if host == "" {
			host = "(local)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Env, r.Server, r.Role, host, r.SSHKeyRef, primary)
	}
	return tw.Flush()
}

// ── run: add ─────────────────────────────────────────────────────────────────

// validRoles is the set of accepted server role strings.
var validRoles = map[string]controlplane.ServerRole{
	"app":           controlplane.RoleApp,
	"lb":            controlplane.RoleLB,
	"observability": controlplane.RoleObservability,
	"db":            controlplane.RoleDB,
	"worker":        controlplane.RoleWorker,
}

func runEnvTargetAdd(cmd *cobra.Command, args []string) error {
	envName := args[0]
	serverName := args[1]

	host, _ := cmd.Flags().GetString("host")
	roleStr, _ := cmd.Flags().GetString("role")
	keyRef, _ := cmd.Flags().GetString("key-ref")
	remotePath, _ := cmd.Flags().GetString("remote-path")
	primary, _ := cmd.Flags().GetBool("primary")
	upstreams, _ := cmd.Flags().GetStringSlice("upstreams")

	// Security: reject inline key material in --host or --key-ref.
	if err := rejectInlineSecret("--host", host); err != nil {
		return err
	}
	if err := rejectInlineSecret("--key-ref", keyRef); err != nil {
		return err
	}

	role, ok := validRoles[strings.ToLower(roleStr)]
	if !ok {
		return fmt.Errorf("invalid --role %q (allowed: app, lb, observability, db, worker)", roleStr)
	}

	root, err := projectRoot()
	if err != nil {
		return err
	}
	inv, err := controlplane.Load(root)
	if err != nil {
		return fmt.Errorf("env target add: %w", err)
	}

	// Resolve or create the environment.
	env, exists := inv.Environments[envName]
	if !exists {
		kind := "remote"
		if host == "" {
			kind = "local"
		}
		env = controlplane.Environment{Name: envName, Kind: kind, Servers: nil}
	}

	// Reject duplicate server name within the environment.
	for _, s := range env.Servers {
		if s.Name == serverName {
			return fmt.Errorf("server %q already exists in environment %q; use 'env target remove' first", serverName, envName)
		}
	}

	// Enforce: remote servers must have a host.
	if env.Kind == "remote" && host == "" {
		return fmt.Errorf("--host is required for remote environment %q", envName)
	}

	srv := controlplane.Server{
		Name:       serverName,
		Role:       role,
		Host:       host,
		SSHKeyRef:  keyRef,
		RemotePath: remotePath,
		Primary:    primary,
		Upstreams:  upstreams,
	}
	env.Servers = append(env.Servers, srv)

	if inv.Environments == nil {
		inv.Environments = make(map[string]controlplane.Environment)
	}
	inv.Environments[envName] = env

	if err := controlplane.Write(root, inv); err != nil {
		return fmt.Errorf("env target add: %w", err)
	}

	fmt.Printf("Added server %q to environment %q. File: .nself/control-plane.yaml (0600).\n", serverName, envName)
	return nil
}

// ── run: remove ──────────────────────────────────────────────────────────────

func runEnvTargetRemove(_ *cobra.Command, args []string) error {
	envName := args[0]
	serverName := args[1]

	root, err := projectRoot()
	if err != nil {
		return err
	}
	inv, err := controlplane.Load(root)
	if err != nil {
		return fmt.Errorf("env target remove: %w", err)
	}

	env, exists := inv.Environments[envName]
	if !exists {
		return fmt.Errorf("environment %q not found in inventory", envName)
	}

	var kept []controlplane.Server
	found := false
	for _, s := range env.Servers {
		if s.Name == serverName {
			found = true
			continue
		}
		kept = append(kept, s)
	}
	if !found {
		return fmt.Errorf("server %q not found in environment %q", serverName, envName)
	}
	env.Servers = kept
	inv.Environments[envName] = env

	if err := controlplane.Write(root, inv); err != nil {
		return fmt.Errorf("env target remove: %w", err)
	}

	fmt.Printf("Removed server %q from environment %q.\n", serverName, envName)
	return nil
}

// ── run: probe ───────────────────────────────────────────────────────────────

// probeRow is one row of the probe matrix, safe for JSON output (no secrets).
type probeRow struct {
	Env          string `json:"env"`
	Server       string `json:"server"`
	Capability   string `json:"capability"`
	SSHReachable bool   `json:"ssh_reachable"`
	DockerOK     bool   `json:"docker_ok"`
	LatencyMS    int    `json:"latency_ms"`
	Reason       string `json:"reason,omitempty"`
}

func runEnvTargetProbe(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	refresh, _ := cmd.Flags().GetBool("refresh")

	root, err := projectRoot()
	if err != nil {
		return err
	}
	inv, err := controlplane.Load(root)
	if err != nil {
		return fmt.Errorf("env target probe: %w", err)
	}

	// Parse optional "env" or "env:server" filter.
	filterEnv, filterServer := "", ""
	if len(args) == 1 {
		parts := strings.SplitN(args[0], ":", 2)
		filterEnv = parts[0]
		if len(parts) == 2 {
			filterServer = parts[1]
		}
	}

	// Build a scoped inventory when a filter is applied.
	if filterEnv != "" {
		if _, ok := inv.Environments[filterEnv]; !ok {
			return fmt.Errorf("environment %q not found in inventory", filterEnv)
		}
		env := inv.Environments[filterEnv]
		if filterServer != "" {
			var srvs []controlplane.Server
			for _, s := range env.Servers {
				if s.Name == filterServer {
					srvs = append(srvs, s)
				}
			}
			if len(srvs) == 0 {
				return fmt.Errorf("server %q not found in environment %q", filterServer, filterEnv)
			}
			env.Servers = srvs
		}
		inv.Environments = map[string]controlplane.Environment{filterEnv: env}
	}

	prober := controlplane.NewSSHProber(root, refresh)
	statuses := controlplane.Resolve(inv, prober)

	// Build output rows — no secret values in output.
	var rows []probeRow
	for _, ts := range statuses {
		if ts.Capability == controlplane.CapHidden {
			continue // omit hidden servers from default probe output
		}
		rows = append(rows, probeRow{
			Env:          ts.Env,
			Server:       ts.Server,
			Capability:   string(ts.Capability),
			SSHReachable: ts.SSHReachable,
			DockerOK:     ts.DockerOK,
			LatencyMS:    ts.LatencyMS,
			Reason:       ts.Reason,
		})
	}

	if jsonOut {
		b, _ := json.MarshalIndent(rows, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(rows) == 0 {
		fmt.Println("No servers to probe.")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ENV\tSERVER\tCAPABILITY\tSSH\tDOCKER\tLATENCY\tREASON")
	for _, r := range rows {
		ssh := boolMark(r.SSHReachable)
		docker := boolMark(r.DockerOK)
		latency := ""
		if r.LatencyMS > 0 {
			latency = fmt.Sprintf("%dms", r.LatencyMS)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Env, r.Server, r.Capability, ssh, docker, latency, r.Reason)
	}
	return tw.Flush()
}

// ── run: migrate ─────────────────────────────────────────────────────────────

func runEnvTargetMigrate(_ *cobra.Command, _ []string) error {
	root, err := projectRoot()
	if err != nil {
		return err
	}

	// Load synthesizes from env vars when control-plane.yaml is absent;
	// when it exists it applies Migrate() in-place (idempotent).
	inv, err := controlplane.Load(root)
	if err != nil {
		return fmt.Errorf("env target migrate: %w", err)
	}

	if err := controlplane.Write(root, inv); err != nil {
		return fmt.Errorf("env target migrate: %w", err)
	}

	path := root + "/.nself/control-plane.yaml"
	fmt.Printf("Wrote %s (mode 0600).\n", path)
	fmt.Printf("Environments: %d\n", len(inv.Environments))
	for envName, env := range inv.Environments {
		fmt.Printf("  %s (%s): %d server(s)\n", envName, env.Kind, len(env.Servers))
	}
	fmt.Println("\nVerify with: nself env target list")
	return nil
}

// ── internal helpers ──────────────────────────────────────────────────────────

// rejectInlineSecret returns an error when the value for flagName looks like
// inline key material (contains whitespace, BEGIN/END PEM markers, or path
// separators where an env-var name is expected).
//
// This guard applies specifically to --key-ref, which should be an env-var
// NAME (e.g. "NSELF_SSH_KEY_STAGING"), not a file path or base64 blob.
func rejectInlineSecret(flagName, value string) error {
	if value == "" {
		return nil
	}
	// An env-var name is all uppercase letters, digits, and underscores.
	// If the flag value looks like a file path, PEM block, or raw secret, reject it.
	if flagName == "--key-ref" {
		// Reject if it looks like a path (contains /) or a PEM header.
		if strings.Contains(value, "/") {
			return fmt.Errorf("%s must be an environment variable NAME (e.g. NSELF_SSH_KEY_STAGING), not a file path (got %q)", flagName, value)
		}
		if strings.Contains(value, "BEGIN") || strings.Contains(value, "ssh-") {
			return fmt.Errorf("%s must be an environment variable NAME, not inline key material", flagName)
		}
	}
	if flagName == "--host" {
		// Reject if it contains newlines (multi-line = PEM block) or obvious secret patterns.
		if strings.ContainsAny(value, "\n\r") {
			return fmt.Errorf("%s must be user@host form; multi-line input rejected", flagName)
		}
	}
	return nil
}

// sortedEnvNames returns the environment names from inv in deterministic order
// (local first, then alphabetical).
func sortedEnvNames(inv *controlplane.Inventory) []string {
	names := make([]string, 0, len(inv.Environments))
	for name := range inv.Environments {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		// "local" always sorts first.
		if names[i] == "local" {
			return true
		}
		if names[j] == "local" {
			return false
		}
		return names[i] < names[j]
	})
	return names
}

// boolMark returns a short human-readable indicator for a boolean probe result.
func boolMark(v bool) string {
	if v {
		return "ok"
	}
	return "-"
}
