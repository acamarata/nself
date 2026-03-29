package compose

// buildNetworks returns the docker-compose network definitions.
// A single bridge network named {ProjectName}_network is created.
func (g *Generator) buildNetworks() map[string]NetworkConfig {
	name := g.cfg.ProjectName + "_network"
	return map[string]NetworkConfig{
		name: {
			Driver: "bridge",
		},
	}
}
