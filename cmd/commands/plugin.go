package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage nSelf plugins",
	Long: `Plugin lifecycle management: list, install, remove, update, start, stop, and status.

Unknown subcommands are proxied to the matching plugin binary:
  nself plugin <name> <action>  →  nself-<name> <action>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If args are present and don't match a subcommand, proxy to the plugin.
		if len(args) >= 1 {
			pluginName := args[0]
			pluginArgs := args[1:]
			return plugin.ProxyCommand(pluginName, pluginArgs)
		}
		return cmd.Help()
	},
}

// --- subcommands ---

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available and installed plugins",
	RunE:  runPluginList,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <plugin> [plugin...]",
	Short: "Install one or more plugins (license check for pro)",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runPluginInstall,
}

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginRemove,
}

var pluginUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update a specific plugin or all plugins",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPluginUpdate,
}

var pluginUpdatesCmd = &cobra.Command{
	Use:   "updates",
	Short: "Check for available plugin updates",
	RunE:  runPluginUpdates,
}

var pluginRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Force refresh the registry cache",
	RunE:  runPluginRefresh,
}

var pluginStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a plugin service",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginStart,
}

var pluginStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a plugin service",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginStop,
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a plugin (excluded from compose on next build)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginDisable,
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Re-enable a previously disabled plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginEnable,
}

var pluginStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show plugin status",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPluginStatus,
}

var pluginInventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "List installed plugins with version, tier, and status",
	RunE:  runPluginInventory,
}

func init() {
	// Flags on the list subcommand.
	pluginListCmd.Flags().Bool("installed", false, "Show only installed plugins")
	pluginListCmd.Flags().Bool("detailed", false, "Show detailed information")
	pluginListCmd.Flags().String("category", "", "Filter by category")
	pluginListCmd.Flags().Bool("show-eol", false, "Include EOL plugins in listing (hidden by default)") // S58-T03

	// Flags on install.
	pluginInstallCmd.Flags().String("key", "", "License key for pro plugins")
	pluginInstallCmd.Flags().String("version", "", "Install a specific version")
	pluginInstallCmd.Flags().Bool("force", false, "Required when using NSELF_LICENSE_SKIP_VERIFY; explicit acknowledgment of skipped validation")
	pluginInstallCmd.Flags().Bool("allow-eol", false, "Allow installing an EOL plugin (not recommended)") // S58-T03

	// Flags on update.
	pluginUpdateCmd.Flags().Bool("allow-eol", false, "Allow updating to/from an EOL plugin (not recommended)") // S58-T03

	// Flags on remove.
	pluginRemoveCmd.Flags().Bool("keep-data", false, "Preserve database data on remove")
	pluginRemoveCmd.Flags().Bool("force", false, "Remove even if other plugins depend on this one")

	// Flags on status.
	pluginStatusCmd.Flags().Bool("detailed", false, "Show detailed status")

	// Register subcommands.
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginUpdateCmd)
	pluginCmd.AddCommand(pluginUpdatesCmd)
	pluginCmd.AddCommand(pluginRefreshCmd)
	pluginCmd.AddCommand(pluginStartCmd)
	pluginCmd.AddCommand(pluginStopCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginStatusCmd)
	pluginCmd.AddCommand(pluginInventoryCmd)

	RootCmd.AddCommand(pluginCmd)
}

// --- helpers ---

// resolvePluginDir returns the plugin directory from config or a sensible default.
func resolvePluginDir() string {
	if d := os.Getenv("NSELF_PLUGIN_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".nself", "plugins")
	}
	return filepath.Join(home, ".nself", "plugins")
}

// loadConfig loads the project config from the current working directory.
func loadConfig() (*config.Config, error) {
	workdir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	cfg, err := config.Load(workdir)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

// --- run functions ---

func runPluginList(cmd *cobra.Command, args []string) error {
	installed, _ := cmd.Flags().GetBool("installed")
	detailed, _ := cmd.Flags().GetBool("detailed")
	category, _ := cmd.Flags().GetString("category")
	showEOL, _ := cmd.Flags().GetBool("show-eol") // S58-T03

	pluginDir := resolvePluginDir()

	plugins, err := plugin.List(pluginDir, installed)
	if err != nil {
		return fmt.Errorf("listing plugins: %w", err)
	}

	if len(plugins) == 0 {
		if installed {
			fmt.Println("No plugins installed.")
		} else {
			fmt.Println("No plugins found in registry.")
		}
		return nil
	}

	for _, p := range plugins {
		// S58-T03: EOL plugins are hidden from listing unless --show-eol is set.
		if p.PublishStatus == "eol" && !showEOL {
			continue
		}
		if category != "" && !strings.EqualFold(p.Category, category) {
			continue
		}
		stateTag := ""
		if p.Running {
			stateTag = " [running]"
		} else if p.Installed {
			stateTag = " [installed]"
		}
		// S58-T03: Full status badge for all 6 lifecycle values.
		statusBadge := ""
		switch p.PublishStatus {
		case "experimental":
			statusBadge = " [experimental]"
		case "planned":
			statusBadge = " [planned]"
		case "beta":
			statusBadge = " [beta]"
		case "deprecated":
			statusBadge = " [DEPRECATED]"
		case "eol":
			statusBadge = " [EOL]"
		// "stable" and "" show no badge — clean install signal
		}
		if detailed {
			fmt.Printf("%-20s %-10s %-12s%s%s\n", p.Name, p.Version, p.Category, stateTag, statusBadge)
		} else {
			fmt.Printf("%-20s%s%s\n", p.Name, stateTag, statusBadge)
		}
	}

	return nil
}

func runPluginInstall(cmd *cobra.Command, args []string) error {
	key, _ := cmd.Flags().GetString("key")
	force, _ := cmd.Flags().GetBool("force")
	allowEOL, _ := cmd.Flags().GetBool("allow-eol") // S58-T03

	// Security gate: NSELF_LICENSE_SKIP_VERIFY=1 requires --force as explicit acknowledgment.
	// Standalone skip (without --force) is rejected to prevent accidental bypass in scripts.
	if os.Getenv("NSELF_LICENSE_SKIP_VERIFY") == "1" && !force {
		return fmt.Errorf("NSELF_LICENSE_SKIP_VERIFY requires --force flag; standalone skip is not permitted")
	}
	if os.Getenv("NSELF_LICENSE_SKIP_VERIFY") == "1" && force {
		fmt.Fprintf(os.Stderr, "warning: license verification bypassed via NSELF_LICENSE_SKIP_VERIFY (--force acknowledged)\n")
	}

	// If a license key is provided via flag, set it in the environment
	// so the plugin manager's license check picks it up.
	if key != "" {
		if err := os.Setenv("NSELF_PLUGIN_LICENSE_KEY", key); err != nil {
			return fmt.Errorf("setting license key: %w", err)
		}
	}

	ctx := context.Background()
	registryURL := os.Getenv("NSELF_PLUGIN_REGISTRY")
	if err := plugin.ValidateNetworkAccess(ctx, registryURL); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// S20: Free-plugin account flow.
	// If no license key is set and all requested plugins are free, prompt the
	// user to create a nself_free_ account for rate-limiting + upsell tracking.
	// Skip when NSELF_LICENSE_SKIP_VERIFY=1 (offline/CI) or when a key already exists.
	pingURL := os.Getenv("NSELF_PING_URL")
	if pingURL == "" {
		pingURL = "https://ping.nself.org"
	}
	existingKey := os.Getenv("NSELF_PLUGIN_LICENSE_KEY")
	if existingKey == "" && os.Getenv("NSELF_LICENSE_SKIP_VERIFY") != "1" {
		if err := maybeRegisterFreeAccount(ctx, pingURL, args); err != nil {
			// Non-fatal: free account creation failure never blocks plugin install.
			fmt.Fprintf(os.Stderr, "note: could not create free account (%v); continuing without one\n", err)
		}
	}

	pluginDir := resolvePluginDir()

	// Install each named plugin. Collect per-plugin errors so that a failure on
	// one plugin does not abort the remaining installs.
	var failures []string
	installedCount := 0
	for _, name := range args {
		// S58-T03: EOL gate — check status before attempting install.
		if eolErr := plugin.CheckEOLBlock(ctx, name, allowEOL); eolErr != nil {
			fmt.Fprintf(os.Stderr, "  %v\n", eolErr)
			failures = append(failures, name)
			continue
		}

		fmt.Fprintf(os.Stderr, "Installing plugin %q...\n", name)
		if err := plugin.Install(ctx, cfg, name, pluginDir); err != nil {
			fmt.Fprintf(os.Stderr, "  error installing %q: %v\n", name, err)
			failures = append(failures, name)
			continue
		}
		fmt.Fprintf(os.Stderr, "Plugin %q installed successfully.\n", name)
		installedCount++
		printPluginPostInstallHint(name)

		// S20: Fire install telemetry for free-tier keys (non-blocking).
		currentKey := os.Getenv("NSELF_PLUGIN_LICENSE_KEY")
		if plugin.IsFreeKey(currentKey) {
			plugin.SendFreeInstallTelemetry(pingURL, currentKey, name)
		}
	}

	// S20: Upsell prompt after 3rd successful free-plugin install.
	if installedCount > 0 {
		maybeShowFreeUpsell(installedCount)
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to install: %s", strings.Join(failures, ", "))
	}
	return nil
}

// maybeRegisterFreeAccount prompts the user to create a free account if they
// do not yet have a license key and are installing only free plugins.
// The generated key is stored in NSELF_PLUGIN_LICENSE_KEY for this session.
func maybeRegisterFreeAccount(ctx context.Context, pingURL string, pluginNames []string) error {
	// Only prompt when all requested plugins are free (not in paidPlugins).
	// If any pro plugin is in the list, the normal license flow handles it.
	allFree := true
	for _, name := range pluginNames {
		if plugin.IsPaidPlugin(name) {
			allFree = false
			break
		}
	}
	if !allFree {
		return nil
	}

	ui.Info("To track your installs and enable usage insights, nSelf uses a free account key.")
	fmt.Fprintf(os.Stderr, "Create a free account? [Y/n] ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil // EOF or no tty — skip silently
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if answer != "" && answer != "y" && answer != "yes" {
		return nil // user declined
	}

	key, err := plugin.RegisterFreeAccount(ctx, pingURL)
	if err != nil {
		return err
	}

	// Store for this session.
	if err := os.Setenv("NSELF_PLUGIN_LICENSE_KEY", key); err != nil {
		return fmt.Errorf("storing free key in env: %w", err)
	}

	ui.Success("Free account created. Run `nself license set " + key + "` to persist it.")
	return nil
}

// freeInstallCounterFile returns the path to the file tracking how many free
// plugins this device has installed (for upsell threshold).
func freeInstallCounterFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nself", "free_install_count")
}

// maybeShowFreeUpsell increments the free-install counter and shows a one-line
// upsell prompt when the 3rd install is reached (items 679/686 from S20).
func maybeShowFreeUpsell(justInstalled int) {
	counterPath := freeInstallCounterFile()

	// Read existing count.
	existing := 0
	if data, err := os.ReadFile(counterPath); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			existing = n
		}
	}

	newCount := existing + justInstalled

	// Write updated count (best-effort; log at debug if it fails).
	_ = os.MkdirAll(filepath.Dir(counterPath), 0700)
	if err := os.WriteFile(counterPath, []byte(strconv.Itoa(newCount)), 0600); err != nil {
		ui.Warn("plugin counter write failed: " + err.Error())
	}

	// Show upsell on reaching (or crossing) the threshold.
	if existing < 3 && newCount >= 3 {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "You've installed 3 free plugins. Unlock all 87 plugins with nSelf+ for $3.99/mo.")
		fmt.Fprintln(os.Stderr, "  nself.org/plus")
		fmt.Fprintln(os.Stderr, "")
	}
}

func runPluginRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	keepData, _ := cmd.Flags().GetBool("keep-data")
	force, _ := cmd.Flags().GetBool("force")

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	pluginDir := resolvePluginDir()
	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "Removing plugin %q...\n", name)
	if err := plugin.Remove(ctx, cfg, name, pluginDir, keepData, force); err != nil {
		return fmt.Errorf("removing plugin %q: %w", name, err)
	}

	fmt.Fprintf(os.Stderr, "Plugin %q removed successfully.\n", name)
	return nil
}

func runPluginUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	registryURL := os.Getenv("NSELF_PLUGIN_REGISTRY")
	if err := plugin.ValidateNetworkAccess(ctx, registryURL); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	pluginDir := resolvePluginDir()

	if len(args) == 1 {
		name := args[0]
		fmt.Fprintf(os.Stderr, "Updating plugin %q...\n", name)
		if err := plugin.Update(ctx, cfg, name, pluginDir); err != nil {
			return fmt.Errorf("updating plugin %q: %w", name, err)
		}
		fmt.Fprintf(os.Stderr, "Plugin %q updated successfully.\n", name)
		return nil
	}

	// Update all installed plugins.
	plugins, err := plugin.List(pluginDir, true)
	if err != nil {
		return fmt.Errorf("listing installed plugins: %w", err)
	}
	if len(plugins) == 0 {
		fmt.Println("No plugins installed.")
		return nil
	}

	var failures []string
	for _, p := range plugins {
		fmt.Fprintf(os.Stderr, "Updating %s...\n", p.Name)
		if err := plugin.Update(ctx, cfg, p.Name, pluginDir); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to update %s: %v\n", p.Name, err)
			failures = append(failures, p.Name)
			continue
		}
		fmt.Fprintf(os.Stderr, "  %s updated.\n", p.Name)
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to update: %s", strings.Join(failures, ", "))
	}
	return nil
}

func runPluginUpdates(cmd *cobra.Command, args []string) error {
	registryURL := os.Getenv("NSELF_PLUGIN_REGISTRY")
	if err := plugin.ValidateNetworkAccess(cmd.Context(), registryURL); err != nil {
		return err
	}

	pluginDir := resolvePluginDir()

	// Get installed plugins.
	installed, err := plugin.List(pluginDir, true)
	if err != nil {
		return fmt.Errorf("listing installed plugins: %w", err)
	}
	if len(installed) == 0 {
		fmt.Println("No plugins installed.")
		return nil
	}

	// Get registry plugins for version comparison.
	registry, err := plugin.List(pluginDir, false)
	if err != nil {
		return fmt.Errorf("fetching registry: %w", err)
	}

	// Build a map of registry versions.
	regVersions := make(map[string]string)
	for _, p := range registry {
		regVersions[p.Name] = p.Version
	}

	hasUpdates := false
	for _, p := range installed {
		regVer, found := regVersions[p.Name]
		if !found {
			continue
		}
		if regVer != p.Version {
			fmt.Printf("%-20s %s -> %s\n", p.Name, p.Version, regVer)
			hasUpdates = true
		}
	}

	if !hasUpdates {
		fmt.Println("All plugins are up to date.")
	}

	return nil
}

func runPluginRefresh(cmd *cobra.Command, args []string) error {
	registryURL := os.Getenv("NSELF_PLUGIN_REGISTRY")
	if err := plugin.ValidateNetworkAccess(cmd.Context(), registryURL); err != nil {
		return err
	}

	ctx := context.Background()
	cacheDir := os.Getenv("NSELF_PLUGIN_CACHE")
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			cacheDir = filepath.Join("/tmp", ".nself", "cache", "plugins")
		} else {
			cacheDir = filepath.Join(home, ".nself", "cache", "plugins")
		}
	}

	// Remove cached registry file to force a fresh fetch.
	cachePath := filepath.Join(cacheDir, "registry.json")
	_ = os.Remove(cachePath)

	fmt.Fprintln(os.Stderr, "Refreshing plugin registry...")
	_, err := plugin.FetchRegistry(ctx, "", cacheDir)
	if err != nil {
		return fmt.Errorf("refreshing registry: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Registry cache refreshed.")
	return nil
}

func runPluginStart(cmd *cobra.Command, args []string) error {
	name := args[0]
	pluginDir := resolvePluginDir()
	ctx := context.Background()

	pluginPath := filepath.Join(pluginDir, name)
	fmt.Fprintf(os.Stderr, "Starting plugin %q...\n", name)
	if err := plugin.Start(ctx, pluginPath, name); err != nil {
		return fmt.Errorf("starting plugin %q: %w", name, err)
	}

	fmt.Fprintf(os.Stderr, "Plugin %q started.\n", name)
	return nil
}

func runPluginStop(cmd *cobra.Command, args []string) error {
	name := args[0]
	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "Stopping plugin %q...\n", name)
	if err := plugin.Stop(ctx, name); err != nil {
		return fmt.Errorf("stopping plugin %q: %w", name, err)
	}

	fmt.Fprintf(os.Stderr, "Plugin %q stopped.\n", name)
	return nil
}

func runPluginInventory(cmd *cobra.Command, args []string) error {
	pluginDir := resolvePluginDir()

	plugins, err := plugin.ListInstalled(pluginDir)
	if err != nil {
		return fmt.Errorf("listing installed plugins: %w", err)
	}

	if len(plugins) == 0 {
		fmt.Println("No plugins installed. Run `nself plugin install <name>` to get started.")
		return nil
	}

	tbl := ui.NewTable("Name", "Version", "Tier", "Status", "Description")
	for _, p := range plugins {
		tbl.AddRow(p.Name, p.Version, p.Tier, p.Status, p.Description)
	}
	tbl.Render()

	return nil
}

func runPluginDisable(cmd *cobra.Command, args []string) error {
	name := args[0]
	pluginDir := resolvePluginDir()

	fmt.Fprintf(os.Stderr, "Disabling plugin %q...\n", name)
	if err := plugin.DisablePlugin(name, pluginDir); err != nil {
		return fmt.Errorf("disabling plugin %q: %w", name, err)
	}

	fmt.Fprintf(os.Stderr, "Plugin %q disabled. Run 'nself build' to update your stack.\n", name)
	return nil
}

func runPluginEnable(cmd *cobra.Command, args []string) error {
	name := args[0]
	pluginDir := resolvePluginDir()

	fmt.Fprintf(os.Stderr, "Enabling plugin %q...\n", name)
	if err := plugin.EnablePlugin(name, pluginDir); err != nil {
		return fmt.Errorf("enabling plugin %q: %w", name, err)
	}

	fmt.Fprintf(os.Stderr, "Plugin %q enabled. Run 'nself build' to update your stack.\n", name)
	return nil
}

func runPluginStatus(cmd *cobra.Command, args []string) error {
	detailed, _ := cmd.Flags().GetBool("detailed")

	if len(args) == 1 {
		name := args[0]
		st, err := plugin.Status(name)
		if err != nil {
			return fmt.Errorf("getting status for %q: %w", name, err)
		}
		if detailed {
			fmt.Printf("Plugin:  %s\nState:   %s\nPID:     %d\n", st.Name, st.State, st.PID)
		} else {
			fmt.Printf("%-20s %s\n", st.Name, st.State)
		}
		return nil
	}

	// No name given: show status for all installed plugins.
	pluginDir := resolvePluginDir()
	plugins, err := plugin.List(pluginDir, true)
	if err != nil {
		return fmt.Errorf("listing installed plugins: %w", err)
	}
	if len(plugins) == 0 {
		fmt.Println("No plugins installed.")
		return nil
	}

	for _, p := range plugins {
		st, err := plugin.Status(p.Name)
		if err != nil {
			fmt.Printf("%-20s unknown\n", p.Name)
			continue
		}
		if detailed {
			fmt.Printf("%-20s %-10s pid=%d\n", st.Name, st.State, st.PID)
		} else {
			fmt.Printf("%-20s %s\n", st.Name, st.State)
		}
	}

	return nil
}
