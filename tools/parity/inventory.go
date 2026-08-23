// Purpose:     load the committed command inventory (tools/cmdinventory's
//
//	output) as the parity matrix's list of top-level commands.
//
// Inputs:      path to .github/command-inventory.json.
// Outputs:     one inventoryEntry per top-level command, sorted as committed.
// Constraints: the JSON shape must mirror tools/cmdinventory.Command; only the
//
//	top-level fields are read here, subcommands are intentionally
//	ignored — CLI-R17 scores top-level commands only.
package main

import (
	"encoding/json"
	"os"
)

// inventoryEntry mirrors the top-level shape tools/cmdinventory emits.
type inventoryEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Short   string `json:"short"`
	Hidden  bool   `json:"hidden"`
	GroupID string `json:"group_id,omitempty"`
}

// loadInventory reads and decodes the committed command inventory.
func loadInventory(path string) ([]inventoryEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []inventoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	out := entries[:0]
	for _, e := range entries {
		if e.Hidden {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}
