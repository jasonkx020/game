ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'active';
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS settings JSONB NOT NULL DEFAULT '{}';

ALTER TABLE game_catalog ADD COLUMN IF NOT EXISTS icon_url VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE game_catalog ADD COLUMN IF NOT EXISTS sort_order INT NOT NULL DEFAULT 0;
ALTER TABLE game_catalog ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS user_game_prefs (
    user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id        VARCHAR(32) NOT NULL REFERENCES game_catalog(game_id) ON DELETE CASCADE,
    visible        BOOLEAN NOT NULL DEFAULT true,
    pinned         BOOLEAN NOT NULL DEFAULT false,
    sort_order     INT NOT NULL DEFAULT 0,
    last_played_at TIMESTAMPTZ,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, game_id)
);

CREATE INDEX IF NOT EXISTS idx_user_game_prefs_user ON user_game_prefs (user_id);

CREATE TABLE IF NOT EXISTS game_client_bundle (
    game_id            VARCHAR(32) PRIMARY KEY REFERENCES game_catalog(game_id) ON DELETE CASCADE,
    bundle_version     VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    bundle_url         VARCHAR(512) NOT NULL DEFAULT '',
    bundle_size_bytes  BIGINT NOT NULL DEFAULT 0,
    bundle_sha256      VARCHAR(64) NOT NULL DEFAULT '',
    entry_scene        VARCHAR(64) NOT NULL DEFAULT '',
    min_host_version   VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

UPDATE game_catalog SET sort_order = 10, description = '经典扑克跑牌' WHERE game_id = 'dawugui';
UPDATE game_catalog SET sort_order = 20, description = '二人棋类对弈' WHERE game_id = 'liuzichong';

INSERT INTO game_client_bundle (game_id, bundle_version, bundle_url, bundle_size_bytes, entry_scene)
VALUES
    ('dawugui', '1.0.0', 'http://localhost:8787/bundles/dawugui', 0, ''),
    ('liuzichong', '1.0.0', 'http://localhost:8787/bundles/liuzichong', 0, 'Liuzichong')
ON CONFLICT (game_id) DO UPDATE SET
    bundle_version = EXCLUDED.bundle_version,
    bundle_url = EXCLUDED.bundle_url,
    entry_scene = EXCLUDED.entry_scene,
    updated_at = now();
