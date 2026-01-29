-- Migration 013: Performance Optimization System
-- Query optimization, caching recommendations, and performance monitoring

BEGIN;

-- ============================================================================
-- SCHEMA: performance
-- Performance tracking and optimization
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS performance;

-- Query performance tracking
CREATE TABLE performance.query_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query_hash TEXT NOT NULL,
    query_text TEXT NOT NULL,

    -- Execution stats
    total_calls INTEGER DEFAULT 0,
    total_time DOUBLE PRECISION DEFAULT 0,
    min_time DOUBLE PRECISION,
    max_time DOUBLE PRECISION,
    mean_time DOUBLE PRECISION,
    stddev_time DOUBLE PRECISION,

    -- Resource usage
    rows_returned BIGINT DEFAULT 0,
    blocks_hit BIGINT DEFAULT 0,
    blocks_read BIGINT DEFAULT 0,

    -- Timestamps
    first_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (query_hash)
);

CREATE INDEX idx_query_stats_hash ON performance.query_stats(query_hash);
CREATE INDEX idx_query_stats_mean_time ON performance.query_stats(mean_time DESC);
CREATE INDEX idx_query_stats_total_time ON performance.query_stats(total_time DESC);
CREATE INDEX idx_query_stats_calls ON performance.query_stats(total_calls DESC);

-- Index recommendations
CREATE TABLE performance.index_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name TEXT NOT NULL,
    column_names TEXT[] NOT NULL,
    index_type TEXT NOT NULL DEFAULT 'btree',

    -- Recommendation details
    reason TEXT NOT NULL,
    estimated_benefit TEXT,
    priority INTEGER DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),

    -- Status
    status TEXT NOT NULL DEFAULT 'recommended' CHECK (status IN ('recommended', 'applied', 'rejected', 'obsolete')),

    -- DDL
    create_statement TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ
);

CREATE INDEX idx_index_recs_table ON performance.index_recommendations(table_name);
CREATE INDEX idx_index_recs_status ON performance.index_recommendations(status);
CREATE INDEX idx_index_recs_priority ON performance.index_recommendations(priority DESC);

-- Cache statistics
CREATE TABLE performance.cache_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cache_key TEXT NOT NULL,
    cache_type TEXT NOT NULL, -- 'query', 'api', 'static'

    -- Hit/miss stats
    hits BIGINT DEFAULT 0,
    misses BIGINT DEFAULT 0,
    evictions BIGINT DEFAULT 0,

    -- Size stats
    avg_size_bytes INTEGER,
    total_size_bytes BIGINT DEFAULT 0,

    -- TTL stats
    avg_ttl_seconds INTEGER,

    -- Timestamps
    first_access TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_access TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (cache_key, cache_type)
);

CREATE INDEX idx_cache_stats_type ON performance.cache_stats(cache_type);
CREATE INDEX idx_cache_stats_last_access ON performance.cache_stats(last_access);

-- Connection pool stats
CREATE TABLE performance.connection_pool_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_name TEXT NOT NULL,

    -- Pool config
    max_connections INTEGER NOT NULL,
    min_connections INTEGER NOT NULL,

    -- Current state
    active_connections INTEGER DEFAULT 0,
    idle_connections INTEGER DEFAULT 0,
    waiting_count INTEGER DEFAULT 0,

    -- Metrics
    total_acquired BIGINT DEFAULT 0,
    total_released BIGINT DEFAULT 0,
    total_timeouts BIGINT DEFAULT 0,
    avg_wait_time_ms DOUBLE PRECISION,

    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conn_pool_name ON performance.connection_pool_stats(pool_name);
CREATE INDEX idx_conn_pool_timestamp ON performance.connection_pool_stats(timestamp);

-- ============================================================================
-- FUNCTIONS: Performance Analysis
-- ============================================================================

-- Function: Capture slow query
CREATE OR REPLACE FUNCTION performance.log_slow_query(
    p_query_text TEXT,
    p_execution_time DOUBLE PRECISION,
    p_rows_returned BIGINT DEFAULT 0
)
RETURNS VOID AS $$
DECLARE
    v_query_hash TEXT;
BEGIN
    -- Generate hash of query (normalized)
    v_query_hash := md5(regexp_replace(p_query_text, '\s+', ' ', 'g'));

    -- Insert or update stats
    INSERT INTO performance.query_stats (
        query_hash, query_text, total_calls, total_time,
        min_time, max_time, mean_time, rows_returned
    )
    VALUES (
        v_query_hash, p_query_text, 1, p_execution_time,
        p_execution_time, p_execution_time, p_execution_time, p_rows_returned
    )
    ON CONFLICT (query_hash) DO UPDATE SET
        total_calls = performance.query_stats.total_calls + 1,
        total_time = performance.query_stats.total_time + p_execution_time,
        min_time = LEAST(performance.query_stats.min_time, p_execution_time),
        max_time = GREATEST(performance.query_stats.max_time, p_execution_time),
        mean_time = (performance.query_stats.total_time + p_execution_time) / (performance.query_stats.total_calls + 1),
        rows_returned = performance.query_stats.rows_returned + p_rows_returned,
        last_seen = NOW();
END;
$$ LANGUAGE plpgsql;

-- Function: Get slow queries
CREATE OR REPLACE FUNCTION performance.get_slow_queries(
    p_min_time DOUBLE PRECISION DEFAULT 1000,
    p_limit INTEGER DEFAULT 20
)
RETURNS TABLE (
    query_text TEXT,
    total_calls INTEGER,
    mean_time DOUBLE PRECISION,
    total_time DOUBLE PRECISION,
    last_seen TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        qs.query_text,
        qs.total_calls,
        qs.mean_time,
        qs.total_time,
        qs.last_seen
    FROM performance.query_stats qs
    WHERE qs.mean_time >= p_min_time
    ORDER BY qs.mean_time DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function: Recommend indexes based on query patterns
CREATE OR REPLACE FUNCTION performance.analyze_and_recommend_indexes()
RETURNS TABLE (
    table_name TEXT,
    columns TEXT[],
    reason TEXT,
    priority INTEGER
) AS $$
BEGIN
    -- This is a simplified version
    -- In production, would analyze pg_stat_statements and actual query plans

    -- Find tables with sequential scans on large tables
    RETURN QUERY
    SELECT
        schemaname || '.' || tablename as table_name,
        ARRAY['id']::text[] as columns,
        'High sequential scan count on large table' as reason,
        8 as priority
    FROM pg_stat_user_tables
    WHERE seq_scan > 1000
    AND n_live_tup > 10000
    AND schemaname NOT IN ('pg_catalog', 'information_schema', 'performance');

    -- Additional analysis would go here
END;
$$ LANGUAGE plpgsql;

-- Function: Get cache hit ratio
CREATE OR REPLACE FUNCTION performance.get_cache_hit_ratio(
    p_cache_type TEXT DEFAULT NULL
)
RETURNS TABLE (
    cache_type TEXT,
    hits BIGINT,
    misses BIGINT,
    hit_ratio DOUBLE PRECISION
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        cs.cache_type,
        SUM(cs.hits) as hits,
        SUM(cs.misses) as misses,
        CASE
            WHEN SUM(cs.hits + cs.misses) > 0
            THEN (SUM(cs.hits)::DOUBLE PRECISION / SUM(cs.hits + cs.misses)) * 100
            ELSE 0
        END as hit_ratio
    FROM performance.cache_stats cs
    WHERE p_cache_type IS NULL OR cs.cache_type = p_cache_type
    GROUP BY cs.cache_type;
END;
$$ LANGUAGE plpgsql;

-- Function: Get database size by schema
CREATE OR REPLACE FUNCTION performance.get_database_sizes()
RETURNS TABLE (
    schema_name TEXT,
    size_bytes BIGINT,
    size_pretty TEXT,
    table_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        schemaname::text,
        SUM(pg_total_relation_size(schemaname || '.' || tablename))::bigint as size_bytes,
        pg_size_pretty(SUM(pg_total_relation_size(schemaname || '.' || tablename))) as size_pretty,
        COUNT(*)::integer as table_count
    FROM pg_tables
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    GROUP BY schemaname
    ORDER BY size_bytes DESC;
END;
$$ LANGUAGE plpgsql;

-- Function: Get table bloat estimate
CREATE OR REPLACE FUNCTION performance.get_table_bloat()
RETURNS TABLE (
    table_name TEXT,
    bloat_ratio DOUBLE PRECISION,
    wasted_bytes BIGINT,
    wasted_pretty TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        schemaname || '.' || tablename as table_name,
        CASE
            WHEN n_live_tup > 0
            THEN (n_dead_tup::DOUBLE PRECISION / n_live_tup) * 100
            ELSE 0
        END as bloat_ratio,
        pg_total_relation_size(schemaname || '.' || tablename) *
            (n_dead_tup::DOUBLE PRECISION / GREATEST(n_live_tup, 1)) as wasted_bytes,
        pg_size_pretty(
            (pg_total_relation_size(schemaname || '.' || tablename) *
            (n_dead_tup::DOUBLE PRECISION / GREATEST(n_live_tup, 1)))::bigint
        ) as wasted_pretty
    FROM pg_stat_user_tables
    WHERE n_dead_tup > 1000
    ORDER BY bloat_ratio DESC
    LIMIT 20;
END;
$$ LANGUAGE plpgsql;

-- Function: Get index usage stats
CREATE OR REPLACE FUNCTION performance.get_unused_indexes()
RETURNS TABLE (
    schema_name TEXT,
    table_name TEXT,
    index_name TEXT,
    index_size TEXT,
    index_scans BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        schemaname::text,
        tablename::text,
        indexname::text,
        pg_size_pretty(pg_relation_size(schemaname || '.' || indexname)) as index_size,
        idx_scan as index_scans
    FROM pg_stat_user_indexes
    WHERE idx_scan < 100
    AND schemaname NOT IN ('pg_catalog', 'information_schema')
    ORDER BY pg_relation_size(schemaname || '.' || indexname) DESC
    LIMIT 20;
END;
$$ LANGUAGE plpgsql;

-- Function: Recommend vacuum operations
CREATE OR REPLACE FUNCTION performance.recommend_vacuum()
RETURNS TABLE (
    table_name TEXT,
    dead_tuples BIGINT,
    live_tuples BIGINT,
    bloat_percent DOUBLE PRECISION,
    recommendation TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        schemaname || '.' || tablename as table_name,
        n_dead_tup as dead_tuples,
        n_live_tup as live_tuples,
        CASE
            WHEN n_live_tup > 0
            THEN (n_dead_tup::DOUBLE PRECISION / n_live_tup) * 100
            ELSE 0
        END as bloat_percent,
        CASE
            WHEN (n_dead_tup::DOUBLE PRECISION / GREATEST(n_live_tup, 1)) > 0.2
            THEN 'VACUUM FULL ' || schemaname || '.' || tablename
            WHEN n_dead_tup > 1000
            THEN 'VACUUM ANALYZE ' || schemaname || '.' || tablename
            ELSE 'No action needed'
        END as recommendation
    FROM pg_stat_user_tables
    WHERE n_dead_tup > 100
    ORDER BY bloat_percent DESC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- VIEWS
-- ============================================================================

-- View: Performance summary
CREATE OR REPLACE VIEW performance.summary AS
SELECT
    (SELECT COUNT(*) FROM performance.query_stats WHERE mean_time > 1000) as slow_queries,
    (SELECT COUNT(*) FROM performance.index_recommendations WHERE status = 'recommended') as pending_index_recs,
    (SELECT SUM(hits + misses) FROM performance.cache_stats) as total_cache_requests,
    (SELECT AVG(
        CASE
            WHEN (hits + misses) > 0
            THEN (hits::DOUBLE PRECISION / (hits + misses)) * 100
            ELSE 0
        END
    ) FROM performance.cache_stats) as avg_cache_hit_ratio,
    (SELECT COUNT(*) FROM performance.get_table_bloat() WHERE bloat_ratio > 20) as bloated_tables;

-- View: Query hotspots
CREATE OR REPLACE VIEW performance.query_hotspots AS
SELECT
    query_hash,
    LEFT(query_text, 100) as query_preview,
    total_calls,
    mean_time,
    total_time,
    last_seen
FROM performance.query_stats
WHERE total_calls > 10
ORDER BY total_time DESC
LIMIT 50;

-- ============================================================================
-- GRANTS
-- ============================================================================

GRANT USAGE ON SCHEMA performance TO hasura;
GRANT SELECT ON ALL TABLES IN SCHEMA performance TO hasura;
GRANT INSERT ON performance.query_stats TO hasura;
GRANT INSERT ON performance.cache_stats TO hasura;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA performance TO hasura;

COMMIT;
