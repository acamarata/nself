package plugin

// Purpose: Plugin domain types — PluginInfo and InstalledPluginInfo.
//          All implementation (install, remove, load, verify, lifecycle) has
//          been decomposed into focused sibling files:
//            installer.go  — Install/Remove/Update + lock + reverse-deps
//            download.go   — downloadPlugin/extractTarGz/rollbackInstall
//            loader.go     — ListInstalled/LoadManifestsFromDir/List + nginx routes
//            security.go   — verifyChecksum/verifyPluginSignature/checkLicense/
//                            CheckEOLBlock/checkAuthorRevocation/postInstallEvent
//            config.go     — DisablePlugin/EnablePlugin/IsDisabled + table conflicts
//            paths.go      — DefaultCacheDir/LicenseCacheDir/pingAPIURL/findPlugin
// Inputs:  (none — type declarations only)
// Outputs: (none — type declarations only)
// Constraints: Keep this file ≤100 lines. Do not add logic here.
// SPORT: plugin-types; callers: all cmd/plugin/*.go commands

// PluginInfo describes a plugin's identity and current state.
type PluginInfo struct {
	Name          string
	Version       string
	Category      string
	Installed     bool
	Running       bool
	PublishStatus string // S58-T01: "experimental"|"planned"|"beta"|"stable"|"deprecated"|"eol" — from registry
	UpdatedAt     string // CLI-R16: per-plugin freshness, empty when the source (registry entry or local plugin.json) doesn't carry one
}

// InstalledPluginInfo is a richer view of an installed plugin used by the
// inventory subcommand. It includes Description and Tier from the manifest.
type InstalledPluginInfo struct {
	Name        string
	Version     string
	Tier        string
	Status      string
	Description string
}
