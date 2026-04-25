package commands

import (
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/ui"
)

// initPreset describes a named project preset for nself init --preset.
type initPreset struct {
	Name        string
	Description string
	// SuggestedPlugins are the pro plugin bundle slugs relevant for this preset.
	SuggestedPlugins []string
	// EnabledServices are the optional service flags this preset recommends enabling.
	EnabledServices []string
	// Notes are printed after init completes.
	Notes []string
}

// initPresets is the canonical registry of built-in presets.
var initPresets = map[string]initPreset{
	"b2b-saas": {
		Name:             "b2b-saas",
		Description:      "Multi-tenant B2B SaaS backend with auth, billing, and team management",
		SuggestedPlugins: []string{"auth", "billing", "notify", "realtime"},
		EnabledServices:  []string{"redis", "functions"},
		Notes: []string{
			"Add team/org tables: nself db migrate",
			"Install billing plugin: nself plugin install billing",
			"Set up Stripe keys in .env.secrets",
		},
	},
	"mobile-backend": {
		Name:             "mobile-backend",
		Description:      "Mobile app backend with push notifications, offline sync, and real-time subscriptions",
		SuggestedPlugins: []string{"notify", "sync", "realtime", "auth"},
		EnabledServices:  []string{"redis"},
		Notes: []string{
			"Configure push provider in .env (FCM/APNS keys)",
			"Install sync plugin: nself plugin install sync",
			"Enable real-time: nself plugin install realtime",
		},
	},
	"ai-assistant": {
		Name:             "ai-assistant",
		Description:      "AI assistant backend with vector search, memory, and multi-model routing (ɳClaw-style)",
		SuggestedPlugins: []string{"ai", "claw", "mux", "browser", "voice"},
		EnabledServices:  []string{"redis", "search"},
		Notes: []string{
			"Set AI provider keys in .env.secrets (OPENAI_API_KEY / ANTHROPIC_API_KEY)",
			"Install AI bundle: nself plugin install ai mux",
			"pgvector is pre-enabled for vector search",
		},
	},
	"community-forum": {
		Name:             "community-forum",
		Description:      "Community forum backend with moderation, voting, and email digests",
		SuggestedPlugins: []string{"notify", "moderation", "cms", "auth"},
		EnabledServices:  []string{"mail", "search"},
		Notes: []string{
			"Configure mail provider in .env (SMTP settings)",
			"Install moderation plugin: nself plugin install moderation",
		},
	},
	"media-hosting": {
		Name:             "media-hosting",
		Description:      "Media hosting and streaming backend with transcoding and CDN integration",
		SuggestedPlugins: []string{"media-processing", "streaming", "file-processing", "subtitle-manager"},
		EnabledServices:  []string{"minio"},
		Notes: []string{
			"MinIO (S3-compatible storage) is enabled — configure MINIO_ACCESS_KEY in .env",
			"Install nTV bundle: nself plugin install media-processing streaming",
		},
	},
}

// listInitPresets prints all available presets to stdout.
func listInitPresets() {
	fmt.Println()
	ui.Section("Available presets (--preset <name>)")
	fmt.Println()
	for _, name := range []string{"b2b-saas", "mobile-backend", "ai-assistant", "community-forum", "media-hosting"} {
		p := initPresets[name]
		fmt.Printf("  %s%-20s%s  %s\n", ui.Bold, p.Name, ui.Reset, p.Description)
		if len(p.SuggestedPlugins) > 0 {
			fmt.Printf("    %sPlugins:%s %s\n", ui.Dim, ui.Reset, strings.Join(p.SuggestedPlugins, ", "))
		}
		fmt.Println()
	}
}

// printPresetPostInit prints preset-specific post-init notes.
func printPresetPostInit(presetName string) {
	p, ok := initPresets[presetName]
	if !ok {
		return
	}
	fmt.Println()
	ui.Section(fmt.Sprintf("Preset: %s — next steps", p.Name))
	for i, note := range p.Notes {
		fmt.Printf("  %s%d.%s %s\n", ui.Bold, i+1, ui.Reset, note)
	}
	fmt.Println()
}
