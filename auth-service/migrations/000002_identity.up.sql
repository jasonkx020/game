BEGIN;

ALTER TABLE auth_accounts ALTER COLUMN phone DROP NOT NULL;

ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS realname_status VARCHAR(16) NOT NULL DEFAULT 'none';
ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS real_name VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS id_card_hash CHAR(64);
ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS id_card_masked VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS birth_date DATE;
ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS is_minor BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE auth_accounts ADD COLUMN IF NOT EXISTS realname_verified_at TIMESTAMPTZ;

-- Allow multiple NULLs while keeping unique non-null phones/emails.
ALTER TABLE auth_accounts DROP CONSTRAINT IF EXISTS auth_accounts_phone_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_accounts_phone ON auth_accounts (phone) WHERE phone IS NOT NULL AND phone <> '';
CREATE UNIQUE INDEX IF NOT EXISTS uq_auth_accounts_email ON auth_accounts (email) WHERE email IS NOT NULL AND email <> '';

CREATE TABLE IF NOT EXISTS auth_oauth_identities (
    id           BIGSERIAL PRIMARY KEY,
    account_id   BIGINT NOT NULL REFERENCES auth_accounts(id) ON DELETE CASCADE,
    provider     VARCHAR(32) NOT NULL,
    platform     VARCHAR(32) NOT NULL,
    open_id      VARCHAR(128) NOT NULL,
    union_id     VARCHAR(128),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, platform, open_id)
);

CREATE INDEX IF NOT EXISTS idx_auth_oauth_union ON auth_oauth_identities (provider, union_id)
    WHERE union_id IS NOT NULL AND union_id <> '';
CREATE INDEX IF NOT EXISTS idx_auth_oauth_account ON auth_oauth_identities (account_id);

COMMIT;
