package config

import (
	"fmt"
	"os"
)

// ParseFrontendApps parses FRONTEND_APP_1 through FRONTEND_APP_20 environment
// variables into FrontendApp structs. Each frontend app is defined by a set of
// per-index env vars:
//
//	FRONTEND_APP_N_DISPLAY_NAME  — required (skip if empty); falls back to FRONTEND_APP_N_NAME (v1 compat)
//	FRONTEND_APP_N_SYSTEM_NAME
//	FRONTEND_APP_N_PORT          — default: 3000 + N - 1
//	FRONTEND_APP_N_ROUTE
//	FRONTEND_APP_N_FRAMEWORK
//	FRONTEND_APP_N_TABLE_PREFIX
//	FRONTEND_APP_N_IMAGE         — optional Docker image reference; hard error if value is invalid
func parseFrontendApps() ([]FrontendApp, error) {
	var apps []FrontendApp
	for i := 1; i <= 20; i++ {
		prefix := fmt.Sprintf("FRONTEND_APP_%d_", i)
		name := os.Getenv(prefix + "DISPLAY_NAME")
		if name == "" {
			// backward compat: v1 used FRONTEND_APP_N_NAME
			name = os.Getenv(prefix + "NAME")
		}
		if name == "" {
			continue // No app at this index
		}

		// Sanitize DisplayName — reject injection characters before building the app.
		sanitizedDisplayName, err := SanitizeName(name)
		if err != nil {
			continue
		}

		app := FrontendApp{
			Index:       i,
			DisplayName: sanitizedDisplayName,
			SystemName:  os.Getenv(prefix + "SYSTEM_NAME"),
			Port:        getEnvInt(prefix+"PORT", 3000+i-1),
			Route:       os.Getenv(prefix + "ROUTE"),
			Framework:   os.Getenv(prefix + "FRAMEWORK"),
			TablePrefix: os.Getenv(prefix + "TABLE_PREFIX"),
		}
		if app.SystemName != "" {
			sanitized, err := SanitizeName(app.SystemName)
			if err != nil {
				continue
			}
			app.SystemName = sanitized
		}
		if app.Route != "" {
			sanitized, err := SanitizeDomain(app.Route)
			if err != nil {
				continue
			}
			app.Route = sanitized
		}

		// Validate image reference — hard error to prevent shell-injection via docker compose.
		app.Image = os.Getenv(prefix + "IMAGE")
		if app.Image != "" {
			if err := validateFrontendAppImage(app.Image); err != nil {
				return nil, fmt.Errorf("FRONTEND_APP_%d_IMAGE: %w", i, err)
			}
		}

		apps = append(apps, app)
	}
	return apps, nil
}
