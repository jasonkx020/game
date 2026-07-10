CREATE TABLE IF NOT EXISTS companion_persona (
    persona_id    VARCHAR(32) PRIMARY KEY,
    name          VARCHAR(64) NOT NULL,
    avatar_url    VARCHAR(512) NOT NULL DEFAULT '',
    system_prompt TEXT NOT NULL,
    voice_style   VARCHAR(32) NOT NULL DEFAULT 'warm',
    enabled       BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS companion_session (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    persona_id    VARCHAR(32) NOT NULL REFERENCES companion_persona(persona_id),
    context_json  JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_companion_session_user ON companion_session (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS companion_message (
    id            BIGSERIAL PRIMARY KEY,
    session_id    BIGINT NOT NULL REFERENCES companion_session(id) ON DELETE CASCADE,
    role          VARCHAR(16) NOT NULL,
    content       TEXT NOT NULL DEFAULT '',
    tool_name     VARCHAR(64) NOT NULL DEFAULT '',
    audit_sn      BIGINT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_companion_message_session ON companion_message (session_id, id);

CREATE TABLE IF NOT EXISTS game_knowledge (
    id            BIGSERIAL PRIMARY KEY,
    game_id       VARCHAR(32) NOT NULL REFERENCES game_catalog(game_id) ON DELETE CASCADE,
    chunk_title   VARCHAR(128) NOT NULL DEFAULT '',
    content       TEXT NOT NULL,
    source        VARCHAR(64) NOT NULL DEFAULT 'manual',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_game_knowledge_game ON game_knowledge (game_id);

CREATE TABLE IF NOT EXISTS game_standalone (
    game_id            VARCHAR(32) PRIMARY KEY REFERENCES game_catalog(game_id) ON DELETE CASCADE,
    deep_link_scheme   VARCHAR(64) NOT NULL DEFAULT '',
    min_host_version   VARCHAR(32) NOT NULL DEFAULT '1.0.0',
    min_lobby_version  VARCHAR(32) NOT NULL DEFAULT '',
    store_listing      VARCHAR(512) NOT NULL DEFAULT '',
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO companion_persona (persona_id, name, avatar_url, system_prompt, voice_style)
VALUES (
    'default',
    '小龟',
    '',
    '你是「小龟」，打乌龟游戏平台的 AI 陪玩伴侣。性格热情、幽默，擅长推荐游戏、讲解规则、陪聊解压。不要讨论提现、赌博、作弊。房卡与游戏金币不可提现。回答简洁口语化，每次 2-4 句为宜。',
    'warm'
) ON CONFLICT (persona_id) DO NOTHING;

INSERT INTO game_knowledge (game_id, chunk_title, content, source) VALUES
('dawugui', '基本规则', '打乌龟是 3-5 人扑克跑牌游戏。目标是最先出完手牌。支持出牌、过牌。报单后需小心被包牌。房卡场消耗平台房卡开房。', 'ops-hooks'),
('dawugui', '陪玩提示', '新手建议先熟悉牌型大小；报单时要谨慎；可观察对手出牌习惯。', 'companion'),
('liuzichong', '基本规则', '六子冲是 2 人棋类游戏，在 16 格棋盘上轮流落子，先连成六子者胜。', 'ops-hooks'),
('liuzichong', '陪玩提示', '占据中心区域通常更有利；注意阻挡对手连线。', 'companion');

INSERT INTO game_standalone (game_id, deep_link_scheme, min_host_version, store_listing)
VALUES
    ('dawugui', 'dawugui', '1.0.0', ''),
    ('liuzichong', 'liuzichong', '1.0.0', '')
ON CONFLICT (game_id) DO NOTHING;
