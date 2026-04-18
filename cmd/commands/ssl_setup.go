package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var sslSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up SSL certificates via DNS-01 challenge",
	Long: `Provision SSL certificates using certbot with DNS-01 validation.

Supports wildcard certificates for *.domain when --wildcard is specified.
Providers: cloudflare (default), route53, digitalocean, custom.

Example:
  nself ssl setup --provider cloudflare --wildcard
  nself ssl setup --provider route53 --domain api.example.com`,
	RunE: runSSLSetup,
}

var sslAddCmd = &cobra.Command{
	Use:   "add <domain>",
	Short: "Provision an SSL certificate for a single domain",
	Long: `Provision an SSL certificate for a custom domain using certbot.

After provisioning, generates an nginx server block and reloads nginx.

Example:
  nself ssl add custom.example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runSSLAdd,
}

func init() {
	sslSetupCmd.Flags().String("provider", "cloudflare", "DNS provider (cloudflare, route53, digitalocean, custom)")
	sslSetupCmd.Flags().Bool("wildcard", false, "Request wildcard certificate (*.domain)")
	sslSetupCmd.Flags().String("email", "", "Email for Let's Encrypt registration")
	sslSetupCmd.Flags().Bool("staging", false, "Use Let's Encrypt staging environment")
	sslSetupCmd.Flags().Bool("install-cron", false, "Install a systemd timer for automatic certificate renewal (Linux only)")
	sslCmd.AddCommand(sslSetupCmd)
	sslCmd.AddCommand(sslAddCmd)
}

// certbotProviderPlugin maps provider names to certbot DNS plugin packages.
var certbotProviderPlugin = map[string]string{
	"cloudflare":   "certbot-dns-cloudflare",
	"route53":      "certbot-dns-route53",
	"digitalocean": "certbot-dns-digitalocean",
}

// certbotProviderFlag maps provider names to certbot --dns-* flag names.
var certbotProviderFlag = map[string]string{
	"cloudflare":   "dns-cloudflare",
	"route53":      "dns-route53",
	"digitalocean": "dns-digitalocean",
}

func runSSLSetup(cmd *cobra.Command, args []string) error {
	provider, _ := cmd.Flags().GetString("provider")
	wildcard, _ := cmd.Flags().GetBool("wildcard")
	email, _ := cmd.Flags().GetString("email")
	staging, _ := cmd.Flags().GetBool("staging")
	installCron, _ := cmd.Flags().GetBool("install-cron")

	ui.CommandHeader("nself ssl setup", "SSL certificate provisioning via DNS-01")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	workdir, err := config.FindNSelfRoot(cwd)
	if err != nil {
		return fmt.Errorf("no nself project found: run 'nself init' first")
	}

	cfg, err := config.Load(workdir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, err := exec.LookPath("certbot"); err != nil {
		return fmt.Errorf("certbot not found in PATH — install certbot and the DNS plugin first")
	}

	domain := cfg.BaseDomain
	if domain == "" || domain == "localhost" {
		return fmt.Errorf("BASE_DOMAIN must be set to a real domain for SSL setup")
	}

	if email == "" {
		email = cfg.AdminEmail
	}
	if email == "" {
		return fmt.Errorf("--email is required (or set ADMIN_EMAIL in .env)")
	}

	// Build certbot command.
	certArgs := []string{
		"certonly",
		"--non-interactive",
		"--agree-tos",
		"--email", email,
	}

	// Provider-specific flags.
	dnsFlag, ok := certbotProviderFlag[provider]
	if !ok {
		return fmt.Errorf("unsupported SSL provider %q (supported: cloudflare, route53, digitalocean)", provider)
	}
	certArgs = append(certArgs, "--"+dnsFlag)

	// Provider credentials file (certbot convention).
	credFile := fmt.Sprintf("/etc/letsencrypt/%s.ini", provider)
	if provider == "cloudflare" {
		certArgs = append(certArgs, "--dns-cloudflare-credentials", credFile)
	} else if provider == "route53" {
		// route53 uses AWS env vars, no credentials flag needed.
		_ = credFile
	} else if provider == "digitalocean" {
		certArgs = append(certArgs, "--dns-digitalocean-credentials", credFile)
	}

	// Domain(s).
	if wildcard {
		certArgs = append(certArgs, "-d", "*."+domain, "-d", domain)
	} else {
		certArgs = append(certArgs, "-d", domain)
		// Add standard subdomains.
		for _, sub := range []string{"api", "auth"} {
			certArgs = append(certArgs, "-d", sub+"."+domain)
		}
	}

	if staging {
		certArgs = append(certArgs, "--staging")
	}

	// Certificate path.
	certArgs = append(certArgs,
		"--cert-path", fmt.Sprintf("/etc/nginx/ssl/%s/fullchain.pem", domain),
		"--key-path", fmt.Sprintf("/etc/nginx/ssl/%s/privkey.pem", domain),
	)

	ui.Info(fmt.Sprintf("Running: certbot %s", strings.Join(certArgs, " ")))

	certCmd := exec.Command("certbot", certArgs...)
	certCmd.Stdout = os.Stdout
	certCmd.Stderr = os.Stderr
	if err := certCmd.Run(); err != nil {
		return fmt.Errorf("certbot failed: %w", err)
	}

	// Set cert file permissions.
	for _, f := range []string{"fullchain.pem", "privkey.pem"} {
		path := fmt.Sprintf("/etc/nginx/ssl/%s/%s", domain, f)
		if chErr := os.Chmod(path, 0600); chErr != nil {
			ui.Warn(fmt.Sprintf("Could not set permissions on %s: %v", path, chErr))
		}
	}

	// Reload nginx.
	reloadCmd := exec.Command("docker", "compose", "exec", "nginx", "nginx", "-s", "reload")
	reloadCmd.Dir = workdir
	reloadCmd.Stdout = os.Stdout
	reloadCmd.Stderr = os.Stderr
	if err := reloadCmd.Run(); err != nil {
		ui.Warn(fmt.Sprintf("Nginx reload failed: %v (container may not be running)", err))
	}

	ui.Success("SSL setup complete.")

	if installCron {
		if err := installSSLRenewalCron(workdir); err != nil {
			ui.Warn(fmt.Sprintf("Could not install renewal timer: %v", err))
			ui.Info("Add manually to crontab: 30 3 * * * certbot renew --quiet")
		} else {
			ui.Success("SSL renewal systemd timer installed (nself-ssl-renew.timer).")
		}
	} else {
		ui.Info("To enable auto-renewal: nself ssl setup --install-cron")
		ui.Info("Or add manually to crontab: 30 3 * * * certbot renew --quiet")
	}
	return nil
}

func runSSLAdd(cmd *cobra.Command, args []string) error {
	domain := args[0]

	ui.CommandHeader("nself ssl add", fmt.Sprintf("Provision certificate for %s", domain))

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	workdir, err := config.FindNSelfRoot(cwd)
	if err != nil {
		return fmt.Errorf("no nself project found: run 'nself init' first")
	}

	cfg, err := config.Load(workdir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, err := exec.LookPath("certbot"); err != nil {
		return fmt.Errorf("certbot not found in PATH")
	}

	email := cfg.AdminEmail
	if email == "" {
		return fmt.Errorf("ADMIN_EMAIL must be set in .env for certificate registration")
	}

	// Use HTTP-01 challenge for single domain adds (simpler, no DNS provider needed).
	certArgs := []string{
		"certonly",
		"--non-interactive",
		"--agree-tos",
		"--email", email,
		"--webroot",
		"--webroot-path", "/var/www/certbot",
		"-d", domain,
	}

	ui.Info(fmt.Sprintf("Provisioning certificate for %s...", domain))
	certCmd := exec.Command("certbot", certArgs...)
	certCmd.Stdout = os.Stdout
	certCmd.Stderr = os.Stderr
	if err := certCmd.Run(); err != nil {
		return fmt.Errorf("certbot failed: %w", err)
	}

	// Validate and reload nginx.
	testCmd := exec.Command("docker", "compose", "exec", "nginx", "nginx", "-t")
	testCmd.Dir = workdir
	if out, testErr := testCmd.CombinedOutput(); testErr != nil {
		return fmt.Errorf("nginx config test failed: %s", string(out))
	}

	reloadCmd := exec.Command("docker", "compose", "exec", "nginx", "nginx", "-s", "reload")
	reloadCmd.Dir = workdir
	reloadCmd.Stdout = os.Stdout
	reloadCmd.Stderr = os.Stderr
	if err := reloadCmd.Run(); err != nil {
		ui.Warn(fmt.Sprintf("Nginx reload failed: %v", err))
	}

	ui.Success(fmt.Sprintf("Certificate provisioned for %s.", domain))
	return nil
}

// sslRenewalServiceContent is the systemd service unit for certbot renewal.
const sslRenewalServiceContent = `[Unit]
Description=nself SSL certificate renewal
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/bin/certbot renew --quiet --post-hook "docker compose exec nginx nginx -s reload"
`

// sslRenewalTimerContent is the systemd timer unit that triggers renewal twice daily.
const sslRenewalTimerContent = `[Unit]
Description=nself SSL certificate renewal timer

[Timer]
OnCalendar=*-*-* 03:30:00
RandomizedDelaySec=3600
Persistent=true

[Install]
WantedBy=timers.target
`

// installSSLRenewalCron installs automatic Let's Encrypt certificate renewal.
// On Linux with systemd, creates and enables a systemd timer unit.
// Returns an error with crontab fallback instructions if installation fails.
func installSSLRenewalCron(_ string) error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemd not available — add to crontab: 30 3 * * * certbot renew --quiet")
	}
	return installSSLRenewalSystemd()
}

// installSSLRenewalSystemd writes and enables the nself-ssl-renew systemd timer.
func installSSLRenewalSystemd() error {
	const unitDir = "/etc/systemd/system"

	servicePath := filepath.Join(unitDir, "nself-ssl-renew.service")
	if err := os.WriteFile(servicePath, []byte(sslRenewalServiceContent), 0644); err != nil {
		return fmt.Errorf("writing service unit: %w", err)
	}

	timerPath := filepath.Join(unitDir, "nself-ssl-renew.timer")
	if err := os.WriteFile(timerPath, []byte(sslRenewalTimerContent), 0644); err != nil {
		return fmt.Errorf("writing timer unit: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := exec.CommandContext(ctx, "systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	if err := exec.CommandContext(ctx2, "systemctl", "enable", "--now", "nself-ssl-renew.timer").Run(); err != nil {
		return fmt.Errorf("enable nself-ssl-renew.timer: %w", err)
	}

	return nil
}
