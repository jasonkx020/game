# 架构决策记录（ADR）

> 技术层 — 服务端与游戏框架选型及约束。索引见 [tech/README.md](../README.md)。

---

## 命名规范

- 文件：`NNN-short-title.md`（三位序号 + 短标题）
- 状态：`Accepted` | `Superseded` | `Deprecated`

---

## ADR 列表

| 序号 | 文档 | 状态 | 说明 |
| :--- | :--- | :--- | :--- |
| 001 | [001-server-stack.md](001-server-stack.md) | Accepted | Gin 平台 HTTP + Pitaya Standalone 游戏实时层 |
| 002 | [002-pluggable-game-constraints.md](002-pluggable-game-constraints.md) | Accepted | 平台/游戏解耦、增游戏白名单、proto route 规范 |
| 004 | [004-pitaya-game-framework.md](004-pitaya-game-framework.md) | Accepted | Pitaya Handler/Groups/Engine 分工与路由表 |
| 005 | [005-ordered-action-log-replay.md](005-ordered-action-log-replay.md) | Accepted | 有序动作日志、action_seq、玩家回放 |
| 006 | [006-game-lobby-dynamic-bundle.md](006-game-lobby-dynamic-bundle.md) | Accepted | 游戏大厅 API 驱动、Remote Bundle、用户偏好 |
| 007 | [007-companion-llm.md](007-companion-llm.md) | Accepted | 智能伴侣 LLM、Tool Calling、SSE |
| 008 | [008-game-host-sdk.md](008-game-host-sdk.md) | Accepted | 独立发布与 GameHostSDK 双入口 |

**已 superseded（未单独成文）：** 自研 WS Gateway + RoomActor + Envelope 主帧方案（见 [proto/DEPRECATED.md](../proto/DEPRECATED.md)）。

---

## 阅读顺序

1. ADR-001 — 整体技术栈
2. ADR-004 — Pitaya 游戏框架（增游戏必读）
3. ADR-005 — 有序日志与回放
4. ADR-002 — 包依赖与接入约束
