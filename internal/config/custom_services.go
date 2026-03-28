package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ParseCustomServices parses CS_1 through CS_10 environment variables into
// CustomService structs. Each CS_N value uses the format:
//
//	name:template[:port][:route]
//
// If port is omitted or zero, it auto-assigns 8000+N.
// Per-service overrides are read from CS_N_PUBLIC, CS_N_MEMORY, CS_N_CPU,
// CS_N_PORT, and CS_N_ROUTE environment variables.
func parseCustomServices() ([]CustomService, error) {
	var services []CustomService
	for i := 1; i <= 10; i++ {
		raw := os.Getenv(fmt.Sprintf("CS_%d", i))
		if raw == "" {
			continue
		}

		// Format: name:template[:port][:route]
		parts := strings.SplitN(raw, ":", 4)
		if len(parts) < 2 {
			return nil, fmt.Errorf("CS_%d has invalid format: expected name:template[:port][:route], got %q", i, raw)
		}

		cs := CustomService{
			Index:    i,
			Name:     parts[0],
			Template: parts[1],
		}

		if err := validateCustomServiceName(cs.Name); err != nil {
			return nil, fmt.Errorf("CS_%d: %w", i, err)
		}
		sanitized, err := SanitizeName(cs.Name)
		if err != nil {
			return nil, fmt.Errorf("CS_%d has invalid name %q: %w", i, cs.Name, err)
		}
		cs.Name = sanitized

		// Optional port
		if len(parts) >= 3 && parts[2] != "" {
			p, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, fmt.Errorf("CS_%d has invalid format: expected numeric port, got %q", i, parts[2])
			}
			cs.Port = p
		}
		if cs.Port == 0 {
			cs.Port = 8000 + i // auto-assign
		}

		// Optional route
		if len(parts) >= 4 && parts[3] != "" {
			cs.Route = parts[3]
		}

		// Additional per-service config
		cs.Public = getEnvBool(fmt.Sprintf("CS_%d_PUBLIC", i), cs.Route != "")
		cs.Memory = getEnvOr(fmt.Sprintf("CS_%d_MEMORY", i), "256m")
		cs.CPU = getEnvOr(fmt.Sprintf("CS_%d_CPU", i), "0.5")
		cs.TablePrefix = os.Getenv(fmt.Sprintf("CS_%d_TABLE_PREFIX", i))
		cs.ExtraEnv = os.Getenv(fmt.Sprintf("CS_%d_ENV", i))

		// Override port/route if explicitly set
		if p := getEnvInt(fmt.Sprintf("CS_%d_PORT", i), 0); p != 0 {
			cs.Port = p
		}
		if r := os.Getenv(fmt.Sprintf("CS_%d_ROUTE", i)); r != "" {
			cs.Route = r
		}

		services = append(services, cs)
	}
	return services, nil
}
