-- QKD Production Database Initialization Script
-- Creates tables for sessions, keys, and audit logs

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Set timezone
SET timezone = 'UTC';

-- Create sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID UNIQUE NOT NULL,
    alice_id VARCHAR(255) NOT NULL,
    bob_id VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    key_length INTEGER NOT NULL,
    backend VARCHAR(100) NOT NULL,
    qber DOUBLE PRECISION,
    raw_key_length INTEGER,
    final_key_length INTEGER,
    is_secure BOOLEAN DEFAULT false,
    message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,

    CONSTRAINT valid_status CHECK (status IN (
        'waiting_for_bob', 'active', 'completed', 'failed', 'expired'
    )),
    CONSTRAINT valid_backend CHECK (backend IN (
        'simulator', 'qiskit', 'braket'
    )),
    CONSTRAINT positive_key_length CHECK (key_length > 0),
    CONSTRAINT valid_qber CHECK (qber IS NULL OR (qber >= 0 AND qber <= 1))
);

-- Create indexes on sessions
CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_sessions_alice_id ON sessions(alice_id);
CREATE INDEX IF NOT EXISTS idx_sessions_bob_id ON sessions(bob_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Create keys table (encrypted storage)
CREATE TABLE IF NOT EXISTS keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_id UUID UNIQUE NOT NULL,
    session_id UUID NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
    key_material BYTEA NOT NULL, -- Encrypted key material
    key_length INTEGER NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT positive_key_length_keys CHECK (key_length > 0)
);

-- Create indexes on keys
CREATE INDEX IF NOT EXISTS idx_keys_key_id ON keys(key_id);
CREATE INDEX IF NOT EXISTS idx_keys_session_id ON keys(session_id);
CREATE INDEX IF NOT EXISTS idx_keys_is_active ON keys(is_active);
CREATE INDEX IF NOT EXISTS idx_keys_created_at ON keys(created_at);

-- Create audit log table
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    user_id VARCHAR(255),
    session_id UUID,
    key_id UUID,
    ip_address INET,
    user_agent TEXT,
    details JSONB,
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on audit logs
CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_session_id ON audit_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Create metrics table for Prometheus
CREATE TABLE IF NOT EXISTS metrics (
    id BIGSERIAL PRIMARY KEY,
    metric_name VARCHAR(255) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    labels JSONB,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on metrics
CREATE INDEX IF NOT EXISTS idx_metrics_metric_name ON metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for sessions table
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
CREATE TRIGGER update_sessions_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create function to cleanup expired sessions
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    WITH deleted AS (
        DELETE FROM sessions
        WHERE expires_at < CURRENT_TIMESTAMP
        AND status != 'completed'
        RETURNING *
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to get session statistics
CREATE OR REPLACE FUNCTION get_session_statistics()
RETURNS TABLE (
    total_sessions BIGINT,
    active_sessions BIGINT,
    completed_sessions BIGINT,
    failed_sessions BIGINT,
    average_qber DOUBLE PRECISION,
    average_key_length DOUBLE PRECISION
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT AS total_sessions,
        COUNT(*) FILTER (WHERE status = 'active')::BIGINT AS active_sessions,
        COUNT(*) FILTER (WHERE status = 'completed')::BIGINT AS completed_sessions,
        COUNT(*) FILTER (WHERE status = 'failed')::BIGINT AS failed_sessions,
        AVG(qber) AS average_qber,
        AVG(final_key_length) AS average_key_length
    FROM sessions;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (adjust as needed for your security requirements)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO qkd_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO qkd_user;

-- Insert sample data for testing (optional - remove in production)
-- INSERT INTO sessions (session_id, alice_id, status, key_length, backend, expires_at)
-- VALUES (
--     uuid_generate_v4(),
--     'alice@example.com',
--     'waiting_for_bob',
--     256,
--     'simulator',
--     CURRENT_TIMESTAMP + INTERVAL '1 hour'
-- );

-- Add comments for documentation
COMMENT ON TABLE sessions IS 'Stores QKD session information';
COMMENT ON TABLE keys IS 'Stores encrypted quantum keys';
COMMENT ON TABLE audit_logs IS 'Audit trail for all QKD operations';
COMMENT ON TABLE metrics IS 'Performance and operational metrics';

-- Create view for active sessions
CREATE OR REPLACE VIEW active_sessions AS
SELECT
    session_id,
    alice_id,
    bob_id,
    status,
    key_length,
    backend,
    created_at,
    expires_at,
    EXTRACT(EPOCH FROM (expires_at - CURRENT_TIMESTAMP)) AS seconds_until_expiry
FROM sessions
WHERE status IN ('waiting_for_bob', 'active')
AND expires_at > CURRENT_TIMESTAMP
ORDER BY created_at DESC;

COMMENT ON VIEW active_sessions IS 'View of currently active QKD sessions';

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'QKD database schema initialized successfully';
END
$$;
