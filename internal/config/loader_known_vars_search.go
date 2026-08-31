package config

// loader_known_vars_search.go — search provider + OAuth + app-level env var
// names (MeiliSearch, OpenSearch, Zinc, Sonic, Dashboard, legacy microservice,
// plugin integrations, social OAuth, push, observability, app-owned, ENV_ALLOWLIST).
// Split from loader_known_vars.go (T-P6-E2-W1-S1-T3).
// Purpose: fourth quarter of the knownEnvVars list, combined in loader_known_vars.go.
// Inputs:  none. Outputs: knownEnvVarsSearch []string.
// Constraints: keep entries verbatim and in original order; see loader_known_vars.go header.

var knownEnvVarsSearch = []string{
	// MeiliSearch search engine (plugin-managed: injected into search compose template).
	"MEILISEARCH_ENABLED",
	"MEILISEARCH_PORT",
	"MEILISEARCH_ROUTE",
	"MEILI_NO_ANALYTICS",
	// OpenSearch search provider (plugin-managed: opensearch plugin compose template).
	"OPENSEARCH_VERSION",
	"OPENSEARCH_PORT",
	"OPENSEARCH_PASSWORD",
	"OPENSEARCH_MEMORY",
	// Zinc search provider (plugin-managed: zinc plugin compose template).
	"ZINC_VERSION",
	"ZINC_PORT",
	"ZINC_ADMIN_USER",
	"ZINC_ADMIN_PASSWORD",
	// Sonic search provider (plugin-managed: sonic plugin compose template).
	"SONIC_VERSION",
	"SONIC_PORT",
	"SONIC_PASSWORD",
	// Dashboard plugin (plugin-managed: dashboard plugin compose template).
	"DASHBOARD_ENABLED",
	"DASHBOARD_VERSION",
	"DASHBOARD_ROUTE",
	"DASHBOARD_PORT",
	// Legacy microservice system (plugin-managed: pre-CS_N system; may appear in old .env files).
	"SERVICES_ENABLED",
	"NESTJS_ENABLED",
	"NESTJS_SERVICES",
	"NESTJS_USE_TYPESCRIPT",
	"NESTJS_PORT_START",
	"BULLMQ_ENABLED",
	"BULLMQ_WORKERS",
	"BULLMQ_DASHBOARD_ENABLED",
	"BULLMQ_DASHBOARD_PORT",
	"BULLMQ_DASHBOARD_ROUTE",
	"GOLANG_ENABLED",
	"GOLANG_SERVICES",
	"GOLANG_PORT_START",
	"PYTHON_ENABLED",
	"PYTHON_SERVICES",
	"PYTHON_FRAMEWORK",
	"PYTHON_PORT_START",
	// Plugin integration vars (plugin-managed: stripe, github, shopify plugin compose templates).
	"STRIPE_API_KEY",
	"STRIPE_WEBHOOK_SECRET",
	"STRIPE_SYNC_INTERVAL",
	"GITHUB_TOKEN",
	"GITHUB_WEBHOOK_SECRET",
	"GITHUB_ORG",
	"GITHUB_REPOS",
	"SHOPIFY_STORE",
	"SHOPIFY_ACCESS_TOKEN",
	"SHOPIFY_API_VERSION",
	"SHOPIFY_WEBHOOK_SECRET",
	"SHOPIFY_SYNC_INTERVAL",
	// Social OAuth env vars — auth server (P2-E2-W3-S5-T16 + P2-E2-W3-S6-T17)
	// Shared
	"OAUTH_REDIRECT_ALLOWLIST",
	"OAUTH_TOKEN_ENCRYPTION_KEY",
	// Tier-1 (T16)
	"GOOGLE_OAUTH_CLIENT_ID",
	"GOOGLE_OAUTH_CLIENT_SECRET",
	"APPLE_OAUTH_CLIENT_ID",
	"APPLE_OAUTH_TEAM_ID",
	"APPLE_OAUTH_KEY_ID",
	"APPLE_OAUTH_PRIVATE_KEY",
	// Note: Apple does not use a static client_secret; the above key material
	// is used to generate a short-lived ES256 JWT per RFC 7636 on every token exchange.
	"GITHUB_OAUTH_CLIENT_ID",
	"GITHUB_OAUTH_CLIENT_SECRET",
	"FACEBOOK_OAUTH_CLIENT_ID",
	"FACEBOOK_OAUTH_CLIENT_SECRET",
	"TWITTER_OAUTH_CLIENT_ID",
	"TWITTER_OAUTH_CLIENT_SECRET",
	// Tier-2 Group A (T17a) — LinkedIn, Discord, Slack, Twitch
	"LINKEDIN_OAUTH_CLIENT_ID",
	"LINKEDIN_OAUTH_CLIENT_SECRET",
	"DISCORD_OAUTH_CLIENT_ID",
	"DISCORD_OAUTH_CLIENT_SECRET",
	"SLACK_OAUTH_CLIENT_ID",
	"SLACK_OAUTH_CLIENT_SECRET",
	"TWITCH_OAUTH_CLIENT_ID",
	"TWITCH_OAUTH_CLIENT_SECRET",
	// Tier-2 Group B (T17b) — Spotify, TikTok, Reddit, Microsoft
	"SPOTIFY_OAUTH_CLIENT_ID",
	"SPOTIFY_OAUTH_CLIENT_SECRET",
	"TIKTOK_OAUTH_CLIENT_ID",
	"TIKTOK_OAUTH_CLIENT_SECRET",
	"REDDIT_OAUTH_CLIENT_ID",
	"REDDIT_OAUTH_CLIENT_SECRET",
	"MICROSOFT_OAUTH_CLIENT_ID",
	"MICROSOFT_OAUTH_CLIENT_SECRET",
	// Tier-2 Gap-list (T17d) — reaching 21 total
	"PINTEREST_OAUTH_CLIENT_ID",
	"PINTEREST_OAUTH_CLIENT_SECRET",
	"DROPBOX_OAUTH_CLIENT_ID",
	"DROPBOX_OAUTH_CLIENT_SECRET",
	"ZOOM_OAUTH_CLIENT_ID",
	"ZOOM_OAUTH_CLIENT_SECRET",
	"ATLASSIAN_OAUTH_CLIENT_ID",
	"ATLASSIAN_OAUTH_CLIENT_SECRET",
	"GITLAB_OAUTH_CLIENT_ID",
	"GITLAB_OAUTH_CLIENT_SECRET",
	"BITBUCKET_OAUTH_CLIENT_ID",
	"BITBUCKET_OAUTH_CLIENT_SECRET",
	"FIGMA_OAUTH_CLIENT_ID",
	"FIGMA_OAUTH_CLIENT_SECRET",
	"NOTION_OAUTH_CLIENT_ID",
	"NOTION_OAUTH_CLIENT_SECRET",
	// Total: 5 Tier-1 + 16 Tier-2 = 21 OAuth providers (full Zernio parity)

	// App-level push notification credentials (mobile/RN apps talking directly
	// to APNs/FCM, distinct from the notify plugin's VAPID web-push keys above).
	// Not read by the CLI loader — listed to suppress false warnings.
	"APNS_KEY_ID",
	"APNS_KEY_PATH",
	"APNS_TEAM_ID",
	"FCM_SERVER_KEY",

	// App-level observability DSN (e.g. Sentry backend project). Not read by
	// the CLI loader — apps wire this into their own error reporting.
	"SENTRY_DSN_BACKEND",

	// App-owned vars commonly present in real app .env files (read by the
	// app's own code / Node runtime / dev scripts, not by the CLI loader).
	// Listed to suppress false "unknown env var" warnings (ntask dogfood
	// gap #19). Project-specific vars beyond these belong in ENV_ALLOWLIST.
	"NODE_ENV",
	"JWT_SECRET",
	"SSL_AUTO_TRUST",
	"COOKIE_SECRET",
	"ENABLE_DEBUG",
	"LOG_LEVEL",
	"NSELF_PROJECT_NAME",

	// ENV_ALLOWLIST itself: comma-separated var names (or prefixes ending
	// in *) that warnUnknownEnvVars treats as app-owned and never warns
	// about. Example: ENV_ALLOWLIST=MY_APP_TOKEN,FEATURE_*
	"ENV_ALLOWLIST",
}
