package build

import (
	"bytes"
	"fmt"
	"regexp"
)

// Purpose: gives a plugin's docker-compose service a `plugin-{name}` DNS
// alias so the CLI-generated PLUGIN_{DEP}_INTERNAL_URL env vars resolve over
// Docker's embedded DNS.
// Inputs: the compose fragment bytes and the plugin's short name.
// Outputs: the (possibly rewritten) compose fragment bytes.
// Constraints: split out of plugins.go (CLI-R12) as a pure move originally;
// P6-E3-W2-S1-T5 (2026-09-03) found the map-form `networks:` rewrite this
// function used to emit was invalid Compose and replaced the body with a
// `hostname:` injection — see the doc comment below. Signature and call
// site (plugins.go DiscoverPluginComposeFiles) are unchanged. Idempotent —
// a no-op when the alias is already present or no networks reference is
// found.

// shortNetworkListRE matches the short-form networks block used by most plugin
// compose fragments:
//
//	networks:
//	  - ${DOCKER_NETWORK}
//
// It captures (1) the indent before "networks:". That indent is reused to
// place the injected `hostname:` line at the same level as its sibling keys
// in the service block.
var shortNetworkListRE = regexp.MustCompile(`(?m)^([ \t]+)networks:\s*\n[ \t]+-[ \t]+(\S+)\s*$`)

// normalizeComposeNetworkAliases ensures a plugin's compose service is
// reachable at the `plugin-{name}` DNS alias used by the CLI-generated
// PLUGIN_<DEP>_INTERNAL_URL env vars.
//
// Background: the nSelf CLI injects environment variables of the form
// `PLUGIN_<DEP>_INTERNAL_URL=http://plugin-<dep>:<port>` for every installed
// plugin dependency. However, most installed plugin compose fragments declare
// the service under the bare plugin name (e.g. `mux`, `ai`, `claw`), which
// means Docker's embedded DNS only resolves the bare hostname, not the
// `plugin-*` alias that callers expect. The result is a "bad address" error
// at runtime even though both containers live on the same network.
//
// A prior version of this function rewrote the fragment's short-form
// `networks: [- ${DOCKER_NETWORK}]` list into the long map form with an
// `aliases:` entry (`${DOCKER_NETWORK}: {aliases: [plugin-<name>]}`). That
// output is invalid: Docker Compose only interpolates `${VAR}` references in
// scalar VALUES, never in YAML mapping KEYS, so `${DOCKER_NETWORK}` as a map
// key is passed to compose-spec's schema validator un-substituted and
// rejected — confirmed against docker compose v5.3.0:
//
//	services.<name>.networks additional properties '${DOCKER_NETWORK}' not allowed
//
// This broke `nself build`'s merged compose for every plugin using the
// short-form list (P6-E3-W2-S1-T5 smoke test, 2026-09-03).
//
// The fix: leave the `networks:` block untouched (its `- ${DOCKER_NETWORK}`
// list form is valid Compose and already connects the service to the shared
// network) and instead inject `hostname: plugin-{name}` as a sibling key in
// the service block. `hostname:` is a plain scalar value (interpolates fine,
// though it needs none here) and Docker's embedded DNS resolves a
// container's `hostname` on a user-defined bridge network exactly like a
// network alias — verified empirically: a sibling container on the same
// compose network resolves `plugin-<name>` via `getent hosts` with no
// `aliases:` block involved at all.
//
// The rewrite is a no-op when:
//   - `hostname: plugin-{name}` is already present (already patched).
//   - The fragment has no short-form `networks:` list to anchor the
//     insertion point on (e.g. a completely different network topology).
//
// Only the first matching short-form networks block per fragment is used to
// anchor the insertion, which matches the one-service-per-plugin convention
// used across the plugins library.
func normalizeComposeNetworkAliases(content []byte, pluginName string) []byte {
	if pluginName == "" {
		return content
	}
	hostnameLine := []byte("hostname: plugin-" + pluginName)
	// Idempotency guard: if the alias hostname is already declared, do nothing.
	if bytes.Contains(content, hostnameLine) {
		return content
	}
	match := shortNetworkListRE.FindSubmatchIndex(content)
	if match == nil {
		return content
	}
	indent := string(content[match[2]:match[3]])
	// Insert the hostname line immediately before the "networks:" line, at
	// the same indent, as a new sibling key in the service block. The
	// networks: block itself (match[0]:match[1]) is copied through unchanged.
	insertion := fmt.Sprintf("%s%s\n", indent, hostnameLine)
	var out bytes.Buffer
	out.Grow(len(content) + len(insertion))
	out.Write(content[:match[0]])
	out.WriteString(insertion)
	out.Write(content[match[0]:])
	return out.Bytes()
}
