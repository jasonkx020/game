BEGIN;

ALTER TABLE players ADD COLUMN IF NOT EXISTS is_minor BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE players ADD COLUMN IF NOT EXISTS risk_status VARCHAR(16) NOT NULL DEFAULT 'active';

CREATE TABLE IF NOT EXISTS player_devices (
    player_id     BIGINT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    device_id     VARCHAR(128) NOT NULL,
    last_ip       VARCHAR(64) NOT NULL DEFAULT '',
    platform      VARCHAR(32) NOT NULL DEFAULT '',
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (device_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_player_devices_player ON player_devices (player_id);
CREATE INDEX IF NOT EXISTS idx_player_devices_device ON player_devices (device_id);

CREATE TABLE IF NOT EXISTS player_playtime_daily (
    player_id BIGINT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    day       DATE NOT NULL,
    minutes   INT NOT NULL DEFAULT 0,
    PRIMARY KEY (player_id, day)
);

CREATE TABLE IF NOT EXISTS risk_alerts (
    id                   BIGSERIAL PRIMARY KEY,
    type                 VARCHAR(32) NOT NULL,
    status               VARCHAR(16) NOT NULL DEFAULT 'open',
    player_id            BIGINT,
    room_id              UUID,
    round_id             UUID,
    settlement_audit_sn  BIGINT,
    meta                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_by          BIGINT,
    reviewed_at          TIMESTAMPTZ,
    review_note          TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_risk_alerts_status_created ON risk_alerts (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_alerts_player ON risk_alerts (player_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_alerts_type ON risk_alerts (type, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_deposits (
    id              BIGSERIAL PRIMARY KEY,
    agent_admin_id  BIGINT NOT NULL REFERENCES admin_users(id),
    kind            VARCHAR(16) NOT NULL, -- deposit | penalty
    amount_cny      NUMERIC(12,2) NOT NULL,
    balance_after   NUMERIC(12,2) NOT NULL,
    reason          TEXT NOT NULL DEFAULT '',
    actor_admin_id  BIGINT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_deposits_agent ON agent_deposits (agent_admin_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_withdrawals (
    id              BIGSERIAL PRIMARY KEY,
    agent_admin_id  BIGINT NOT NULL REFERENCES admin_users(id),
    amount_cny      NUMERIC(12,2) NOT NULL,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    eligible_at     TIMESTAMPTZ NOT NULL,
    reviewed_by     BIGINT,
    reviewed_at     TIMESTAMPTZ,
    review_note     TEXT NOT NULL DEFAULT '',
    force_approve   BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_agent_withdrawals_agent ON agent_withdrawals (agent_admin_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_withdrawals_status ON agent_withdrawals (status, requested_at DESC);

COMMIT;
