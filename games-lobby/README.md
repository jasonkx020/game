# games-lobby 独立大厅工程（迁移指南）

> Phase 1 目标：将平台壳从 GameS 单仓拆分为独立可发布 App。

## 迁出范围

从 [client/](../client/) 迁出：

```
assets/platform/sdk/
assets/platform/lobby/
assets/platform/companion/
assets/platform/host/
assets/platform/generated/
assets/scripts/scenes/Launch|Lobby|Room
```

**保留在 GameS**：

```
client/assets/games/          # 各游戏 Remote Bundle
internal/                     # 平台后端 + Pitaya
migrations/
```

## 发布产物

| 产物 | 包名 | 说明 |
| :--- | :--- | :--- |
| 游戏大厅 App | `com.games.lobby` | 含 Companion + GameShelf + GameHostSDK |
| 打乌龟独立 App | `com.games.dawugui` | 见 `StandaloneEntry.ts` |

## 同步依赖

- PlatformSDK：git submodule 或 npm `@games/platform-sdk`
- GameHostSDK 契约：[docs/tech/game-host-sdk.md](../docs/tech/game-host-sdk.md)
- API：`platform-api` 同一 JWT 与 HMAC

## 构建

1. Cocos Creator 打开 `games-lobby/client/`
2. 场景：Launch → Lobby（含 CompanionPanel + GameShelf）
3. **不包含** `assets/games/` 游戏 Bundle
4. 游戏按需从 CDN 下载（见 ADR-006）

## Deep Link（POC）

- 大厅唤起游戏：`gameslobby://game/dawugui`
- 独立游戏：`dawugui://play`

配置见 `game_standalone` 表。
