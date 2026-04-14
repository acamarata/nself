package aiprofile

import (
	"os"
	"strings"
)

// FeatureFlags holds the AI_FEATURE_* master switches (T-05-08).
// All default to true (on). Flippable at runtime via admin settings.
type FeatureFlags struct {
	AutoProvision    bool // AI_FEATURE_AUTO_PROVISION (A2)
	LocalAutoInstall bool // AI_FEATURE_LOCAL_AUTO_INSTALL (A1)
	RoutingUI        bool // AI_FEATURE_ROUTING_UI (A3 UI)
	HealthDaemon     bool // AI_FEATURE_HEALTH_DAEMON (A5)
}

// LoadFeatureFlags reads AI_FEATURE_* env vars. Missing or empty = true (on).
func LoadFeatureFlags() FeatureFlags {
	return FeatureFlags{
		AutoProvision:    envBool("AI_FEATURE_AUTO_PROVISION", true),
		LocalAutoInstall: envBool("AI_FEATURE_LOCAL_AUTO_INSTALL", true),
		RoutingUI:        envBool("AI_FEATURE_ROUTING_UI", true),
		HealthDaemon:     envBool("AI_FEATURE_HEALTH_DAEMON", true),
	}
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}
