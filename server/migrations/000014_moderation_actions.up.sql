CREATE TABLE IF NOT EXISTS moderation_actions (
    id              BIGSERIAL PRIMARY KEY,
    actor_admin_id  BIGINT NOT NULL,
    target_type     VARCHAR(16) NOT NULL,
    target_id       BIGINT NOT NULL,
    action          VARCHAR(32) NOT NULL,
    reason          TEXT NOT NULL DEFAULT '',
    meta            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_moderation_actions_target
    ON moderation_actions (target_type, target_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_moderation_actions_actor
    ON moderation_actions (actor_admin_id, created_at DESC);
