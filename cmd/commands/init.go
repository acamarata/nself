package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"nself/internal/config"
	"nself/internal/migration"
	"nself/internal/setup"
	"nself/internal/ui"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new nSelf project",
	Long: `Interactive setup wizard that creates a pristine .env configuration
for a new nSelf project.

Entry points:
  nself init                 Minimal setup with smart defaults
  nself init --wizard        Full 10-step interactive wizard
  nself init --demo          All services enabled, pre-configured
  nself init --full          Creates .env.dev, .env.staging, .env.prod, .env.secrets
  nself init --fast          Skip advanced options, use smart defaults
  nself init --non-interactive  Use all defaults without prompts

After init, run:
  Edit .env      Customize your project settings
  nself start    Boot your backend stack`,
	RunE: runInit,
}

func init() {
	f := initCmd.Flags()
	f.Bool("fast", false, "Skip advanced options, use smart defaults")
	f.Bool("interactive", false, "Explicitly enable interactive wizard")
	f.Bool("non-interactive", false, "Use all defaults without prompts")
	f.String("template", "", "Use specific template (express, fastapi, go, rust)")
	f.Bool("skip-validation", false, "Skip configuration validation")
	f.Bool("wizard", false, "Run the full 10-step interactive wizard")
	f.Bool("demo", false, "Auto-configure with all services enabled")
	f.Bool("full", false, "Create all environment files (.env.dev, .env.staging, .env.prod, .env.secrets)")
	f.Bool("force", false, "Overwrite existing configuration")
	f.Bool("quiet", false, "Suppress output messages")
	f.String("name", "", "Project name (sets PROJECT_NAME in generated .env)")
	f.String("domain", "", "Base domain (skips interactive domain selection, e.g. myapp.dev)")

	RootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	fast, _ := cmd.Flags().GetBool("fast")
	interactive, _ := cmd.Flags().GetBool("interactive")
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
	template, _ := cmd.Flags().GetString("template")
	skipValidation, _ := cmd.Flags().GetBool("skip-validation")
	wizard, _ := cmd.Flags().GetBool("wizard")
	demo, _ := cmd.Flags().GetBool("demo")
	full, _ := cmd.Flags().GetBool("full")
	force, _ := cmd.Flags().GetBool("force")
	quiet, _ := cmd.Flags().GetBool("quiet")
	name, _ := cmd.Flags().GetString("name")
	domainFlag, _ := cmd.Flags().GetString("domain")

	// Sanitize user-supplied --name before it enters the config system.
	if name != "" {
		sanitized, err := config.SanitizeName(name)
		if err != nil {
			return fmt.Errorf("--name: PROJECT_NAME contains no valid characters after sanitization")
		}
		name = sanitized
	}

	// Sanitize user-supplied --domain before it enters the config system.
	if domainFlag != "" {
		sanitized, err := config.SanitizeDomain(domainFlag)
		if err != nil {
			return fmt.Errorf("--domain: BASE_DOMAIN contains no valid characters after sanitization")
		}
		domainFlag = sanitized
	}

	if template != "" {
		if err := validateTemplate(template); err != nil {
			return err
		}
	}

	// v1 migration check: if .nself/ does not exist this may be a first run on an
	// existing v1 project. Scan for legacy artifacts and warn the user before init.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	if _, statErr := os.Stat(cwd + "/.nself"); os.IsNotExist(statErr) {
		if artifacts := migration.Detect(cwd); len(artifacts) > 0 {
			fmt.Fprintf(os.Stderr, "\n%s Legacy nSelf v1 artifacts detected:\n", ui.C(ui.Yellow, ui.IconWarning))
			for _, a := range artifacts {
				fmt.Fprintf(os.Stderr, "   %s %s  %s %s\n",
					ui.C(ui.Yellow, ui.IconBullet), a.Path,
					ui.C(ui.Dim, "\u2014"), ui.C(ui.Dim, a.Description+" ("+a.Action+")"),
				)
			}
			fmt.Fprintf(os.Stderr, "\nRun `nself migrate` to upgrade this project to v2.\n\n")
		}
	}

	if !quiet {
		ui.CommandHeader("nself init", "Initialize a new nSelf project")
	}

	// Resolve domain: --domain flag takes precedence; otherwise prompt
	// interactively when running in a TTY. Non-TTY / --non-interactive /
	// --fast paths skip the prompt and let setup.go apply its defaults.
	selectedDomain := domainFlag
	selectedDomainComment := ""
	if selectedDomain != "" {
		// --domain flag provided: validate (non-empty, no spaces).
		if err := validateDomain(selectedDomain); err != nil {
			return err
		}
	} else if !nonInteractive && !fast && term.IsTerminal(int(os.Stdin.Fd())) {
		d, comment, err := promptDomainPattern()
		if err != nil {
			return err
		}
		selectedDomain = d
		selectedDomainComment = comment
	}

	opts := setup.Options{
		Fast:           fast,
		Interactive:    interactive,
		NonInteractive: nonInteractive,
		Template:       template,
		SkipValidation: skipValidation,
		Wizard:         wizard,
		Demo:           demo,
		Full:           full,
		Force:          force,
		Quiet:          quiet,
		Name:           name,
		Domain:         selectedDomain,
		DomainComment:  selectedDomainComment,
	}

	result, err := setup.Initialize(opts)
	if err != nil {
		if !quiet {
			ui.Error(fmt.Sprintf("Init failed: %v", err))
		}
		return err
	}

	if quiet {
		return nil
	}

	// Summary.
	items := []string{
		fmt.Sprintf("Project:  %s", result.ProjectName),
		fmt.Sprintf("Domain:   %s", result.BaseDomain),
		fmt.Sprintf("Env:      %s", result.Env),
		fmt.Sprintf("Files:    %d created", len(result.FilesCreated)),
	}
	if name != "" {
		items = append(items, fmt.Sprintf("Name:     %s (from --name flag)", name))
	}
	if result.Demo {
		items = append(items, "Mode:     demo (all services enabled)")
	}
	ui.SummaryBox("Configuration Generated", items)

	// Next steps.
	ui.Section("Next steps")
	fmt.Printf("  1. Edit %s to customize your project\n", ui.C(ui.Bold, ".env"))
	fmt.Printf("  2. %s  %s Launch your backend\n", ui.C(ui.Bold, "nself start"), ui.C(ui.Dim, ui.IconArrow))
	fmt.Println()

	return nil
}

// domainOption represents a single selectable domain pattern in the wizard.
type domainOption struct {
	label   string // display label shown in the menu
	value   string // BASE_DOMAIN value written to .env
	comment string // comment written above BASE_DOMAIN in .env
	custom  bool   // when true, prompt user for a custom value
}

// domainOptions lists the preset domain patterns presented to the user.
var domainOptions = []domainOption{
	{
		label:   "*.local.nself.org    — Works with local.nself.org wildcard DNS",
		value:   "local.nself.org",
		comment: "# Domain: *.local.nself.org (wildcard — requires DNS resolution)",
	},
	{
		label:   "*.localhost           — Browser-only, no TLS in some browsers",
		value:   "localhost",
		comment: "# Domain: *.localhost (browser-only local dev)",
	},
	{
		label:   "*.127.0.0.1.nip.io   — Works on any machine, uses wildcard DNS",
		value:   "127.0.0.1.nip.io",
		comment: "# Domain: *.127.0.0.1.nip.io (wildcard DNS via nip.io)",
	},
	{
		label:  "Custom domain         — Enter your own (e.g. myapp.dev)",
		custom: true,
	},
}

// promptDomainPattern presents a numbered menu to the user and returns the
// selected BASE_DOMAIN value and its associated .env comment.
// It reads from os.Stdin and writes the prompt to os.Stdout.
func promptDomainPattern() (domain, comment string, err error) {
	fmt.Println()
	fmt.Printf("%sChoose a domain pattern:%s\n\n", ui.Bold, ui.Reset)
	for i, opt := range domainOptions {
		fmt.Printf("  %s%d%s) %s\n", ui.Cyan, i+1, ui.Reset, opt.label)
	}
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("Enter choice [1-%d] (default 1): ", len(domainOptions))
		if !scanner.Scan() {
			// EOF or error — fall back to default.
			return domainOptions[0].value, domainOptions[0].comment, nil
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			// User pressed Enter — use default.
			return domainOptions[0].value, domainOptions[0].comment, nil
		}

		// Parse the selection.
		choice := 0
		for _, ch := range input {
			if ch < '0' || ch > '9' {
				choice = -1
				break
			}
			choice = choice*10 + int(ch-'0')
		}

		if choice < 1 || choice > len(domainOptions) {
			fmt.Fprintf(os.Stderr, "  Invalid choice. Please enter a number between 1 and %d.\n", len(domainOptions))
			continue
		}

		selected := domainOptions[choice-1]
		if !selected.custom {
			return selected.value, selected.comment, nil
		}

		// Custom domain: prompt for the value.
		fmt.Printf("  Enter your domain (e.g. myapp.dev): ")
		if !scanner.Scan() {
			return "", "", fmt.Errorf("reading custom domain: unexpected EOF")
		}
		custom := strings.TrimSpace(scanner.Text())
		if err := validateDomain(custom); err != nil {
			fmt.Fprintf(os.Stderr, "  %s\n", err)
			continue
		}
		return custom, "", nil
	}
}

func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain must not be empty")
	}
	if strings.ContainsAny(domain, " \t\n\r") {
		return fmt.Errorf("domain must not contain whitespace: %q", domain)
	}
	return nil
}

func validateTemplate(name string) error {
	valid := []string{"postgres", "express", "fastapi", "go", "rust"}
	for _, v := range valid {
		if v == name {
			return nil
		}
	}
	return fmt.Errorf("template %q not found. Available templates: %s", name, strings.Join(valid, ", "))
}
