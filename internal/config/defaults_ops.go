package config

// defaults_ops.go — default value application for Monitoring, Docker and start/stop behaviour.
//
// Purpose: Fill in unset Monitoring (toggles and ports), Docker and start/stop fields on a loaded Config with the CLI's standard defaults, split out of defaults.go for file size.
// Inputs: a *Config already populated by the loader.
// Outputs: the same *Config with these operational fields defaulted in place.
// Constraints: pure move from defaults.go (CLI-R12 Batch F); no behaviour change. Keep in sync with ApplyDefaults in defaults.go, which calls these in order.

import (
	"fmt"
	"log/slog"
)

// applyDefaultsMonitoring auto-enables monitoring sub-services and sets port defaults.
func applyDefaultsMonitoring(cfg *Config) {
	applyDefaultsMonitoringToggles(cfg)
	applyDefaultsMonitoringPorts(cfg)
}

// applyDefaultsMonitoringToggles enables all sub-services when the master toggle is on.
func applyDefaultsMonitoringToggles(cfg *Config) {
	if !cfg.Monitoring.Enabled {
		return
	}
	if !cfg.Monitoring.PrometheusEnabled {
		cfg.Monitoring.PrometheusEnabled = true
	}
	if !cfg.Monitoring.GrafanaEnabled {
		cfg.Monitoring.GrafanaEnabled = true
	}
	if !cfg.Monitoring.LokiEnabled {
		cfg.Monitoring.LokiEnabled = true
	}
	if !cfg.Monitoring.PromtailEnabled {
		cfg.Monitoring.PromtailEnabled = true
	}
	if !cfg.Monitoring.TempoEnabled {
		cfg.Monitoring.TempoEnabled = true
	}
	if !cfg.Monitoring.AlertmanagerEnabled {
		cfg.Monitoring.AlertmanagerEnabled = true
	}
	if !cfg.Monitoring.CadvisorEnabled {
		cfg.Monitoring.CadvisorEnabled = true
	}
	if !cfg.Monitoring.NodeExporterEnabled {
		cfg.Monitoring.NodeExporterEnabled = true
	}
	if !cfg.Monitoring.PGExporterEnabled {
		cfg.Monitoring.PGExporterEnabled = true
	}
	if !cfg.Monitoring.RedisExporterEnabled {
		cfg.Monitoring.RedisExporterEnabled = true
	}
}

// applyDefaultsMonitoringPorts fills monitoring service port defaults (always, regardless of enabled state).
func applyDefaultsMonitoringPorts(cfg *Config) {
	// Ports (always fill regardless of enabled state)
	if cfg.Monitoring.PrometheusPort == 0 {
		cfg.Monitoring.PrometheusPort = 9090
		slog.Debug("default", "key", "PROMETHEUS_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.PrometheusPort))
	}
	if cfg.Monitoring.GrafanaPort == 0 {
		cfg.Monitoring.GrafanaPort = 3001
		slog.Debug("default", "key", "GRAFANA_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.GrafanaPort))
	}
	if cfg.Monitoring.GrafanaAdminUser == "" {
		cfg.Monitoring.GrafanaAdminUser = "admin"
		slog.Debug("default", "key", "GRAFANA_ADMIN_USER", "value", cfg.Monitoring.GrafanaAdminUser)
	}
	if cfg.Monitoring.GrafanaRoute == "" {
		cfg.Monitoring.GrafanaRoute = "grafana"
		slog.Debug("default", "key", "GRAFANA_ROUTE", "value", cfg.Monitoring.GrafanaRoute)
	}
	if cfg.Monitoring.LokiPort == 0 {
		cfg.Monitoring.LokiPort = 3100
		slog.Debug("default", "key", "LOKI_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.LokiPort))
	}
	if cfg.Monitoring.TempoPort == 0 {
		cfg.Monitoring.TempoPort = 3200
		slog.Debug("default", "key", "TEMPO_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.TempoPort))
	}
	if cfg.Monitoring.AlertmanagerPort == 0 {
		cfg.Monitoring.AlertmanagerPort = 9093
		slog.Debug("default", "key", "ALERTMANAGER_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.AlertmanagerPort))
	}
	if cfg.Monitoring.CadvisorPort == 0 {
		cfg.Monitoring.CadvisorPort = 8082
		slog.Debug("default", "key", "CADVISOR_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.CadvisorPort))
	}
	if cfg.Monitoring.NodeExporterPort == 0 {
		cfg.Monitoring.NodeExporterPort = 9100
		slog.Debug("default", "key", "NODE_EXPORTER_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.NodeExporterPort))
	}
	if cfg.Monitoring.PGExporterPort == 0 {
		cfg.Monitoring.PGExporterPort = 9187
		slog.Debug("default", "key", "PG_EXPORTER_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.PGExporterPort))
	}
	if cfg.Monitoring.RedisExporterPort == 0 {
		cfg.Monitoring.RedisExporterPort = 9121
		slog.Debug("default", "key", "REDIS_EXPORTER_PORT", "value", fmt.Sprintf("%d", cfg.Monitoring.RedisExporterPort))
	}
}

// applyDefaultsDocker sets Docker network name and logging defaults.
func applyDefaultsDocker(cfg *Config) {
	// DockerNetwork is always computed from ProjectName.
	cfg.DockerNetwork = cfg.ProjectName + "_network"

	if cfg.DockerLogMaxSize == "" {
		cfg.DockerLogMaxSize = "10m"
		slog.Debug("default", "key", "DOCKER_LOG_MAX_SIZE", "value", cfg.DockerLogMaxSize)
	}
	if cfg.DockerLogMaxFile == "" {
		cfg.DockerLogMaxFile = "3"
		slog.Debug("default", "key", "DOCKER_LOG_MAX_FILE", "value", cfg.DockerLogMaxFile)
	}
	if cfg.DockerStopGrace == "" {
		cfg.DockerStopGrace = "30s"
		slog.Debug("default", "key", "DOCKER_STOP_GRACE", "value", cfg.DockerStopGrace)
	}
	if cfg.DockerBuildTimeout == 0 {
		cfg.DockerBuildTimeout = 300
		slog.Debug("default", "key", "DOCKER_BUILD_TIMEOUT", "value", fmt.Sprintf("%d", cfg.DockerBuildTimeout))
	}
}

// applyDefaultsStartStop sets lifecycle control defaults (start mode, health checks, logging).
func applyDefaultsStartStop(cfg *Config) {
	if cfg.StartMode == "" {
		cfg.StartMode = "smart"
		slog.Debug("default", "key", "START_MODE", "value", cfg.StartMode)
	}
	if cfg.HealthCheckTimeout == 0 {
		cfg.HealthCheckTimeout = 120
		slog.Debug("default", "key", "HEALTH_CHECK_TIMEOUT", "value", fmt.Sprintf("%d", cfg.HealthCheckTimeout))
	}
	if cfg.HealthCheckInterval == 0 {
		cfg.HealthCheckInterval = 2
		slog.Debug("default", "key", "HEALTH_CHECK_INTERVAL", "value", fmt.Sprintf("%d", cfg.HealthCheckInterval))
	}
	if cfg.HealthCheckRequired == 0 {
		cfg.HealthCheckRequired = 80
		slog.Debug("default", "key", "HEALTH_CHECK_REQUIRED", "value", fmt.Sprintf("%d", cfg.HealthCheckRequired))
	}
	if cfg.CleanupOnStart == "" {
		cfg.CleanupOnStart = "auto"
		slog.Debug("default", "key", "CLEANUP_ON_START", "value", cfg.CleanupOnStart)
	}
	if cfg.ParallelLimit == 0 {
		cfg.ParallelLimit = 5
		slog.Debug("default", "key", "PARALLEL_LIMIT", "value", fmt.Sprintf("%d", cfg.ParallelLimit))
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
		slog.Debug("default", "key", "NSELF_LOG_LEVEL", "value", cfg.LogLevel)
	}
	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = 30
		slog.Debug("default", "key", "STOP_TIMEOUT", "value", fmt.Sprintf("%d", cfg.StopTimeout))
	}
}
