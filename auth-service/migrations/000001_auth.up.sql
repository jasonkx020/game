CREATE TABLE IF NOT EXISTS auth_accounts (
    id              BIGSERIAL PRIMARY KEY,
    principal_type  VARCHAR(16) NOT NULL,
    phone           VARCHAR(32) NOT NULL,
    role            VARCHAR(32) NOT NULL DEFAULT 'player',
    nickname        VARCHAR(64) NOT NULL DEFAULT '',
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at   TIMESTAMPTZ,
    UNIQUE (phone)
);

CREATE INDEX IF NOT EXISTS idx_auth_accounts_type ON auth_accounts (principal_type);

CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
    account_id      BIGINT PRIMARY KEY REFERENCES auth_accounts(id) ON DELETE CASCADE,
    token_hash      CHAR(64) NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_refresh_tokens_hash ON auth_refresh_tokens (token_hash);
