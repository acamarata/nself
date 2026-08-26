package compose

import "testing"

// TestNginxMasterRunsAsRoot pins the fix for the TLS crash-loop.
//
// The nginx.conf this same tool generates declares `user nginx;` on line 4,
// which only takes effect when the MASTER process starts as root: it reads the
// TLS material and binds 80/443, then drops the workers to the nginx user.
//
// Setting User: "101:101" on the compose service contradicted that config, and
// nginx said so on every boot ("the \"user\" directive makes sense only if the
// master process runs with super-user privileges, ignored"). It also broke TLS
// outright. ./ssl is bind-mounted, so certificates keep their host ownership
// and mode; uid 101 is neither their owner nor in their group, so nginx died
// with
//
//	[emerg] cannot load certificate ".../fullchain.pem": Permission denied
//
// and crash-looped on every host whose invoking uid is not 101, which on Linux
// is all of them. Docker Desktop on macOS remaps ownership and hid it, so this
// only ever surfaced on Linux and in CI, where it timed out the golden path's
// service-health wait for weeks.
//
// If a User override is reintroduced here, this fails instead of shipping a
// stack whose reverse proxy never starts.
func TestNginxMasterRunsAsRoot(t *testing.T) {
	cfg := minimalCfg()
	g := NewGenerator(cfg)
	svc := g.buildNginxService(&DockerCompose{Services: map[string]ServiceConfig{}})

	if svc.User != "" {
		t.Errorf("nginx service sets User=%q, but the generated nginx.conf "+
			"declares `user nginx;` and needs a root master to read the "+
			"bind-mounted TLS material and bind 80/443. A non-root master "+
			"cannot read fullchain.pem and nginx crash-loops.", svc.User)
	}
}

// TestNginxWorkersStayUnprivileged guards the other half: removing the User
// override must not also remove the tmpfs uid/gid that match the unprivileged
// workers the config drops to.
func TestNginxWorkersStayUnprivileged(t *testing.T) {
	cfg := minimalCfg()
	g := NewGenerator(cfg)
	svc := g.buildNginxService(&DockerCompose{Services: map[string]ServiceConfig{}})

	if len(svc.Tmpfs) == 0 {
		t.Fatal("nginx service declares no tmpfs; workers need writable cache and run dirs")
	}
	for _, m := range svc.Tmpfs {
		if !contains(m, "uid=101") || !contains(m, "gid=101") {
			t.Errorf("tmpfs %q should be owned by the unprivileged nginx worker (uid=101,gid=101)", m)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
