CREATE TABLE IF NOT EXISTS wallet_room_card (
    user_id     BIGINT PRIMARY KEY REFERENCES users(id),
    balance     BIGINT NOT NULL DEFAULT 10 CHECK (balance >= 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wallet_game_coin (
    user_id     BIGINT NOT NULL REFERENCES users(id),
    game_id     VARCHAR(32) NOT NULL,
    balance     BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, game_id)
);

CREATE TABLE IF NOT EXISTS wallet_ledger (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id),
    wallet_type VARCHAR(16) NOT NULL,
    game_id     VARCHAR(32),
    delta       BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    reason      VARCHAR(64) NOT NULL,
    ref_id      UUID,
    audit_sn    BIGINT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_wallet_ledger_user ON wallet_ledger (user_id, created_at DESC);
