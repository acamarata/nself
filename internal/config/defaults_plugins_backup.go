package config

// defaults_plugins_backup.go — default value application for the plugin system, backups and email.
//
// Purpose: Fill in unset plugin system (registry, limits, secrets), backup and email fields on a loaded Config with the CLI's standard defaults, split out of defaults.go for file size.
// Inputs: a *Config already populated by the loader.
// Outputs: the same *Config with these fields defaulted in place.
// Constraints: pure move from defaults.go (CLI-R12 Batch F); no behaviour change. Keep in sync with ApplyDefaults in defaults.go, which calls these in order.

import (
	"fmt"
	"log/slog"
)

// applyDefaultsPlugins sets plugin system registry, cache, and pro-plugin secret defaults.
func applyDefaultsPlugins(cfg *Config) error {
	applyDefaultsPluginSystem(cfg)
	return applyDefaultsPluginSecrets(cfg)
}

// applyDefaultsPluginSystem sets plugin directory, registry, URL, routing, and resource limit defaults.
func applyDefaultsPluginSystem(cfg *Config) {
	applyDefaultsPluginSystemRegistry(cfg)
	applyDefaultsPluginSystemLimits(cfg)
}

// applyDefaultsPluginSystemRegistry sets plugin dir, cache, registry URLs, and routing.
func applyDefaultsPluginSystemRegistry(cfg *Config) {
	if cfg.PluginSystem.Dir == "" {
		cfg.PluginSystem.Dir = "~/.nself/plugins"
		slog.Debug("default", "key", "PLUGIN_DIR", "value", cfg.PluginSystem.Dir)
	}
	if cfg.PluginSystem.Cache == "" {
		cfg.PluginSystem.Cache = "~/.nself/cache/plugins"
		slog.Debug("default", "key", "PLUGIN_CACHE", "value", cfg.PluginSystem.Cache)
	}
	if cfg.PluginSystem.Registry == "" {
		cfg.PluginSystem.Registry = "https://plugins.nself.org"
		slog.Debug("default", "key", "PLUGIN_REGISTRY", "value", cfg.PluginSystem.Registry)
	}
	if cfg.PluginSystem.CacheTTL == 0 {
		cfg.PluginSystem.CacheTTL = 300
		slog.Debug("default", "key", "PLUGIN_CACHE_TTL", "value", fmt.Sprintf("%d", cfg.PluginSystem.CacheTTL))
	}
	if cfg.PluginSystem.PingURL == "" {
		cfg.PluginSystem.PingURL = "https://ping.nself.org"
		slog.Debug("default", "key", "PLUGIN_PING_URL", "value", cfg.PluginSystem.PingURL)
	}
	if cfg.PluginSystem.PricingURL == "" {
		cfg.PluginSystem.PricingURL = "https://nself.org/pricing"
		slog.Debug("default", "key", "PLUGIN_PRICING_URL", "value", cfg.PluginSystem.PricingURL)
	}
	if cfg.PluginConfig.NotifyPort == 0 {
		cfg.PluginConfig.NotifyPort = 3712
		slog.Debug("default", "key", "PLUGIN_NOTIFY_PORT", "value", fmt.Sprintf("%d", cfg.PluginConfig.NotifyPort))
	}
	if cfg.PluginConfig.NotifyRoute == "" {
		cfg.PluginConfig.NotifyRoute = "notify"
		slog.Debug("default", "key", "PLUGIN_NOTIFY_ROUTE", "value", cfg.PluginConfig.NotifyRoute)
	}
	if cfg.PluginConfig.CronPort == 0 {
		cfg.PluginConfig.CronPort = 3713
		slog.Debug("default", "key", "PLUGIN_CRON_PORT", "value", fmt.Sprintf("%d", cfg.PluginConfig.CronPort))
	}
	if cfg.PluginConfig.CronRetention == 0 {
		cfg.PluginConfig.CronRetention = 90
		slog.Debug("default", "key", "PLUGIN_CRON_RETENTION", "value", fmt.Sprintf("%d", cfg.PluginConfig.CronRetention))
	}
}

// applyDefaultsPluginSystemLimits sets per-plugin memory and CPU resource limits.
func applyDefaultsPluginSystemLimits(cfg *Config) {
	if cfg.PluginConfig.AIMemLimit == "" {
		cfg.PluginConfig.AIMemLimit = "1g"
		slog.Debug("default", "key", "PLUGIN_AI_MEM_LIMIT", "value", cfg.PluginConfig.AIMemLimit)
	}
	if cfg.PluginConfig.AICPULimit == "" {
		cfg.PluginConfig.AICPULimit = "1.0"
		slog.Debug("default", "key", "PLUGIN_AI_CPU_LIMIT", "value", cfg.PluginConfig.AICPULimit)
	}
	if cfg.PluginConfig.MuxMemLimit == "" {
		cfg.PluginConfig.MuxMemLimit = "512m"
		slog.Debug("default", "key", "PLUGIN_MUX_MEM_LIMIT", "value", cfg.PluginConfig.MuxMemLimit)
	}
	if cfg.PluginConfig.MuxCPULimit == "" {
		cfg.PluginConfig.MuxCPULimit = "0.5"
		slog.Debug("default", "key", "PLUGIN_MUX_CPU_LIMIT", "value", cfg.PluginConfig.MuxCPULimit)
	}
	if cfg.PluginConfig.ClawMemLimit == "" {
		cfg.PluginConfig.ClawMemLimit = "512m"
		slog.Debug("default", "key", "PLUGIN_CLAW_MEM_LIMIT", "value", cfg.PluginConfig.ClawMemLimit)
	}
	if cfg.PluginConfig.ClawCPULimit == "" {
		cfg.PluginConfig.ClawCPULimit = "0.5"
		slog.Debug("default", "key", "PLUGIN_CLAW_CPU_LIMIT", "value", cfg.PluginConfig.ClawCPULimit)
	}
	if cfg.PluginConfig.DefaultMemLimit == "" {
		cfg.PluginConfig.DefaultMemLimit = "512m"
		slog.Debug("default", "key", "PLUGIN_DEFAULT_MEM_LIMIT", "value", cfg.PluginConfig.DefaultMemLimit)
	}
	if cfg.PluginConfig.DefaultCPULimit == "" {
		cfg.PluginConfig.DefaultCPULimit = "0.5"
		slog.Debug("default", "key", "PLUGIN_DEFAULT_CPU_LIMIT", "value", cfg.PluginConfig.DefaultCPULimit)
	}
}

// applyDefaultsPluginSecrets generates internal plugin communication secrets when missing.
func applyDefaultsPluginSecrets(cfg *Config) error {
	if cfg.PluginConfig.NotifySecret == "" {
		secret, err := generateSecureRandom(32)
		if err != nil {
			return fmt.Errorf("generating NOTIFY_INTERNAL_SECRET: %w", err)
		}
		cfg.PluginConfig.NotifySecret = secret
		slog.Debug("default", "key", "NOTIFY_INTERNAL_SECRET", "value", "[generated]")
	}
	if cfg.PluginConfig.CronSecret == "" {
		secret, err := generateSecureRandom(32)
		if err != nil {
			return fmt.Errorf("generating CRON_INTERNAL_SECRET: %w", err)
		}
		cfg.PluginConfig.CronSecret = secret
		slog.Debug("default", "key", "CRON_INTERNAL_SECRET", "value", "[generated]")
	}
	if cfg.PluginSystem.InternalSecret == "" {
		secret, err := generateSecureRandom(32)
		if err != nil {
			return fmt.Errorf("generating PLUGIN_INTERNAL_SECRET: %w", err)
		}
		cfg.PluginSystem.InternalSecret = secret
		slog.Debug("default", "key", "PLUGIN_INTERNAL_SECRET", "value", "[generated]")
	}
	return nil
}

// applyDefaultsBackup sets backup schedule, retention, and PITR defaults.
func applyDefaultsBackup(cfg *Config) {
	if cfg.Backup.Dir == "" {
		cfg.Backup.Dir = "./backups"
		slog.Debug("default", "key", "BACKUP_DIR", "value", cfg.Backup.Dir)
	}
	if cfg.Backup.RetentionDays == 0 {
		cfg.Backup.RetentionDays = 30
		slog.Debug("default", "key", "BACKUP_RETENTION_DAYS", "value", fmt.Sprintf("%d", cfg.Backup.RetentionDays))
	}
	if cfg.Backup.Schedule == "" {
		cfg.Backup.Schedule = "0 2 * * *"
		slog.Debug("default", "key", "BACKUP_SCHEDULE", "value", cfg.Backup.Schedule)
	}
	if cfg.Backup.ScheduleFull == "" {
		cfg.Backup.ScheduleFull = "0 3 * * *"
		slog.Debug("default", "key", "BACKUP_SCHEDULE_FULL", "value", cfg.Backup.ScheduleFull)
	}
	if cfg.Backup.WALInterval == 0 {
		cfg.Backup.WALInterval = 60
		slog.Debug("default", "key", "BACKUP_WAL_INTERVAL_SECONDS", "value", "60")
	}
	if cfg.Backup.RetentionDaily == 0 {
		cfg.Backup.RetentionDaily = 7
		slog.Debug("default", "key", "BACKUP_RETENTION_DAILY", "value", "7")
	}
	if cfg.Backup.RetentionWeekly == 0 {
		cfg.Backup.RetentionWeekly = 4
		slog.Debug("default", "key", "BACKUP_RETENTION_WEEKLY", "value", "4")
	}
	if cfg.Backup.RetentionMonthly == 0 {
		cfg.Backup.RetentionMonthly = 12
		slog.Debug("default", "key", "BACKUP_RETENTION_MONTHLY", "value", "12")
	}
	if cfg.Backup.RestoreTestSchedule == "" {
		cfg.Backup.RestoreTestSchedule = "0 5 * * 0"
		slog.Debug("default", "key", "BACKUP_RESTORE_TEST_SCHEDULE", "value", cfg.Backup.RestoreTestSchedule)
	}
}

// applyDefaultsEmail sets email provider and SMTP defaults.
func applyDefaultsEmail(cfg *Config) {
	if cfg.Email.Provider == "" {
		cfg.Email.Provider = "mailpit"
		slog.Debug("default", "key", "EMAIL_PROVIDER", "value", cfg.Email.Provider)
	}
	if cfg.Email.From == "" {
		cfg.Email.From = "noreply@" + cfg.BaseDomain
		slog.Debug("default", "key", "EMAIL_FROM", "value", cfg.Email.From)
	}
	if cfg.Email.AWSRegion == "" {
		cfg.Email.AWSRegion = "us-east-1"
		slog.Debug("default", "key", "EMAIL_AWS_REGION", "value", cfg.Email.AWSRegion)
	}
	if cfg.Email.SMTPPort == 0 {
		cfg.Email.SMTPPort = 587
		slog.Debug("default", "key", "SMTP_PORT", "value", "587")
	}
}
