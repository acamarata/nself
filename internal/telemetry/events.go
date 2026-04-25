package telemetry

// Event catalog for the CLI telemetry client (Q07 / T01-T05).
//
// Five anonymous events are defined. The server de-duplicates "per-install-ID"
// events via a UNIQUE index on (install_id, event_type) for the three
// "first_*" variants. The CLI sends every time; ping_api discards duplicates
// server-side.
//
// All event names are snake_case strings. No PII, no paths, no free-form text.
const (
	// EventCLIInstall fires on nself install completion.
	// Metadata: version, os, arch, via (brew/script/binary).
	EventCLIInstall = "cli.install"

	// EventCLIFirstInit fires on the first nself init per install-ID.
	// Metadata: version.
	EventCLIFirstInit = "cli.first_init"

	// EventCLIFirstStart fires on the first nself start per install-ID.
	// Metadata: version, services_count.
	EventCLIFirstStart = "cli.first_start"

	// EventPluginInstall fires on every nself plugin install <name>.
	// Metadata: plugin_name, plugin_version, tier (free/pro).
	EventPluginInstall = "plugin.install"

	// EventEndpointFirstHit fires on the first successful Hasura query per install-ID.
	// Metadata: version.
	EventEndpointFirstHit = "endpoint.first_hit"
)
