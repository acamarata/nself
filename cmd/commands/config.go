package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"nself/internal/config"
	"nself/internal/security"
	"nself/internal/ui"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// configCmd is the parent command for all configuration management.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage project configuration",
	Long: `Manage configuration with interactive wizard, environment switching,
secret rotation, and validation.

Subcommands:
  config show      Display all key=value pairs (masked by default)
  config get       Get a single configuration value
  config set       Update a configuration value
  config list      List all known config keys with current values
  config validate  Validate configuration against all registered rules
  config export    Export config to a file
  config import    Import config from a file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// --- Subcommand declarations ---

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all config key=value pairs (masked by default)",
	Long: `Load the .env file (or .env.{env} when --env is set) and display all
key=value pairs sorted alphabetically. Secret values are masked as **** by
default; use --reveal to print plaintext values.

Secret keys are any key whose name contains: SECRET, PASSWORD, KEY, TOKEN.`,
	RunE: runConfigShow,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a single configuration value",
	Long: `Return the raw value of KEY from .env (or .env.{env} with --env).
Output is the plain value only, making it safe for scripting.

Exits non-zero if the key is not found.`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Update a configuration value (writes to .env)",
	Long: `Update KEY=VALUE in .env (or .env.{env} with --env) in-place.
If the key does not exist, it is appended to the end of the file.
Existing comments and formatting are preserved.

NOTE: config set always writes to .env, not .env.dev or other cascade files.`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all known config keys with current values",
	Long: `Display a table of KEY | VALUE | SOURCE for every known configuration
variable. Unknown keys are omitted. Keys with no value show the default or
(unset). Use --env to select the environment file.`,
	RunE: runConfigList,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration against all registered rules",
	Long: `Load config and run all registered validators. Failures are printed
to stderr.

Exits 0 when all validators pass; exits 1 if any fail.`,
	RunE: runConfigValidate,
}

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export current config to a file or stdout",
	Long: `Write all key=value pairs from .env (or .env.{env}) to --output file
or stdout. Use --format to select env (default), json, or yaml output.
Secret values are included in the export. Use standard file permissions
to protect the output file.`,
	Args: cobra.NoArgs,
	RunE: runConfigExport,
}

var configImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import config from a file into .env",
	Long: `Read key=value pairs from FILE and merge them into .env (or .env.{env}).
Keys that already exist are overwritten; new keys are appended.
Prompts for confirmation before overwriting differing keys.
Use --force to skip the prompt.`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigImport,
}

// --- init: register commands and flags ---

func init() {
	// Persistent flags inherited by all subcommands.
	configCmd.PersistentFlags().String("env", "", "Target environment (reads .env.{env})")
	configCmd.PersistentFlags().Bool("reveal", false, "Show secret values in plaintext")
	configCmd.PersistentFlags().Bool("json", false, "JSON output")

	// Show-specific flags.
	configShowCmd.Flags().String("format", "table", "Output format: table|yaml|json")

	// Import-specific flags.
	configImportCmd.Flags().Bool("force", false, "Skip confirmation prompt when overwriting keys")
	configImportCmd.Flags().Bool("dry-run", false, "Show what would change without writing")

	// Export-specific flags.
	configExportCmd.Flags().String("format", "env", "Output format: env|json|yaml")
	configExportCmd.Flags().String("output", "", "Output file path (default: stdout)")

	// Wire all subcommands to parent.
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)

	RootCmd.AddCommand(configCmd)
}

// --- Helpers ---

// secretKeyParts contains substrings whose presence in a key name indicates
// a secret value that should be masked by default.
var secretKeyParts = []string{"SECRET", "PASSWORD", "KEY", "TOKEN"}

// isSecretKey returns true when the key name contains any of the secret
// indicator substrings (case-insensitive comparison against uppercased key).
func isSecretKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, part := range secretKeyParts {
		if strings.Contains(upper, part) {
			return true
		}
	}
	return false
}

// maskValue returns "****" when reveal is false and the key is a secret key.
func maskValue(key, value string, reveal bool) string {
	if !reveal && isSecretKey(key) && value != "" {
		return "***"
	}
	return value
}

// envFileName returns the .env filename to use for the given env flag value.
// An empty envFlag means use the plain ".env" file.
func envFileName(projectDir, envFlag string) string {
	if envFlag == "" {
		return filepath.Join(projectDir, ".env")
	}
	return filepath.Join(projectDir, ".env."+envFlag)
}

// resolveProjectDir returns the nself project root, used by all config
// subcommands that need to locate .env files.
func resolveProjectDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	dir, err := config.FindNSelfRoot(cwd)
	if err != nil {
		return "", fmt.Errorf("no nself project found in current directory or parents — run 'nself init' first")
	}
	return dir, nil
}

// --- S4-T01: config show ---

func runConfigShow(cmd *cobra.Command, args []string) error {
	reveal, _ := cmd.Flags().GetBool("reveal")
	envFlag, _ := cmd.Flags().GetString("env")
	format, _ := cmd.Flags().GetString("format")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	envFile := envFileName(projectDir, envFlag)
	pairs, err := godotenv.Read(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("env file not found: %s", envFile)
		}
		return fmt.Errorf("reading %s: %w", envFile, err)
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	switch format {
	case "yaml":
		for _, k := range keys {
			v := maskValue(k, pairs[k], reveal)
			if strings.ContainsAny(v, ": \t#\"'\\") || v == "" {
				fmt.Printf("%s: %q\n", k, v)
			} else {
				fmt.Printf("%s: %s\n", k, v)
			}
		}
	case "json":
		m := make(map[string]string, len(pairs))
		for _, k := range keys {
			m[k] = maskValue(k, pairs[k], reveal)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	default: // table
		for _, k := range keys {
			fmt.Printf("%s=%s\n", k, maskValue(k, pairs[k], reveal))
		}
	}
	return nil
}

// --- S4-T02: config get ---

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	reveal, _ := cmd.Flags().GetBool("reveal")
	envFlag, _ := cmd.Flags().GetString("env")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	envFile := envFileName(projectDir, envFlag)
	pairs, err := godotenv.Read(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("env file not found: %s", envFile)
		}
		return fmt.Errorf("reading %s: %w", envFile, err)
	}

	val, ok := pairs[key]
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}

	fmt.Println(maskValue(key, val, reveal))
	return nil
}

// --- S4-T03: config set ---

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]
	envFlag, _ := cmd.Flags().GetString("env")

	// Validate key: only [A-Z0-9_] allowed, max 128 chars.
	if err := validateConfigKey(key); err != nil {
		return err
	}
	// Validate value: no null bytes.
	if err := validateConfigValue(value); err != nil {
		return err
	}

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	envFile := envFileName(projectDir, envFlag)

	// Read the existing file line by line and update in-place.
	// If the key doesn't exist, append it.
	updated, err := setEnvFileLine(envFile, key, value)
	if err != nil {
		return err
	}

	if updated {
		ui.Success(fmt.Sprintf("Updated %s in %s", key, filepath.Base(envFile)))
	} else {
		ui.Success(fmt.Sprintf("Added %s to %s", key, filepath.Base(envFile)))
	}
	return nil
}

// configKeyRe matches valid config keys: uppercase letters, digits, underscores only.
var configKeyRe = regexp.MustCompile(`^[A-Z0-9_]+$`)

// validateConfigKey returns an error if key contains invalid characters,
// is empty, or exceeds 128 characters.
func validateConfigKey(key string) error {
	if len(key) == 0 {
		return fmt.Errorf("config key cannot be empty")
	}
	if len(key) > 128 {
		return fmt.Errorf("config key too long (max 128 characters)")
	}
	if !configKeyRe.MatchString(key) {
		return fmt.Errorf("config key %q contains invalid characters (only A-Z, 0-9, _ allowed)", key)
	}
	return nil
}

// validateConfigValue returns an error if value contains null bytes.
func validateConfigValue(value string) error {
	for i, b := range []byte(value) {
		if b == 0 {
			return fmt.Errorf("config value contains null byte at position %d", i)
		}
	}
	return nil
}

// setEnvFileLine updates KEY=VALUE in the named file in-place, preserving all
// comments and blank lines. If the key is not found it is appended. Returns
// true when an existing line was replaced, false when the key was appended.
func setEnvFileLine(envFile, key, value string) (updated bool, err error) {
	// quoteIfNeeded wraps value in double-quotes when it contains spaces,
	// special characters, or is empty.
	quoteIfNeeded := func(v string) string {
		if v == "" || strings.ContainsAny(v, " \t#\"'\\$`") {
			return `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
		}
		return v
	}
	newLine := key + "=" + quoteIfNeeded(value)

	// If the file does not exist, create it with just this key.
	if _, statErr := os.Stat(envFile); os.IsNotExist(statErr) {
		if writeErr := os.WriteFile(envFile, []byte(newLine+"\n"), 0600); writeErr != nil {
			return false, fmt.Errorf("creating %s: %w", envFile, writeErr)
		}
		if chmodErr := security.EnforceFilePermissions(envFile, 0600); chmodErr != nil {
			return false, fmt.Errorf("enforcing permissions on %s: %w", envFile, chmodErr)
		}
		return false, nil
	}

	// Read existing file.
	f, openErr := os.Open(envFile)
	if openErr != nil {
		return false, fmt.Errorf("opening %s: %w", envFile, openErr)
	}
	defer f.Close()

	var lines []string
	found := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		// Skip comment and blank lines.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lines = append(lines, line)
			continue
		}
		// Match KEY= or KEY =
		eqIdx := strings.IndexByte(trimmed, '=')
		if eqIdx > 0 {
			lineKey := strings.TrimSpace(trimmed[:eqIdx])
			if lineKey == key {
				lines = append(lines, newLine)
				found = true
				continue
			}
		}
		lines = append(lines, line)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return false, fmt.Errorf("scanning %s: %w", envFile, scanErr)
	}
	f.Close()

	if !found {
		lines = append(lines, newLine)
	}

	content := strings.Join(lines, "\n") + "\n"
	if writeErr := os.WriteFile(envFile, []byte(content), 0600); writeErr != nil {
		return false, fmt.Errorf("writing %s: %w", envFile, writeErr)
	}
	if chmodErr := security.EnforceFilePermissions(envFile, 0600); chmodErr != nil {
		return false, fmt.Errorf("enforcing permissions on %s: %w", envFile, chmodErr)
	}
	return found, nil
}

// --- S4-T04: config list ---

// knownVarsExported mirrors loader.go's knownEnvVars. We access it through the
// exported function below rather than duplicating the list.

func runConfigList(cmd *cobra.Command, args []string) error {
	envFlag, _ := cmd.Flags().GetString("env")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	envFile := envFileName(projectDir, envFlag)

	// Read the env file if it exists. Missing file is not an error for list.
	var pairs map[string]string
	if _, statErr := os.Stat(envFile); statErr == nil {
		pairs, err = godotenv.Read(envFile)
		if err != nil {
			return fmt.Errorf("reading %s: %w", envFile, err)
		}
	} else {
		pairs = make(map[string]string)
	}

	known := config.KnownEnvVars()

	// Print header.
	fmt.Printf("%-45s %-30s %s\n", "KEY", "VALUE", "SOURCE")
	fmt.Println(strings.Repeat("-", 90))

	for _, k := range known {
		val, ok := pairs[k]
		source := filepath.Base(envFile)
		displayVal := val
		if !ok || val == "" {
			displayVal = config.DefaultFor(k)
			if displayVal != "" {
				displayVal = "(default: " + displayVal + ")"
				source = "default"
			} else {
				displayVal = "(unset)"
				source = "unset"
			}
		} else {
			displayVal = maskValue(k, val, false)
		}
		fmt.Printf("%-45s %-30s %s\n", k, displayVal, source)
	}
	return nil
}

// --- S4-T05: config validate ---

func runConfigValidate(cmd *cobra.Command, args []string) error {
	envFlag, _ := cmd.Flags().GetString("env")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	// For validate, use the cascade loader so all defaults apply.
	// Temporarily honour --env by setting ENV if provided.
	if envFlag != "" {
		os.Setenv("ENV", envFlag)
	}

	cfg, err := config.Load(projectDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Run validators individually to report per-validator pass/fail.
	results := config.RunAllWithResults(cfg)
	errorCount := 0
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %s\n", r.Name, r.Err.Error())
			errorCount++
		} else {
			fmt.Printf("[PASS] %s\n", r.Name)
		}
	}

	if errorCount == 0 {
		fmt.Println("config OK")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n%d error(s) found\n", errorCount)
	return fmt.Errorf("one or more validators failed")
}

// --- S4-T06: config export ---

func runConfigExport(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile == "" && len(args) > 0 {
		outputFile = args[0]
	}
	reveal, _ := cmd.Flags().GetBool("reveal")
	envFlag, _ := cmd.Flags().GetString("env")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	envFile := envFileName(projectDir, envFlag)
	pairs, err := godotenv.Read(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("env file not found: %s", envFile)
		}
		return fmt.Errorf("reading %s: %w", envFile, err)
	}

	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder

	switch format {
	case "json":
		m := make(map[string]string, len(pairs))
		for _, k := range keys {
			m[k] = maskValue(k, pairs[k], reveal)
		}
		data, jsonErr := json.MarshalIndent(m, "", "  ")
		if jsonErr != nil {
			return fmt.Errorf("encoding json: %w", jsonErr)
		}
		sb.Write(data)
		sb.WriteString("\n")
	case "yaml":
		for _, k := range keys {
			v := maskValue(k, pairs[k], reveal)
			if strings.ContainsAny(v, ": \t#\"'\\") || v == "" {
				sb.WriteString(fmt.Sprintf("%s: %q\n", k, v))
			} else {
				sb.WriteString(fmt.Sprintf("%s: %s\n", k, v))
			}
		}
	default: // env
		sb.WriteString("# Exported by nself config export\n")
		for _, k := range keys {
			sb.WriteString(k + "=" + maskValue(k, pairs[k], reveal) + "\n")
		}
	}

	output := sb.String()

	if outputFile != "" {
		if writeErr := os.WriteFile(outputFile, []byte(output), 0600); writeErr != nil {
			return fmt.Errorf("writing export file %s: %w", outputFile, writeErr)
		}
		ui.Success(fmt.Sprintf("Config exported to %s (%d keys)", outputFile, len(keys)))
	} else {
		fmt.Print(output)
	}

	return nil
}

// --- config import ---

func runConfigImport(cmd *cobra.Command, args []string) error {
	srcFile := args[0]
	envFlag, _ := cmd.Flags().GetString("env")
	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	projectDir, err := resolveProjectDir()
	if err != nil {
		return err
	}

	// Read source file.
	incoming, err := godotenv.Read(srcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("import file not found: %s", srcFile)
		}
		return fmt.Errorf("reading import file %s: %w", srcFile, err)
	}

	envFile := envFileName(projectDir, envFlag)

	// Read current env file (may not exist yet).
	current := make(map[string]string)
	if _, statErr := os.Stat(envFile); statErr == nil {
		current, err = godotenv.Read(envFile)
		if err != nil {
			return fmt.Errorf("reading %s: %w", envFile, err)
		}
	}

	if dryRun {
		// Sort keys for deterministic output.
		incomingKeys := make([]string, 0, len(incoming))
		for k := range incoming {
			incomingKeys = append(incomingKeys, k)
		}
		sort.Strings(incomingKeys)

		for _, k := range incomingKeys {
			newVal := incoming[k]
			if existing, ok := current[k]; ok {
				if existing != newVal {
					fmt.Printf("update %s: %s -> %s\n", k, maskValue(k, existing, false), maskValue(k, newVal, false))
				}
			} else {
				fmt.Printf("add %s=%s\n", k, maskValue(k, newVal, false))
			}
		}
		fmt.Println("(dry run - no changes written)")
		return nil
	}

	// Find keys that would be overwritten with a different value.
	var conflicts []string
	for k, newVal := range incoming {
		if existing, ok := current[k]; ok && existing != newVal {
			conflicts = append(conflicts, k)
		}
	}
	sort.Strings(conflicts)

	if len(conflicts) > 0 && !force {
		fmt.Fprintf(cmd.ErrOrStderr(), "The following keys in %s will be overwritten:\n", filepath.Base(envFile))
		for _, k := range conflicts {
			fmt.Fprintf(cmd.ErrOrStderr(), "  %s: %s -> %s\n", k, maskValue(k, current[k], false), maskValue(k, incoming[k], false))
		}
		fmt.Fprint(cmd.ErrOrStderr(), "Continue? [y/N] ")
		var response string
		if _, scanErr := fmt.Scanln(&response); scanErr != nil || strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Fprintln(cmd.ErrOrStderr(), "Import cancelled.")
			return nil
		}
	}

	// Apply all incoming keys into the target file.
	newCount := 0
	updateCount := 0
	for k, v := range incoming {
		updated, setErr := setEnvFileLine(envFile, k, v)
		if setErr != nil {
			return setErr
		}
		if updated {
			updateCount++
		} else {
			newCount++
		}
	}

	ui.Success(fmt.Sprintf("Import complete: %d updated, %d added to %s", updateCount, newCount, filepath.Base(envFile)))
	return nil
}
