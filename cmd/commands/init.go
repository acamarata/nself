package commands

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/migration"
	"github.com/nself-org/cli/internal/scaffold"
	"github.com/nself-org/cli/internal/setup"
	"github.com/nself-org/cli/internal/telemetry"
	clonetemplate "github.com/nself-org/cli/internal/templates/clone"
	"github.com/nself-org/cli/internal/ui"

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
	f.String("template", "", "Use a built-in, clone, or marketplace template (e.g. airbnb-clone, go, rust)")
	f.Bool("no-seed", false, "Skip seed data when scaffolding a clone template")
	f.Bool("dry-run", false, "Print files that would be written without writing them (clone templates only)")
	f.Bool("skip-validation", false, "Skip configuration validation")
	f.Bool("wizard", false, "Run the full 10-step interactive wizard")
	f.Bool("demo", false, "Auto-configure with all services enabled")
	f.Bool("full", false, "Create all environment files (.env.dev, .env.staging, .env.prod, .env.secrets)")
	f.Bool("force", false, "Overwrite existing configuration")
	f.Bool("quiet", false, "Suppress output messages")
	f.String("name", "", "Project name (sets PROJECT_NAME in generated .env)")
	f.String("domain", "", "Base domain (skips interactive domain selection, e.g. myapp.dev)")
	f.String("profile", "", "Resource profile: 'tiny' for small VPS (Postgres+nginx only)")
	f.Bool("no-pgvector", false, "Skip pgvector extension and RAG scaffold tables (sets PGVECTOR_ENABLED=false)")
	f.String("preset", "", "Use a project-type preset: b2b-saas, mobile-backend, ai-assistant, community-forum, media-hosting, dev, sentry, nclaw-app")
	f.Bool("list-presets", false, "List all available project presets and exit")
	f.String("cs-template", "", "Scaffold a custom service at init time: specify language (go, node, python, rust, other)")

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
	noPgvector, _ := cmd.Flags().GetBool("no-pgvector")
	preset, _ := cmd.Flags().GetString("preset")
	listPresets, _ := cmd.Flags().GetBool("list-presets")
	csTemplate, _ := cmd.Flags().GetString("cs-template")

	// --list-presets: print preset catalog and exit.
	if listPresets {
		listInitPresets()
		return nil
	}

	// Validate --cs-template if given.
	if csTemplate != "" && !scaffold.IsValidLang(csTemplate) {
		return fmt.Errorf("--cs-template %q is not a supported language; choose one of: %s",
			csTemplate, strings.Join(scaffold.SupportedLangs(), ", "))
	}

	// Validate --preset if given.
	if preset != "" {
		if _, ok := initPresets[preset]; !ok {
			fmt.Fprintf(os.Stderr, "%s Unknown preset %q.\n", ui.C(ui.Yellow, ui.IconWarning), preset)
			listInitPresets()
			return fmt.Errorf("unknown preset %q — see presets above", preset)
		}
	}

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

		// Clone templates are embedded in the binary and scaffold directly.
		if clonetemplate.IsCloneTemplate(template) {
			destDir := "."
			if len(args) > 0 {
				destDir = args[0]
			}
			noSeed, _ := cmd.Flags().GetBool("no-seed")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			return runInitCloneTemplate(template, destDir, noSeed, dryRun, force, quiet)
		}

		// Marketplace slug (not a built-in language template): fetch from registry.
		if !isBuiltinTemplate(template) {
			destDir := "."
			if len(args) > 0 {
				destDir = args[0]
			}
			return runInitMarketplaceTemplate(template, destDir, force, quiet)
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

	// Tiny-VPS preflight: warn when available RAM is below the recommended
	// minimum (1 GB). Detection failure is non-fatal — we warn-and-continue.
	if ramMB, ramErr := getTotalMemoryMB(); ramErr == nil {
		const tinyThresholdMB = 1024
		const warnThresholdMB = 512
		if ramMB < warnThresholdMB {
			fmt.Fprintf(os.Stderr,
				"\n%s Detected %d MB RAM. nself default stack needs 1 GB+.\n"+
					"   For small VPS, run: nself init --profile=tiny\n"+
					"   The tiny profile starts Postgres + nginx only; Hasura/Auth are opt-in.\n"+
					"   See: https://github.com/nself-org/cli/wiki/install/tiny-vps\n\n",
				ui.C(ui.Yellow, ui.IconWarning), ramMB)
		} else if ramMB < tinyThresholdMB {
			fmt.Fprintf(os.Stderr,
				"\n%s Detected %d MB RAM. 1 GB+ recommended for the full stack.\n"+
					"   For small VPS, run: nself init --profile=tiny\n\n",
				ui.C(ui.Yellow, ui.IconWarning), ramMB)
		}
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
		NoPgvector:     noPgvector,
	}

	// Wizard progress: show step-by-step display when --wizard is active.
	var wizardSteps *ui.InitSteps
	if wizard && !quiet {
		wizardSteps = ui.NewInitSteps(false,
			"Validate inputs",
			"Generate secrets",
			"Write .env files",
			"Write .env.example",
			"Update .gitignore",
			"Create .nself/ directory",
		)
		fmt.Println()
		ui.Section("Wizard — initializing your nSelf project")
		fmt.Println()
		wizardSteps.Next() // step 1: inputs already validated above
		wizardSteps.Next() // step 2: generating secrets (happens inside Initialize)
	}

	// Telemetry: record start time for duration measurement.
	initStart := time.Now()

	result, err := setup.Initialize(opts)

	if wizardSteps != nil {
		// Advance remaining steps to reflect work completed inside Initialize.
		wizardSteps.Next() // step 3: .env files
		wizardSteps.Next() // step 4: .env.example
		wizardSteps.Next() // step 5: .gitignore
		wizardSteps.Next() // step 6: .nself/
		wizardSteps.Done()
		fmt.Println()
	}

	// Telemetry: emit init_complete event (opt-in only; silently no-ops when unset).
	if telemetry.IsOptedIn() {
		wizardMode := "default"
		switch {
		case fast:
			wizardMode = "fast"
		case wizard:
			wizardMode = "wizard"
		case demo:
			wizardMode = "demo"
		case nonInteractive:
			wizardMode = "non-interactive"
		}

		errCategory := ""
		if err != nil {
			errCategory = classifyInitError(err)
		}

		telemetry.Send("init_complete", map[string]any{
			"wizard_mode":  wizardMode,
			"duration_ms":  time.Since(initStart).Milliseconds(),
			"success":      err == nil,
			"err_category": errCategory,
		})
	}

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
	fmt.Printf("  1. %s  %s Verify environment (Docker, ports, config)\n",
		ui.C(ui.Bold, "nself doctor"), ui.C(ui.Dim, ui.IconArrow))
	fmt.Printf("  2. %s  %s Launch your backend stack\n",
		ui.C(ui.Bold, "nself start"), ui.C(ui.Dim, ui.IconArrow))
	fmt.Printf("  3. %s  %s Live health dashboard\n",
		ui.C(ui.Bold, "nself status"), ui.C(ui.Dim, ui.IconArrow))
	fmt.Printf("  4. %s  %s Browse available plugins\n",
		ui.C(ui.Bold, "nself plugin list"), ui.C(ui.Dim, ui.IconArrow))
	if result.Demo {
		fmt.Printf("  5. %s  %s Open the admin panel (demo mode)\n",
			ui.C(ui.Bold, "nself admin start"), ui.C(ui.Dim, ui.IconArrow))
	}
	fmt.Println()

	// Preset-specific next steps.
	if preset != "" {
		printPresetPostInit(preset)
	}

	// --cs-template: scaffold a custom service into the newly initialised project.
	if csTemplate != "" {
		svcName := "myservice"
		if name != "" {
			svcName = name
		}
		svcOpts := scaffold.Options{
			Name:       svcName,
			Lang:       csTemplate,
			ProjectDir: cwd,
		}
		if !quiet {
			ui.Section(fmt.Sprintf("Scaffolding custom service %q (%s)", svcName, csTemplate))
		}
		csResult, csErr := scaffold.Run(svcOpts)
		if csErr != nil {
			// Non-fatal: report but do not fail init.
			fmt.Fprintf(os.Stderr, "Warning: --cs-template scaffold failed: %v\n", csErr)
			fmt.Fprintln(os.Stderr, "Run 'nself service add <name> --lang "+csTemplate+"' to retry.")
		} else if !quiet {
			fmt.Printf("  Service %q scaffolded at %s\n", svcName, csResult.ServiceDir)
			fmt.Printf("  %s=%s written to %s\n", csResult.EnvKey, csResult.EnvValue, csResult.EnvFile)
			fmt.Printf("  Run 'nself build' then 'nself start' to launch.\n\n")
		}
	}

	return nil
}

// classifyInitError maps an init error to a categorical string safe for telemetry.
// Enumerated values: timeout, permission-denied, docker-not-found, other.
// Must never include the error message text (may contain file paths).
func classifyInitError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case containsAny(msg, "timeout", "deadline exceeded", "context deadline"):
		return "timeout"
	case containsAny(msg, "permission denied", "access denied", "EACCES"):
		return "permission-denied"
	case containsAny(msg, "docker", "Docker", "cannot connect to the Docker"):
		return "docker-not-found"
	default:
		return "other"
	}
}

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
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

// builtinTemplates are the scaffolding language templates bundled with the CLI.
var builtinTemplates = []string{"postgres", "express", "fastapi", "go", "rust"}

func validateTemplate(name string) error {
	for _, v := range builtinTemplates {
		if v == name {
			return nil
		}
	}
	// Not a built-in; treat as a marketplace slug — validation deferred to registry lookup.
	return nil
}

// isBuiltinTemplate reports whether name is one of the bundled language or clone templates.
func isBuiltinTemplate(name string) bool {
	for _, v := range builtinTemplates {
		if v == name {
			return true
		}
	}
	return clonetemplate.IsCloneTemplate(name)
}

// runInitCloneTemplate scaffolds one of the six embedded app-clone templates
// into destDir. No network access is required.
func runInitCloneTemplate(name, destDir string, noSeed, dryRun, force, quiet bool) error {
	if !quiet {
		verb := "Scaffolding"
		if dryRun {
			verb = "Dry-run for"
		}
		ui.CommandHeader("nself init --template", fmt.Sprintf("%s clone template %q", verb, name))
	}

	m, err := clonetemplate.GetManifest(name)
	if err != nil {
		return fmt.Errorf("loading template manifest: %w", err)
	}

	// Resolve destination directory.
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolving destination: %w", err)
	}

	if !dryRun {
		if !force {
			if entries, readErr := os.ReadDir(absDest); readErr == nil && len(entries) > 0 {
				return fmt.Errorf("destination %s is not empty; use --force to overwrite", absDest)
			}
		}
		if mkErr := os.MkdirAll(absDest, 0755); mkErr != nil {
			return fmt.Errorf("creating destination directory: %w", mkErr)
		}
	}

	written, scaffoldErr := clonetemplate.Scaffold(name, clonetemplate.ScaffoldOptions{
		TargetDir: absDest,
		NoSeed:    noSeed,
		DryRun:    dryRun,
	})
	if scaffoldErr != nil {
		return fmt.Errorf("scaffolding template: %w", scaffoldErr)
	}

	if dryRun {
		fmt.Printf("\n  %s Files that would be written to %s:\n\n", ui.C(ui.Blue, ui.IconInfo), absDest)
		for _, f := range written {
			fmt.Printf("    %s\n", f)
		}
		fmt.Printf("\n  %d file(s). Run without --dry-run to write them.\n\n", len(written))
		return nil
	}

	if !quiet {
		fmt.Printf("\n  %s Template %q scaffolded into %s\n\n",
			ui.C(ui.Green, ui.IconSuccess), name, absDest)
		fmt.Printf("  Files written: %d\n", len(written))
		if len(m.RequiredPlugins) > 0 {
			fmt.Printf("  Required plugins: %s\n", strings.Join(m.RequiredPlugins, ", "))
			fmt.Printf("\n  Install plugins:\n")
			fmt.Printf("    %s\n", ui.C(ui.Cyan, fmt.Sprintf("nself plugin install %s", strings.Join(m.RequiredPlugins, " "))))
		}
		fmt.Printf("\n  Next steps:\n")
		fmt.Printf("    cd %s\n", absDest)
		fmt.Printf("    nself start\n")
		fmt.Printf("    nself db migrate\n")
		fmt.Printf("    nself hasura metadata apply --file hasura/metadata.json\n")
		if len(m.RequiredPlugins) > 0 {
			fmt.Printf("    nself plugin install %s\n", strings.Join(m.RequiredPlugins, " "))
		}
		fmt.Printf("    cd flutter && flutter run\n\n")
	}
	return nil
}

// runInitMarketplaceTemplate downloads a marketplace template tarball, verifies
// its SHA256 checksum, and extracts it into destDir.
func runInitMarketplaceTemplate(slug, destDir string, force, quiet bool) error {
	if !quiet {
		ui.CommandHeader("nself init --template", fmt.Sprintf("Installing marketplace template %q", slug))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	baseURL := resolveTemplateRegistryURL()
	t, err := fetchTemplateSingle(ctx, baseURL, slug)
	if err != nil {
		return fmt.Errorf("template %q not found in registry.\n"+
			"Browse available templates: nself template list\n"+
			"Or visit: https://nself.org/templates\n"+
			"Original error: %w", slug, err)
	}

	// Resolve destination directory.
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolving destination: %w", err)
	}
	if !force {
		if entries, readErr := os.ReadDir(absDest); readErr == nil && len(entries) > 0 {
			return fmt.Errorf("destination %s is not empty; use --force to overwrite", absDest)
		}
	}
	if mkErr := os.MkdirAll(absDest, 0755); mkErr != nil {
		return fmt.Errorf("creating destination directory: %w", mkErr)
	}

	// License check for paid templates.
	if t.PriceUSD > 0 {
		licenseKey := os.Getenv("NSELF_LICENSE_KEY")
		if licenseKey == "" {
			return fmt.Errorf(
				"template %q costs $%.2f and requires a valid license.\n"+
					"Set your license key: nself license set nself_pro_<your-key>\n"+
					"Get a license at: https://nself.org/pricing",
				slug, t.PriceUSD,
			)
		}
	}

	if !quiet {
		ui.Info(fmt.Sprintf("Downloading %s v%s...", t.DisplayName, t.TemplateVersion))
	}

	// Download tarball.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.TarballURL, nil)
	if err != nil {
		return fmt.Errorf("preparing download request: %w", err)
	}
	req.Header.Set("User-Agent", "nself-cli")
	if licKey := os.Getenv("NSELF_LICENSE_KEY"); licKey != "" {
		req.Header.Set("X-Nself-License", licKey)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("downloading template tarball: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("access denied for template %q — check your license key", slug)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Buffer the body so we can verify SHA256 before extraction.
	tarData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading tarball: %w", err)
	}

	// Verify SHA256.
	if t.TarballSHA256 != "" {
		h := sha256.New()
		h.Write(tarData)
		got := hex.EncodeToString(h.Sum(nil))
		if got != t.TarballSHA256 {
			return fmt.Errorf(
				"SHA256 mismatch for template %q\n  expected: %s\n  got:      %s\nThe download may be corrupted or tampered with.",
				slug, t.TarballSHA256, got,
			)
		}
		if !quiet {
			ui.Success("Checksum verified.")
		}
	}

	// Extract tarball into destDir.
	if !quiet {
		ui.Info(fmt.Sprintf("Extracting into %s...", absDest))
	}

	gr, err := gzip.NewReader(strings.NewReader(string(tarData)))
	if err != nil {
		return fmt.Errorf("opening gzip stream: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, tarErr := tr.Next()
		if tarErr == io.EOF {
			break
		}
		if tarErr != nil {
			return fmt.Errorf("reading tarball entry: %w", tarErr)
		}

		// Guard against path traversal.
		target := filepath.Join(absDest, filepath.Clean("/" + header.Name)[1:])
		if !strings.HasPrefix(target, absDest+string(os.PathSeparator)) && target != absDest {
			return fmt.Errorf("illegal path in tarball: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if mkErr := os.MkdirAll(target, 0755); mkErr != nil {
				return fmt.Errorf("creating directory %s: %w", target, mkErr)
			}
		case tar.TypeReg:
			if mkErr := os.MkdirAll(filepath.Dir(target), 0755); mkErr != nil {
				return fmt.Errorf("creating parent directory: %w", mkErr)
			}
			f, createErr := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if createErr != nil {
				return fmt.Errorf("creating file %s: %w", target, createErr)
			}
			if _, copyErr := io.Copy(f, tr); copyErr != nil {
				f.Close()
				return fmt.Errorf("writing file %s: %w", target, copyErr)
			}
			f.Close()
		}
	}

	if !quiet {
		ui.Success(fmt.Sprintf("Template %q installed in %s", slug, absDest))
		fmt.Println()
		if len(t.RequiredPlugins) > 0 {
			fmt.Printf("Required plugins: %s\n", strings.Join(t.RequiredPlugins, ", "))
			fmt.Printf("Install with: nself plugin install %s\n", strings.Join(t.RequiredPlugins, " "))
			fmt.Println()
		}
		fmt.Println("Next steps:")
		fmt.Printf("  cd %s\n", destDir)
		fmt.Println("  nself start")
	}

	return nil
}
