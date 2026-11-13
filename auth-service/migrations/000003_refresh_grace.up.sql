-- Refresh 轮换宽限：客户端未落盘新 refresh 时，可在短窗内用上一枚 refresh 再换一对。
ALTER TABLE auth_refresh_tokens
    ADD COLUMN IF NOT EXISTS prev_token_hash CHAR(64),
    ADD COLUMN IF NOT EXISTS prev_expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_auth_refresh_prev_hash
    ON auth_refresh_tokens (prev_token_hash)
    WHERE prev_token_hash IS NOT NULL;
