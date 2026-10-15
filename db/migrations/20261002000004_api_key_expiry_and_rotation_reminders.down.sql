-- Revert API key expiry policy and rotation tracking
DROP INDEX IF EXISTS idx_api_keys_active_expiry;
DROP INDEX IF EXISTS idx_api_keys_expires_at;

ALTER TABLE api_keys DROP COLUMN IF EXISTS last_rotation_reminded_at;
ALTER TABLE api_keys DROP COLUMN IF EXISTS rotation_reminder_days;
ALTER TABLE api_keys DROP COLUMN IF EXISTS expires_at;
