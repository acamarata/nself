package commands

// sentry_server.go — 'nself sentry-server' command group.
//
// Purpose: end-to-end provisioning of an ops/sentry server. Orchestrates
// Terraform infrastructure provisioning, secrets push, and ops-profile
// deployment in a single workflow. Intended for spinning up ɳSentry servers
// on Hetzner Cloud.
//
// Commands:
//   nself sentry-server provision <project>
//
// Inputs:
//   <project> positional arg (required)
//   --location, --server-type, --dry-run, --host, --key-path flags
// Outputs: provisioned server running the ops docker-compose profile.
// Constraints: terraform, ssh, scp, rsync must be in PATH.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/deploy"
	"github.com/nself-org/cli/internal/infra"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var sentryServerCmd = &cobra.Command{
	Use:   "sentry-server",
	Short: "Provision and manage ɳSentry ops servers",
	Long: `Provision and manage ɳSentry observability/ops servers.

The sentry-server command orchestrates the full lifecycle of an ops server:
Terraform provisioning, secrets push, and ops-profile deployment.

Examples:
  nself sentry-server provision myproject
  nself sentry-server provision myproject --dry-run
  nself sentry-server provision myproject --host nself@1.2.3.4 --key-path ~/.ssh/ops_key`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sentryServerProvisionCmd = &cobra.Command{
	Use:   "provision <project>",
	Short: "Provision an ops/sentry server for a project",
	Long: `Provision a full ɳSentry ops server for the given project.

Steps:
  1. Provision Hetzner server via Terraform (skipped when --host is given)
  2. Wait for SSH to become reachable (up to 3 minutes)
  3. Push secrets (.env.ops or .env) to the remote host via scp
  4. Deploy with the ops profile via rsync + docker compose

Use --host to skip Terraform and redeploy to an existing server.
Use --dry-run to preview steps without executing any real operations.`,
	Args: cobra.ExactArgs(1),
	RunE: runSentryServerProvision,
}

func init() {
	sentryServerProvisionCmd.Flags().String("location", "nbg1", "Hetzner datacenter location (nbg1, fsn1, hel1)")
	sentryServerProvisionCmd.Flags().String("server-type", "cpx21", "Hetzner server type (e.g. cpx21, cx22)")
	sentryServerProvisionCmd.Flags().Bool("dry-run", false, "Print what would happen without executing")
	sentryServerProvisionCmd.Flags().String("host", "", "Skip Terraform and deploy to this existing host (user@host:/path)")
	sentryServerProvisionCmd.Flags().String("key-path", "", "SSH key path (defaults to NSELF_DEPLOY_KEY_PATH env var)")

	sentryServerCmd.AddCommand(sentryServerProvisionCmd)
	RootCmd.AddCommand(sentryServerCmd)
}

// runSentryServerProvision implements 'nself sentry-server provision <project>'.
//
// Purpose:  Full ops-server provisioning pipeline: Terraform → SSH wait →
//
//	secrets push → ops deploy.
//
// Inputs:   project arg, flags (location, server-type, dry-run, host, key-path).
// Outputs:  Provisioned and running ops server.
// Constraints: Steps are sequential; failure at any step returns immediately.
func runSentryServerProvision(cmd *cobra.Command, args []string) error {
	project := args[0]

	location, _ := cmd.Flags().GetString("location")
	serverType, _ := cmd.Flags().GetString("server-type")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	host, _ := cmd.Flags().GetString("host")
	keyPath, _ := cmd.Flags().GetString("key-path")

	// Resolve SSH key from flag → env → default.
	if keyPath == "" {
		keyPath = os.Getenv("NSELF_DEPLOY_KEY_PATH")
	}
	if keyPath == "" {
		home, _ := os.UserHomeDir()
		keyPath = home + "/.ssh/id_ed25519"
	}

	ui.Info(fmt.Sprintf("Provisioning ops/sentry server for project %s...", project))

	// ── Step 1: Terraform (unless --host skips it) ─────────────────────────
	if host == "" {
		if err := runSentryTerraform(cmd.Context(), project, location, serverType, keyPath, dryRun); err != nil {
			return fmt.Errorf("sentry-server provision: terraform: %w", err)
		}
		if dryRun {
			ui.Info("DRY-RUN: would wait for SSH, push secrets, and deploy ops profile")
			return nil
		}
		// After Terraform, the host is read from env (set by the module output or
		// the NSELF_DEPLOY_HOST_OPS env var the operator configures).
		host = os.Getenv("NSELF_DEPLOY_HOST_OPS")
		if host == "" {
			host = os.Getenv("OPS_DEPLOY_HOST")
		}
		if host == "" {
			return fmt.Errorf("sentry-server provision: terraform complete but NSELF_DEPLOY_HOST_OPS is not set; export it and re-run with --host")
		}
	}

	cfg := deploy.SSHConfig{
		Host:    host,
		KeyPath: keyPath,
	}

	// ── Step 2: Wait for SSH ───────────────────────────────────────────────
	if dryRun {
		ui.Info(fmt.Sprintf("DRY-RUN: would wait for SSH on %s (up to 3 minutes)", host))
	} else {
		ui.Info(fmt.Sprintf("Waiting for SSH on %s (up to 3 minutes)...", host))
		if err := waitForSSH(cmd.Context(), host, keyPath, 3*time.Minute); err != nil {
			return fmt.Errorf("sentry-server provision: SSH wait: %w", err)
		}
		ui.Success("SSH reachable")
	}

	// ── Step 3: Push secrets ───────────────────────────────────────────────
	workdir, err := projectRoot()
	if err != nil {
		return fmt.Errorf("sentry-server provision: %w", err)
	}

	pushOpts := deploy.PushSecretsOptions{
		Target:      "ops",
		DryRun:      dryRun,
		ProjectRoot: workdir,
	}
	ui.Info("Pushing secrets to remote host...")
	if err := deploy.PushSecrets(cmd.Context(), cfg, pushOpts); err != nil {
		return fmt.Errorf("sentry-server provision: push secrets: %w", err)
	}

	// ── Step 4: Deploy ops profile ─────────────────────────────────────────
	ui.Info("Deploying ops profile...")
	if dryRun {
		ui.Info(fmt.Sprintf("DRY-RUN: would run ops deploy to %s", host))
	} else {
		if err := runCLISelf(cmd.Context(), workdir, "ops", "deploy"); err != nil {
			return fmt.Errorf("sentry-server provision: ops deploy: %w", err)
		}
	}

	if !dryRun {
		ui.Success(fmt.Sprintf("ɳSentry ops server for project %q is live at %s", project, host))
	}
	return nil
}

// runSentryTerraform runs terraform apply for the Hetzner ops module.
//
// Purpose:  Provision Hetzner infrastructure for the ops server.
// Inputs:   project name, location, serverType, keyPath (SSH private key path), dryRun flag.
// Outputs:  Terraform apply executed against terraform/modules/hetzner.
// Constraints: HETZNER_NSELF_TOKEN or HCLOUD_TOKEN must be set.
func runSentryTerraform(ctx context.Context, project, location, serverType, keyPath string, dryRun bool) error {
	// Passthrough: HETZNER_NSELF_TOKEN → HCLOUD_TOKEN.
	if tok := os.Getenv("HETZNER_NSELF_TOKEN"); tok != "" && os.Getenv("HCLOUD_TOKEN") == "" {
		os.Setenv("HCLOUD_TOKEN", tok)
	}

	// Read the public key for the deploy user's authorized_keys.
	pubKeyPath := keyPath + ".pub"
	deployPubKey := ""
	if data, err := os.ReadFile(pubKeyPath); err == nil {
		deployPubKey = strings.TrimSpace(string(data))
	}

	vars := map[string]string{}
	if location != "" {
		vars["location"] = location
	}
	if serverType != "" {
		vars["server_type"] = serverType
	}
	if project != "" {
		vars["project"] = project
	}
	if deployPubKey != "" {
		vars["deploy_ssh_pubkey"] = deployPubKey
	}

	opts := infra.ApplyOptions{
		PlanOptions: infra.PlanOptions{
			Provider:   infra.ProviderHetzner,
			Domain:     project,
			ModulesDir: "terraform/modules/hetzner",
		},
		AutoApprove: true,
		Vars:        vars,
	}

	if dryRun {
		ui.Info(fmt.Sprintf("DRY-RUN: would run terraform apply (provider=hetzner, location=%s, server-type=%s)", location, serverType))
		return nil
	}

	ui.Info(fmt.Sprintf("Running terraform apply (location=%s, server-type=%s)...", location, serverType))
	return infra.Apply(ctx, opts)
}

// waitForSSH polls the given host until SSH is reachable or the timeout expires.
//
// Purpose:  Block until a freshly provisioned server accepts SSH connections.
// Inputs:   host in user@host or user@host:/path format, keyPath, timeout.
// Outputs:  nil when SSH is ready; error after timeout.
// Constraints: polls every 10 seconds; requires ssh binary in PATH.
func waitForSSH(ctx context.Context, host, keyPath string, timeout time.Duration) error {
	// Extract the SSH target (strip any :/path suffix).
	sshTarget := host
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			sshTarget = host[:i]
			break
		}
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		args := []string{
			"-i", keyPath,
			"-o", "ConnectTimeout=5",
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "ForwardAgent=no",
			"-o", "BatchMode=yes",
			sshTarget,
			"true",
		}
		c := exec.CommandContext(ctx, "ssh", args...)
		c.Env = os.Environ()
		if err := c.Run(); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for SSH")
		case <-time.After(10 * time.Second):
		}
	}
	return fmt.Errorf("SSH on %s did not become reachable within %s", sshTarget, timeout)
}
