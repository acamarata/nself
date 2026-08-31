package config

// loader_known_vars.go — authoritative list of known environment variable names.
//
// Purpose: Enumerate every env var name that the nSelf config loader and
//          ApplyDefaults recognise. Used by warnUnknownEnvVars to surface
//          typos in user .env files (any key not in this list and not matching
//          a dynamic prefix emits a slog.Warn — it never fails the load).
// Inputs:  none (package-level var, referenced by loader.go and defaults.go).
// Outputs: knownEnvVars []string — consumed by warnUnknownEnvVars in warn.go.
// Constraints: Keep in sync with parseEnvToConfig (loader_parse_env.go) and
//              ApplyDefaults (defaults.go). Plugin-managed vars that the CLI
//              loader does NOT read are included at the bottom to suppress false
//              "unknown env var" warnings from compose-injected config.
// SPORT:   cli/internal/config — decomposed from loader.go (T-E2-06), further
//          split into loader_known_vars_{core,storage,ops,search}.go for
//          300-line compliance (T-P6-E2-W1-S1-T3).

// knownEnvVars is the concatenation of the four category slices below, in
// original list order. combineKnownEnvVars keeps that concatenation in one
// place so the split is purely mechanical (no entry moved, reordered, or
// changed).
var knownEnvVars = combineKnownEnvVars()

func combineKnownEnvVars() []string {
	out := make([]string, 0, len(knownEnvVarsCore)+len(knownEnvVarsStorage)+len(knownEnvVarsOps)+len(knownEnvVarsSearch))
	out = append(out, knownEnvVarsCore...)
	out = append(out, knownEnvVarsStorage...)
	out = append(out, knownEnvVarsOps...)
	out = append(out, knownEnvVarsSearch...)
	return out
}
