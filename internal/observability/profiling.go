// profiling.go provides pprof endpoint registration with token-based auth
// and optional Pyroscope continuous profiling agent integration.
package observability

import (
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
)

// ProfilingConfig configures pprof and Pyroscope integration.
type ProfilingConfig struct {
	// BindAddr for pprof HTTP server (default 127.0.0.1:6060).
	BindAddr string
	// Token required in X-Profile-Token header. Empty disables auth.
	Token string
	// PyroscopeEnabled enables continuous profiling push to Pyroscope.
	PyroscopeEnabled bool
	// PyroscopeServerURL is the Pyroscope server address.
	PyroscopeServerURL string
	// ApplicationName identifies this service in Pyroscope.
	ApplicationName string
}

// DefaultProfilingConfig reads config from environment variables.
func DefaultProfilingConfig() ProfilingConfig {
	addr := os.Getenv("NSELF_PPROF_BIND")
	if addr == "" {
		addr = "127.0.0.1:6060"
	}
	appName := os.Getenv("PYROSCOPE_APPLICATION_NAME")
	if appName == "" {
		appName = os.Getenv("OTEL_SERVICE_NAME")
	}
	return ProfilingConfig{
		BindAddr:           addr,
		Token:              os.Getenv("NSELF_PROFILING_TOKEN"),
		PyroscopeEnabled:   os.Getenv("PYROSCOPE_ENABLED") == "true",
		PyroscopeServerURL: os.Getenv("PYROSCOPE_SERVER_URL"),
		ApplicationName:    appName,
	}
}

// ServeProfiling starts a pprof HTTP server on the configured bind address.
// It blocks, so call in a goroutine. The server is bound to 127.0.0.1 only
// and is never exposed publicly via nginx.
func ServeProfiling(cfg ProfilingConfig) {
	if cfg.BindAddr == "" {
		cfg.BindAddr = "127.0.0.1:6060"
	}

	mux := http.NewServeMux()

	// Wrap all pprof handlers with token auth.
	wrap := func(h http.HandlerFunc) http.Handler {
		return tokenAuth(cfg.Token, cfg.BindAddr, h)
	}

	mux.Handle("/debug/pprof/", wrap(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", wrap(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", wrap(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", wrap(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", wrap(pprof.Trace))

	slog.Info("serving pprof profiles", "addr", cfg.BindAddr)
	if err := http.ListenAndServe(cfg.BindAddr, Recoverer(mux)); err != nil {
		slog.Error("pprof server failed", "error", err)
	}
}

// PprofHandler returns an http.Handler that serves pprof endpoints with
// token authentication. Useful for registering on an existing mux.
//
// bindAddr is the address (host:port) the caller intends to serve this
// handler on — e.g. the same value passed to http.ListenAndServe. It is
// used only to evaluate the NSELF_PPROF_DEV escape hatch (see tokenAuth);
// pass "" if unknown, which is treated as non-loopback (escape hatch
// never applies).
func PprofHandler(token, bindAddr string) http.Handler {
	mux := http.NewServeMux()
	wrap := func(h http.HandlerFunc) http.Handler {
		return tokenAuth(token, bindAddr, h)
	}
	mux.Handle("/debug/pprof/", wrap(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", wrap(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", wrap(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", wrap(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", wrap(pprof.Trace))
	return mux
}

// isLoopbackBind reports whether addr (a host:port pair, as passed to
// http.ListenAndServe) resolves to a loopback-only bind. An empty host
// (e.g. ":6060", meaning "all interfaces") and any host that is not a
// literal loopback IP or "localhost" are treated as non-loopback.
func isLoopbackBind(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil || host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// tokenAuth middleware checks the X-Profile-Token header.
//
// Contract (P4 deferred-backlog row 20, revised per manager decision after
// cli#354 CI failure): empty token = pprof CLOSED (403) by default — the
// original "empty token = open" behavior was a default-deny gap. A narrow
// dev escape hatch exists: empty token is allowed ONLY when both
// NSELF_PPROF_DEV=1 is set AND bindAddr is loopback-only (127.0.0.1, ::1,
// or "localhost") — never on a non-loopback bind, even with the env var
// set. Every request served through the escape hatch logs a one-line
// warning so it is never silently active.
func tokenAuth(token, bindAddr string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			if os.Getenv("NSELF_PPROF_DEV") == "1" && isLoopbackBind(bindAddr) {
				slog.Warn("pprof serving without a token — NSELF_PPROF_DEV=1 escape hatch active on loopback bind", "addr", bindAddr)
				next(w, r)
				return
			}
			http.Error(w, "Forbidden: profiling disabled (set NSELF_PROFILING_TOKEN to enable, or NSELF_PPROF_DEV=1 on a loopback bind for local dev)", http.StatusForbidden)
			return
		}
		if r.Header.Get("X-Profile-Token") != token {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

// Recoverer is an HTTP middleware that catches panics from handlers, logs the
// panic value, and returns a 500 Internal Server Error instead of letting the
// panic propagate and kill the process. Apply it as the outermost middleware
// on every http.ServeMux to provide a process-wide safety net.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("http handler panic", "panic", rec, "path", r.URL.Path, "method", r.Method)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
