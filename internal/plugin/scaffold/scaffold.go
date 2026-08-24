// Package scaffold provides the canonical plugin scaffolding logic shared
// between the nself CLI (plugin new command) and the standalone new-plugin
// binary in plugin-sdk-go/devkit.
//
// Both entry points call scaffold.Run with a Params struct; the output is
// identical so that plugin authors get the same result regardless of whether
// they have nself installed or use the SDK devkit directly.
//
// Purpose: Core types and all logic functions for plugin scaffold generation.
//
//	Template strings are in scaffold_templates_infra.go (infrastructure,
//	devops, metadata templates) and scaffold_templates_code.go (Go code
//	templates: main, config, server, server_test).
//
// Inputs:  Options struct — name, tier, language, tenancy mode, overrides.
// Outputs: Result struct — output directory path and list of emitted files.
// Constraints: Must remain import-compatible with plugin-sdk-go/devkit.
//
//	Template strings must live in the _templates_*.go files, not here.
//
// SPORT:   cli/internal/plugin/scaffold — decomposed from scaffold.go (T-E2-06).
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SlugRE is the valid plugin name regexp.
// A slug must start with a lowercase letter, be at least 2 chars, at most 41
// chars total, contain only lowercase letters, digits, and internal hyphens,
// and must NOT end with a hyphen.
var SlugRE = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}[a-z0-9]$`)

// TenancyMode controls which multi-tenant column(s) the scaffold emits.
// Matches the --tenancy flag and the interactive prompt choices.
type TenancyMode string

const (
	// TenancyNone omits all tenancy columns. Use for plugins with no per-user Postgres tables.
	TenancyNone TenancyMode = "none"
	// TenancyAppIsolation emits source_account_id TEXT NOT NULL DEFAULT 'primary'.
	// Correct for multi-app isolation within one nSelf deploy.
	TenancyAppIsolation TenancyMode = "app-isolation"
	// TenancyCloudTenant emits tenant_id UUID (nullable) + Hasura row filter.
	// Correct for Cloud SaaS where each paying customer is isolated.
	TenancyCloudTenant TenancyMode = "cloud-tenant"
	// TenancyBoth emits both columns. Use when unsure — the developer can remove one later.
	TenancyBoth TenancyMode = "both"
)

// Params carries all values available inside scaffold templates.
type Params struct {
	Name        string // plugin slug, e.g. "mywidget"
	PascalName  string // e.g. "Mywidget"
	EnvPrefix   string // e.g. "MYWIDGET" (upper-cased, dashes to underscores)
	RepoBucket  string // "paid" or "free"
	Tier        string // "free" or "pro"
	Bundle      string // bundle display name, e.g. "nClaw" (empty allowed for free)
	Description string
	Author      string
	License     string
	Language    string // "go" (default), "rust", "node", "static"
	MinCLI      string
	MinSDK      string
	Category    string
	Port        int
	Year        int
	Tenancy     TenancyMode // multi-tenant column choice; empty == TenancyNone
}

// Options configures a scaffold run.
type Options struct {
	// Name is required. Must match SlugRE.
	Name string
	// Tier is "free" or "pro". Default "free".
	Tier string
	// Bundle is the bundle display name (optional for free plugins).
	Bundle string
	// Description defaults to "nSelf <name> plugin."
	Description string
	// Author is optional.
	Author string
	// Category defaults to "custom".
	Category string
	// Language is the plugin language: go, rust, node, static. Default "go".
	Language string
	// MinCLI is the minimum nSelf CLI version required. Default "1.0.9".
	MinCLI string
	// MinSDK is the minimum plugin-sdk-go version required. Default "0.1.0".
	MinSDK string
	// Port is the default listen port. Default 8080.
	Port int
	// OutDir overrides the output directory. Default: ./<name>.
	OutDir string
	// Force allows overwriting an existing directory.
	Force bool
	// Tenancy controls multi-tenant column scaffolding. Default TenancyNone.
	// When empty string it is treated as TenancyNone.
	Tenancy TenancyMode
}

// Result describes what was emitted.
type Result struct {
	Dir   string
	Files []string
}

// Run executes the scaffold and returns the result.
func Run(opts Options) (*Result, error) {
	// Apply defaults.
	if !SlugRE.MatchString(opts.Name) {
		return nil, fmt.Errorf("invalid plugin name %q: must match %s", opts.Name, SlugRE)
	}
	if opts.Tier == "" {
		opts.Tier = "free"
	}
	if opts.Tier != "free" && opts.Tier != "pro" {
		return nil, fmt.Errorf("--tier must be 'free' or 'pro', got %q", opts.Tier)
	}
	if opts.Language == "" {
		opts.Language = "go"
	}
	if opts.Category == "" {
		opts.Category = "custom"
	}
	if opts.MinCLI == "" {
		opts.MinCLI = "1.0.9"
	}
	if opts.MinSDK == "" {
		opts.MinSDK = "0.1.0"
	}
	if opts.Port == 0 {
		opts.Port = 8080
	}
	if opts.Description == "" {
		opts.Description = fmt.Sprintf("nSelf %s plugin.", opts.Name)
	}
	if opts.Tenancy == "" {
		opts.Tenancy = TenancyNone
	}
	switch opts.Tenancy {
	case TenancyNone, TenancyAppIsolation, TenancyCloudTenant, TenancyBoth:
	default:
		return nil, fmt.Errorf("--tenancy must be none, app-isolation, cloud-tenant, or both; got %q", opts.Tenancy)
	}

	repoBucket := "paid"
	if opts.Tier == "free" {
		repoBucket = "free"
	}

	outDir := opts.OutDir
	if outDir == "" {
		outDir = filepath.Join(".", opts.Name)
	}

	// Safety check.
	if !opts.Force {
		if entries, err := os.ReadDir(outDir); err == nil && len(entries) > 0 {
			return nil, fmt.Errorf("destination %q is not empty (use Force to overwrite)", outDir)
		}
	}

	params := Params{
		Name:        opts.Name,
		PascalName:  toPascal(opts.Name),
		EnvPrefix:   toEnvPrefix(opts.Name),
		RepoBucket:  repoBucket,
		Tier:        opts.Tier,
		Bundle:      opts.Bundle,
		Description: opts.Description,
		Author:      opts.Author,
		License:     licenseForTier(opts.Tier),
		Language:    opts.Language,
		Category:    opts.Category,
		MinCLI:      opts.MinCLI,
		MinSDK:      opts.MinSDK,
		Port:        opts.Port,
		Year:        time.Now().Year(),
		Tenancy:     opts.Tenancy,
	}

	fileList, err := buildFiles(params)
	if err != nil {
		return nil, fmt.Errorf("scaffold: build files: %w", err)
	}

	var emitted []string
	for relPath, content := range fileList {
		fullPath := filepath.Join(outDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, fmt.Errorf("creating dir for %s: %w", relPath, err)
		}
		mode := os.FileMode(0644)
		if strings.HasSuffix(relPath, ".sh") {
			mode = 0750
		}
		if err := os.WriteFile(fullPath, []byte(content), mode); err != nil {
			return nil, fmt.Errorf("writing %s: %w", relPath, err)
		}
		emitted = append(emitted, relPath)
	}

	return &Result{Dir: outDir, Files: emitted}, nil
}
