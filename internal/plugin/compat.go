package plugin

import (
	"fmt"
	"strconv"
	"strings"
)

// CompatBlock describes the compatibility requirements declared by a plugin
// manifest. The Nself field is a semver range constraint for the CLI version
// (e.g. ">=1.0.0 <2.0.0"). Requires maps service names to their own semver
// range constraints (e.g. "postgres": ">=14", "hasura": ">=2.30").
type CompatBlock struct {
	Nself    string            `json:"nself,omitempty" yaml:"nself,omitempty"`
	Requires map[string]string `json:"requires,omitempty" yaml:"requires,omitempty"`
}

// semverConstraint represents a single version comparison operator + version.
type semverConstraint struct {
	op      string // ">=", "<=", ">", "<", "=", "!="
	version [3]int
}

// CheckCLICompat checks whether cliVersion satisfies the plugin's compat.nself
// range. Returns nil if no compat block is set or if the version is in range.
func CheckCLICompat(compat *CompatBlock, cliVersion string) error {
	if compat == nil || compat.Nself == "" {
		return nil
	}
	ok, err := SatisfiesRange(cliVersion, compat.Nself)
	if err != nil {
		return fmt.Errorf("parsing compat range %q: %w", compat.Nself, err)
	}
	if !ok {
		return fmt.Errorf("CLI version %s does not satisfy plugin requirement %q", cliVersion, compat.Nself)
	}
	return nil
}

// CheckServiceCompat checks whether each service version in actual satisfies
// the corresponding constraint in compat.requires. Returns a list of services
// that fail the check.
func CheckServiceCompat(compat *CompatBlock, actual map[string]string) []string {
	if compat == nil || len(compat.Requires) == 0 {
		return nil
	}
	var failures []string
	for svc, constraint := range compat.Requires {
		ver, ok := actual[svc]
		if !ok {
			failures = append(failures, fmt.Sprintf("%s: not available (requires %s)", svc, constraint))
			continue
		}
		sat, err := SatisfiesRange(ver, constraint)
		if err != nil || !sat {
			failures = append(failures, fmt.Sprintf("%s: %s does not satisfy %s", svc, ver, constraint))
		}
	}
	return failures
}

// SatisfiesRange checks whether version satisfies a space-separated list of
// semver constraints. All constraints must be satisfied (AND logic).
// Supported operators: >=, <=, >, <, =, !=
// Examples: ">=1.0.0 <2.0.0", ">=14", ">=2.30"
func SatisfiesRange(version, rangeStr string) (bool, error) {
	ver, err := parseSemverTuple(version)
	if err != nil {
		return false, fmt.Errorf("parsing version %q: %w", version, err)
	}

	constraints, err := parseConstraints(rangeStr)
	if err != nil {
		return false, err
	}

	for _, c := range constraints {
		if !c.check(ver) {
			return false, nil
		}
	}
	return true, nil
}

// IncompatiblePlugins checks all installed plugins and returns a list of those
// whose compat.nself range will not be satisfied by newCLIVersion. This is used
// by `nself upgrade` to warn the user before upgrading.
func IncompatiblePlugins(pluginDir, newCLIVersion string) []string {
	installed, err := ListInstalled(pluginDir)
	if err != nil {
		return nil
	}
	var incompatible []string
	for _, p := range installed {
		manifest, err := parseManifest(pluginDir + "/" + p.Name + "/plugin.json")
		if err != nil {
			continue
		}
		if manifest.Compat == nil || manifest.Compat.Nself == "" {
			continue
		}
		ok, err := SatisfiesRange(newCLIVersion, manifest.Compat.Nself)
		if err != nil || !ok {
			incompatible = append(incompatible, fmt.Sprintf("%s (requires nself %s)", p.Name, manifest.Compat.Nself))
		}
	}
	return incompatible
}

// --- parsing helpers ---

// parseSemverTuple parses a version string like "1.2.3", "v1.2.3", "14", or
// "2.30" into a [3]int tuple. Missing segments default to 0.
func parseSemverTuple(s string) ([3]int, error) {
	s = strings.TrimPrefix(s, "v")
	// Strip pre-release suffix (e.g. "-beta.1") for comparison purposes.
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		s = s[:idx]
	}
	// Strip build metadata (e.g. "+build123").
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		s = s[:idx]
	}

	parts := strings.Split(s, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return result, fmt.Errorf("invalid version segment %q", parts[i])
		}
		result[i] = n
	}
	return result, nil
}

// parseConstraints splits a range string on whitespace and parses each token
// as an operator+version constraint. Bare versions (no operator) are treated
// as "=" (exact match).
func parseConstraints(rangeStr string) ([]semverConstraint, error) {
	tokens := strings.Fields(strings.TrimSpace(rangeStr))
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty constraint range")
	}

	var constraints []semverConstraint
	for _, tok := range tokens {
		c, err := parseOneConstraint(tok)
		if err != nil {
			return nil, fmt.Errorf("parsing constraint %q: %w", tok, err)
		}
		constraints = append(constraints, c)
	}
	return constraints, nil
}

func parseOneConstraint(s string) (semverConstraint, error) {
	var op string
	var verStr string

	for _, prefix := range []string{">=", "<=", "!=", ">", "<", "="} {
		if strings.HasPrefix(s, prefix) {
			op = prefix
			verStr = s[len(prefix):]
			break
		}
	}
	if op == "" {
		// Bare version: treat as exact match.
		op = "="
		verStr = s
	}

	ver, err := parseSemverTuple(verStr)
	if err != nil {
		return semverConstraint{}, err
	}
	return semverConstraint{op: op, version: ver}, nil
}

func (c semverConstraint) check(ver [3]int) bool {
	cmp := compareTuples(ver, c.version)
	switch c.op {
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case "=":
		return cmp == 0
	case "!=":
		return cmp != 0
	default:
		return false
	}
}

func compareTuples(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
