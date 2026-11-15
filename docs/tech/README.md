# 技术架构文档索引

> **Go 平台（Gin + OpenAPI）** + **Pitaya 游戏实时层** + **Cocos Creator 3**。  
> 架构决策：[adr/README.md](adr/README.md)  
> **在线阅读（推荐）**：运营后台 [`http://localhost:5173/docs`](http://localhost:5173/docs)（免登录，协议与接入文档中心）

---

## 阅读顺序

| 顺序 | 文档 | 适合读者 |
| :--- | :--- | :--- |
| 1 | [adr/001-server-stack.md](adr/001-server-stack.md) | 技术栈总览 |
| 2 | [platform-architecture.md](platform-architecture.md) | 后端、架构 |
| 3 | [adr/004-pitaya-game-framework.md](adr/004-pitaya-game-framework.md) | Pitaya + 增游戏 |
| 4 | [adr/005-ordered-action-log-replay.md](adr/005-ordered-action-log-replay.md) | 有序日志与回放 |
| 5 | [audit-action-log.md](audit-action-log.md) | 日志 DDL |
| 6 | [replay.md](replay.md) | 战绩回放 API |
| 7 | [protocol.md](protocol.md) | 全栈协议 |
| 8 | [pitaya-client.md](pitaya-client.md) | Cocos PitayaClient |
| 9 | [game-framework.md](game-framework.md) | 游戏逻辑 |
| 10 | [client-architecture.md](client-architecture.md) | 客户端 |
| 11 | [adr/006-game-lobby-dynamic-bundle.md](adr/006-game-lobby-dynamic-bundle.md) | 游戏大厅与动态 Bundle |
| 12 | [auth-service.md](auth-service.md) / [oauth.md](oauth.md) | 独立 Auth、微信 OAuth 与表归属 |
| 13 | [gateway.md](gateway.md) | nginx 单端口分流（auth / game / admin） |

---

## 技术栈摘要

| 层级 | 选型 |
| :--- | :--- |
| HTTP | **Gin** + OpenAPI 3.0 + HMAC |
| 游戏实时 | **Pitaya v2 Standalone** + Protobuf |
| 持久化 | PostgreSQL 16+、Redis |
| 客户端 | Cocos Creator 3 + PitayaClient TS |
| 审计 | audit_sn + action_seq（EventMeta） |
| 回放 | HTTP replay + Pitaya sync |

---

## 目录结构

```
docs/tech/
├── README.md
├── adr/                     # 架构决策
├── platform-architecture.md
├── game-framework.md
├── client-architecture.md
├── protocol.md
├── pitaya-client.md
├── auth-service.md          # 独立 Auth
├── oauth.md                 # 微信 OAuth 表与约束
├── audit-action-log.md      # 有序日志 DDL
├── replay.md                # 战绩回放 API
├── openapi/                 # HTTP OpenAPI 3.0
└── proto/
    ├── pitaya/              # Pitaya route proto（当前）
    └── DEPRECATED.md        # 旧 Envelope 方案
```

---

## 新游戏接入

1. `server/internal/game/{id}/` GameEngine
2. `server/internal/pitaya/handlers/{id}/` Handler
3. `server/proto/pitaya/{id}.proto`
4. `docs/games/{id}/ops-hooks.md`
5. Cocos `client/assets/games/{id}/` + PitayaClient Push 订阅 + GameEvent 映射

详见 [adr/002-pluggable-game-constraints.md](adr/002-pluggable-game-constraints.md)。

---

## 相关文档

| 文档 | 内容 |
| :--- | :--- |
| [../../README.md](../../README.md) | **本地启动**（含 `client/` Cocos SDK） |
| [../../client/README.md](../../client/README.md) | 客户端工程与场景挂载 |
| [auth-service.md](auth-service.md) | 独立 Auth（sign-in / 实名 / Token） |
| [oauth.md](oauth.md) | 微信 OAuth、涉及表、AI 修改约束 |
| [games-lobby-migration.md](games-lobby-migration.md) | 独立大厅 App 迁出指南 |
| [运营手册.md](../../运营手册.md) | 总索引 |
| [games/README.md](../games/README.md) | 游戏接入 |
