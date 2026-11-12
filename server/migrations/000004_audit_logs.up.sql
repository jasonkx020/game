CREATE TABLE IF NOT EXISTS game_round (
    round_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id         UUID NOT NULL REFERENCES room(room_id),
    round_no        INT NOT NULL,
    game_id         VARCHAR(32) NOT NULL,
    status          VARCHAR(16) NOT NULL,
    config_snapshot JSONB NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at        TIMESTAMPTZ,
    winner_user_ids BIGINT[],
    settlement_audit_sn BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (room_id, round_no)
);

CREATE INDEX idx_game_round_room ON game_round (room_id, round_no DESC);

CREATE TABLE IF NOT EXISTS game_action_log (
    id              BIGSERIAL PRIMARY KEY,
    room_id         UUID NOT NULL,
    round_id        UUID NOT NULL REFERENCES game_round(round_id),
    action_seq      INT NOT NULL,
    audit_sn        BIGINT NOT NULL,
    event_type      VARCHAR(32) NOT NULL,
    actor_user_id   BIGINT,
    seat            SMALLINT,
    payload         BYTEA NOT NULL,
    push_route      VARCHAR(64) NOT NULL,
    server_ts       TIMESTAMPTZ NOT NULL DEFAULT now(),
    c2s_route       VARCHAR(64),
    c2s_request_id  UUID,
    UNIQUE (round_id, action_seq),
    UNIQUE (audit_sn)
);

CREATE INDEX idx_action_log_round_seq ON game_action_log (round_id, action_seq);

CREATE TABLE IF NOT EXISTS room_event_log (
    id              BIGSERIAL PRIMARY KEY,
    room_id         UUID NOT NULL,
    room_seq        INT NOT NULL,
    event_type      VARCHAR(32) NOT NULL,
    user_id         BIGINT,
    audit_sn        BIGINT NOT NULL UNIQUE,
    payload         JSONB,
    server_ts       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (room_id, room_seq)
);

CREATE INDEX idx_room_event_room ON room_event_log (room_id, room_seq);

CREATE TABLE IF NOT EXISTS settlement_record (
    id              BIGSERIAL PRIMARY KEY,
    room_id         UUID NOT NULL REFERENCES room(room_id),
    round_id        UUID REFERENCES game_round(round_id),
    game_id         VARCHAR(32) NOT NULL,
    audit_sn        BIGINT UNIQUE,
    payload         JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
