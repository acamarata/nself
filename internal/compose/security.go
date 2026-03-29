package compose

// ServiceSecurity holds Docker security configuration for a service.
type ServiceSecurity struct {
	CapDrop     []string
	CapAdd      []string
	SecurityOpt []string
}

// DefaultSecurity returns the security config applied to all services.
func DefaultSecurity() ServiceSecurity {
	return ServiceSecurity{
		CapDrop:     []string{"ALL"},
		SecurityOpt: []string{"no-new-privileges:true"},
	}
}

// PostgresSecurity returns the security config for the PostgreSQL service.
// PostgreSQL needs IPC_LOCK for shared memory management.
func PostgresSecurity() ServiceSecurity {
	return ServiceSecurity{
		CapDrop:     []string{"ALL"},
		CapAdd:      []string{"IPC_LOCK"},
		SecurityOpt: []string{"no-new-privileges:true"},
	}
}

// applySecurityToService merges a ServiceSecurity into a ServiceConfig.
func applySecurityToService(svc *ServiceConfig, sec ServiceSecurity) {
	svc.CapDrop = sec.CapDrop
	if len(sec.CapAdd) > 0 {
		svc.CapAdd = sec.CapAdd
	}
	svc.SecurityOpt = sec.SecurityOpt
}
