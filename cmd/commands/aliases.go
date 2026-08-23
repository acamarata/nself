package commands

import (
	"os"
	"path/filepath"
	"strings"
)

// normalizeInvokedBinary rewrites os.Args when the binary is invoked through
// a product-alias name, so a symlink gives a dedicated CLI for free:
//
//	ln -s $(which nself) /usr/local/bin/nsentry
//	nsentry monitors list   ≡   nself sentry monitors list
//
// Called from Execute() before any arg parsing (never from init(): init must
// only do cobra registration). Idempotent: skips when the subcommand is
// already present.
func normalizeInvokedBinary() {
	if len(os.Args) == 0 {
		return
	}
	base := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	if base != "nsentry" {
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "sentry" {
		return
	}
	rewritten := make([]string, 0, len(os.Args)+1)
	rewritten = append(rewritten, os.Args[0], "sentry")
	rewritten = append(rewritten, os.Args[1:]...)
	os.Args = rewritten
}

// NOTE on `up` / `down`:
//
// These used to be registered here as hidden sibling commands wrapping
// startCmd.RunE / stopCmd.RunE. They were dead code: startCmd and stopCmd
// already declare Aliases{"up"} / Aliases{"down"}, and cobra resolves an alias
// to the real command before it ever reaches a same-named sibling. Worse, the
// wrappers copied only RunE, so had they ever been reached, `nself up
// --allow-legacy` and every other start flag would have failed to parse.
//
// The canonical aliases live on startCmd/stopCmd. Their deprecation warnings
// come from internal/deprecation/registry.yaml, keyed on the spelling the user
// typed — see invokedCommandPath in root.go.
