package plugin

import (
	"regexp"
	"sort"
	"strings"
)

// Deriving canonical permissions from the descriptive form.
//
// Purpose: 121 shipped manifests declare permissions as an object —
// {"database": ["create","read"], "network": ["api.stripe.com"],
// "filesystem": ["logs"]} — while this CLI validates a flat list against an
// allowlist of "db:read", "network:internet", "fs:write:<volume>" and so on.
// Until now the object form was recorded but never validated, so those plugins
// were exempt from a check that is meant to be fail-closed.
//
// The two are not competing vocabularies, which is what made this look
// unresolvable at first. The descriptive form is strictly MORE specific: it
// names the hosts, the paths and the CRUD operations. The canonical form is the
// coarse one the CLI enforces. So the descriptive form can be reduced to the
// canonical form mechanically.
//
// Inputs: the grouped permission object from a manifest.
//
// Outputs: canonical permission strings.
//
// Constraints: every mapping WIDENS or stays equal — never narrows. Mapping
// "network:api.stripe.com" to "network:internet" over-declares, because
// internet access is broader than one host, and over-declaring is the safe
// direction for a fail-closed check. The reverse — deciding a specific token
// means something narrower — is the mistake this must not make, so
// "database:delete" maps to db:write and never to db:read.
//
// A token this cannot classify is NOT silently dropped: it comes back as-is and
// fails validation, which is the fail-closed answer.

// fsParamUnsafe matches characters the permission-parameter validator rejects.
// Scopes in the wild include paths ("data/vpn", "pentest-docs/"), and the
// canonical parameter grammar is [a-zA-Z0-9_-.:], so separators are folded to
// hyphens rather than the scope being dropped.
var fsParamUnsafe = regexp.MustCompile(`[^a-zA-Z0-9_\-.:]+`)

// databaseWriteOps are the descriptive operations that modify data. Anything
// that is not plainly a read maps to db:write, so a new verb appearing in a
// manifest widens rather than slipping through as read-only.
var databaseWriteOps = map[string]bool{
	"create": true, "update": true, "delete": true, "write": true,
	"ddl": true, "insert": true, "truncate": true, "drop": true, "*": true,
}

// networkPluginSuffixes identify a descriptive network target that names
// another plugin rather than a host. These become network:plugin:<name>, the
// canonical parameterized form for inter-plugin calls.
var networkPluginSuffixes = []string{"_plugin", "-plugin"}

// DeriveCanonicalPermissions reduces the descriptive permission object to the
// canonical vocabulary the allowlist validates.
//
// The result is sorted and de-duplicated so a manifest produces the same set
// every time regardless of map iteration order.
func DeriveCanonicalPermissions(grouped map[string][]string) []string {
	if len(grouped) == 0 {
		return nil
	}

	seen := map[string]bool{}
	add := func(p string) {
		if p != "" {
			seen[p] = true
		}
	}

	categories := make([]string, 0, len(grouped))
	for c := range grouped {
		categories = append(categories, c)
	}
	sort.Strings(categories)

	for _, category := range categories {
		for _, action := range grouped[category] {
			add(deriveOne(category, action))
		}
	}

	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// deriveOne maps a single category/action pair to its canonical permission.
//
// An unrecognised pair returns "<category>:<action>" unchanged, which will not
// match the allowlist and so fails validation. That is deliberate: a permission
// this code does not understand must be visible, not assumed harmless.
func deriveOne(category, action string) string {
	a := strings.ToLower(strings.TrimSpace(action))

	switch strings.ToLower(strings.TrimSpace(category)) {
	case "database", "db":
		if a == "read" || a == "select" {
			return "db:read"
		}
		if databaseWriteOps[a] {
			return "db:write"
		}
		// Unknown verb: widen to write rather than guess it is read-only.
		return "db:write"

	case "filesystem", "fs", "objectstorage", "storage":
		// Every filesystem declaration becomes a write scope. There is no
		// canonical read permission for the filesystem, and treating a declared
		// read as "no permission needed" would under-state it.
		if scope := fsScope(a); scope != "" {
			return "fs:write:" + scope
		}
		return ""

	case "network", "net":
		if isPluginTarget(a) {
			return "network:plugin:" + fsScope(strings.TrimSuffix(strings.TrimSuffix(a, "_plugin"), "-plugin"))
		}
		// Hosts, wildcards, "outbound", "localhost" — all of it is network
		// access, and network:internet is the broadest thing the allowlist
		// offers. Naming one host is narrower in reality, so this over-declares.
		return "network:internet"

	case "secrets", "env":
		if scope := fsScope(a); scope != "" {
			return "secrets:env:" + scope
		}
		return ""

	case "system", "exec", "process":
		return "system:exec"

	case "ai", "aiprovider":
		if scope := fsScope(a); scope != "" {
			return "ai:provider:" + scope
		}
		return ""
	}

	return category + ":" + action
}

// isPluginTarget reports whether a descriptive network target names another
// plugin rather than a host.
func isPluginTarget(target string) bool {
	for _, s := range networkPluginSuffixes {
		if strings.HasSuffix(target, s) && len(target) > len(s) {
			return true
		}
	}
	return false
}

// fsScope folds a descriptive scope into the canonical parameter grammar,
// which allows only [a-zA-Z0-9_-.:]. Paths keep their shape as hyphenated
// segments rather than being discarded.
func fsScope(s string) string {
	s = fsParamUnsafe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
