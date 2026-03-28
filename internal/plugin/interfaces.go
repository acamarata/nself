package plugin

import "context"

// RegistryClient abstracts plugin registry HTTP operations for testability.
type RegistryClient interface {
	Fetch(ctx context.Context) (*Registry, error)
	GetPlugin(ctx context.Context, name string) (*PluginManifest, error)
}

// LicenseClient abstracts license validation for testability.
type LicenseClient interface {
	Validate(ctx context.Context, key string) (bool, error)
}

// Registry represents the full plugin registry response.
type Registry struct {
	Plugins []PluginManifest
}

// EnvVar describes an environment variable required or used by a plugin.
type EnvVar struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
}

// CLICommand describes a CLI command provided by a plugin.
type CLICommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SystemDependency describes a system-level dependency for a plugin.
type SystemDependency struct {
	Name          string `json:"name"`
	Verify        string `json:"verify"`
	MinVersion    string `json:"minVersion,omitempty"`
	Apt           string `json:"apt,omitempty"`
	Brew          string `json:"brew,omitempty"`
	CustomInstall string `json:"custom_install,omitempty"`
}

// SystemDependencies groups required and recommended system deps.
type SystemDependencies struct {
	Required    []SystemDependency `json:"required,omitempty"`
	Recommended []SystemDependency `json:"recommended,omitempty"`
}

// MultiApp describes multi-tenancy configuration for a plugin.
type MultiApp struct {
	Supported       bool   `json:"supported"`
	IsolationColumn string `json:"isolationColumn,omitempty"`
	PKStrategy      string `json:"pkStrategy,omitempty"`
	DefaultValue    string `json:"defaultValue,omitempty"`
}

// PluginManifest describes a single plugin, parsed from plugin.json.
type PluginManifest struct {
	// Required fields
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Category    string `json:"category"`
	License     string `json:"license"`

	// Optional metadata
	Author               string   `json:"author,omitempty"`
	Homepage             string   `json:"homepage,omitempty"`
	Repository           string   `json:"repository,omitempty"`
	Tags                 []string `json:"tags,omitempty"`
	IsCommercial         bool     `json:"isCommercial,omitempty"`
	LicenseType          string   `json:"licenseType,omitempty"`
	RequiredEntitlements []string `json:"requiredEntitlements,omitempty"`
	RequiresLicense      bool     `json:"requires_license,omitempty"`
	MinNselfVersion      string   `json:"minNselfVersion,omitempty"`
	MinNodeVersion       string   `json:"minNodeVersion,omitempty"`
	ArchSupport          []string `json:"arch_support,omitempty"`

	// Implementation
	Language       string `json:"language,omitempty"`
	Runtime        string `json:"runtime,omitempty"`
	Port           int    `json:"port,omitempty"`
	EntryPoint     string `json:"entryPoint,omitempty"`
	CLI            string `json:"cli,omitempty"`
	HealthEndpoint string `json:"health_endpoint,omitempty"`
	PackageManager string `json:"packageManager,omitempty"`
	Framework      string `json:"framework,omitempty"`

	// Database
	Tables []string `json:"tables,omitempty"`
	Views  []string `json:"views,omitempty"`

	// API
	APIEndpoints []string     `json:"apiEndpoints,omitempty"`
	Webhooks     []string     `json:"webhooks,omitempty"`
	CLICommands  []CLICommand `json:"cliCommands,omitempty"`

	// Environment
	EnvVars []EnvVar `json:"envVars,omitempty"`

	// Dependencies
	Dependencies         []string           `json:"dependencies,omitempty"`
	OptionalDependencies []string           `json:"optionalDependencies,omitempty"`
	SystemDependencies   SystemDependencies `json:"systemDependencies,omitempty"`

	// Multi-tenancy
	MultiApp MultiApp `json:"multiApp,omitempty"`

	// Permissions
	Permissions []string `json:"permissions,omitempty"`

	// Registry-specific fields (not in plugin.json, populated by registry)
	Tier     string `json:"tier,omitempty"`
	Checksum string `json:"checksum,omitempty"`
}
