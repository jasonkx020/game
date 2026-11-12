BEGIN;

DROP INDEX IF EXISTS idx_admin_users_auth_account_id;
DROP INDEX IF EXISTS idx_players_auth_account_id;
ALTER TABLE admin_users DROP COLUMN IF EXISTS auth_account_id;
ALTER TABLE players DROP COLUMN IF EXISTS auth_account_id;

COMMIT;
