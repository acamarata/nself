// Purpose:     decide whether the env vars a command's own source reads are
//
//	documented in .github/wiki/Config-Env-Vars.md.
//
// Inputs:      cmd/commands/<name>*.go (per cli/.claude/rules/go.md, a
//
//	command's subcommands split into "<name>_<subgroup>.go" files
//	sharing the same prefix — the same convention tools/wikigen and
//	tools/cmdinventory rely on for one-command-per-file), plus the
//	Config-Env-Vars.md wiki page.
//
// Outputs:     scoreEnvVars returns "n/a" (no direct env reads found),
//
//	"documented" (every found var is documented), or
//	"undocumented: VAR, VAR" (lists the gaps).
//
// Constraints: DELIBERATELY CONSERVATIVE / UNDER-COUNTS. Only direct
//
//	os.Getenv/os.LookupEnv/viper.Get*/viper.IsSet/viper.BindEnv calls
//	in the command's own cmd/commands files are seen — env vars read
//	transitively through internal/* packages (the common case; most
//	commands delegate to internal/config, internal/database, etc.)
//	are invisible to this check and will not appear here. That is a
//	known, documented limitation, not a bug: a false "n/a" undercounts
//	but never wrongly flags a documented var as missing. This is why
//	the CI gate treats this column as a WARN, never a FAIL.
package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var envReadRe = regexp.MustCompile(
	`(?:os\.(?:Getenv|LookupEnv)|viper\.(?:Get\w*|IsSet|BindEnv))\(\s*"([A-Z][A-Z0-9_]*)"`,
)

// envVarsForCommand globs cmd/commands/<name>*.go and returns the sorted,
// de-duplicated set of env var names those files read directly.
func envVarsForCommand(commandsDir, name string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(commandsDir, name+"*.go"))
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, path := range matches {
		if filepath.Ext(path) != ".go" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue // best-effort: a glob hit that vanished mid-scan is not fatal
		}
		for _, m := range envReadRe.FindAllStringSubmatch(string(data), -1) {
			seen[m[1]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

// scoreEnvVars cross-checks vars against the Config-Env-Vars.md content
// (matched as a `VARNAME` code span, the page's documented convention).
func scoreEnvVars(vars []string, envDoc []byte) string {
	if len(vars) == 0 {
		return "n/a"
	}
	doc := string(envDoc)
	var missing []string
	for _, v := range vars {
		if !containsCodeSpan(doc, v) {
			missing = append(missing, v)
		}
	}
	if len(missing) == 0 {
		return "documented"
	}
	return "undocumented: " + joinComma(missing)
}

func containsCodeSpan(doc, name string) bool {
	return strings.Contains(doc, "`"+name+"`")
}

func joinComma(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
