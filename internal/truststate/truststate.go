package truststate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// TrustState records the last BASE_DOMAIN for which trust was configured.
type TrustState struct {
	TrustedDomain string `json:"trusted_domain"`
}

// statePath returns ~/.nself/trust-state.json
func statePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nself", "trust-state.json"), nil
}

// Load reads the trust state from disk. Returns zero-value TrustState if not found.
func Load() (TrustState, error) {
	path, err := statePath()
	if err != nil {
		return TrustState{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TrustState{}, nil
		}
		return TrustState{}, err
	}
	var state TrustState
	if err := json.Unmarshal(data, &state); err != nil {
		return TrustState{}, err
	}
	return state, nil
}

// Save writes the trust state to disk. Creates ~/.nself/ if needed.
func Save(state TrustState) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
