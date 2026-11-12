BEGIN;

ALTER TABLE players ADD COLUMN role VARCHAR(32) NOT NULL DEFAULT 'player';

INSERT INTO players (id, phone, nickname, role, avatar_url, status, settings, created_at, updated_at, last_login_at)
SELECT id, phone, nickname, role, avatar_url, status, settings, created_at, updated_at, last_login_at
FROM admin_users
ON CONFLICT (phone) DO UPDATE SET
    role = EXCLUDED.role,
    nickname = EXCLUDED.nickname,
    avatar_url = EXCLUDED.avatar_url,
    status = EXCLUDED.status,
    settings = EXCLUDED.settings,
    updated_at = EXCLUDED.updated_at,
    last_login_at = EXCLUDED.last_login_at;

ALTER TABLE club DROP CONSTRAINT IF EXISTS club_agent_id_fkey;
ALTER TABLE club DROP CONSTRAINT IF EXISTS club_owner_admin_id_fkey;

ALTER TABLE club ADD COLUMN owner_user_id BIGINT;
UPDATE club SET owner_user_id = owner_admin_id;
ALTER TABLE club DROP COLUMN owner_admin_id;
ALTER TABLE club ALTER COLUMN owner_user_id SET NOT NULL;
ALTER TABLE club ADD CONSTRAINT club_owner_user_id_fkey
    FOREIGN KEY (owner_user_id) REFERENCES players(id);

DROP INDEX IF EXISTS idx_club_owner;
CREATE INDEX idx_club_owner ON club (owner_user_id);

ALTER TABLE players RENAME TO users;
ALTER INDEX idx_players_phone RENAME TO idx_users_phone;

DROP TABLE admin_users;

COMMIT;
