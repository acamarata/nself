-- ============================================================================
-- nself White-Label System Database Migration
-- Sprint 14: White-Label & Customization (60pts) for v0.9.0
-- ============================================================================
-- Description: Creates tables for white-label branding, custom domains,
--              themes, email templates, and asset management
-- ============================================================================

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- Brands Table
-- ============================================================================
-- Stores white-label brand configurations for multi-tenant support

CREATE TABLE IF NOT EXISTS whitelabel_brands (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id VARCHAR(255) UNIQUE NOT NULL DEFAULT 'default',
  brand_name VARCHAR(255) NOT NULL,
  tagline TEXT,
  description TEXT,

  -- Contact Information
  company_address TEXT,
  support_email VARCHAR(255),
  support_url TEXT,

  -- Branding Configuration
  primary_color VARCHAR(7) DEFAULT '#0066cc',
  secondary_color VARCHAR(7) DEFAULT '#ff6600',
  accent_color VARCHAR(7) DEFAULT '#00cc66',
  background_color VARCHAR(7) DEFAULT '#ffffff',
  text_color VARCHAR(7) DEFAULT '#333333',

  -- Fonts
  primary_font VARCHAR(255) DEFAULT 'Inter, system-ui, sans-serif',
  secondary_font VARCHAR(255) DEFAULT 'Georgia, serif',
  code_font VARCHAR(255) DEFAULT 'Fira Code, Consolas, monospace',

  -- Logo References
  logo_main_id UUID,
  logo_icon_id UUID,
  logo_email_id UUID,
  logo_favicon_id UUID,

  -- Custom CSS
  custom_css_id UUID,

  -- Theme
  active_theme_id UUID,

  -- Status
  is_active BOOLEAN DEFAULT true,
  is_primary BOOLEAN DEFAULT false,

  -- Metadata
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  created_by UUID,
  updated_by UUID,

  -- Constraints
  CONSTRAINT valid_primary_color CHECK (primary_color ~ '^#[0-9A-Fa-f]{6}$'),
  CONSTRAINT valid_secondary_color CHECK (secondary_color ~ '^#[0-9A-Fa-f]{6}$'),
  CONSTRAINT valid_accent_color CHECK (accent_color ~ '^#[0-9A-Fa-f]{6}$'),
  CONSTRAINT valid_background_color CHECK (background_color ~ '^#[0-9A-Fa-f]{6}$'),
  CONSTRAINT valid_text_color CHECK (text_color ~ '^#[0-9A-Fa-f]{6}$')
);

-- Index for tenant lookups
CREATE INDEX idx_whitelabel_brands_tenant ON whitelabel_brands(tenant_id);
CREATE INDEX idx_whitelabel_brands_active ON whitelabel_brands(is_active);

COMMENT ON TABLE whitelabel_brands IS 'White-label brand configurations for multi-tenant support';

-- ============================================================================
-- Custom Domains Table
-- ============================================================================
-- Manages custom domains with SSL and DNS verification

CREATE TABLE IF NOT EXISTS whitelabel_domains (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  brand_id UUID REFERENCES whitelabel_brands(id) ON DELETE CASCADE,

  -- Domain Configuration
  domain VARCHAR(255) UNIQUE NOT NULL,
  is_primary BOOLEAN DEFAULT false,

  -- DNS Configuration
  dns_verified BOOLEAN DEFAULT false,
  dns_verification_token VARCHAR(255),
  dns_verification_method VARCHAR(50) DEFAULT 'txt', -- txt, cname, a
  dns_verified_at TIMESTAMP WITH TIME ZONE,

  -- SSL Configuration
  ssl_enabled BOOLEAN DEFAULT false,
  ssl_provider VARCHAR(50) DEFAULT 'letsencrypt', -- letsencrypt, selfsigned, custom
  ssl_issuer VARCHAR(255),
  ssl_issued_at TIMESTAMP WITH TIME ZONE,
  ssl_expiry_date TIMESTAMP WITH TIME ZONE,
  ssl_auto_renew BOOLEAN DEFAULT true,
  ssl_last_renewed_at TIMESTAMP WITH TIME ZONE,

  -- Certificate Storage (references to whitelabel_assets)
  ssl_cert_id UUID,
  ssl_key_id UUID,
  ssl_chain_id UUID,

  -- Health Status
  health_status VARCHAR(50) DEFAULT 'unknown', -- healthy, degraded, unhealthy, unknown
  last_health_check_at TIMESTAMP WITH TIME ZONE,
  health_check_interval INTEGER DEFAULT 300, -- seconds

  -- HTTP Configuration
  redirect_to_https BOOLEAN DEFAULT true,
  redirect_www_to_apex BOOLEAN DEFAULT false,

  -- Status
  status VARCHAR(50) DEFAULT 'pending', -- pending, verified, active, suspended, failed
  is_active BOOLEAN DEFAULT true,

  -- Metadata
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  created_by UUID,
  updated_by UUID,

  -- Constraints
  CONSTRAINT valid_domain_format CHECK (domain ~ '^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]?\.[a-zA-Z]{2,}$'),
  CONSTRAINT valid_ssl_provider CHECK (ssl_provider IN ('letsencrypt', 'selfsigned', 'custom')),
  CONSTRAINT valid_health_status CHECK (health_status IN ('healthy', 'degraded', 'unhealthy', 'unknown')),
  CONSTRAINT valid_status CHECK (status IN ('pending', 'verified', 'active', 'suspended', 'failed'))
);

-- Indexes for domain lookups
CREATE INDEX idx_whitelabel_domains_brand ON whitelabel_domains(brand_id);
CREATE INDEX idx_whitelabel_domains_domain ON whitelabel_domains(domain);
CREATE INDEX idx_whitelabel_domains_status ON whitelabel_domains(status);
CREATE INDEX idx_whitelabel_domains_health ON whitelabel_domains(health_status);
CREATE INDEX idx_whitelabel_domains_ssl_expiry ON whitelabel_domains(ssl_expiry_date) WHERE ssl_enabled = true;

COMMENT ON TABLE whitelabel_domains IS 'Custom domains with SSL and DNS verification';

-- ============================================================================
-- Themes Table
-- ============================================================================
-- Manages UI themes with CSS variables and dark/light modes

CREATE TABLE IF NOT EXISTS whitelabel_themes (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  brand_id UUID REFERENCES whitelabel_brands(id) ON DELETE CASCADE,

  -- Theme Information
  theme_name VARCHAR(255) NOT NULL,
  display_name VARCHAR(255) NOT NULL,
  description TEXT,
  version VARCHAR(50) DEFAULT '1.0.0',
  author VARCHAR(255),

  -- Theme Mode
  mode VARCHAR(50) DEFAULT 'light', -- light, dark, auto

  -- Color Variables (JSON)
  colors JSONB DEFAULT '{
    "primary": "#0066cc",
    "secondary": "#6c757d",
    "accent": "#00cc66",
    "background": "#ffffff",
    "surface": "#ffffff",
    "text": "#212529",
    "border": "#dee2e6",
    "success": "#28a745",
    "warning": "#ffc107",
    "error": "#dc3545",
    "info": "#17a2b8"
  }'::jsonb,

  -- Typography Variables (JSON)
  typography JSONB DEFAULT '{
    "fontFamily": "-apple-system, BlinkMacSystemFont, sans-serif",
    "fontFamilyMono": "Courier New, monospace",
    "fontSize": "16px",
    "fontWeight": "400",
    "lineHeight": "1.5"
  }'::jsonb,

  -- Spacing Variables (JSON)
  spacing JSONB DEFAULT '{
    "xs": "4px",
    "sm": "8px",
    "md": "16px",
    "lg": "24px",
    "xl": "32px"
  }'::jsonb,

  -- Border Variables (JSON)
  borders JSONB DEFAULT '{
    "radius": "4px",
    "width": "1px"
  }'::jsonb,

  -- Shadow Variables (JSON)
  shadows JSONB DEFAULT '{
    "sm": "0 1px 3px rgba(0,0,0,0.12)",
    "md": "0 4px 6px rgba(0,0,0,0.1)",
    "lg": "0 10px 20px rgba(0,0,0,0.15)"
  }'::jsonb,

  -- Custom CSS
  custom_css TEXT,

  -- Compiled CSS (generated from variables)
  compiled_css TEXT,

  -- Status
  is_active BOOLEAN DEFAULT false,
  is_default BOOLEAN DEFAULT false,
  is_system BOOLEAN DEFAULT false, -- true for built-in themes

  -- Metadata
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  created_by UUID,
  updated_by UUID,

  -- Constraints
  CONSTRAINT unique_theme_per_brand UNIQUE(brand_id, theme_name),
  CONSTRAINT valid_theme_mode CHECK (mode IN ('light', 'dark', 'auto'))
);

-- Indexes for theme lookups
CREATE INDEX idx_whitelabel_themes_brand ON whitelabel_themes(brand_id);
CREATE INDEX idx_whitelabel_themes_active ON whitelabel_themes(is_active);
CREATE INDEX idx_whitelabel_themes_system ON whitelabel_themes(is_system);

COMMENT ON TABLE whitelabel_themes IS 'UI themes with CSS variables and styling';

-- ============================================================================
-- Email Templates Table
-- ============================================================================
-- Manages custom email templates with multi-language support

CREATE TABLE IF NOT EXISTS whitelabel_email_templates (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  brand_id UUID REFERENCES whitelabel_brands(id) ON DELETE CASCADE,

  -- Template Information
  template_name VARCHAR(255) NOT NULL, -- welcome, password-reset, verify-email, etc.
  display_name VARCHAR(255) NOT NULL,
  description TEXT,
  category VARCHAR(100), -- authentication, notifications, alerts, etc.

  -- Language Support
  language_code VARCHAR(10) DEFAULT 'en', -- ISO 639-1 language code

  -- Email Configuration
  subject VARCHAR(500) NOT NULL,
  from_name VARCHAR(255),
  from_email VARCHAR(255),
  reply_to VARCHAR(255),

  -- Template Content
  html_content TEXT NOT NULL,
  text_content TEXT NOT NULL,

  -- Variables (JSON array of variable names)
  variables JSONB DEFAULT '[]'::jsonb,

  -- Sample Data (for preview)
  sample_data JSONB,

  -- Compiled Templates (with variables replaced)
  compiled_html TEXT,
  compiled_text TEXT,

  -- Status
  is_active BOOLEAN DEFAULT true,
  is_default BOOLEAN DEFAULT false,
  is_system BOOLEAN DEFAULT false,

  -- Usage Statistics
  sent_count INTEGER DEFAULT 0,
  last_sent_at TIMESTAMP WITH TIME ZONE,

  -- Metadata
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  created_by UUID,
  updated_by UUID,

  -- Constraints
  CONSTRAINT unique_template_per_brand_language UNIQUE(brand_id, template_name, language_code)
);

-- Indexes for template lookups
CREATE INDEX idx_whitelabel_email_templates_brand ON whitelabel_email_templates(brand_id);
CREATE INDEX idx_whitelabel_email_templates_name ON whitelabel_email_templates(template_name);
CREATE INDEX idx_whitelabel_email_templates_language ON whitelabel_email_templates(language_code);
CREATE INDEX idx_whitelabel_email_templates_category ON whitelabel_email_templates(category);
CREATE INDEX idx_whitelabel_email_templates_active ON whitelabel_email_templates(is_active);

COMMENT ON TABLE whitelabel_email_templates IS 'Custom email templates with multi-language support';

-- ============================================================================
-- Assets Table
-- ============================================================================
-- Manages logos, images, fonts, and other white-label assets

CREATE TABLE IF NOT EXISTS whitelabel_assets (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  brand_id UUID REFERENCES whitelabel_brands(id) ON DELETE CASCADE,

  -- Asset Information
  asset_name VARCHAR(255) NOT NULL,
  asset_type VARCHAR(100) NOT NULL, -- logo, image, font, css, certificate, key
  asset_category VARCHAR(100), -- main, icon, email, favicon, etc.

  -- File Information
  file_name VARCHAR(255) NOT NULL,
  file_path TEXT NOT NULL,
  file_size BIGINT, -- bytes
  mime_type VARCHAR(255),
  file_extension VARCHAR(50),

  -- Image Metadata (for images)
  image_width INTEGER,
  image_height INTEGER,
  image_format VARCHAR(50),

  -- Storage Information
  storage_provider VARCHAR(100) DEFAULT 'local', -- local, s3, minio, etc.
  storage_bucket VARCHAR(255),
  storage_key TEXT,

  -- CDN Information
  cdn_url TEXT,
  cdn_enabled BOOLEAN DEFAULT false,

  -- Access Control
  is_public BOOLEAN DEFAULT true,
  access_url TEXT,

  -- Version Control
  version INTEGER DEFAULT 1,
  previous_version_id UUID,

  -- Status
  is_active BOOLEAN DEFAULT true,

  -- Metadata
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  created_by UUID,
  updated_by UUID,

  -- Constraints
  CONSTRAINT valid_asset_type CHECK (asset_type IN ('logo', 'image', 'font', 'css', 'certificate', 'key', 'other'))
);

-- Indexes for asset lookups
CREATE INDEX idx_whitelabel_assets_brand ON whitelabel_assets(brand_id);
CREATE INDEX idx_whitelabel_assets_type ON whitelabel_assets(asset_type);
CREATE INDEX idx_whitelabel_assets_category ON whitelabel_assets(asset_category);
CREATE INDEX idx_whitelabel_assets_active ON whitelabel_assets(is_active);

COMMENT ON TABLE whitelabel_assets IS 'Logos, images, fonts, and other white-label assets';

-- ============================================================================
-- Foreign Key Constraints
-- ============================================================================
-- Add foreign key constraints from brands table to assets/themes

ALTER TABLE whitelabel_brands
  ADD CONSTRAINT fk_brand_logo_main FOREIGN KEY (logo_main_id) REFERENCES whitelabel_assets(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_brand_logo_icon FOREIGN KEY (logo_icon_id) REFERENCES whitelabel_assets(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_brand_logo_email FOREIGN KEY (logo_email_id) REFERENCES whitelabel_assets(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_brand_logo_favicon FOREIGN KEY (logo_favicon_id) REFERENCES whitelabel_assets(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_brand_custom_css FOREIGN KEY (custom_css_id) REFERENCES whitelabel_assets(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_brand_active_theme FOREIGN KEY (active_theme_id) REFERENCES whitelabel_themes(id) ON DELETE SET NULL;

ALTER TABLE whitelabel_domains
  ADD CONSTRAINT fk_domain_ssl_cert FOREIGN KEY (ssl_cert_id) REFERENCES whitelabel_assets(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_domain_ssl_key FOREIGN KEY (ssl_key_id) REFERENCES whitelabel_assets(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_domain_ssl_chain FOREIGN KEY (ssl_chain_id) REFERENCES whitelabel_assets(id) ON DELETE SET NULL;

-- ============================================================================
-- Triggers for Updated Timestamp
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_whitelabel_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to all whitelabel tables
CREATE TRIGGER trigger_whitelabel_brands_updated_at
  BEFORE UPDATE ON whitelabel_brands
  FOR EACH ROW EXECUTE FUNCTION update_whitelabel_updated_at();

CREATE TRIGGER trigger_whitelabel_domains_updated_at
  BEFORE UPDATE ON whitelabel_domains
  FOR EACH ROW EXECUTE FUNCTION update_whitelabel_updated_at();

CREATE TRIGGER trigger_whitelabel_themes_updated_at
  BEFORE UPDATE ON whitelabel_themes
  FOR EACH ROW EXECUTE FUNCTION update_whitelabel_updated_at();

CREATE TRIGGER trigger_whitelabel_email_templates_updated_at
  BEFORE UPDATE ON whitelabel_email_templates
  FOR EACH ROW EXECUTE FUNCTION update_whitelabel_updated_at();

CREATE TRIGGER trigger_whitelabel_assets_updated_at
  BEFORE UPDATE ON whitelabel_assets
  FOR EACH ROW EXECUTE FUNCTION update_whitelabel_updated_at();

-- ============================================================================
-- Default Data Insertion
-- ============================================================================

-- Insert default brand
INSERT INTO whitelabel_brands (
  tenant_id,
  brand_name,
  tagline,
  description,
  is_primary,
  is_active
) VALUES (
  'default',
  'nself',
  'Powerful Backend for Modern Applications',
  'Open-source backend infrastructure platform',
  true,
  true
) ON CONFLICT (tenant_id) DO NOTHING;

-- Get default brand ID for subsequent inserts
DO $$
DECLARE
  default_brand_id UUID;
BEGIN
  SELECT id INTO default_brand_id FROM whitelabel_brands WHERE tenant_id = 'default';

  -- Insert default themes
  INSERT INTO whitelabel_themes (
    brand_id,
    theme_name,
    display_name,
    description,
    mode,
    is_default,
    is_system,
    is_active
  ) VALUES
    (default_brand_id, 'light', 'Light Theme', 'Clean and bright light theme', 'light', true, true, true),
    (default_brand_id, 'dark', 'Dark Theme', 'Easy on the eyes dark theme', 'dark', false, true, false),
    (default_brand_id, 'high-contrast', 'High Contrast', 'Maximum contrast for accessibility', 'dark', false, true, false)
  ON CONFLICT (brand_id, theme_name) DO NOTHING;

  -- Insert default email templates
  INSERT INTO whitelabel_email_templates (
    brand_id,
    template_name,
    display_name,
    description,
    category,
    subject,
    html_content,
    text_content,
    is_system,
    is_default
  ) VALUES
    (
      default_brand_id,
      'welcome',
      'Welcome Email',
      'Welcome email sent to new users',
      'authentication',
      'Welcome to {{BRAND_NAME}}!',
      '<h1>Welcome!</h1><p>Hi {{USER_NAME}}, welcome to {{BRAND_NAME}}.</p>',
      'Welcome! Hi {{USER_NAME}}, welcome to {{BRAND_NAME}}.',
      true,
      true
    ),
    (
      default_brand_id,
      'password-reset',
      'Password Reset',
      'Password reset email with secure link',
      'security',
      'Reset Your Password',
      '<h1>Password Reset</h1><p>Click here to reset: {{RESET_URL}}</p>',
      'Password Reset - Click here: {{RESET_URL}}',
      true,
      true
    )
  ON CONFLICT (brand_id, template_name, language_code) DO NOTHING;
END $$;

-- ============================================================================
-- Permissions (Hasura Integration)
-- ============================================================================

-- Grant permissions for authenticated users
-- Note: Adjust these based on your authentication system

COMMENT ON COLUMN whitelabel_brands.tenant_id IS 'Unique tenant identifier for multi-tenant support';
COMMENT ON COLUMN whitelabel_domains.domain IS 'Custom domain name (e.g., app.company.com)';
COMMENT ON COLUMN whitelabel_themes.colors IS 'JSON object containing color variables';
COMMENT ON COLUMN whitelabel_email_templates.variables IS 'Array of variable names used in template';

-- ============================================================================
-- Views for Common Queries
-- ============================================================================

-- View for active brands with their themes and domains
CREATE OR REPLACE VIEW whitelabel_brands_full AS
SELECT
  b.*,
  t.theme_name,
  t.display_name as theme_display_name,
  t.mode as theme_mode,
  array_agg(DISTINCT d.domain) FILTER (WHERE d.domain IS NOT NULL) as domains,
  COUNT(DISTINCT d.id) as domain_count
FROM whitelabel_brands b
LEFT JOIN whitelabel_themes t ON b.active_theme_id = t.id
LEFT JOIN whitelabel_domains d ON b.id = d.brand_id AND d.is_active = true
WHERE b.is_active = true
GROUP BY b.id, t.id;

COMMENT ON VIEW whitelabel_brands_full IS 'Complete brand information with themes and domains';

-- ============================================================================
-- Migration Complete
-- ============================================================================

-- Log migration completion
DO $$
BEGIN
  RAISE NOTICE 'White-Label System Migration 016 completed successfully';
  RAISE NOTICE 'Created tables: whitelabel_brands, whitelabel_domains, whitelabel_themes, whitelabel_email_templates, whitelabel_assets';
  RAISE NOTICE 'Created view: whitelabel_brands_full';
END $$;
