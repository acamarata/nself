package config

import (
	"fmt"
	"strings"
)

// ValidatorFunc pairs a human-readable name with a validation function.
// The name is used in error messages and test assertions.
type ValidatorFunc struct {
	Name string
	Fn   func(*Config) error
}

// registry holds all validators added via Register. Order follows
// file-alphabetical init() ordering within the config package — callers
// must not rely on a specific validator running before another.
var registry []ValidatorFunc

// register appends a named validator to the registry. It is designed to be
// called from init() functions so that each file owning a validator can
// self-register without a central dispatch table.
func register(name string, fn func(*Config) error) {
	registry = append(registry, ValidatorFunc{Name: name, Fn: fn})
}

// runAll executes every registered validator in order and collects all
// errors. If one or more validators fail, the returned error lists every
// failure separated by newlines so the caller sees the complete picture
// rather than just the first problem. Returns nil when all pass.
func runAll(cfg *Config) error {
	var failures []string
	for _, v := range registry {
		if err := v.Fn(cfg); err != nil {
			failures = append(failures, fmt.Sprintf("[%s] %s", v.Name, err.Error()))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "\n"))
	}
	return nil
}

// ValidatorResult pairs a validator name with its outcome.
type ValidatorResult struct {
	Name string
	Err  error
}

// RunAllWithResults executes every registered validator and returns one
// ValidatorResult per validator, preserving the original name and error.
// Unlike RunAll, no error is returned — the caller inspects results directly.
func RunAllWithResults(cfg *Config) []ValidatorResult {
	results := make([]ValidatorResult, 0, len(registry))
	for _, v := range registry {
		results = append(results, ValidatorResult{
			Name: v.Name,
			Err:  v.Fn(cfg),
		})
	}
	return results
}
