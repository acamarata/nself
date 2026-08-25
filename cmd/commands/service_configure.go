package commands

// Purpose: Implements `nself service configure email --provider <name>`:
// applies known SMTP host/port presets for common email providers (Postmark,
// SendGrid, Resend, Mailgun, SES, SparkPost, generic SMTP) into the env
// file. Split out of service.go (CLI-R12) to separate this configure
// subcommand's cobra wiring and handler from the add/upgrade/list/enable/
// disable/lifecycle handlers in the other service_*.go files.
// Inputs: the serviceConfigureCmd cobra.Command + args (subject, e.g.
// "email") and the --provider / --env flags.
// Outputs: SMTP_HOST / SMTP_PORT (and related) env file entries written via
// setEnvKeyInFile (service_add_upgrade.go).
// Constraints: pure move — no behavior changes. serviceConfigureCmd keeps
// its own init() (Go permits multiple init funcs per file/package); it is
// registered onto serviceCmd exactly as before.

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// service configure (email provider presets)
// ---------------------------------------------------------------------------

// emailProviderPreset holds the SMTP env var values for a known email provider.
type emailProviderPreset struct {
	Host string
	Port string
	// User and Pass are intentionally empty — users must supply credentials.
}

// emailProviderPresets maps provider names to their SMTP relay settings.
var emailProviderPresets = map[string]emailProviderPreset{
	"postmark":     {Host: "smtp.postmarkapp.com", Port: "587"},
	"sendgrid":     {Host: "smtp.sendgrid.net", Port: "587"},
	"resend":       {Host: "smtp.resend.com", Port: "465"},
	"mailgun":      {Host: "smtp.mailgun.org", Port: "587"},
	"ses":          {Host: "email-smtp.us-east-1.amazonaws.com", Port: "587"},
	"sparkpost":    {Host: "smtp.sparkpostmail.com", Port: "587"},
	"smtp-generic": {Host: "", Port: "587"},
	"mailchimp":    {Host: "smtp.mandrillapp.com", Port: "587"},
	"brevo":        {Host: "smtp-relay.brevo.com", Port: "587"},
	"mailersend":   {Host: "smtp.mailersend.net", Port: "587"},
	"smtp2go":      {Host: "mail.smtp2go.com", Port: "2525"},
	"zoho":         {Host: "smtp.zoho.com", Port: "587"},
	"outlook":      {Host: "smtp.office365.com", Port: "587"},
	"gmail":        {Host: "smtp.gmail.com", Port: "587"},
	"elasticemail": {Host: "smtp.elasticemail.com", Port: "2525"},
	"socketlabs":   {Host: "smtp.socketlabs.com", Port: "587"},
}

var serviceConfigureCmd = &cobra.Command{
	Use:   "configure <service>",
	Short: "Configure service settings (e.g. email provider presets)",
	Long: `Configure service-specific settings.

Currently supports email provider presets:

  nself service configure email --provider postmark
  nself service configure email --provider sendgrid
  nself service configure email --provider resend
  nself service configure email --provider mailgun
  nself service configure email --provider ses
  nself service configure email --provider sparkpost
  nself service configure email --provider smtp-generic

The command writes AUTH_SMTP_HOST and AUTH_SMTP_PORT to your .env file.
You must also set AUTH_SMTP_USER and AUTH_SMTP_PASS (credentials not stored here).`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceConfigure,
}

func init() {
	serviceConfigureCmd.Flags().String("provider", "", "Email provider preset name")
}

func runServiceConfigure(cmd *cobra.Command, args []string) error {
	svcName := strings.ToLower(strings.TrimSpace(args[0]))

	// Resolve aliases.
	if canonical, ok := serviceAliases[svcName]; ok {
		svcName = canonical
	}

	if svcName != "email" {
		return fmt.Errorf("configure only supports the 'email' service currently; got %q", svcName)
	}

	provider, _ := cmd.Flags().GetString("provider")
	if provider == "" {
		// List available providers and exit.
		fmt.Println("Available email providers:")
		for name := range emailProviderPresets {
			p := emailProviderPresets[name]
			if p.Host != "" {
				fmt.Printf("  %-16s  host: %s  port: %s\n", name, p.Host, p.Port)
			} else {
				fmt.Printf("  %-16s  (custom SMTP — set AUTH_SMTP_HOST manually)\n", name)
			}
		}
		fmt.Println("\nUsage: nself service configure email --provider <name>")
		return nil
	}

	provider = strings.ToLower(strings.TrimSpace(provider))
	preset, ok := emailProviderPresets[provider]
	if !ok {
		return fmt.Errorf("unknown provider %q; run without --provider to list all providers", provider)
	}

	envFlag, _ := cmd.Flags().GetString("env")
	envFile, err := resolveEnvFile(envFlag)
	if err != nil {
		return err
	}

	if preset.Host != "" {
		if err := setEnvKeyInFile(envFile, "AUTH_SMTP_HOST", preset.Host); err != nil {
			return fmt.Errorf("writing AUTH_SMTP_HOST: %w", err)
		}
	}
	if err := setEnvKeyInFile(envFile, "AUTH_SMTP_PORT", preset.Port); err != nil {
		return fmt.Errorf("writing AUTH_SMTP_PORT: %w", err)
	}

	fmt.Printf("Email provider '%s' configured:\n", provider)
	if preset.Host != "" {
		fmt.Printf("  AUTH_SMTP_HOST=%s\n", preset.Host)
	}
	fmt.Printf("  AUTH_SMTP_PORT=%s\n", preset.Port)
	fmt.Println("\nNext steps:")
	fmt.Println("  Set AUTH_SMTP_USER=<your-username>")
	fmt.Println("  Set AUTH_SMTP_PASS=<your-password>")
	fmt.Println("  Run `nself build` to apply changes.")
	return nil
}
