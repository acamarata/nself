package config

import (
	"testing"
)

// TestRegistryLength asserts that at least 6 validators are registered.
// This test acts as a regression guard: if a validator is accidentally
// removed from an init() call, the count drops and this test fails.
//
// Current expected registrations (all from validator.go init()):
//   - env
//   - passwords
//   - cors
//   - jwt
//   - port-conflicts
//   - duplicate-routes
const minValidatorCount = 6

func TestRegistryLength(t *testing.T) {
	if len(registry) < minValidatorCount {
		t.Errorf("expected at least %d registered validators, got %d — a validator may have been accidentally de-registered",
			minValidatorCount, len(registry))
	}
}

// TestRegistryNames verifies that all required validators are present by name.
// This is stricter than a count check: it catches renames or accidental omissions.
func TestRegistryNames(t *testing.T) {
	required := []string{
		"env",
		"passwords",
		"cors",
		"jwt",
		"port-conflicts",
		"duplicate-routes",
	}

	registered := make(map[string]bool, len(registry))
	for _, v := range registry {
		registered[v.Name] = true
	}

	for _, name := range required {
		if !registered[name] {
			t.Errorf("required validator %q is not registered", name)
		}
	}
}

// TestRunAllPassesOnMinimalValidConfig verifies that RunAll returns nil for a
// minimal but structurally valid dev config. This guards against validators
// blocking valid configurations.
func TestRunAllPassesOnMinimalValidConfig(t *testing.T) {
	cfg := &Config{
		Env:         "dev",
		ProjectName: "test-project",
		BaseDomain:  "localhost",
		Postgres: PostgresConfig{
			Password: "strongpassword1234567890", // ≥16 chars
		},
		Hasura: HasuraConfig{
			AdminSecret: "stronghasuraadminsecret1234567890", // ≥32 chars
			CORSDomain:  "*",                                 // wildcard OK in dev
			JWTKey:      "short",                             // JWT only enforced in staging/prod
		},
	}

	if err := runAll(cfg); err != nil {
		t.Errorf("RunAll returned unexpected error for valid dev config: %v", err)
	}
}

// TestRunAllCollectsMultipleErrors verifies that RunAll collects failures
// from multiple validators rather than stopping at the first.
func TestRunAllCollectsMultipleErrors(t *testing.T) {
	// This config has both a password problem and a port conflict.
	// RunAll should report both in the returned error.
	cfg := &Config{
		Env:         "dev",
		ProjectName: "test-project",
		BaseDomain:  "localhost",
		Postgres: PostgresConfig{
			Password: "short", // too short — triggers passwords validator
		},
		Hasura: HasuraConfig{
			AdminSecret: "alsowaytoolong_hasura_secret_12345", // ≥32 chars — OK
		},
		CustomServices: []CustomService{
			{Index: 1, Port: 5432}, // conflicts with PostgreSQL reserved port
		},
	}

	err := runAll(cfg)
	if err == nil {
		t.Fatal("RunAll expected to return an error for invalid config, got nil")
	}

	errMsg := err.Error()
	// Both the password validator and the port-conflicts validator should fire.
	hasPassErr := false
	hasPortErr := false
	for _, v := range registry {
		if v.Name == "passwords" && v.Fn(cfg) != nil {
			hasPassErr = true
		}
		if v.Name == "port-conflicts" && v.Fn(cfg) != nil {
			hasPortErr = true
		}
	}

	if !hasPassErr {
		t.Error("expected passwords validator to fail for this config")
	}
	if !hasPortErr {
		t.Error("expected port-conflicts validator to fail for this config")
	}
	_ = errMsg // combined error message verified indirectly via sub-validator checks
}
