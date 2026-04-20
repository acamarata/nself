package doctor

import (
	"context"
	"testing"
)

// TestCheckModerationWired_PublicNoMod: public bind + ai loaded + no moderation → WARN.
func TestCheckModerationWired_PublicNoMod(t *testing.T) {
	t.Setenv("NSELF_BIND_ADDRESS", "0.0.0.0")
	t.Setenv("NSELF_AI_LOADED", "1")
	t.Setenv("PLUGIN_AI_INTERNAL_URL", "")
	t.Setenv("NSELF_MODERATION_LOADED", "")
	t.Setenv("PLUGIN_MODERATION_INTERNAL_URL", "")

	r := CheckModerationWired(context.Background())
	if r.Status != "warn" {
		t.Errorf("public + ai + no mod: want warn, got %q", r.Status)
	}
	if !containsStr(r.Message, "moderation not wired") {
		t.Errorf("want 'moderation not wired' in message, got %q", r.Message)
	}
}

// TestCheckModerationWired_PublicWithMod: public bind + ai + moderation loaded → pass.
func TestCheckModerationWired_PublicWithMod(t *testing.T) {
	t.Setenv("NSELF_BIND_ADDRESS", "0.0.0.0")
	t.Setenv("NSELF_AI_LOADED", "1")
	t.Setenv("PLUGIN_AI_INTERNAL_URL", "")
	t.Setenv("NSELF_MODERATION_LOADED", "1")
	t.Setenv("PLUGIN_MODERATION_INTERNAL_URL", "")

	r := CheckModerationWired(context.Background())
	if r.Status != "pass" {
		t.Errorf("public + ai + mod: want pass, got %q", r.Status)
	}
}

// TestCheckModerationWired_Loopback: loopback bind + ai + no moderation → pass.
func TestCheckModerationWired_Loopback(t *testing.T) {
	t.Setenv("NSELF_BIND_ADDRESS", "127.0.0.1")
	t.Setenv("NSELF_AI_LOADED", "1")
	t.Setenv("PLUGIN_AI_INTERNAL_URL", "")
	t.Setenv("NSELF_MODERATION_LOADED", "")
	t.Setenv("PLUGIN_MODERATION_INTERNAL_URL", "")

	r := CheckModerationWired(context.Background())
	if r.Status != "pass" {
		t.Errorf("loopback + ai + no mod: want pass, got %q", r.Status)
	}
}

// TestCheckModerationWired_NoAI: no ai plugin → pass (not applicable).
func TestCheckModerationWired_NoAI(t *testing.T) {
	t.Setenv("NSELF_BIND_ADDRESS", "0.0.0.0")
	t.Setenv("NSELF_AI_LOADED", "")
	t.Setenv("PLUGIN_AI_INTERNAL_URL", "")
	t.Setenv("NSELF_MODERATION_LOADED", "")
	t.Setenv("PLUGIN_MODERATION_INTERNAL_URL", "")

	r := CheckModerationWired(context.Background())
	if r.Status != "pass" {
		t.Errorf("no ai: want pass, got %q", r.Status)
	}
}

func containsStr(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}())
}
