-- 014_create_security_system.sql
-- Advanced Security System for nself
-- Sprint 17: Advanced Security (25pts)
--
-- Features:
-- - WebAuthn/FIDO2 support for hardware security keys
-- - Device management and tracking
-- - Suspicious activity detection
-- - Security event logging and audit trail
-- - Security incident management
-- - Password strength analysis
-- - Session security monitoring

-- ============================================================================
-- WebAuthn/FIDO2 Hardware Key Support
-- ============================================================================

-- Store WebAuthn credentials (hardware keys, Touch ID, Face ID, etc.)
CREATE TABLE IF NOT EXISTS auth.webauthn_credentials (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  credential_id TEXT NOT NULL UNIQUE, -- Base64URL encoded credential ID
  public_key TEXT NOT NULL, -- Base64URL encoded public key
  counter BIGINT NOT NULL DEFAULT 0, -- Signature counter to prevent replay attacks
  credential_type TEXT NOT NULL DEFAULT 'public-key',
  transports TEXT[], -- ['usb', 'nfc', 'ble', 'internal']
  authenticator_attachment TEXT, -- 'platform' or 'cross-platform'

  -- User-friendly metadata
  name TEXT, -- User-assigned name (e.g., "YubiKey 5", "Touch ID")
  aaguid UUID, -- Authenticator Attestation GUID

  -- Security flags
  is_backup_eligible BOOLEAN DEFAULT false,
  is_backup_state BOOLEAN DEFAULT false,

  -- Timestamps
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_used_at TIMESTAMPTZ,

  -- Metadata
  user_agent TEXT,
  ip_address INET,

  CONSTRAINT valid_credential_id CHECK (length(credential_id) > 0),
  CONSTRAINT valid_public_key CHECK (length(public_key) > 0)
);

-- Index for fast lookups
CREATE INDEX idx_webauthn_credentials_user_id ON auth.webauthn_credentials(user_id);
CREATE INDEX idx_webauthn_credentials_credential_id ON auth.webauthn_credentials(credential_id);
CREATE INDEX idx_webauthn_credentials_last_used ON auth.webauthn_credentials(last_used_at DESC);

-- Enable RLS
ALTER TABLE auth.webauthn_credentials ENABLE ROW LEVEL SECURITY;

-- Users can only see and manage their own credentials
CREATE POLICY webauthn_credentials_user_policy ON auth.webauthn_credentials
  FOR ALL
  USING (user_id = auth.uid());

-- ============================================================================
-- Device Management
-- ============================================================================

-- Track user devices for security monitoring
CREATE TABLE IF NOT EXISTS auth.user_devices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,

  -- Device identification
  device_id TEXT NOT NULL, -- Fingerprint hash of device characteristics
  device_name TEXT, -- User-assigned name (e.g., "iPhone 14", "MacBook Pro")
  device_type TEXT, -- 'mobile', 'tablet', 'desktop', 'other'

  -- Device characteristics
  user_agent TEXT,
  os TEXT, -- Operating system
  os_version TEXT,
  browser TEXT,
  browser_version TEXT,

  -- Network information
  last_ip_address INET,
  last_location JSONB, -- {country, region, city, lat, lon}

  -- Security status
  is_trusted BOOLEAN DEFAULT false, -- User explicitly trusted this device
  is_current BOOLEAN DEFAULT true, -- Is this the current session device?
  risk_score INTEGER DEFAULT 0 CHECK (risk_score >= 0 AND risk_score <= 100),

  -- Timestamps
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Metadata
  login_count INTEGER DEFAULT 1,
  failed_login_attempts INTEGER DEFAULT 0,

  UNIQUE(user_id, device_id)
);

-- Indexes
CREATE INDEX idx_user_devices_user_id ON auth.user_devices(user_id);
CREATE INDEX idx_user_devices_device_id ON auth.user_devices(device_id);
CREATE INDEX idx_user_devices_last_seen ON auth.user_devices(last_seen_at DESC);
CREATE INDEX idx_user_devices_risk_score ON auth.user_devices(risk_score DESC);

-- Enable RLS
ALTER TABLE auth.user_devices ENABLE ROW LEVEL SECURITY;

-- Users can only see their own devices
CREATE POLICY user_devices_user_policy ON auth.user_devices
  FOR ALL
  USING (user_id = auth.uid());

-- ============================================================================
-- Security Events and Audit Trail
-- ============================================================================

-- Log all security-related events
CREATE TABLE IF NOT EXISTS auth.security_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Event details
  event_type TEXT NOT NULL, -- 'login', 'logout', 'password_change', 'mfa_enable', etc.
  severity TEXT NOT NULL DEFAULT 'info', -- 'info', 'warning', 'critical'
  category TEXT NOT NULL DEFAULT 'authentication', -- 'authentication', 'authorization', 'device', 'suspicious'

  -- User and device
  user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
  device_id UUID REFERENCES auth.user_devices(id) ON DELETE SET NULL,
  session_id UUID,

  -- Event data
  description TEXT NOT NULL,
  details JSONB DEFAULT '{}', -- Additional context

  -- Network information
  ip_address INET,
  user_agent TEXT,
  location JSONB, -- {country, region, city}

  -- Risk assessment
  risk_score INTEGER DEFAULT 0 CHECK (risk_score >= 0 AND risk_score <= 100),
  is_suspicious BOOLEAN DEFAULT false,

  -- Resolution
  status TEXT DEFAULT 'open', -- 'open', 'investigating', 'resolved', 'false_positive'
  resolved_at TIMESTAMPTZ,
  resolved_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
  resolution_notes TEXT,

  -- Timestamps
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Metadata
  metadata JSONB DEFAULT '{}'
);

-- Indexes for fast queries
CREATE INDEX idx_security_events_user_id ON auth.security_events(user_id);
CREATE INDEX idx_security_events_created_at ON auth.security_events(created_at DESC);
CREATE INDEX idx_security_events_event_type ON auth.security_events(event_type);
CREATE INDEX idx_security_events_severity ON auth.security_events(severity);
CREATE INDEX idx_security_events_is_suspicious ON auth.security_events(is_suspicious) WHERE is_suspicious = true;
CREATE INDEX idx_security_events_status ON auth.security_events(status) WHERE status = 'open';

-- Enable RLS
ALTER TABLE auth.security_events ENABLE ROW LEVEL SECURITY;

-- Users can only see their own security events
CREATE POLICY security_events_user_policy ON auth.security_events
  FOR SELECT
  USING (user_id = auth.uid());

-- Admins can see all events
CREATE POLICY security_events_admin_policy ON auth.security_events
  FOR ALL
  USING (
    EXISTS (
      SELECT 1 FROM auth.users
      WHERE id = auth.uid()
      AND role IN ('admin', 'security_admin')
    )
  );

-- ============================================================================
-- Security Incidents
-- ============================================================================

-- Track security incidents and investigations
CREATE TABLE IF NOT EXISTS auth.security_incidents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Incident details
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'medium', -- 'low', 'medium', 'high', 'critical'
  category TEXT NOT NULL, -- 'breach_attempt', 'suspicious_activity', 'credential_stuffing', etc.

  -- Affected entities
  affected_user_ids UUID[],
  affected_device_ids UUID[],
  related_event_ids UUID[], -- Links to security_events

  -- Status tracking
  status TEXT NOT NULL DEFAULT 'open', -- 'open', 'investigating', 'contained', 'resolved', 'false_positive'
  priority TEXT NOT NULL DEFAULT 'normal', -- 'low', 'normal', 'high', 'urgent'

  -- Assignment
  assigned_to UUID REFERENCES auth.users(id) ON DELETE SET NULL,
  assigned_at TIMESTAMPTZ,

  -- Resolution
  resolution TEXT,
  resolved_at TIMESTAMPTZ,
  resolved_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,

  -- Timestamps
  detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Metadata
  metadata JSONB DEFAULT '{}',
  tags TEXT[] DEFAULT '{}'
);

-- Indexes
CREATE INDEX idx_security_incidents_status ON auth.security_incidents(status);
CREATE INDEX idx_security_incidents_severity ON auth.security_incidents(severity);
CREATE INDEX idx_security_incidents_detected_at ON auth.security_incidents(detected_at DESC);
CREATE INDEX idx_security_incidents_assigned_to ON auth.security_incidents(assigned_to);

-- Enable RLS
ALTER TABLE auth.security_incidents ENABLE ROW LEVEL SECURITY;

-- Only security admins can view incidents
CREATE POLICY security_incidents_admin_policy ON auth.security_incidents
  FOR ALL
  USING (
    EXISTS (
      SELECT 1 FROM auth.users
      WHERE id = auth.uid()
      AND role IN ('admin', 'security_admin')
    )
  );

-- ============================================================================
-- Security Analytics and Metrics
-- ============================================================================

-- Store security metrics for dashboards and monitoring
CREATE TABLE IF NOT EXISTS auth.security_metrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Metric details
  metric_name TEXT NOT NULL,
  metric_type TEXT NOT NULL, -- 'counter', 'gauge', 'histogram'
  metric_value NUMERIC NOT NULL,

  -- Dimensions
  user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
  device_id UUID REFERENCES auth.user_devices(id) ON DELETE SET NULL,

  -- Context
  tags JSONB DEFAULT '{}',

  -- Timestamp
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for time-series queries
CREATE INDEX idx_security_metrics_name_time ON auth.security_metrics(metric_name, recorded_at DESC);
CREATE INDEX idx_security_metrics_user_id ON auth.security_metrics(user_id, recorded_at DESC);

-- Partitioned for performance (optional - can be enabled later)
-- CREATE INDEX idx_security_metrics_recorded_at ON auth.security_metrics(recorded_at DESC);

-- Enable RLS
ALTER TABLE auth.security_metrics ENABLE ROW LEVEL SECURITY;

-- Only admins can view metrics
CREATE POLICY security_metrics_admin_policy ON auth.security_metrics
  FOR SELECT
  USING (
    EXISTS (
      SELECT 1 FROM auth.users
      WHERE id = auth.uid()
      AND role IN ('admin', 'security_admin')
    )
  );

-- ============================================================================
-- Password Security
-- ============================================================================

-- Track password history to prevent reuse
CREATE TABLE IF NOT EXISTS auth.password_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,

  -- Password data (hashed)
  password_hash TEXT NOT NULL,

  -- Metadata
  changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  changed_by UUID REFERENCES auth.users(id) ON DELETE SET NULL, -- Admin or self
  reason TEXT, -- 'reset', 'change', 'forced', 'expired'

  -- Context
  ip_address INET,
  user_agent TEXT
);

-- Index for history lookups
CREATE INDEX idx_password_history_user_id ON auth.password_history(user_id, changed_at DESC);

-- Enable RLS
ALTER TABLE auth.password_history ENABLE ROW LEVEL SECURITY;

-- Users can only see their own password history (metadata only, not hashes)
CREATE POLICY password_history_user_policy ON auth.password_history
  FOR SELECT
  USING (user_id = auth.uid());

-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Function: Log a security event
CREATE OR REPLACE FUNCTION auth.log_security_event(
  p_event_type TEXT,
  p_severity TEXT,
  p_description TEXT,
  p_user_id UUID DEFAULT NULL,
  p_details JSONB DEFAULT '{}'
) RETURNS UUID AS $$
DECLARE
  v_event_id UUID;
BEGIN
  INSERT INTO auth.security_events (
    event_type,
    severity,
    description,
    user_id,
    details,
    ip_address,
    user_agent
  ) VALUES (
    p_event_type,
    p_severity,
    p_description,
    COALESCE(p_user_id, auth.uid()),
    p_details,
    inet_client_addr(),
    current_setting('request.headers', true)::json->>'user-agent'
  ) RETURNING id INTO v_event_id;

  RETURN v_event_id;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function: Calculate device risk score
CREATE OR REPLACE FUNCTION auth.calculate_device_risk_score(
  p_device_id UUID
) RETURNS INTEGER AS $$
DECLARE
  v_risk_score INTEGER := 0;
  v_device RECORD;
BEGIN
  SELECT * INTO v_device FROM auth.user_devices WHERE id = p_device_id;

  IF NOT FOUND THEN
    RETURN 0;
  END IF;

  -- New device (first time seen)
  IF v_device.first_seen_at > NOW() - INTERVAL '1 hour' THEN
    v_risk_score := v_risk_score + 20;
  END IF;

  -- Failed login attempts
  IF v_device.failed_login_attempts > 0 THEN
    v_risk_score := v_risk_score + (v_device.failed_login_attempts * 10);
  END IF;

  -- Not trusted
  IF NOT v_device.is_trusted THEN
    v_risk_score := v_risk_score + 10;
  END IF;

  -- Inactive device (not seen in 30 days)
  IF v_device.last_seen_at < NOW() - INTERVAL '30 days' THEN
    v_risk_score := v_risk_score + 15;
  END IF;

  -- Cap at 100
  IF v_risk_score > 100 THEN
    v_risk_score := 100;
  END IF;

  -- Update device risk score
  UPDATE auth.user_devices
  SET risk_score = v_risk_score
  WHERE id = p_device_id;

  RETURN v_risk_score;
END;
$$ LANGUAGE plpgsql;

-- Function: Detect suspicious activity
CREATE OR REPLACE FUNCTION auth.detect_suspicious_activity(
  p_user_id UUID,
  p_event_type TEXT,
  p_ip_address INET DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
  v_suspicious BOOLEAN := false;
  v_recent_count INTEGER;
  v_different_locations INTEGER;
BEGIN
  -- Check for rapid-fire events (rate limiting)
  SELECT COUNT(*) INTO v_recent_count
  FROM auth.security_events
  WHERE user_id = p_user_id
    AND event_type = p_event_type
    AND created_at > NOW() - INTERVAL '5 minutes';

  IF v_recent_count > 5 THEN
    v_suspicious := true;
  END IF;

  -- Check for impossible travel (multiple locations in short time)
  IF p_ip_address IS NOT NULL THEN
    SELECT COUNT(DISTINCT ip_address) INTO v_different_locations
    FROM auth.security_events
    WHERE user_id = p_user_id
      AND created_at > NOW() - INTERVAL '1 hour'
      AND ip_address IS NOT NULL
      AND ip_address != p_ip_address;

    IF v_different_locations > 3 THEN
      v_suspicious := true;
    END IF;
  END IF;

  RETURN v_suspicious;
END;
$$ LANGUAGE plpgsql;

-- Function: Get weak passwords (for scanning)
CREATE OR REPLACE FUNCTION auth.get_weak_passwords()
RETURNS TABLE (
  user_id UUID,
  email TEXT,
  password_age INTERVAL,
  has_mfa BOOLEAN
) AS $$
BEGIN
  RETURN QUERY
  SELECT
    u.id,
    u.email,
    NOW() - COALESCE(ph.changed_at, u.created_at) as password_age,
    EXISTS(SELECT 1 FROM auth.mfa_factors WHERE user_id = u.id AND verified = true) as has_mfa
  FROM auth.users u
  LEFT JOIN LATERAL (
    SELECT changed_at
    FROM auth.password_history
    WHERE user_id = u.id
    ORDER BY changed_at DESC
    LIMIT 1
  ) ph ON true
  WHERE u.encrypted_password IS NOT NULL -- Only email/password users
    AND (
      -- Password older than 90 days
      NOW() - COALESCE(ph.changed_at, u.created_at) > INTERVAL '90 days'
      OR
      -- No MFA enabled
      NOT EXISTS(SELECT 1 FROM auth.mfa_factors WHERE user_id = u.id AND verified = true)
    )
  ORDER BY password_age DESC;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- ============================================================================
-- Triggers
-- ============================================================================

-- Auto-update updated_at on security_incidents
CREATE OR REPLACE FUNCTION auth.update_security_incident_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER security_incidents_updated_at
  BEFORE UPDATE ON auth.security_incidents
  FOR EACH ROW
  EXECUTE FUNCTION auth.update_security_incident_updated_at();

-- Auto-detect suspicious activity on security events
CREATE OR REPLACE FUNCTION auth.auto_detect_suspicious_activity()
RETURNS TRIGGER AS $$
BEGIN
  -- Mark as suspicious if detection function returns true
  NEW.is_suspicious := auth.detect_suspicious_activity(
    NEW.user_id,
    NEW.event_type,
    NEW.ip_address
  );

  -- Set risk score
  IF NEW.is_suspicious THEN
    NEW.risk_score := GREATEST(NEW.risk_score, 50);
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER security_events_auto_detect
  BEFORE INSERT ON auth.security_events
  FOR EACH ROW
  EXECUTE FUNCTION auth.auto_detect_suspicious_activity();

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE auth.webauthn_credentials IS 'WebAuthn/FIDO2 hardware security keys and platform authenticators';
COMMENT ON TABLE auth.user_devices IS 'Tracked user devices for security monitoring';
COMMENT ON TABLE auth.security_events IS 'Security event audit trail';
COMMENT ON TABLE auth.security_incidents IS 'Security incidents and investigations';
COMMENT ON TABLE auth.security_metrics IS 'Security metrics for monitoring and dashboards';
COMMENT ON TABLE auth.password_history IS 'Password change history for preventing reuse';

COMMENT ON FUNCTION auth.log_security_event IS 'Log a security event with automatic context capture';
COMMENT ON FUNCTION auth.calculate_device_risk_score IS 'Calculate risk score for a device based on behavior';
COMMENT ON FUNCTION auth.detect_suspicious_activity IS 'Detect suspicious activity patterns';
COMMENT ON FUNCTION auth.get_weak_passwords IS 'Find users with weak password security (old passwords, no MFA)';
