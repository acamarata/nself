-- nself Database Migration: 018_rollback_whitelabel_system.sql
-- Rollback of 016_create_whitelabel_system.sql
--
-- Safely removes all white-label system tables, views, and functions
-- in reverse dependency order with proper error handling.
--
-- Author: nself
-- Version: 0.9.0
-- Date: 2026-01-30
-- ============================================================================

BEGIN TRANSACTION;

-- ============================================================================
-- Drop Triggers
-- ============================================================================

DROP TRIGGER IF EXISTS trigger_whitelabel_brands_updated_at ON whitelabel_brands;
DROP TRIGGER IF EXISTS trigger_whitelabel_domains_updated_at ON whitelabel_domains;
DROP TRIGGER IF EXISTS trigger_whitelabel_themes_updated_at ON whitelabel_themes;
DROP TRIGGER IF EXISTS trigger_whitelabel_email_templates_updated_at ON whitelabel_email_templates;
DROP TRIGGER IF EXISTS trigger_whitelabel_assets_updated_at ON whitelabel_assets;

-- ============================================================================
-- Drop Functions
-- ============================================================================

DROP FUNCTION IF EXISTS update_whitelabel_updated_at();

-- ============================================================================
-- Drop Views (must be before dropping referenced tables)
-- ============================================================================

DROP VIEW IF EXISTS whitelabel_brands_full;

-- ============================================================================
-- Drop Foreign Key Constraints (before dropping tables)
-- ============================================================================

-- Drop foreign keys from whitelabel_brands table (added via ALTER)
ALTER TABLE IF EXISTS whitelabel_brands
  DROP CONSTRAINT IF EXISTS fk_brand_logo_main,
  DROP CONSTRAINT IF EXISTS fk_brand_logo_icon,
  DROP CONSTRAINT IF EXISTS fk_brand_logo_email,
  DROP CONSTRAINT IF EXISTS fk_brand_logo_favicon,
  DROP CONSTRAINT IF EXISTS fk_brand_custom_css,
  DROP CONSTRAINT IF EXISTS fk_brand_active_theme;

-- Drop foreign keys from whitelabel_domains table (added via ALTER)
ALTER TABLE IF EXISTS whitelabel_domains
  DROP CONSTRAINT IF EXISTS fk_domain_ssl_cert,
  DROP CONSTRAINT IF EXISTS fk_domain_ssl_key,
  DROP CONSTRAINT IF EXISTS fk_domain_ssl_chain;

-- ============================================================================
-- Drop Tables (in reverse dependency order)
-- ============================================================================

-- Tables with no ON DELETE CASCADE dependencies
DROP TABLE IF EXISTS whitelabel_assets;
DROP TABLE IF EXISTS whitelabel_email_templates;
DROP TABLE IF EXISTS whitelabel_themes;
DROP TABLE IF EXISTS whitelabel_domains;
DROP TABLE IF EXISTS whitelabel_brands;

-- ============================================================================
-- Drop Extensions (only if created by this migration)
-- ============================================================================

-- Note: Only drop uuid-ossp if no other tables depend on it
-- DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;

-- ============================================================================
-- Log Migration Completion
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 018_rollback_whitelabel_system.sql completed successfully';
    RAISE NOTICE 'Removed white-label system with:';
    RAISE NOTICE '  - whitelabel_brands table';
    RAISE NOTICE '  - whitelabel_domains table';
    RAISE NOTICE '  - whitelabel_themes table';
    RAISE NOTICE '  - whitelabel_email_templates table';
    RAISE NOTICE '  - whitelabel_assets table';
    RAISE NOTICE '  - whitelabel_brands_full view';
    RAISE NOTICE '  - All associated triggers and functions';
END $$;

COMMIT;

-- ============================================================================
-- Verification (uncomment to check state after rollback)
-- ============================================================================

-- SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whitelabel_brands');
-- SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whitelabel_domains');
-- SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whitelabel_themes');
-- SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whitelabel_email_templates');
-- SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whitelabel_assets');
-- SELECT EXISTS (SELECT 1 FROM information_schema.views WHERE table_name = 'whitelabel_brands_full');
