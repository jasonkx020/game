# ADR-008：独立发布与 GameHostSDK

| 项 | 值 |
| :--- | :--- |
| **Status** | Accepted |
| **Date** | 2026-07 |
| **Depends on** | ADR-006, ADR-007 |

---

## 背景

游戏需支持：**从大厅进入** 与 **独立 App 发布** 双入口，统一账号与 Pitaya 协议。

---

## 决策

### GameHostSDK

客户端 `platform/host/GameHost.ts` 提供统一启动：

| mode | 行为 |
| :--- | :--- |
| `lobby` | 大厅已开房或带 room 上下文进入 |
| `standalone` | 独立 App 自行 login + createRoom + Room |

### 数据

`game_standalone` 表：`deep_link_scheme`、`min_host_version`、`store_listing`

### 产物

- **Lobby App**：不含游戏资源，Companion + GameShelf
- **Game App**：单游戏 Bundle + StandaloneEntry

---

## 相关

- [game-host-sdk.md](../game-host-sdk.md)
- [games-lobby/README.md](../../../games-lobby/README.md)
