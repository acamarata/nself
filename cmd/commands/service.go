package commands

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/scaffold"
	"github.com/nself-org/cli/internal/security"
	"github.com/nself-org/cli/internal/service"
	"github.com/nself-org/cli/internal/ui"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// versionPattern validates service version strings: semver, tags like "latest",
// "alpine", "16.3", "v2.40.0", and simple numeric versions.
var versionPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-]*$`)

// serviceEntry describes a single optional service and its governing env var.
type serviceEntry struct {
	Name   string
	EnvVar string
	Port   int // 0 means N/A (or "multi" for monitoring)
}

// knownServices is the ordered list of optional services managed by `nself service`.
// MLflow was reclassified to a free plugin in v1.1.0 — use `nself plugin install mlflow`.
var knownServices = []serviceEntry{
	{Name: "redis", EnvVar: "REDIS_ENABLED", Port: 6379},
	{Name: "minio", EnvVar: "MINIO_ENABLED", Port: 9000},
	{Name: "email", EnvVar: "MAILPIT_ENABLED", Port: 8025},
	{Name: "functions", EnvVar: "FUNCTIONS_ENABLED", Port: 3008},
	{Name: "search", EnvVar: "SEARCH_ENABLED", Port: 7700},
	{Name: "monitoring", EnvVar: "MONITORING_ENABLED", Port: 0},
	{Name: "admin", EnvVar: "NSELF_ADMIN_ENABLED", Port: 3021},
}

// serviceAliases maps alternate names to canonical service names.
var serviceAliases = map[string]string{
	"storage":     "minio",
	"mail":        "email",
	"mailpit":     "email",
	"meilisearch": "search",
}

// serviceDependents maps a service name to the services that depend on it.
var serviceDependents = map[string][]string{
	"redis":  {"functions", "notify", "cron"},
	"minio":  {"storage"},
	"search": {"cms"},
}

// serviceCmd is the parent command for optional service management.
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage optional services",
	Long: `Enable, disable, and list optional nSelf services (6 total).

Available services:
  redis        Cache and queue (REDIS_ENABLED)
  minio        S3-compatible storage (MINIO_ENABLED), alias: storage
  email        Email testing via Mailpit (MAILPIT_ENABLED), aliases: mail, mailpit
  functions    Serverless runtime (FUNCTIONS_ENABLED)
  search       Full-text search via MeiliSearch (SEARCH_ENABLED), alias: meilisearch
  monitoring   Prometheus/Grafana stack (MONITORING_ENABLED)
  admin        nSelf Admin GUI (NSELF_ADMIN_ENABLED)

MLflow is now a free plugin: run 'nself plugin install mlflow'

After enabling or disabling a service, run 'nself build' to apply changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all optional services with enabled/disabled status",
	RunE:  runServiceList,
}

var serviceEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable an optional service",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceEnable,
}

var serviceDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable an optional service",
	Args:  cobra.ExactArgs(1),
	RunE:  runServiceDisable,
}

// serviceVersionKeys maps a canonical service name to its .env version key.
var serviceVersionKeys = map[string]string{
	"postgres":  "POSTGRES_VERSION",
	"hasura":    "HASURA_VERSION",
	"auth":      "AUTH_VERSION",
	"nginx":     "NGINX_VERSION",
	"redis":     "REDIS_VERSION",
	"minio":     "MINIO_VERSION",
	"email":     "MAILPIT_VERSION",
	"functions": "FUNCTIONS_VERSION",
	"search":    "MEILISEARCH_VERSION",
	"admin":     "NSELF_ADMIN_VERSION",
}

var serviceUpgradeCmd = &cobra.Command{
	Use:   "upgrade <name> <version>",
	Short: "Pin a service to a specific image version",
	Long: `Write the version tag for the named service into your .env file and prompt
you to run 'nself build' to apply the change.

Examples:
  nself service upgrade postgres 16.3
  nself service upgrade hasura v2.40.0
  nself service upgrade auth latest`,
	Args: cobra.ExactArgs(2),
	RunE: runServiceUpgrade,
}

// serviceStartCmd starts a named nSelf service via `docker compose up -d --no-deps`.
var serviceStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a named nSelf service",
	Long: `Start a named nSelf service using the existing docker-compose.yml.

The service must already be configured in the stack (run 'nself build' first).
This command is equivalent to 'docker compose up -d --no-deps <service>'.

Examples:
  nself service start redis
  nself service start minio
  nself service start search`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceStart,
}

// serviceStopCmd stops a named nSelf service (container preserved).
var serviceStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a named nSelf service (container preserved)",
	Long: `Stop a named nSelf service without removing its container.

The container state is preserved so 'nself service start <name>' can resume it
without re-creating volumes or losing data.

Examples:
  nself service stop redis
  nself service stop minio`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceStop,
}

// serviceRestartCmd restarts a named nSelf service.
var serviceRestartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart a named nSelf service",
	Long: `Restart a running nSelf service. Equivalent to 'docker compose restart <service>'.

Examples:
  nself service restart redis
  nself service restart hasura`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceRestart,
}

// servicePsCmd shows current status of all nSelf stack services.
var servicePsCmd = &cobra.Command{
	Use:   "ps",
	Short: "Show status of all nSelf stack services",
	Long: `Show the current status of every service in the running nSelf stack.

Reads live container state from 'docker compose ps'. Run 'nself start' first
if no services appear.`,
	RunE: runServicePs,
}

// serviceUpdateCmd pulls the latest image for a service and restarts it.
var serviceUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Pull the latest image for a service and restart it",
	Long: `Pull the latest image for a named service and restart it.

Equivalent to:
  docker compose pull <service>
  docker compose up -d --no-deps <service>

Examples:
  nself service update redis
  nself service update hasura`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceUpdate,
}

// serviceScaleCmd adjusts the replica count for a named service.
var serviceScaleCmd = &cobra.Command{
	Use:   "scale <name> <replicas>",
	Short: "Set the replica count for a named service",
	Long: `Set the number of replicas for a named nSelf service.

Replicas must be at least 1. This wraps 'docker compose up -d --scale <service>=<n>'.

Examples:
  nself service scale functions 3
  nself service scale redis 1`,
	Args: cobra.ExactArgs(2),
	RunE: runServiceScale,
}

// serviceAddCmd scaffolds a new CS_N custom service into the current nSelf project.
var serviceAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Scaffold a custom service (CS_N slot) into the current project",
	Long: `Scaffold a new custom service into the current nSelf project.

The command:
  1. Finds the next available CS_N slot (1-10) in .env.dev
  2. Creates a services/<name>/ directory with starter files
  3. Writes CS_N=<name>:<template>:<port> and <NAME>_PORT=<port> into .env.dev

Run 'nself build' after adding a service to regenerate docker-compose.yml.

Supported templates: go (default), node, python, static, rust, other

Examples:
  nself service add myapi
  nself service add myapi --template python
  nself service add myapi --template node --dry-run
  nself service add mysite --template static`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceAdd,
}

func init() {
	serviceCmd.PersistentFlags().String("env", "", "Target environment (reads .env.{env})")
	serviceListCmd.Flags().Bool("json", false, "Output as JSON array")

	// service add flags
	serviceAddCmd.Flags().String("template", "go", "Service template: go, node, python, static, rust, other")
	serviceAddCmd.Flags().String("lang", "", "Alias for --template (deprecated, use --template)")
	serviceAddCmd.Flags().MarkHidden("lang")
	serviceAddCmd.Flags().Bool("force", false, "Overwrite existing service directory")
	serviceAddCmd.Flags().Bool("dry-run", false, "Print what would be done without writing files")

	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceEnableCmd)
	serviceCmd.AddCommand(serviceDisableCmd)
	serviceCmd.AddCommand(serviceUpgradeCmd)
	serviceCmd.AddCommand(serviceConfigureCmd)
	serviceCmd.AddCommand(serviceAddCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(servicePsCmd)
	serviceCmd.AddCommand(serviceUpdateCmd)
	serviceCmd.AddCommand(serviceScaleCmd)

	RootCmd.AddCommand(serviceCmd)
}

// runServiceAdd implements 'nself service add <name>'.
func runServiceAdd(cmd *cobra.Command, args []string) error {
	name := strings.ToLower(strings.TrimSpace(args[0]))
	// --template is the canonical flag; --lang is a hidden backward-compat alias.
	lang, _ := cmd.Flags().GetString("template")
	if legacyLang, _ := cmd.Flags().GetString("lang"); legacyLang != "" {
		lang = legacyLang
	}
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if name == "" {
		return fmt.Errorf("service name must not be empty")
	}

	// Validate name: reuse the same rules as CS_N parsing (lowercase alphanumeric + hyphens/underscores, 2-63 chars).
	csNameRe := regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,61}[a-z0-9]$`)
	if !csNameRe.MatchString(name) {
		return fmt.Errorf("invalid service name %q: must be lowercase alphanumeric with hyphens/underscores, 2-63 chars", name)
	}

	if !scaffold.IsValidLang(lang) {
		return fmt.Errorf("unsupported language %q; choose one of: %s",
			lang, strings.Join(scaffold.SupportedLangs(), ", "))
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	opts := scaffold.Options{
		Name:       name,
		Lang:       lang,
		ProjectDir: cwd,
		Force:      force,
		DryRun:     dryRun,
	}

	result, err := scaffold.Run(opts)
	if err != nil {
		return err
	}

	if dryRun {
		ui.Info(fmt.Sprintf("Dry run — no files written. Would scaffold service %q (%s):", name, lang))
		fmt.Printf("  Slot:     %s (port %d)\n", result.EnvKey, 8000+result.Slot)
		fmt.Printf("  Env:      %s=%s\n", result.EnvKey, result.EnvValue)
		fmt.Printf("  EnvFile:  %s\n", result.EnvFile)
		fmt.Printf("  Dir:      %s\n", result.ServiceDir)
		fmt.Printf("  Files:\n")
		for _, f := range result.Files {
			fmt.Printf("    %s\n", f)
		}
		return nil
	}

	ui.Success(fmt.Sprintf("Custom service %q scaffolded.", name))
	fmt.Printf("  Slot:    %s = %s\n", result.EnvKey, result.EnvValue)
	fmt.Printf("  Dir:     %s\n", result.ServiceDir)
	fmt.Printf("  EnvFile: %s (updated)\n", result.EnvFile)
	fmt.Println()
	fmt.Printf("Next steps:\n")
	fmt.Printf("  Edit %s and implement your service\n", result.ServiceDir)
	fmt.Printf("  Run 'nself build' to regenerate docker-compose.yml\n")
	fmt.Printf("  Run 'nself start' to launch the full stack\n")
	return nil
}

// runServiceUpgrade writes <NAME>_VERSION=<ver> into the .env file.
func runServiceUpgrade(cmd *cobra.Command, args []string) error {
	name := strings.ToLower(strings.TrimSpace(args[0]))
	version := strings.TrimSpace(args[1])

	// Resolve aliases (e.g. "mailpit" → "email", "meilisearch" → "search").
	if canonical, ok := serviceAliases[name]; ok {
		name = canonical
	}

	// Validate: "mailpit" also maps to email via reverse lookup in knownServices.
	// For core services (postgres, hasura, auth, nginx) accept them directly.
	envKey, knownVersion := serviceVersionKeys[name]
	if !knownVersion {
		return fmt.Errorf("unknown service %q; upgradeable services: %s",
			name, strings.Join(upgradeableServiceNames(), ", "))
	}

	// Sanitize version string: alphanumeric plus ., -, _ only.
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("invalid version string %q: only alphanumeric, dots, hyphens, and underscores are allowed", version)
	}
	if len(version) > 128 {
		return fmt.Errorf("version string too long (max 128 chars)")
	}

	envFlag, _ := cmd.Flags().GetString("env")
	envFile, err := resolveEnvFile(envFlag)
	if err != nil {
		return err
	}

	if err := setEnvKeyInFile(envFile, envKey, version); err != nil {
		return fmt.Errorf("setting version for %s: %w", name, err)
	}

	fmt.Printf("%s version pinned to %s (%s=%s)\n", name, version, envKey, version)
	fmt.Println("Run `nself build` to apply the change.")
	return nil
}

// upgradeableServiceNames returns a sorted list of services that support version pinning.
func upgradeableServiceNames() []string {
	names := make([]string, 0, len(serviceVersionKeys))
	for k := range serviceVersionKeys {
		names = append(names, k)
	}
	return names
}

// --- helpers ---

// resolveEnvFile returns the .env filename for the given --env flag value.
// An empty env string resolves to ".env" in the current working directory.
func resolveEnvFile(envFlag string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	filename := ".env"
	if envFlag != "" {
		filename = ".env." + envFlag
	}
	return cwd + "/" + filename, nil
}

// mlflowRedirectMsg is returned when a user tries to enable/disable mlflow via
// the service command after the v1.1.0 reclassification.
const mlflowRedirectMsg = "mlflow is now a plugin: run `nself plugin install mlflow`\n" +
	"To uninstall: `nself plugin uninstall mlflow`"

// canonicalServiceName resolves aliases and validates the service name.
// Returns the canonical name and the serviceEntry, or an error if unknown.
// Returns a special errMLflowRedirect sentinel for the mlflow redirect case.
func canonicalServiceName(input string) (string, serviceEntry, error) {
	lower := strings.ToLower(strings.TrimSpace(input))

	// MLflow redirect: was an optional service, now a free plugin.
	if lower == "mlflow" {
		return "", serviceEntry{}, fmt.Errorf("%s", mlflowRedirectMsg)
	}

	// Resolve alias to canonical name.
	if canonical, ok := serviceAliases[lower]; ok {
		lower = canonical
	}

	for _, svc := range knownServices {
		if svc.Name == lower {
			return lower, svc, nil
		}
	}

	// Build list of valid names including aliases.
	validNames := make([]string, 0, len(knownServices)+len(serviceAliases))
	for _, svc := range knownServices {
		validNames = append(validNames, svc.Name)
	}
	for alias := range serviceAliases {
		validNames = append(validNames, alias)
	}
	return "", serviceEntry{}, fmt.Errorf("unknown service %q\nValid names: %s", input, strings.Join(validNames, ", "))
}

// readEnvValues reads key=value pairs from the given file.
// Returns an empty map (not an error) if the file does not exist.
func readEnvValues(filename string) (map[string]string, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	values, err := godotenv.Read(filename)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}
	return values, nil
}

// setEnvKeyInFile reads filename, replaces or appends KEY=value, and writes it back.
// Preserves comments and line ordering. Safe for files that do not yet exist.
func setEnvKeyInFile(filename, key, value string) error {
	// Read existing content (if any).
	var lines []string
	if _, err := os.Stat(filename); err == nil {
		f, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("opening %s: %w", filename, err)
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading %s: %w", filename, err)
		}
	}

	// Search for an existing line that sets this key.
	prefix := key + "="
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Match KEY=... lines (skip comments).
		if strings.HasPrefix(trimmed, prefix) || trimmed == key {
			lines[i] = key + "=" + config.QuoteEnvValue(value)
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, key+"="+config.QuoteEnvValue(value))
	}

	// Write back. Env files must be 0600 (user-readable only).
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}
	if err := security.EnforceFilePermissions(filename, 0600); err != nil {
		return fmt.Errorf("enforcing permissions on %s: %w", filename, err)
	}
	return nil
}

// --- run functions ---

// serviceStatusRow holds data for a single row in the service list output.
type serviceStatusRow struct {
	Service string `json:"service"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
	Port    string `json:"port"`
	EnvVar  string `json:"env_var"`
}

func runServiceList(cmd *cobra.Command, args []string) error {
	envFlag, _ := cmd.Flags().GetString("env")
	jsonOut, _ := cmd.Flags().GetBool("json")

	envFile, err := resolveEnvFile(envFlag)
	if err != nil {
		return err
	}

	values, err := readEnvValues(envFile)
	if err != nil {
		return err
	}

	// Emit deprecation warnings for legacy env vars.
	if v, ok := values["MAIL_ENABLED"]; ok && strings.ToLower(strings.TrimSpace(v)) == "true" {
		fmt.Fprintln(os.Stderr, "DEPRECATED: MAIL_ENABLED is deprecated; use EMAIL_ENABLED (or MAILPIT_ENABLED)")
	}
	if v, ok := values["MLFLOW_ENABLED"]; ok && strings.ToLower(strings.TrimSpace(v)) == "true" {
		fmt.Fprintln(os.Stderr, "DEPRECATED: MLFLOW_ENABLED is no longer an optional service. Run:")
		fmt.Fprintln(os.Stderr, "  nself plugin install mlflow")
		fmt.Fprintln(os.Stderr, "Then remove MLFLOW_ENABLED from your .env")
	}

	rows := make([]serviceStatusRow, 0, len(knownServices))
	for _, svc := range knownServices {
		enabled := false
		if v, ok := values[svc.EnvVar]; ok && strings.ToLower(strings.TrimSpace(v)) == "true" {
			enabled = true
		}
		status := "disabled"
		if enabled {
			status = "enabled"
		}
		enabledStr := "no"
		if enabled {
			enabledStr = "yes"
		}

		portStr := fmt.Sprintf("%d", svc.Port)
		if svc.Name == "monitoring" {
			portStr = "multi"
		} else if svc.Port == 0 {
			portStr = "N/A"
		}

		_ = enabledStr // used in table below
		rows = append(rows, serviceStatusRow{
			Service: svc.Name,
			Enabled: enabled,
			Status:  status,
			Port:    portStr,
			EnvVar:  svc.EnvVar,
		})
	}

	if jsonOut {
		return ui.PrintJSON(rows)
	}

	tbl := ui.NewTable("SERVICE", "STATUS", "ENABLED", "PORT", "ENV_VAR")
	for _, r := range rows {
		enabledStr := "no"
		if r.Enabled {
			enabledStr = "yes"
		}
		tbl.AddRow(r.Service, r.Status, enabledStr, r.Port, r.EnvVar)
	}
	tbl.Render()

	return nil
}

func runServiceEnable(cmd *cobra.Command, args []string) error {
	envFlag, _ := cmd.Flags().GetString("env")

	canonName, svc, err := canonicalServiceName(args[0])
	if err != nil {
		return err
	}

	envFile, err := resolveEnvFile(envFlag)
	if err != nil {
		return err
	}

	// Check current state — idempotency guard.
	values, err := readEnvValues(envFile)
	if err != nil {
		return err
	}
	if v, ok := values[svc.EnvVar]; ok && strings.ToLower(strings.TrimSpace(v)) == "true" {
		fmt.Printf("%s is already enabled\n", canonName)
		return nil
	}

	if err := setEnvKeyInFile(envFile, svc.EnvVar, "true"); err != nil {
		return fmt.Errorf("enabling %s: %w", canonName, err)
	}

	fmt.Printf("%s enabled. Run `nself build` to apply changes.\n", canonName)
	return nil
}

func runServiceDisable(cmd *cobra.Command, args []string) error {
	envFlag, _ := cmd.Flags().GetString("env")

	canonName, svc, err := canonicalServiceName(args[0])
	if err != nil {
		return err
	}

	envFile, err := resolveEnvFile(envFlag)
	if err != nil {
		return err
	}

	// Dependency warning: check if any enabled services depend on this one.
	if deps, ok := serviceDependents[canonName]; ok {
		values, _ := readEnvValues(envFile)
		var activeDeps []string
		for _, dep := range deps {
			for _, depSvc := range knownServices {
				if depSvc.Name == dep {
					if v, ok := values[depSvc.EnvVar]; ok && strings.ToLower(strings.TrimSpace(v)) == "true" {
						activeDeps = append(activeDeps, dep)
					}
				}
			}
		}
		if len(activeDeps) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: the following services depend on %s: %s\n", canonName, strings.Join(activeDeps, ", "))
			fmt.Fprint(os.Stderr, "Continue? [y/N] ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(strings.TrimSpace(response)) != "y" {
				fmt.Fprintln(os.Stderr, "Aborted.")
				return nil
			}
		}
	}

	if err := setEnvKeyInFile(envFile, svc.EnvVar, "false"); err != nil {
		return fmt.Errorf("disabling %s: %w", canonName, err)
	}

	fmt.Printf("%s disabled. Run `nself build` to apply changes.\n", canonName)
	return nil
}

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

// ---------------------------------------------------------------------------
// service start / stop / restart / ps / update / scale
// ---------------------------------------------------------------------------

func runServiceStart(cmd *cobra.Command, args []string) error {
	return service.Start(cmd.Context(), args[0])
}

func runServiceStop(cmd *cobra.Command, args []string) error {
	return service.Stop(cmd.Context(), args[0])
}

func runServiceRestart(cmd *cobra.Command, args []string) error {
	return service.Restart(cmd.Context(), args[0])
}

func runServicePs(cmd *cobra.Command, args []string) error {
	entries, err := service.PS(cmd.Context())
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No services running. Run `nself start` to boot the stack.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSERVICE\tSTATUS\tHEALTH\tID")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.Name, e.Service, e.Status, e.Health, e.ID)
	}
	return w.Flush()
}

func runServiceUpdate(cmd *cobra.Command, args []string) error {
	return service.Update(cmd.Context(), args[0])
}

func runServiceScale(cmd *cobra.Command, args []string) error {
	var replicas int
	if _, err := fmt.Sscanf(args[1], "%d", &replicas); err != nil {
		return fmt.Errorf("replicas must be an integer, got %q", args[1])
	}
	return service.Scale(cmd.Context(), args[0], replicas)
}
