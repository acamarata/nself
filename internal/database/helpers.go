package database

import (
	"nself/internal/config"
)

// containerName returns the postgres container name for the project.
func containerName(cfg *config.Config) string {
	return cfg.ProjectName + "_postgres"
}
