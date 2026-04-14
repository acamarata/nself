// profiling.go provides pprof endpoint registration with token-based auth
// and optional Pyroscope continuous profiling agent integration.
package observability

import (
	"log/slog"
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
		return tokenAuth(cfg.Token, h)
	}

	mux.Handle("/debug/pprof/", wrap(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", wrap(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", wrap(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", wrap(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", wrap(pprof.Trace))

	slog.Info("serving pprof profiles", "addr", cfg.BindAddr)
	if err := http.ListenAndServe(cfg.BindAddr, mux); err != nil {
		slog.Error("pprof server failed", "error", err)
	}
}

// PprofHandler returns an http.Handler that serves pprof endpoints
// with token authentication. Useful for registering on an existing mux.
func PprofHandler(token string) http.Handler {
	mux := http.NewServeMux()
	wrap := func(h http.HandlerFunc) http.Handler {
		return tokenAuth(token, h)
	}
	mux.Handle("/debug/pprof/", wrap(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", wrap(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", wrap(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", wrap(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", wrap(pprof.Trace))
	return mux
}

// tokenAuth middleware checks X-Profile-Token header. If token is empty,
// auth is disabled (dev mode).
func tokenAuth(token string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("X-Profile-Token") != token {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}
