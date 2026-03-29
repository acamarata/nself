package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/ssl"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var dnsSetupCmd = &cobra.Command{
	Use:   "dns-setup",
	Short: "Add project domains to /etc/hosts (run with sudo)",
	Long: `Add all project domains to /etc/hosts so local *.custom-domain URLs resolve.

This command reads your project's .env, collects every domain that nself
generates nginx config for, and appends missing entries to /etc/hosts.

Because /etc/hosts requires root access, run this once with sudo:

  sudo nself dns-setup

Wildcard entries (e.g. *.pro.ummat.local) cannot go in /etc/hosts. For
wildcard subdomains, install dnsmasq and add:

  address=/.pro.ummat.local/127.0.0.1

Only needed for custom BASE_DOMAIN values (e.g. ummat.local). Standard
nself.org and localhost setups do not need this.`,
	RunE: runDNSSetup,
}

func init() {
	dnsSetupCmd.Flags().BoolP("dry-run", "n", false, "Print entries that would be added without writing")
	RootCmd.AddCommand(dnsSetupCmd)
}

func runDNSSetup(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	ui.CommandHeader("nself dns-setup", "Add project domains to /etc/hosts")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	workdir, err := config.FindNSelfRoot(cwd)
	if err != nil {
		return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
	}

	cfg, err := config.Load(workdir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	gen := ssl.NewGenerator(cfg)
	domains := gen.CollectDomains()

	// Filter to only the entries that can go in /etc/hosts (no wildcards, no IPs).
	// We surface wildcards separately so the user knows what needs dnsmasq.
	var wildcards []string
	var hostable []string
	for _, d := range domains {
		if strings.HasPrefix(d, "*.") || strings.Contains(d, "*") {
			wildcards = append(wildcards, d)
		} else {
			hostable = append(hostable, d)
		}
	}

	if len(hostable) == 0 && len(wildcards) == 0 {
		ui.Info("No domains to configure for " + cfg.BaseDomain)
		return nil
	}

	// Report wildcard entries separately — they need dnsmasq, not /etc/hosts.
	if len(wildcards) > 0 {
		ui.Warn("Wildcard domains require dnsmasq (cannot go in /etc/hosts):")
		for _, w := range wildcards {
			fmt.Printf("  address=/%s/127.0.0.1\n", strings.TrimPrefix(w, "*."))
		}
		fmt.Println()
	}

	if dryRun {
		fmt.Println("Would add to /etc/hosts:")
		for _, h := range hostable {
			fmt.Printf("  127.0.0.1\t%s\n", h)
		}
		return nil
	}

	added, err := ssl.AddHosts(hostable)
	if err != nil {
		if errors.Is(err, ssl.ErrSudoRequired) {
			ui.Error("Permission denied — /etc/hosts requires root. Run: sudo nself dns-setup")
			return fmt.Errorf("permission denied writing /etc/hosts")
		}
		return fmt.Errorf("updating /etc/hosts: %w", err)
	}

	if added == 0 {
		ui.Success("All domains already in /etc/hosts — nothing to do.")
	} else {
		ui.Success(fmt.Sprintf("Added %d domain(s) to /etc/hosts", added))
	}

	if len(wildcards) > 0 {
		ui.Info("Remember to configure dnsmasq for wildcard domains (see above).")
	}

	return nil
}
