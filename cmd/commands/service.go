package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nself-org/cli/internal/security"
	"github.com/nself-org/cli/internal/ui"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// serviceEntry describes a single optional service and its governing env var.
type serviceEntry struct {
	Name   string
	EnvVar string
	Port   int // 0 means N/A (or "multi" for monitoring)
}

// knownServices is the ordered list of optional services managed by `nself service`.
var knownServices = []serviceEntry{
	{Name: "redis", EnvVar: "REDIS_ENABLED", Port: 6379},
	{Name: "minio", EnvVar: "MINIO_ENABLED", Port: 9000},
	{Name: "mailpit", EnvVar: "MAILPIT_ENABLED", Port: 8025},
	{Name: "functions", EnvVar: "FUNCTIONS_ENABLED", Port: 3008},
	{Name: "search", EnvVar: "SEARCH_ENABLED", Port: 7700},
	{Name: "mlflow", EnvVar: "MLFLOW_ENABLED", Port: 5000},
	{Name: "monitoring", EnvVar: "MONITORING_ENABLED", Port: 0},
	{Name: "admin", EnvVar: "NSELF_ADMIN_ENABLED", Port: 3021},
}

// serviceAliases maps alternate names to canonical service names.
var serviceAliases = map[string]string{
	"storage":     "minio",
	"email":       "mailpit",
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
	Long: `Enable, disable, and list optional nSelf services.

Available services:
  redis        Cache and queue (REDIS_ENABLED)
  minio        S3-compatible storage (MINIO_ENABLED), alias: storage
  mailpit      Local email testing (MAILPIT_ENABLED), alias: email
  functions    Serverless runtime (FUNCTIONS_ENABLED)
  search       Full-text search (SEARCH_ENABLED), alias: meilisearch
  mlflow       ML experiment tracking (MLFLOW_ENABLED)
  monitoring   Prometheus/Grafana stack (MONITORING_ENABLED)
  admin        nSelf Admin GUI (NSELF_ADMIN_ENABLED)

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

func init() {
	serviceCmd.PersistentFlags().String("env", "", "Target environment (reads .env.{env})")
	serviceListCmd.Flags().Bool("json", false, "Output as JSON array")

	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceEnableCmd)
	serviceCmd.AddCommand(serviceDisableCmd)

	RootCmd.AddCommand(serviceCmd)
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

// canonicalServiceName resolves aliases and validates the service name.
// Returns the canonical name and the serviceEntry, or an error if unknown.
func canonicalServiceName(input string) (string, serviceEntry, error) {
	lower := strings.ToLower(strings.TrimSpace(input))

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
			lines[i] = key + "=" + value
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, key+"="+value)
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
