# 技术架构文档索引

> **Go 平台（Gin + OpenAPI）** + **Pitaya 游戏实时层** + **Cocos Creator 3**。  
> 架构决策：[adr/README.md](adr/README.md)

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
├── audit-action-log.md      # 有序日志 DDL
├── replay.md                # 战绩回放 API
├── openapi/                 # HTTP OpenAPI 3.0
└── proto/
    ├── pitaya/              # Pitaya route proto（当前）
    └── DEPRECATED.md        # 旧 Envelope 方案
```

---

## 新游戏接入

1. `internal/game/{id}/` GameEngine
2. `internal/pitaya/handlers/{id}/` Handler
3. `proto/pitaya/{id}.proto`
4. `docs/games/{id}/ops-hooks.md`
5. Cocos `games/{id}/` + PitayaClient Push 订阅 + GameEvent 映射

详见 [adr/002-pluggable-game-constraints.md](adr/002-pluggable-game-constraints.md)。

---

## 相关文档

| 文档 | 内容 |
| :--- | :--- |
| [../../README.md](../../README.md) | **本地启动**（含 `client/` Cocos SDK） |
| [../../client/README.md](../../client/README.md) | 客户端工程与场景挂载 |
| [运营手册.md](../../运营手册.md) | 总索引 |
| [games/README.md](../games/README.md) | 游戏接入 |
