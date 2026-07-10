# ADR-007：智能伴侣（LLM）架构

| 项 | 值 |
| :--- | :--- |
| **Status** | Accepted |
| **Date** | 2026-07 |
| **Depends on** | ADR-006 |

---

## 背景

游戏大厅升级为「陪玩陪聊」主阵地，需接入 LLM 实现对话、推荐、规则讲解与 Tool 开房。

---

## 决策

### 服务位置

- MVP：`internal/platform/companion` 并入 `platform-api`
- LLM：OpenAI 兼容 HTTP（`LLM_BASE_URL` + `LLM_API_KEY`）
- 无 API Key 时：启发式回复 + Tool 直连（开发可用）

### 数据

- `companion_persona` / `companion_session` / `companion_message`
- `game_knowledge`：规则 RAG 文本（P2 embedding）
- 全量消息落库 + `audit_sn`

### API

| 路径 | 说明 |
| :--- | :--- |
| POST `/v1/companion/sessions` | 创建会话 |
| POST `/v1/companion/sessions/{id}/chat` | SSE 流式 |
| GET `/v1/lobby/recommendations` | 推荐游戏 |
| PUT `/v1/user/settings` | 伴侣偏好 |

### Tool Calling

`list_games`、`create_room`、`explain_rules`、`recommend_games` — 服务端执行，校验 JWT user_id。

### 边界

- `internal/platform` **不** import 游戏引擎（ADR-002）
- 伴侣不得读取非公开手牌；仅 Push 公开信息 + 规则知识

---

## 相关

- [008-game-host-sdk.md](008-game-host-sdk.md)
- [006-game-lobby-dynamic-bundle.md](006-game-lobby-dynamic-bundle.md)
