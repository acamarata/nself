package compose

// Package compose: service-profile registry.
//
// Purpose: define curated subsets of nSelf services for different deployment
// roles. A profile controls which service builders fire inside buildDockerCompose.
// This lets operators run e.g. an "ops" server (observability + CI + registry)
// without any app-specific services (minio, mailpit, admin, functions, search).
//
// Inputs:  ProfileName constant (string) selected via --profile flag or NSELF_PROFILE.
// Outputs: ServiceSet describing which service groups are allowed.
// Constraints:
//   - ProfileApp ("app") is the default and must be fully backward-compatible.
//   - New profiles are additive (data-driven via profileRegistry map).
//   - The registry is the single source of truth — do not gate on profile strings
//     in generator.go directly; always call ProfileForName.
//
// SPORT: see F08-SERVICE-INVENTORY for canonical service names.

// ProfileName identifies a named service profile.
type ProfileName string

const (
	// ProfileApp is the default profile: every service enabled by config
	// is included. Identical to pre-profile behaviour — no regression.
	ProfileApp ProfileName = "app"

	// ProfileOps is the observability + CI + functions + registry profile.
	// Core data services (postgres, hasura, auth, nginx) plus the monitoring
	// stack are included. App-specific services (minio, mailpit, admin,
	// functions-v8 SaaS, search, nself-admin) are excluded.
	//
	// Reserve service name stubs for ops-only services whose compose fragments
	// will be contributed by dedicated plugins (forgejo, container-registry,
	// functions-v8). Listing them here documents the intended topology even
	// before those plugins exist.
	ProfileOps ProfileName = "ops"
)

// ServiceSet describes which service families are allowed in a profile.
// Every bool field corresponds to a builder or conditional block in
// generator.go. Fields default to false; set only what the profile needs.
type ServiceSet struct {
	// CoreDB enables postgres (unless EmbeddedPG is active).
	CoreDB bool
	// Hasura enables the Hasura GraphQL engine.
	Hasura bool
	// Auth enables the nhost/hasura-auth service.
	Auth bool
	// Nginx enables the nginx reverse proxy (always last in the compose file).
	Nginx bool

	// Redis enables the Redis cache/queue service (when cfg.Redis.Enabled).
	Redis bool
	// Minio enables the MinIO object storage service (when cfg.Minio.Enabled).
	Minio bool
	// Mailpit enables the Mailpit dev SMTP server (when cfg.Mailpit.Enabled).
	Mailpit bool
	// AdminUI enables the nSelf Admin web UI (when cfg.Admin.Enabled).
	AdminUI bool
	// Functions enables the Functions runtime (when cfg.Functions.Enabled).
	Functions bool
	// Search enables the search engine (meilisearch/typesense) when configured.
	Search bool

	// Monitoring enables the monitoring stack (prometheus, grafana, loki, etc.)
	// when cfg.Monitoring.Enabled. The compose services are contributed by the
	// nself-monitoring plugin; this flag only gates the config rendering step.
	Monitoring bool

	// OpsServices is true when ops-profile-specific reserved service stubs are
	// declared. Actual compose fragments for forgejo, container-registry, and
	// functions-v8 are contributed by dedicated plugins; this flag marks intent.
	//
	// Reserved ops service names (F08-SERVICE-INVENTORY):
	//   - forgejo          (lightweight Gitea-compatible forge for CI/CD)
	//   - container-registry (OCI-compatible image registry, e.g. Zot/Distribution)
	//   - functions-v8     (Deno V8-isolate function runtime, ops-server variant)
	OpsServices bool
}

// profileRegistry maps profile names to their service sets.
// Adding a new profile means adding a single entry here.
var profileRegistry = map[ProfileName]ServiceSet{
	ProfileApp: {
		CoreDB:     true,
		Hasura:     true,
		Auth:       true,
		Nginx:      true,
		Redis:      true,
		Minio:      true,
		Mailpit:    true,
		AdminUI:    true,
		Functions:  true,
		Search:     true,
		Monitoring: true,
	},
	ProfileOps: {
		// Core data layer — always required; hasura-auth is needed for API
		// authentication even on an ops-only server.
		CoreDB: true,
		Hasura: true,
		Auth:   true,
		Nginx:  true,

		// Monitoring stack is the primary payload of the ops profile.
		Monitoring: true,

		// Ops-specific services (forgejo / container-registry / functions-v8)
		// contributed by plugins; flag them as enabled here for documentation.
		OpsServices: true,

		// App services explicitly excluded:
		// Redis:     false  — no BullMQ/queues on an ops server
		// Minio:     false  — no object storage for apps
		// Mailpit:   false  — no SMTP sandbox for apps
		// AdminUI:   false  — nSelf Admin not needed on an ops server
		// Functions: false  — SaaS function runtime excluded (ops has functions-v8)
		// Search:    false  — no app-level search engine
	},
}

// ProfileForName returns the ServiceSet for the given profile name.
// Falls back to the ProfileApp set if the name is unrecognised, and returns
// false as the second value so callers can warn the user.
//
// Callers: compose.Generator, build.Build.
func ProfileForName(name ProfileName) (ServiceSet, bool) {
	if name == "" {
		return profileRegistry[ProfileApp], true
	}
	set, ok := profileRegistry[name]
	if !ok {
		return profileRegistry[ProfileApp], false
	}
	return set, true
}

// ValidProfiles returns the list of known profile names in deterministic order.
// Used by --help text and shell completions.
func ValidProfiles() []string {
	return []string{
		string(ProfileApp),
		string(ProfileOps),
	}
}
