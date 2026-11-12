CREATE TABLE IF NOT EXISTS club (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(128) NOT NULL,
    owner_user_id   BIGINT NOT NULL REFERENCES users(id),
    agent_id        BIGINT,
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_club_owner ON club (owner_user_id);

CREATE TABLE IF NOT EXISTS club_member (
    club_id     BIGINT NOT NULL REFERENCES club(id),
    user_id     BIGINT NOT NULL REFERENCES users(id),
    role        VARCHAR(16) NOT NULL DEFAULT 'member',
    status      VARCHAR(16) NOT NULL DEFAULT 'active',
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (club_id, user_id)
);

CREATE INDEX idx_club_member_user ON club_member (user_id);

CREATE TABLE IF NOT EXISTS club_room_card_pool (
    club_id     BIGINT PRIMARY KEY REFERENCES club(id),
    balance     BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE room ADD COLUMN IF NOT EXISTS club_id BIGINT REFERENCES club(id);
