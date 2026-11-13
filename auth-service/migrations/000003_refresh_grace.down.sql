DROP INDEX IF EXISTS idx_auth_refresh_prev_hash;
ALTER TABLE auth_refresh_tokens
    DROP COLUMN IF EXISTS prev_expires_at,
    DROP COLUMN IF EXISTS prev_token_hash;
