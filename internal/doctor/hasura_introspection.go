package doctor

import (
	"context"
	"os"
	"strings"
)

// CheckHasuraIntrospection implements SEC-HASURA-INTRO-01.
// It warns if HASURA_GRAPHQL_DISABLE_INTROSPECTION is not set to "true"
// in a production environment (NSELF_ENV=production or NODE_ENV=production).
//
// The check is:
//   - status "skip": environment is dev/staging or NSELF_ENV is unset/empty
//   - status "pass": HASURA_GRAPHQL_DISABLE_INTROSPECTION=true in production
//   - status "warn": HASURA_GRAPHQL_DISABLE_INTROSPECTION absent or not "true" in production
//
// The check is read-only — it never modifies Hasura configuration.
// S70-T06
func CheckHasuraIntrospection(_ context.Context) CheckResult {
	const (
		checkID   = "SEC-HASURA-INTRO-01"
		section   = "security"
		checkName = checkID + ": Hasura introspection disabled in production"
	)

	// Determine environment. Accept NSELF_ENV first, fall back to NODE_ENV.
	env := strings.ToLower(strings.TrimSpace(os.Getenv("NSELF_ENV")))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("NODE_ENV")))
	}

	// Only enforce in production environments.
	if env != "production" && env != "prod" {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "skip",
			Message: "not a production environment — introspection check skipped",
		}
	}

	// In production: HASURA_GRAPHQL_DISABLE_INTROSPECTION must be "true".
	val := strings.TrimSpace(os.Getenv("HASURA_GRAPHQL_DISABLE_INTROSPECTION"))
	if strings.EqualFold(val, "true") {
		return CheckResult{
			Section: section,
			Name:    checkName,
			Status:  "pass",
			Message: "HASURA_GRAPHQL_DISABLE_INTROSPECTION=true in production",
		}
	}

	// Missing or not "true" in production.
	msg := "HASURA_GRAPHQL_DISABLE_INTROSPECTION is not set to 'true' in production — schema enumeration is possible"
	if val != "" {
		msg = "HASURA_GRAPHQL_DISABLE_INTROSPECTION=" + val + " (expected 'true') in production"
	}

	return CheckResult{
		Section: section,
		Name:    checkName,
		Status:  "warn",
		Message: msg,
		FixCmd:  "nself env set HASURA_GRAPHQL_DISABLE_INTROSPECTION=true --env prod",
	}
}
