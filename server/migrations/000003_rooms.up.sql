CREATE TABLE IF NOT EXISTS room (
    room_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        BIGINT NOT NULL REFERENCES users(id),
    game_id         VARCHAR(32) NOT NULL,
    room_mode       VARCHAR(16) NOT NULL DEFAULT 'room_card',
    status          VARCHAR(16) NOT NULL DEFAULT 'waiting',
    player_count    INT NOT NULL DEFAULT 4,
    config          JSONB NOT NULL DEFAULT '{}',
    ws_url          VARCHAR(256) NOT NULL DEFAULT '',
    idempotency_key UUID UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_room_owner ON room (owner_id, created_at DESC);
