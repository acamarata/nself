// Purpose:     decide what the OpenAPI Route column reports.
//
// Inputs:      none at runtime — this file records a one-time investigation
//
//	of internal/apidocs/{openapi.go,plugin_routes.go} done for CLI-R17.
//
// Outputs:     openAPIColumnValue, the constant string used for every row.
//
// Constraints: internal/apidocs DOES exist (openapi.go, openapi_test.go,
//
//	plugin_routes.go, scalar.html) and IS wired into the CLI, from
//	internal/build/orchestrator.go (apidocs.Generate /
//	apidocs.CollectPluginRoutes / apidocs.NginxConf are all called from
//	there, guarded by cfg.ApiDocs.Enabled during `nself build`). An
//	earlier version of this file wrongly claimed the package did not
//	exist; it does, and the finding below is corrected accordingly.
//
//	What it does NOT do is describe the CLI's own commands. Reading
//	internal/apidocs/openapi.go's buildSpec: the paths it emits are
//	(a) five hardcoded /auth/v1/* endpoints, (b) /v1/graphql (+ its
//	/subscriptions variant) when cfg.GraphQLEnabled, and (c) REST
//	routes read from each installed plugin's plugin.json rest_routes
//	key at build time (plugin_routes.go's CollectPluginRoutes walks
//	~/.nself/plugins/*/plugin.json, not this repo's source tree). All
//	three describe the HTTP surface of the *generated backend stack*
//	nself provisions (Hasura/Auth/GraphQL/plugin REST) — served at
//	docs.<domain>/api-docs — not the `nself <command>` CLI surface
//	this matrix's other three columns score. There is no field on
//	either side (cobra.Command, OpenAPIOperation, PluginRoute) linking
//	a CLI command name to an OpenAPI path, so no per-command match is
//	possible without inventing one. Grepped cmd/commands/api.go too
//	(the closest-named top-level command): it never references the
//	apidocs package, confirming the two surfaces are unrelated.
package main

// openAPIColumnValue is applied to every row: the finding above is a
// repo-wide architectural fact, not a per-command one, so all 84 rows
// carry the same accurate reason rather than a per-row guess.
const openAPIColumnValue = "n/a (see below)"

// openAPIFinding is reproduced in the generated page header so the reason
// travels with the data instead of living only in this source comment.
const openAPIFinding = "internal/apidocs exists (openapi.go, plugin_routes.go) and is wired into " +
	"`nself build` via internal/build/orchestrator.go, but it documents the generated " +
	"backend's HTTP surface (hardcoded `/auth/v1/*` endpoints, `/v1/graphql`, and REST " +
	"routes read from each installed plugin's `plugin.json` at build time) — not the CLI's " +
	"own commands. No command-to-route mapping exists on either side, so every row is `n/a`."
