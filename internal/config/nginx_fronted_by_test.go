package config

// nginx_fronted_by_test.go — coverage for NGINX_FRONTED_BY, the env var that
// declares this project's ingress as another stack's nginx (2026-09-03
// staging incident: ntask was fronted by nself-web's nginx but still
// generated an nginx service that could never bind 80/443).
//
// Purpose: prove the var is parsed into NginxConfig.FrontedBy and is
// registered in knownEnvVars, so setting it does not make every build print
// a false "unknown env var" warning.
// Inputs: the process environment (t.Setenv) and the knownEnvVars list.
// Outputs: pass/fail on parse and on schema registration.
// Constraints: parseEnvCore reads os.Getenv directly, so this exercises it
// through t.Setenv rather than a fixture file.

import "testing"

// TestFrontedByParsedFromEnv verifies NGINX_FRONTED_BY reaches
// NginxConfig.FrontedBy, and that leaving it unset yields the empty default
// every existing project relies on.
func TestFrontedByParsedFromEnv(t *testing.T) {
	t.Setenv("NGINX_FRONTED_BY", "nself-web")
	cfg := &Config{}
	parseEnvCore(cfg)
	if cfg.Nginx.FrontedBy != "nself-web" {
		t.Errorf("Nginx.FrontedBy = %q, want %q", cfg.Nginx.FrontedBy, "nself-web")
	}

	t.Setenv("NGINX_FRONTED_BY", "")
	unset := &Config{}
	parseEnvCore(unset)
	if unset.Nginx.FrontedBy != "" {
		t.Errorf("Nginx.FrontedBy = %q with the var unset, want empty", unset.Nginx.FrontedBy)
	}
}

// TestFrontedByIsAKnownEnvVar guards the half of the change that is easy to
// forget: a var the loader reads but knownEnvVars does not list makes
// warnUnknownEnvVars print a warning for correct configuration on every
// single build.
func TestFrontedByIsAKnownEnvVar(t *testing.T) {
	for _, k := range knownEnvVars {
		if k == "NGINX_FRONTED_BY" {
			return
		}
	}
	t.Error("NGINX_FRONTED_BY is read by parseEnvCore but is missing from knownEnvVars — every build that sets it would print a false unknown-var warning")
}
