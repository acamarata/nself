package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var urlsCmd = &cobra.Command{
	Use:   "urls",
	Short: "Display all service URLs with route conflict detection",
	Long: `Show all dynamically assigned proxy URLs for the current project,
grouped by service type: Required, Optional, Custom, and Frontend.

URLs are computed from BASE_DOMAIN and each service's route configuration.
Use --check-conflicts to detect overlapping routes.`,
	RunE: runURLs,
}

func init() {
	urlsCmd.Flags().BoolP("all", "a", false, "Show all routes including internal")
	urlsCmd.Flags().Bool("json", false, "JSON output")
	urlsCmd.Flags().String("env", "", "Show URLs for specific environment")
	urlsCmd.Flags().String("diff", "", "Compare URLs between environments")
	urlsCmd.Flags().Bool("check-conflicts", false, "Check for route conflicts")

	RootCmd.AddCommand(urlsCmd)
}

// serviceURL represents a single resolved URL entry for display and JSON output.
type serviceURL struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Internal bool   `json:"internal,omitempty"`
	Group    string `json:"group"`
}

// urlsOutput is the JSON-serializable result of the urls command.
type urlsOutput struct {
	BaseDomain       string       `json:"base_domain"`
	Env              string       `json:"env"`
	RequiredServices []serviceURL `json:"required_services"`
	OptionalServices []serviceURL `json:"optional_services,omitempty"`
	CustomServices   []serviceURL `json:"custom_services,omitempty"`
	FrontendApps     []serviceURL `json:"frontend_apps,omitempty"`
	InternalRoutes   []serviceURL `json:"internal_routes,omitempty"`
	TotalRoutes      int          `json:"total_routes"`
	Conflicts        []conflict   `json:"conflicts,omitempty"`
}

// conflict describes two services whose routes overlap.
type conflict struct {
	Route    string `json:"route"`
	Service1 string `json:"service_1"`
	Service2 string `json:"service_2"`
}

func runURLs(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	showAll, _ := cmd.Flags().GetBool("all")
	envFlag, _ := cmd.Flags().GetString("env")
	diffFlag, _ := cmd.Flags().GetString("diff")
	checkConflicts, _ := cmd.Flags().GetBool("check-conflicts")

	if !jsonOut {
		ui.CommandHeaderStderr("nself urls", "Service URL overview")
	}

	rawCwd, err := os.Getwd()
	if err != nil {
		if jsonOut {
			ui.PrintJSONError(err)
		} else {
			ui.Error("Failed to determine working directory")
		}
		return fmt.Errorf("getting working directory: %w", err)
	}
	workdir, err := config.FindNSelfRoot(rawCwd)
	if err != nil {
		return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
	}

	// If --env is set, override the ENV variable before loading config.
	if envFlag != "" {
		_ = os.Setenv("ENV", envFlag)
	}

	// --diff: load two environments and compare.
	if diffFlag != "" {
		return runURLsDiff(workdir, envFlag, diffFlag, jsonOut)
	}

	cfg, err := config.Load(workdir)
	if err != nil {
		if jsonOut {
			ui.PrintJSONError(err)
		} else {
			ui.Error(fmt.Sprintf("Failed to load config: %v", err))
		}
		return err
	}

	output := buildURLOutput(cfg, showAll)

	if checkConflicts {
		output.Conflicts = detectConflicts(output)
	}

	if jsonOut {
		return ui.PrintJSON(output)
	}

	printURLGroups(output, showAll)

	if checkConflicts && len(output.Conflicts) > 0 {
		fmt.Println()
		ui.Warn(fmt.Sprintf("%d route conflict(s) detected:", len(output.Conflicts)))
		for _, c := range output.Conflicts {
			fmt.Printf("  %s: %s vs %s\n",
				ui.C(ui.Yellow, c.Route),
				ui.C(ui.Cyan, c.Service1),
				ui.C(ui.Cyan, c.Service2),
			)
		}
	} else if checkConflicts {
		fmt.Println()
		ui.Success("No route conflicts detected")
	}

	return nil
}

// buildURLOutput computes all URLs from a loaded config.
func buildURLOutput(cfg *config.Config, showAll bool) urlsOutput {
	bd := cfg.BaseDomain
	scheme := urlScheme(cfg)

	out := urlsOutput{
		BaseDomain: bd,
		Env:        cfg.Env,
	}

	// -- Required Services (always present) --
	out.RequiredServices = []serviceURL{
		{
			Name:     "PostgreSQL",
			URL:      fmt.Sprintf("127.0.0.1:%d", cfg.Postgres.Port),
			Internal: true,
			Group:    "Required Services",
		},
		{
			Name:  "Hasura GraphQL",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Hasura.Route, bd),
			Group: "Required Services",
		},
		{
			Name:  "Auth",
			URL:   resolveRoute(scheme, cfg.Auth.Route, bd),
			Group: "Required Services",
		},
		{
			Name:  "Nginx",
			URL:   fmt.Sprintf("%s://%s", scheme, bd),
			Group: "Required Services",
		},
	}

	// -- Optional Services --
	if cfg.Redis.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:     "Redis",
			URL:      fmt.Sprintf("127.0.0.1:%d", cfg.Redis.Port),
			Internal: true,
			Group:    "Optional Services",
		})
	}
	if cfg.Minio.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "MinIO Storage",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Minio.StorageRoute, bd),
			Group: "Optional Services",
		})
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "MinIO Console",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Minio.ConsoleRoute, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Mailpit.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Mailpit UI",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Mailpit.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Functions.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Functions",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Functions.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.MLflow.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "MLflow",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.MLflow.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Admin.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Admin",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Admin.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Search.Enabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Search",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Search.Route, bd),
			Group: "Optional Services",
		})
	}
	if cfg.Monitoring.Enabled && cfg.Monitoring.GrafanaEnabled {
		out.OptionalServices = append(out.OptionalServices, serviceURL{
			Name:  "Grafana",
			URL:   fmt.Sprintf("%s://%s.%s", scheme, cfg.Monitoring.GrafanaRoute, bd),
			Group: "Optional Services",
		})
	}

	// -- Custom Services --
	for _, cs := range cfg.CustomServices {
		if cs.Route != "" {
			out.CustomServices = append(out.CustomServices, serviceURL{
				Name:  cs.Name,
				URL:   fmt.Sprintf("%s://%s.%s", scheme, cs.Route, bd),
				Group: "Custom Services",
			})
		} else if showAll {
			out.CustomServices = append(out.CustomServices, serviceURL{
				Name:     cs.Name,
				URL:      fmt.Sprintf("127.0.0.1:%d", cs.Port),
				Internal: true,
				Group:    "Custom Services",
			})
		}
	}

	// -- Frontend Apps --
	for _, fa := range cfg.FrontendApps {
		if fa.Route != "" {
			out.FrontendApps = append(out.FrontendApps, serviceURL{
				Name:  fa.DisplayName,
				URL:   fmt.Sprintf("%s://%s.%s", scheme, fa.Route, bd),
				Group: "Frontend Apps",
			})
		}
	}

	// -- Internal admin endpoints (only with --all) --
	if showAll {
		// Hasura admin console (direct port access, not via proxy)
		out.InternalRoutes = append(out.InternalRoutes, serviceURL{
			Name:     "Hasura Admin",
			URL:      fmt.Sprintf("http://localhost:%d/console", cfg.Hasura.Port),
			Internal: true,
			Group:    "Internal Endpoints",
		})
		// Auth service direct port
		out.InternalRoutes = append(out.InternalRoutes, serviceURL{
			Name:     "Auth (direct)",
			URL:      fmt.Sprintf("http://localhost:%d", cfg.Auth.Port),
			Internal: true,
			Group:    "Internal Endpoints",
		})
		if cfg.Minio.Enabled {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "MinIO Console (direct)",
				URL:      fmt.Sprintf("http://localhost:%d", cfg.Minio.ConsolePort),
				Internal: true,
				Group:    "Internal Endpoints",
			})
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "MinIO API (direct)",
				URL:      fmt.Sprintf("http://localhost:%d", cfg.Minio.Port),
				Internal: true,
				Group:    "Internal Endpoints",
			})
		}
		if cfg.Monitoring.Enabled && cfg.Monitoring.GrafanaEnabled {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "Grafana (direct)",
				URL:      fmt.Sprintf("http://localhost:%d", cfg.Monitoring.GrafanaPort),
				Internal: true,
				Group:    "Internal Endpoints",
			})
		}
		if cfg.Mailpit.Enabled {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "Mailpit UI (direct)",
				URL:      fmt.Sprintf("http://localhost:%d", cfg.Mailpit.UIPort),
				Internal: true,
				Group:    "Internal Endpoints",
			})
		}
		if cfg.Redis.Enabled {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     "Redis (direct)",
				URL:      fmt.Sprintf("127.0.0.1:%d", cfg.Redis.Port),
				Internal: true,
				Group:    "Internal Endpoints",
			})
		}
	}

	// -- Internal Routes (only with --all) --
	if showAll {
		for _, ir := range cfg.InternalRoutes {
			out.InternalRoutes = append(out.InternalRoutes, serviceURL{
				Name:     ir.Name,
				URL:      fmt.Sprintf("%s://%s.%s", scheme, ir.Subdomain, bd),
				Internal: true,
				Group:    "Internal Routes",
			})
		}
	}

	// Count total public routes.
	out.TotalRoutes = countRoutes(out)

	return out
}

// resolveRoute handles routes that may already include the base domain
// (e.g., "auth.local.nself.org") vs bare subdomain prefixes (e.g., "auth").
func resolveRoute(scheme, route, bd string) string {
	if strings.Contains(route, ".") {
		// Fully qualified: auth.local.nself.org
		return fmt.Sprintf("%s://%s", scheme, route)
	}
	return fmt.Sprintf("%s://%s.%s", scheme, route, bd)
}

// urlScheme returns "https" unless SSL is disabled.
func urlScheme(cfg *config.Config) string {
	if cfg.SSLMode == "none" {
		return "http"
	}
	return "https"
}

// countRoutes tallies all non-internal URLs across all groups.
func countRoutes(out urlsOutput) int {
	total := 0
	for _, groups := range [][]serviceURL{
		out.RequiredServices,
		out.OptionalServices,
		out.CustomServices,
		out.FrontendApps,
		out.InternalRoutes,
	} {
		for _, s := range groups {
			if !s.Internal {
				total++
			}
		}
	}
	return total
}

// detectConflicts finds routes that resolve to the same subdomain.
func detectConflicts(out urlsOutput) []conflict {
	seen := make(map[string]string) // route -> first service name
	var conflicts []conflict

	allEntries := make([]serviceURL, 0, 32)
	allEntries = append(allEntries, out.RequiredServices...)
	allEntries = append(allEntries, out.OptionalServices...)
	allEntries = append(allEntries, out.CustomServices...)
	allEntries = append(allEntries, out.FrontendApps...)
	allEntries = append(allEntries, out.InternalRoutes...)

	for _, entry := range allEntries {
		if entry.Internal {
			continue
		}
		route := entry.URL
		if prev, ok := seen[route]; ok {
			conflicts = append(conflicts, conflict{
				Route:    route,
				Service1: prev,
				Service2: entry.Name,
			})
		} else {
			seen[route] = entry.Name
		}
	}

	return conflicts
}

// printURLGroups renders the grouped URL listing to the terminal.
func printURLGroups(out urlsOutput, showAll bool) {
	printGroup("Required Services", out.RequiredServices)
	printGroup("Optional Services", out.OptionalServices)
	printGroup("Custom Services", out.CustomServices)
	printGroup("Frontend Apps", out.FrontendApps)
	if showAll {
		printGroup("Internal Routes", out.InternalRoutes)
	}

	fmt.Printf("\n%d routes on %s\n", out.TotalRoutes, ui.C(ui.Cyan, out.BaseDomain))
}

// printGroup renders a single URL group section.
func printGroup(title string, entries []serviceURL) {
	if len(entries) == 0 {
		return
	}
	ui.Section(title)

	// Calculate max name width for alignment.
	maxName := 0
	for _, e := range entries {
		if len(e.Name) > maxName {
			maxName = len(e.Name)
		}
	}

	for _, e := range entries {
		name := fmt.Sprintf("%-*s", maxName, e.Name)
		if e.Internal {
			fmt.Printf("  %s   %s   %s\n",
				ui.C(ui.Dim, name),
				ui.C(ui.Dim, e.URL),
				ui.C(ui.Dim, "(internal only)"),
			)
		} else {
			fmt.Printf("  %s   %s\n",
				ui.C(ui.Cyan, name),
				ui.C(ui.White, e.URL),
			)
		}
	}
}

// runURLsDiff loads two environments and compares their URLs side-by-side.
func runURLsDiff(workdir, envA, envB string, jsonOut bool) error {
	if envA == "" {
		envA = "dev"
	}

	// Load environment A.
	_ = os.Setenv("ENV", envA)
	cfgA, err := config.Load(workdir)
	if err != nil {
		return fmt.Errorf("loading env %q: %w", envA, err)
	}
	outA := buildURLOutput(cfgA, true)

	// Load environment B.
	_ = os.Setenv("ENV", envB)
	cfgB, err := config.Load(workdir)
	if err != nil {
		return fmt.Errorf("loading env %q: %w", envB, err)
	}
	outB := buildURLOutput(cfgB, true)

	if jsonOut {
		diff := map[string]interface{}{
			envA: outA,
			envB: outB,
		}
		return ui.PrintJSON(diff)
	}

	// Build maps for comparison.
	mapA := urlMap(outA)
	mapB := urlMap(outB)

	// Collect all service names.
	nameSet := make(map[string]bool)
	for k := range mapA {
		nameSet[k] = true
	}
	for k := range mapB {
		nameSet[k] = true
	}
	names := make([]string, 0, len(nameSet))
	for k := range nameSet {
		names = append(names, k)
	}
	sort.Strings(names)

	fmt.Printf("%-20s  %-35s  %s\n",
		ui.C(ui.Bold, "Service"),
		ui.C(ui.Bold, envA),
		ui.C(ui.Bold, envB),
	)
	ui.Separator()

	for _, name := range names {
		a := mapA[name]
		b := mapB[name]
		if a == b {
			fmt.Printf("%-20s  %-35s  %s\n", name, a, b)
		} else {
			aDisp := a
			bDisp := b
			if a == "" {
				aDisp = ui.C(ui.Dim, "(none)")
			}
			if b == "" {
				bDisp = ui.C(ui.Dim, "(none)")
			}
			fmt.Printf("%-20s  %-35s  %s  %s\n",
				ui.C(ui.Yellow, name),
				aDisp,
				bDisp,
				ui.C(ui.Yellow, ui.IconWarning),
			)
		}
	}

	return nil
}

// urlMap flattens all URL entries into a name->url map for diff comparison.
func urlMap(out urlsOutput) map[string]string {
	m := make(map[string]string)
	for _, groups := range [][]serviceURL{
		out.RequiredServices,
		out.OptionalServices,
		out.CustomServices,
		out.FrontendApps,
		out.InternalRoutes,
	} {
		for _, e := range groups {
			m[e.Name] = e.URL
		}
	}
	return m
}
