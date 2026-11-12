BEGIN;

ALTER TABLE players ADD COLUMN IF NOT EXISTS auth_account_id BIGINT UNIQUE;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS auth_account_id BIGINT UNIQUE;

CREATE INDEX IF NOT EXISTS idx_players_auth_account_id ON players (auth_account_id);
CREATE INDEX IF NOT EXISTS idx_admin_users_auth_account_id ON admin_users (auth_account_id);

COMMIT;
