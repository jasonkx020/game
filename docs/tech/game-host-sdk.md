# GameHostSDK 契约

> 游戏大厅与独立游戏 App 的统一启动协议。ADR：[008-game-host-sdk.md](adr/008-game-host-sdk.md)

---

## 启动模式

### lobby — 从大厅进入

```typescript
await GameHost.launch({
  mode: 'lobby',
  gameId: 'dawugui',
  game: lobbyGameItem,       // GET /v1/lobby/games 项
  companionSessionId?: number // 局内伴侣上下文
})
```

流程：`GameBundleManager.ensureLoaded` → `POST /v1/rooms` → `RoomScene`

### standalone — 独立 App

```typescript
await GameHost.launch({
  mode: 'standalone',
  gameId: 'dawugui',
})
```

流程：确保已登录 → 加载 Bundle → `createRoom` → `RoomScene`

实现：[client/assets/platform/host/GameHost.ts](../../client/assets/platform/host/GameHost.ts)

---

## 游戏包要求

1. `{GameId}Module.ts` 注册 `GameModuleRegistry`
2. `GameEntry.ts`（Bundle 入口）
3. 可选 `StandaloneEntry.ts`（独立 App 根）
4. 不 import 大厅 UI

---

## Deep Link

| scheme | 用途 |
| :--- | :--- |
| `gameslobby://game/{id}` | 大厅唤起 |
| `{gameId}://play` | 独立游戏（见 `game_standalone`） |

---

## 版本

- `game_client_bundle.min_host_version`：游戏 Bundle 要求的平台壳版本
- `game_standalone.min_lobby_version`：从大厅启动该独立包的最低大厅版本
