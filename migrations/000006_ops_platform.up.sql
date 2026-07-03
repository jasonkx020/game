ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(32) NOT NULL DEFAULT 'player';

CREATE TABLE IF NOT EXISTS game_catalog (
    game_id      VARCHAR(32) PRIMARY KEY,
    name         VARCHAR(64) NOT NULL,
    min_players  INT NOT NULL,
    max_players  INT NOT NULL,
    enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game_ops_config (
    game_id    VARCHAR(32) PRIMARY KEY REFERENCES game_catalog(game_id),
    config     JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS recharge_order (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id),
    product_id   VARCHAR(32) NOT NULL,
    amount_cny   INT NOT NULL,
    cards        INT NOT NULL,
    audit_sn     BIGINT NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_recharge_user ON recharge_order (user_id, created_at DESC);
