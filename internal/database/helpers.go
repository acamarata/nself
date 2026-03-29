package database

import (
	"github.com/nself-org/cli/internal/config"
)

// containerName returns the postgres container name for the project.
func containerName(cfg *config.Config) string {
	return cfg.ProjectName + "_postgres"
}
