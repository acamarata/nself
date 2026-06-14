package scaffold

// scaffold_templates_code.go — Go source code template strings.
//
// Purpose: Hold all Go const template strings that emit plugin source code:
//          cmd/main.go, internal/config/config.go, internal/server/server.go,
//          and internal/server/server_test.go. Separated from
//          scaffold_templates_infra.go (infra/devops templates) and
//          scaffold.go (logic) so each file has a single clear concern.
// Inputs:  none (package-level consts, referenced by addGoFiles in scaffold.go).
// Outputs: Exported const strings consumed by render at run time.
// Constraints: All consts here are Go text/template strings that render valid
//              Go source code. Use backtick literals only. No logic allowed.
// SPORT:   cli/internal/plugin/scaffold — decomposed from scaffold.go (T-E2-06).

const tmplMain = `// Package main is the entrypoint for the {{.Name}} plugin.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nself-org/{{.RepoBucket}}/{{.Name}}/internal/config"
	"github.com/nself-org/{{.RepoBucket}}/{{.Name}}/internal/server"

	"github.com/nself-org/plugin-sdk-go/logger"
)

// Version is stamped at build time via -ldflags.
var Version = "0.1.0"

func main() {
	cfg := config.FromEnv()
	log := logger.New(logger.Options{
		Plugin:  "{{.Name}}",
		Version: Version,
		Level:   logger.ParseLevel(cfg.LogLevel),
	})

	srv := server.New(server.Deps{Config: cfg, Logger: log, Version: Version})

	httpSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("server goroutine panic: %v", r)
			}
		}()
		log.Info("{{.Name}} listening", "addr", cfg.ListenAddr)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		return
	}
	log.Info("{{.Name}} stopped cleanly")
}
`

const tmplConfig = `// Package config loads {{.Name}} config from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds runtime config.
type Config struct {
	ListenAddr  string
	LogLevel    string
	DatabaseURL string
}

// FromEnv reads config from env vars with sensible defaults.
func FromEnv() Config {
	return Config{
		ListenAddr:  envOr("{{.EnvPrefix}}_LISTEN_ADDR", ":{{.Port}}"),
		LogLevel:    envOr("LOG_LEVEL", "info"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}

// Validate returns an error if required fields are missing.
func (c Config) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("{{.Name}}: listen address must not be empty")
	}
	return nil
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
`

const tmplServer = `// Package server wires the HTTP router for the {{.Name}} plugin.
package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nself-org/{{.RepoBucket}}/{{.Name}}/internal/config"

	sdkmetrics "github.com/nself-org/plugin-sdk-go/metrics"
	sdkserver "github.com/nself-org/plugin-sdk-go/server"
)

// Deps wires runtime dependencies.
type Deps struct {
	Config  config.Config
	Logger  *slog.Logger
	Version string
}

type readyFn func(ctx context.Context) error

func (f readyFn) Ready(ctx context.Context) error { return f(ctx) }

// New returns a ready-to-serve http.Handler.
func New(d Deps) http.Handler {
	return sdkserver.New(sdkserver.Options{
		Plugin:  "{{.Name}}",
		Version: d.Version,
		Ready: readyFn(func(ctx context.Context) error {
			return d.Config.Validate()
		}),
		Routes: func(r chi.Router, m *sdkmetrics.Registry) {
			r.Route("/v1", func(r chi.Router) {
				r.With(m.Middleware("/v1/hello")).Get("/hello", helloHandler(d.Logger))
			})
		},
	})
}

func helloHandler(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if log != nil {
			log.Info("hello called", "method", r.Method, "path", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(` + "`" + `{"plugin":"{{.Name}}","hello":"world"}` + "`" + `))
	}
}
`

const tmplServerTest = `package server

import (
	"net/http/httptest"
	"testing"

	"github.com/nself-org/{{.RepoBucket}}/{{.Name}}/internal/config"
)

func TestHelloEndpoint(t *testing.T) {
	h := New(Deps{Config: config.Config{ListenAddr: ":{{.Port}}"}, Version: "test"})
	req := httptest.NewRequest("GET", "/v1/hello", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestHealthz(t *testing.T) {
	h := New(Deps{Config: config.Config{ListenAddr: ":{{.Port}}"}, Version: "test"})
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200 from /healthz, got %d", rr.Code)
	}
}
`
