package commands

// Purpose: small helpers for filtering and counting servers in a control-plane
// inventory. Inputs are a *controlplane.Inventory and a server name filter;
// outputs are a filtered inventory or a server count.
// Constraints: split out of deploy.go (CLI-R12) as a pure move, no behavior change.

import (
	"github.com/nself-org/cli/internal/controlplane"
)

// filterInventoryByServer returns a shallow copy of inv retaining only the
// server whose Name matches serverName across all environments.
func filterInventoryByServer(inv *controlplane.Inventory, serverName string) *controlplane.Inventory {
	filtered := &controlplane.Inventory{
		SchemaVersion: inv.SchemaVersion,
		Project:       inv.Project,
		Environments:  make(map[string]controlplane.Environment, len(inv.Environments)),
	}
	for envName, env := range inv.Environments {
		var kept []controlplane.Server
		for _, srv := range env.Servers {
			if srv.Name == serverName {
				kept = append(kept, srv)
			}
		}
		if len(kept) > 0 {
			filtered.Environments[envName] = controlplane.Environment{
				Name:    env.Name,
				Kind:    env.Kind,
				Servers: kept,
			}
		}
	}
	return filtered
}

// totalServers returns the total number of servers across all environments.
func totalServers(inv *controlplane.Inventory) int {
	n := 0
	for _, env := range inv.Environments {
		n += len(env.Servers)
	}
	return n
}
