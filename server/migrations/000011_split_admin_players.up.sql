BEGIN;

CREATE TABLE admin_users (
    id            BIGSERIAL PRIMARY KEY,
    phone         VARCHAR(20) NOT NULL UNIQUE,
    nickname      VARCHAR(64) NOT NULL DEFAULT '',
    role          VARCHAR(32) NOT NULL CHECK (role IN ('platform_admin', 'club_admin', 'agent')),
    avatar_url    VARCHAR(512) NOT NULL DEFAULT '',
    status        VARCHAR(16) NOT NULL DEFAULT 'active',
    settings      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ
);

CREATE INDEX idx_admin_users_phone ON admin_users (phone);

INSERT INTO admin_users (id, phone, nickname, role, avatar_url, status, settings, created_at, updated_at, last_login_at)
SELECT id, phone, nickname, role, avatar_url, status, settings, created_at, updated_at, last_login_at
FROM users
WHERE role IN ('platform_admin', 'club_admin', 'agent');

SELECT setval(
    pg_get_serial_sequence('admin_users', 'id'),
    COALESCE((SELECT MAX(id) FROM admin_users), 1)
);

DELETE FROM wallet_ledger WHERE user_id IN (SELECT id FROM admin_users);
DELETE FROM wallet_game_coin WHERE user_id IN (SELECT id FROM admin_users);
DELETE FROM wallet_room_card WHERE user_id IN (SELECT id FROM admin_users);
DELETE FROM user_game_prefs WHERE user_id IN (SELECT id FROM admin_users);
DELETE FROM companion_session WHERE user_id IN (SELECT id FROM admin_users);
DELETE FROM recharge_order WHERE user_id IN (SELECT id FROM admin_users);
DELETE FROM room WHERE owner_id IN (SELECT id FROM admin_users);
DELETE FROM club_member WHERE user_id IN (SELECT id FROM admin_users);

ALTER TABLE club ADD COLUMN owner_admin_id BIGINT;

UPDATE club SET owner_admin_id = owner_user_id
WHERE owner_user_id IN (SELECT id FROM admin_users);

DELETE FROM club WHERE owner_admin_id IS NULL;

ALTER TABLE club DROP COLUMN owner_user_id;
ALTER TABLE club ALTER COLUMN owner_admin_id SET NOT NULL;
ALTER TABLE club ADD CONSTRAINT club_owner_admin_id_fkey
    FOREIGN KEY (owner_admin_id) REFERENCES admin_users(id);

DROP INDEX IF EXISTS idx_club_owner;
CREATE INDEX idx_club_owner ON club (owner_admin_id);

ALTER TABLE club ADD CONSTRAINT club_agent_id_fkey
    FOREIGN KEY (agent_id) REFERENCES admin_users(id);

DELETE FROM users WHERE role IN ('platform_admin', 'club_admin', 'agent');

ALTER TABLE users RENAME TO players;
ALTER INDEX idx_users_phone RENAME TO idx_players_phone;
ALTER TABLE players DROP COLUMN role;

INSERT INTO players (phone, nickname)
VALUES ('13800000001', '测试玩家')
ON CONFLICT (phone) DO NOTHING;

INSERT INTO wallet_room_card (user_id, balance)
SELECT id, 100 FROM players WHERE phone = '13800000001'
ON CONFLICT (user_id) DO UPDATE SET balance = GREATEST(wallet_room_card.balance, 100);

INSERT INTO wallet_game_coin (user_id, game_id, balance)
SELECT id, 'dawugui', 0 FROM players WHERE phone = '13800000001'
ON CONFLICT DO NOTHING;

COMMIT;
