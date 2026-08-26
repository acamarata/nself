package compose

// ServiceSecurity holds Docker security configuration for a service.
type ServiceSecurity struct {
	CapDrop     []string
	CapAdd      []string
	SecurityOpt []string
	ReadOnly    bool
	Tmpfs       []string
	User        string
}

// DefaultSecurity returns the security config applied to most services.
// read_only root FS with /tmp and /run as tmpfs.
func DefaultSecurity() ServiceSecurity {
	return ServiceSecurity{
		CapDrop:     []string{"ALL"},
		SecurityOpt: []string{"no-new-privileges:true"},
		ReadOnly:    true,
		Tmpfs:       []string{"/tmp", "/run"},
	}
}

// PostgresSecurity returns the security config for the PostgreSQL service.
// PostgreSQL needs IPC_LOCK for shared memory and CHOWN/SETUID/SETGID for
// initdb. Root FS is read-only; data dir is a writable volume.
func PostgresSecurity() ServiceSecurity {
	return ServiceSecurity{
		CapDrop:     []string{"ALL"},
		CapAdd:      []string{"CHOWN", "SETUID", "SETGID", "IPC_LOCK"},
		SecurityOpt: []string{"no-new-privileges:true"},
		ReadOnly:    true,
		Tmpfs:       []string{"/tmp", "/run/postgresql"},
	}
}

// MinioSecurity returns the security config for the MinIO service.
// MinIO needs CHOWN/SETUID/SETGID for data directory ownership.
func MinioSecurity() ServiceSecurity {
	return ServiceSecurity{
		CapDrop:     []string{"ALL"},
		CapAdd:      []string{"CHOWN", "SETUID", "SETGID"},
		SecurityOpt: []string{"no-new-privileges:true"},
		ReadOnly:    true,
		Tmpfs:       []string{"/tmp", "/run"},
	}
}

// NginxSecurity returns the security config for the Nginx service.
// Nginx needs NET_BIND_SERVICE for ports 80/443 when running non-root.
//
// DAC_READ_SEARCH is required because CapDrop: ALL takes away root's usual
// ability to ignore file permissions, and the TLS material is bind-mounted
// from the host with host ownership. privkey.pem is written 0600 by
// mkcert/openssl and owned by whoever ran `nself build`, so without this the
// master cannot open it and nginx exits with
//
//	[emerg] cannot load certificate ".../fullchain.pem": Permission denied
//
// then crash-loops. The alternative was relaxing the private key to be
// world-readable on the host, which is a worse trade: this grants read and
// traverse bypass to one container that must read TLS material at startup,
// whereas DAC_OVERRIDE would also grant write bypass, and a 0644 key would
// expose it to every user on the machine.
func NginxSecurity() ServiceSecurity {
	return ServiceSecurity{
		CapDrop:     []string{"ALL"},
		CapAdd:      []string{"NET_BIND_SERVICE", "DAC_READ_SEARCH"},
		SecurityOpt: []string{"no-new-privileges:true"},
		ReadOnly:    true,
		// nginx tmpfs handled separately in service builder (uid/gid specific)
	}
}

// RedisSecurity returns the security config for the Redis service.
func RedisSecurity() ServiceSecurity {
	return ServiceSecurity{
		CapDrop:     []string{"ALL"},
		SecurityOpt: []string{"no-new-privileges:true"},
		ReadOnly:    true,
		Tmpfs:       []string{"/tmp"},
	}
}

// applySecurityToService merges a ServiceSecurity into a ServiceConfig.
func applySecurityToService(svc *ServiceConfig, sec ServiceSecurity) {
	svc.CapDrop = sec.CapDrop
	if len(sec.CapAdd) > 0 {
		svc.CapAdd = sec.CapAdd
	}
	svc.SecurityOpt = sec.SecurityOpt
	if sec.ReadOnly {
		svc.ReadOnly = true
	}
	// Merge tmpfs: keep any existing tmpfs entries and add security ones.
	if len(sec.Tmpfs) > 0 {
		existing := make(map[string]bool)
		for _, t := range svc.Tmpfs {
			existing[t] = true
		}
		for _, t := range sec.Tmpfs {
			if !existing[t] {
				svc.Tmpfs = append(svc.Tmpfs, t)
			}
		}
	}
}
