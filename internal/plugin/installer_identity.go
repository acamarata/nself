package plugin

// Purpose: best-effort per-plugin Ed25519 identity keypair generation and
// ping_api registration, run as Step 7b of installLocked. Split out of
// installer_locked.go for file size (engineering-standard <=300 lines, ASI
// Policy 3) when the P6-E3-W2-S1-T5 install-path-nesting fix (Step 6a,
// flattenExtractedPlugin) pushed that file over the cap.
// Inputs: context, pluginDir (doubles as the identity data root), plugin
// name.
// Outputs: none — failures are logged via slog.Warn, never returned, since
// the plugin still functions without JWT auth until Phase B-3 strict mode.
// Constraints: pure move (Q01); no behavior change. Only runs when
// PLUGIN_INTERNAL_SECRET is set (the operator has opted into the
// inter-plugin JWT system) and does not re-generate an existing identity key.

import (
	"context"
	"log/slog"
	"os"
)

// registerPluginIdentityIfEnabled generates a per-plugin Ed25519 identity
// keypair and registers the public key with ping_api, when the operator has
// opted into the inter-plugin JWT system (PLUGIN_INTERNAL_SECRET set) and no
// identity key already exists for this plugin. Best-effort: a failure here
// does not fail the install.
func registerPluginIdentityIfEnabled(ctx context.Context, pluginDir, name string) {
	if os.Getenv("PLUGIN_INTERNAL_SECRET") == "" {
		return
	}
	// pluginDir doubles as the identity data root — each plugin's keypair
	// is stored at pluginDir/<name>/identity.key alongside its manifest.
	if IdentityKeyExists(pluginDir, name) {
		return
	}
	pubKey, idErr := GenerateEd25519Keypair(pluginDir, name)
	if idErr != nil {
		slog.Warn("plugin identity key generation failed — inter-plugin JWT auth unavailable until resolved",
			"plugin", name, "error", idErr)
		return
	}
	if regErr := RegisterIdentity(ctx, name, pubKey); regErr != nil {
		slog.Warn("plugin identity registration with ping_api failed — JWT auth unavailable until resolved",
			"plugin", name, "error", regErr)
		return
	}
	slog.Info("plugin.identity.registered", "plugin", name)
}
