BEGIN;

DROP INDEX IF EXISTS idx_auth_oauth_account;
DROP INDEX IF EXISTS idx_auth_oauth_union;
DROP TABLE IF EXISTS auth_oauth_identities;

DROP INDEX IF EXISTS uq_auth_accounts_email;
DROP INDEX IF EXISTS uq_auth_accounts_phone;

ALTER TABLE auth_accounts DROP COLUMN IF EXISTS realname_verified_at;
ALTER TABLE auth_accounts DROP COLUMN IF EXISTS is_minor;
ALTER TABLE auth_accounts DROP COLUMN IF EXISTS birth_date;
ALTER TABLE auth_accounts DROP COLUMN IF EXISTS id_card_masked;
ALTER TABLE auth_accounts DROP COLUMN IF EXISTS id_card_hash;
ALTER TABLE auth_accounts DROP COLUMN IF EXISTS real_name;
ALTER TABLE auth_accounts DROP COLUMN IF EXISTS realname_status;
ALTER TABLE auth_accounts DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE auth_accounts DROP COLUMN IF EXISTS email;

UPDATE auth_accounts SET phone = COALESCE(phone, '') WHERE phone IS NULL;
ALTER TABLE auth_accounts ALTER COLUMN phone SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS auth_accounts_phone_key ON auth_accounts (phone);

COMMIT;
