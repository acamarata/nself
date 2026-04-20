package doctor

import (
	"context"
	"testing"
)

// TestCheckHasuraIntrospection_WarnWhenAbsentInProd verifies that
// SEC-HASURA-INTRO-01 returns "warn" when the flag is absent in production.
// S70-T06.
func TestCheckHasuraIntrospection_WarnWhenAbsentInProd(t *testing.T) {
	t.Setenv("NSELF_ENV", "production")
	t.Setenv("HASURA_GRAPHQL_DISABLE_INTROSPECTION", "")

	result := CheckHasuraIntrospection(context.Background())

	if result.Status != "warn" {
		t.Errorf("expected status 'warn' when flag absent in production, got %q", result.Status)
	}
	if result.Section != "security" {
		t.Errorf("expected section 'security', got %q", result.Section)
	}
	if result.FixCmd == "" {
		t.Error("expected a FixCmd to be provided when warn")
	}
}

// TestCheckHasuraIntrospection_PassWhenSetInProd verifies that
// SEC-HASURA-INTRO-01 returns "pass" when the flag is "true" in production.
// S70-T06.
func TestCheckHasuraIntrospection_PassWhenSetInProd(t *testing.T) {
	t.Setenv("NSELF_ENV", "production")
	t.Setenv("HASURA_GRAPHQL_DISABLE_INTROSPECTION", "true")

	result := CheckHasuraIntrospection(context.Background())

	if result.Status != "pass" {
		t.Errorf("expected status 'pass' when flag is true in production, got %q", result.Status)
	}
}

// TestCheckHasuraIntrospection_SkipInDev verifies that
// SEC-HASURA-INTRO-01 returns "skip" in development environments.
// S70-T06.
func TestCheckHasuraIntrospection_SkipInDev(t *testing.T) {
	for _, env := range []string{"development", "dev", "staging", ""} {
		t.Run("NSELF_ENV="+env, func(t *testing.T) {
			t.Setenv("NSELF_ENV", env)
			t.Setenv("NODE_ENV", "")
			t.Setenv("HASURA_GRAPHQL_DISABLE_INTROSPECTION", "")

			result := CheckHasuraIntrospection(context.Background())

			if result.Status != "skip" {
				t.Errorf("expected status 'skip' for env %q, got %q", env, result.Status)
			}
		})
	}
}

// TestCheckHasuraIntrospection_NodeEnvFallback verifies that
// NODE_ENV=production is used when NSELF_ENV is unset.
// S70-T06.
func TestCheckHasuraIntrospection_NodeEnvFallback(t *testing.T) {
	t.Setenv("NSELF_ENV", "")
	t.Setenv("NODE_ENV", "production")
	t.Setenv("HASURA_GRAPHQL_DISABLE_INTROSPECTION", "")

	result := CheckHasuraIntrospection(context.Background())

	if result.Status != "warn" {
		t.Errorf("expected 'warn' when NODE_ENV=production and flag absent, got %q", result.Status)
	}
}
