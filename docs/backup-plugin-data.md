# Backup and Plugin Data

`nself backup create` uses pg_dumpall which includes ALL tables, including np_* plugin tables.

Plugin data preserved in backups:
- np_mux_rules, np_mux_inbox, np_mux_outbox (mux routing rules + message history)
- np_claw_memory, np_claw_sessions (AI assistant memory and sessions)
- np_ai_usage, np_ai_providers (AI usage stats + provider config)
- np_google_accounts, np_google_tokens (Google OAuth accounts)
- np_notification_subscriptions, np_notification_log (notification subscriptions)
- np_jobs, np_job_runs (cron jobs and execution history)
- np_voice_sessions, np_voice_recordings (voice session history)
- np_browser_sessions, np_browser_screenshots (browser session history)
- np_plugin_registry (installed plugins + versions)

After `nself backup restore`, run `nself plugin status` to verify all plugins show healthy.
