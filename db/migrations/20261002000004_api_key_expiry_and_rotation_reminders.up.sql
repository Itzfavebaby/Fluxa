-- Add API key expiry policy and rotation tracking
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS rotation_reminder_days INT DEFAULT 7;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS last_rotation_reminded_at TIMESTAMPTZ;

-- Indices for efficient expiry checks and rotation reminders
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys (expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_api_keys_active_expiry ON api_keys (tenant_id, mode, expires_at) WHERE revoked_at IS NULL;
