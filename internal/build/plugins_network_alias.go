package build

import (
	"bytes"
	"fmt"
	"regexp"
)

// Purpose: rewrites a plugin's short-form docker-compose networks block into
// the long form with a `plugin-{name}` alias, so the CLI-generated
// PLUGIN_{DEP}_INTERNAL_URL env vars resolve over Docker's embedded DNS.
// Inputs: the compose fragment bytes and the plugin's short name.
// Outputs: the (possibly rewritten) compose fragment bytes.
// Constraints: split out of plugins.go (CLI-R12) as a pure move; no behavior
// changed. Idempotent — a no-op when the alias is already present or no
// short-form networks block is found.

// shortNetworkListRE matches the short-form networks block used by most plugin
// compose fragments:
//
//	networks:
//	  - ${DOCKER_NETWORK}
//
// It captures (1) the indent before "networks:", (2) the network reference
// after the list dash. The indent is preserved so the rewritten long form
// lines up with the original service block.
var shortNetworkListRE = regexp.MustCompile(`(?m)^([ \t]+)networks:\s*\n[ \t]+-[ \t]+(\S+)\s*$`)

// longNetworkAliasGuardRE checks whether the normalized fragment already
// contains an "aliases:" entry pointing at the plugin short name, so the
// normalizer is idempotent and safe to run on already-patched files.
var aliasAlreadyPresentRE = regexp.MustCompile(`aliases:\s*\n[ \t]+-[ \t]+plugin-`)

// normalizeComposeNetworkAliases rewrites the short-form networks list used
// by plugin compose fragments into the long form with a network alias that
// matches the inter-plugin hostname convention (`plugin-{name}`).
//
// Background: the nSelf CLI injects environment variables of the form
// `PLUGIN_<DEP>_INTERNAL_URL=http://plugin-<dep>:<port>` for every installed
// plugin dependency. However, most installed plugin compose fragments declare
// the service under the bare plugin name (e.g. `mux`, `ai`, `claw`), which
// means Docker's embedded DNS only resolves the bare hostname, not the
// `plugin-*` alias that callers expect. The result is a "bad address" error
// at runtime even though both containers live on the same network.
//
// The fix is to ensure the plugin container is reachable at BOTH its bare
// service name (for local compose references and backward compatibility) AND
// the `plugin-{name}` alias used by the CLI-generated env vars. This
// normalizer performs that rewrite at build time so no plugin re-release is
// required.
//
// The rewrite is a no-op when:
//   - The fragment already uses the long form with an explicit aliases block.
//   - The fragment does not contain a short-form networks list (e.g. already
//     rewritten, or uses a completely different network topology).
//
// Only the first matching short-form networks block per fragment is rewritten,
// which matches the one-service-per-plugin convention used across the
// plugins-pro library.
func normalizeComposeNetworkAliases(content []byte, pluginName string) []byte {
	if pluginName == "" {
		return content
	}
	// Idempotency guard: if a plugin- alias is already present, do nothing.
	if aliasAlreadyPresentRE.Match(content) {
		return content
	}
	match := shortNetworkListRE.FindSubmatchIndex(content)
	if match == nil {
		return content
	}
	indent := string(content[match[2]:match[3]])
	netRef := string(content[match[4]:match[5]])
	// Build the replacement using the captured indent so the YAML stays aligned
	// with the surrounding service block.
	innerIndent := indent + "  "
	aliasIndent := innerIndent + "  "
	aliasItemIndent := aliasIndent + "  "
	replacement := fmt.Sprintf(
		"%snetworks:\n%s%s:\n%saliases:\n%s- plugin-%s\n",
		indent,
		innerIndent, netRef,
		aliasIndent,
		aliasItemIndent, pluginName,
	)
	var out bytes.Buffer
	out.Grow(len(content) + len(replacement))
	out.Write(content[:match[0]])
	out.WriteString(replacement)
	out.Write(content[match[1]:])
	return out.Bytes()
}
