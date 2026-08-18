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
	sslAddCmd.Flags().String("upstream", "", "Backend service to proxy to (host:port), e.g. app:3000")
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

	// No --cert-path/--key-path: certbot ignores them for `certonly` and always
	// writes to /etc/letsencrypt/live/<domain>/. They previously pointed at
	// /etc/nginx/ssl/<dotted-domain>/, which is a CONTAINER path being handed to
	// a certbot process running on the HOST, so it neither placed nor found
	// anything. The certificate is installed explicitly below instead.
	ui.Info(fmt.Sprintf("Running: certbot %s", strings.Join(certArgs, " ")))

	certCmd := exec.Command("certbot", certArgs...)
	certCmd.Stdout = os.Stdout
	certCmd.Stderr = os.Stderr
	if err := certCmd.Run(); err != nil {
		return fmt.Errorf("certbot failed: %w", err)
	}

	// Install into the tree compose mounts at /etc/nginx/ssl, using the same
	// certificates/<domain-safe> layout internal/ssl and `ssl add` use.
	// For a wildcard request the lineage is still named after the bare domain.
	certDir := filepath.Join(workdir, "ssl", "certificates", domainToFilesafe(domain))
	if err := os.MkdirAll(certDir, 0750); err != nil {
		return fmt.Errorf("creating cert directory: %w", err)
	}
	if err := installIssuedCert(domain, certDir); err != nil {
		return fmt.Errorf("installing certificate for %s: %w", domain, err)
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
	upstream, _ := cmd.Flags().GetString("upstream")

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

	// Create the cert output directory so certbot can write into it.
	//
	// This must match the layout the nginx container actually sees. Compose
	// mounts "./ssl:/etc/nginx/ssl:ro" and internal/ssl writes certificates to
	// ssl/certificates/<dir>, where <dir> is the domain with dots replaced by
	// dashes. Writing to ssl/<dotted-domain> instead produced a cert on disk
	// that the generated server block could never reference, so `ssl add`
	// reported success while nginx kept serving the self-signed wildcard.
	domainSafe := domainToFilesafe(domain)
	certDir := filepath.Join(workdir, "ssl", "certificates", domainSafe)
	if err := os.MkdirAll(certDir, 0750); err != nil {
		return fmt.Errorf("creating cert directory: %w", err)
	}

	// Use HTTP-01 challenge for single domain adds (simpler, no DNS provider needed).
	//
	// --cert-path/--key-path are deliberately NOT passed: certbot ignores them
	// for `certonly` (they apply to `install`), which is why earlier versions
	// appeared to place the certificate where nginx expected it while certbot
	// actually wrote to /etc/letsencrypt/live/<domain>/ and nothing ever bridged
	// the two. We copy explicitly below instead.
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

	// Install the issued certificate into the tree nginx actually reads.
	// Without this the cert exists only under /etc/letsencrypt and nginx keeps
	// serving the self-signed wildcard, which is a silent no-op from the
	// operator's point of view.
	if err := installIssuedCert(domain, certDir); err != nil {
		return fmt.Errorf("installing certificate for %s: %w", domain, err)
	}

	// Write the nginx server block for this custom domain.
	if err := writeCustomDomainConf(workdir, domain, upstream); err != nil {
		return fmt.Errorf("writing nginx conf: %w", err)
	}

	// Validate nginx config before reloading.
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

	ui.Info(fmt.Sprintf("Custom domain conf written to %s",
		filepath.Join(workdir, "nginx", "conf.d", fmt.Sprintf("custom-%s.conf", domainSafe))))
	ui.Success(fmt.Sprintf("Certificate provisioned for %s.", domain))
	return nil
}

// domainToFilesafe replaces dots and colons with dashes so a domain name is
// safe to embed in a filename. e.g. "my.custom.com" -> "my-custom-com".
func domainToFilesafe(domain string) string {
	r := strings.NewReplacer(".", "-", ":", "-")
	return r.Replace(domain)
}

// letsEncryptLiveDir is where certbot stores issued certificates. Declared as a
// variable so tests can point it at a temp dir.
var letsEncryptLiveDir = "/etc/letsencrypt/live"

// installIssuedCert copies the certificate certbot just issued for domain into
// certDir, which is the directory the generated nginx server block references.
//
// certbot `certonly` always writes to /etc/letsencrypt/live/<domain>/ and
// ignores --cert-path/--key-path, so without this step the certificate is
// issued but invisible to nginx. Files are written 0600 because privkey.pem is
// a private key; the containing directory is created by the caller.
//
// Copies rather than symlinks: /etc/letsencrypt/live entries are themselves
// symlinks into ../archive, and a bind-mounted symlink chain does not resolve
// inside the nginx container.
func installIssuedCert(domain, certDir string) error {
	liveDir := filepath.Join(letsEncryptLiveDir, domain)

	for _, name := range []string{"fullchain.pem", "privkey.pem"} {
		src := filepath.Join(liveDir, name)
		data, err := os.ReadFile(src) //nolint:gosec // path derived from the validated domain
		if err != nil {
			return fmt.Errorf("reading %s (did certbot succeed?): %w", src, err)
		}
		dst := filepath.Join(certDir, name)
		if err := os.WriteFile(dst, data, 0600); err != nil {
			return fmt.Errorf("writing %s: %w", dst, err)
		}
	}
	return nil
}

// writeCustomDomainConf generates an nginx server block for domain and writes
// it to nginx/conf.d/custom-{domain-safe}.conf inside workdir.
// When upstream is non-empty the server block proxy_passes to it; otherwise it
// returns a 200 informational response until --upstream is configured.
//
// server_name keeps the real dotted domain, but ssl_certificate must point at
// ssl/certificates/{domain-safe} — the layout internal/ssl writes and that
// compose mounts at /etc/nginx/ssl. Using the dotted domain here produced a
// path nginx could not resolve even once the conf was in place.
func writeCustomDomainConf(workdir, domain, upstream string) error {
	confDir := filepath.Join(workdir, "nginx", "conf.d")
	if err := os.MkdirAll(confDir, 0750); err != nil {
		return fmt.Errorf("creating nginx/conf.d: %w", err)
	}

	domainSafe := domainToFilesafe(domain)
	confPath := filepath.Join(confDir, fmt.Sprintf("custom-%s.conf", domainSafe))

	var locationBlock string
	if upstream != "" {
		locationBlock = fmt.Sprintf(`    location / {
        proxy_pass http://%s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }`, upstream)
	} else {
		locationBlock = `    location / {
        return 200 'nself custom domain — configure --upstream to proxy to a backend service';
        add_header Content-Type text/plain;
    }`
	}

	conf := fmt.Sprintf(`# Generated by nself ssl add
server {
    listen 80;
    server_name %s;

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    http2 on;
    server_name %s;

    ssl_certificate     /etc/nginx/ssl/certificates/%s/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/certificates/%s/privkey.pem;

    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;

%s
}
`, domain, domain, domainSafe, domainSafe, locationBlock)

	if err := os.WriteFile(confPath, []byte(conf), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", confPath, err)
	}
	return nil
}

// sslRenewalServiceUnit builds the systemd service unit for certbot renewal,
// rooted at workdir.
//
// Two things this has to get right, both of which were previously wrong:
//
//   - WorkingDirectory. The reload runs `docker compose exec`, which can only
//     find the project's compose file from the project root. Without it the
//     post-hook silently failed and nginx kept the old certificate loaded.
//
//   - deploy-hook. `certbot renew` refreshes /etc/letsencrypt/live/<domain>/
//     but nothing copied the result into ssl/certificates/<domain-safe>/, so a
//     renewed certificate never reached nginx and the served cert would simply
//     expire ~90 days after issue. The hook mirrors installIssuedCert: it runs
//     only for lineages that actually renewed, derives the same dash-safe
//     directory name, and installs both files 0600.
func sslRenewalServiceUnit(workdir string) string {
	deployHook := fmt.Sprintf(
		`safe=$(basename "$RENEWED_LINEAGE" | tr '.:' '--'); `+
			`dest=%s/ssl/certificates/$safe; `+
			`mkdir -p "$dest" && `+
			`install -m 600 "$RENEWED_LINEAGE/fullchain.pem" "$RENEWED_LINEAGE/privkey.pem" "$dest/"`,
		workdir)

	return fmt.Sprintf(`[Unit]
Description=nself SSL certificate renewal
After=network.target

[Service]
Type=oneshot
WorkingDirectory=%s
ExecStart=/usr/bin/certbot renew --quiet --deploy-hook "%s" --post-hook "docker compose exec nginx nginx -s reload"
`, workdir, deployHook)
}

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
func installSSLRenewalCron(workdir string) error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemd not available — add to crontab: 30 3 * * * certbot renew --quiet")
	}
	return installSSLRenewalSystemd(workdir)
}

// installSSLRenewalSystemd writes and enables the nself-ssl-renew systemd timer.
func installSSLRenewalSystemd(workdir string) error {
	const unitDir = "/etc/systemd/system"

	servicePath := filepath.Join(unitDir, "nself-ssl-renew.service")
	if err := os.WriteFile(servicePath, []byte(sslRenewalServiceUnit(workdir)), 0644); err != nil {
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
