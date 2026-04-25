// Package db provides pgx pool helpers used by every plugin that talks to
// Postgres. Centralizing here avoids the 10+ copies of the same DSN parsing /
// pool-config code across plugins.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds the tunable pool parameters. Zero values get nSelf defaults.
type PoolConfig struct {
	DSN             string        // postgres://... required
	MaxConns        int32         // default 10
	MinConns        int32         // default 2
	MaxConnLifetime time.Duration // default 30m
	MaxConnIdleTime time.Duration // default 5m
	ConnectTimeout  time.Duration // default 10s
}

// Open parses the DSN, applies defaults, and returns a ready pool.
// Callers must Close() the pool on shutdown.
func Open(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("db: PoolConfig.DSN is required")
	}

	pc, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: parse DSN: %w", err)
	}

	if cfg.MaxConns > 0 {
		pc.MaxConns = cfg.MaxConns
	} else {
		pc.MaxConns = 10
	}
	if cfg.MinConns > 0 {
		pc.MinConns = cfg.MinConns
	} else {
		pc.MinConns = 2
	}
	if cfg.MaxConnLifetime > 0 {
		pc.MaxConnLifetime = cfg.MaxConnLifetime
	} else {
		pc.MaxConnLifetime = 30 * time.Minute
	}
	if cfg.MaxConnIdleTime > 0 {
		pc.MaxConnIdleTime = cfg.MaxConnIdleTime
	} else {
		pc.MaxConnIdleTime = 5 * time.Minute
	}

	connectCtx := ctx
	if cfg.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, cfg.ConnectTimeout)
		defer cancel()
	}

	pool, err := pgxpool.NewWithConfig(connectCtx, pc)
	if err != nil {
		return nil, fmt.Errorf("db: new pool: %w", err)
	}
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

// HealthCheck runs a fast SELECT 1 against the pool. Used by /readyz endpoints.
func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("db: pool is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var v int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&v); err != nil {
		return fmt.Errorf("db: health query: %w", err)
	}
	if v != 1 {
		return fmt.Errorf("db: health query returned %d, expected 1", v)
	}
	return nil
}
