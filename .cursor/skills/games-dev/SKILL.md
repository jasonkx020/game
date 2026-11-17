---
name: games-dev
description: Guides Go/Pitaya/Cocos development for the GameS platform: dual-process architecture, GameEngine constraints, proto generation, local dev, lobby, dynamic bundles, and new-game onboarding. Use when modifying backend, client, admin, proto, migrations, or adding a new game in this repository.
---

# GameS 技术开发

## 项目速览

| 组件 | 路径 | 端口 |
| :--- | :--- | :--- |
| Auth | `auth-service/cmd/auth-api` | :8081 |
| 游戏 HTTP | `server/cmd/game-api` | :8080 |
| Pitaya 游戏 | `server/cmd/game` | :3250 WS |
| 运营后台 | `admin/cmd/admin-api`（嵌入 `web/` SPA） | :8082（**非游戏关键路径**） |
| 运营 HMR（可选） | `admin/web/` Vite | :5173 |
| Cocos 客户端 | `client/` | Cocos 3.8.8 |

技术栈：Go 1.22+ / Gin + OpenAPI / Pitaya v2 Standalone / PostgreSQL + Redis / Cocos Creator 3 + Vue 3。

本地启动见 [README.md](../../README.md)。

## 游戏大厅与动态 Bundle

- 大厅 API：`GET/PUT /v1/lobby/games`、`GET /v1/lobby/recommendations`
- **智能伴侣**：`POST /v1/companion/sessions`、`POST .../chat`（SSE）、`server/internal/platform/companion/`
- 客户端：`platform/companion/`（CompanionPanel）、`platform/lobby/GameShelf`、`platform/host/GameHost`
- 新游戏：实现 `{GameId}Module.ts` + Remote Bundle + 可选 `companionHooks`
- 本地 Bundle 服务：`make serve-bundles`（:8787）
- ADR：[006](../../docs/tech/adr/006-game-lobby-dynamic-bundle.md) [007](../../docs/tech/adr/007-companion-llm.md) [008](../../docs/tech/adr/008-game-host-sdk.md)

## 架构硬约束

摘自 [ADR-002](../../docs/tech/adr/002-pluggable-game-constraints.md)，**违反即拒绝合并**：

| 规则 | 说明 |
| :--- | :--- |
| `server/internal/platform/*`、`server/pkg/*` | **禁止** import `internal/game/{具体游戏}` |
| `server/internal/game/{id}/` | **禁止** import Pitaya / Gin / Redis；**禁止**跨游戏互引 |
| `server/internal/game/{id}/internal/` | 仅本游戏可引用；对外只暴露 `GameID` / `New` / `NewBotPicker` |
| `handlers/room`、`pitaya/bot` | 只许 `engine` + `game.Get`/`Bot`；bot 走 `BotPicker` |
| `server/cmd/game-api` | **禁止** import pitaya handlers；**禁止**依赖 `admin-api` |
| `admin/` | 运营独立工程；停服不得影响玩家路径 |
| `server/internal/game/engine` | 仅 interface 与类型，无 IO |
| `server/internal/pitaya/handlers/room` | 默认禁止改；须 [room-change-checklist](../../docs/tech/room-change-checklist.md) + CODEOWNERS |
| `server/internal/pitaya/handlers/*` | 薄层：Session/Group/Push/audit，调用 Engine |

**职责分工**：GameEngine = 纯规则（服务端权威）；Pitaya Handler = IO + `commitEvents()` 落库与 Push。边界检查：`make lint` / `.\scripts\dev.ps1 lint`。  
可选能力（Phased / Snapshotter / SpectatorViewer / BotPicker、旁观托管重连）见 [new-game-guide](../../docs/tech/new-game-guide.md) 与 ADR-002。

## 本地开发命令

### Windows

```powershell
.\scripts\dev.ps1 up          # Postgres + Redis
.\scripts\dev.ps1 seed-dev    # 迁移 + 种子
.\scripts\dev.ps1 gen-proto   # Go proto
.\scripts\dev.ps1 run-auth       # :8081
.\scripts\dev.ps1 run-game-api        # :8080
.\scripts\dev.ps1 run-game       # :3250
.\scripts\dev.ps1 run-admin      # :8082 API+SPA（可选；需先 run-auth）
.\scripts\dev.ps1 run-admin-web  # :5173 HMR（可选；另开 run-admin-api）
.\scripts\dev.ps1 test           # go test ./...
.\scripts\dev.ps1 lint           # 游戏模块 import 边界
```

### Linux / macOS

```bash
make up && make seed-dev
make gen-proto
make run-auth   # 终端 1
make run-game-api    # 终端 2
make run-game   # 终端 3
# 可选：make run-admin   # :8082 API+SPA
make test
make lint
```

### Proto 与客户端

```bash
make gen-proto              # Go: server/internal/gen/pitaya/
make gen-client-proto       # TS: client/assets/platform/generated/pitaya/
cd client && npm test       # Vitest（不依赖 Cocos 编辑器）
```

### 生产构建

```bash
make build-linux    # server/bin/game-api, server/bin/game
make docker-build   # Linux 镜像（context=server/）
```

环境变量：复制 `server/.env.example` → `server/.env`。Access JWT 为 ES256（Auth 私钥 + 业务公钥）；生产勿用 dev 默认 `HMAC_SECRET` / dev PEM。

## 目录地图

```
auth-service/           # 登录（:8081）
server/                 # 游戏平台（独立 go.mod）
  cmd/game-api/     玩家 HTTP :8080
  cmd/game/             Pitaya WS
  pkg/                  共享领域包（admin replace）
  internal/platform/    房间/大厅/companion 等
  internal/game/        GameEngine
  internal/pitaya/      Handler + commitEvents + bot
  migrations/ deploy/
admin/                  # 运营独立工程
  cmd/admin-api/        :8082（API + 嵌入 SPA）
  web/                  Vue 源码
  internal/webui/       go:embed dist
client/                 # Cocos + Platform SDK（目录约束见 docs/tech/client-structure.md）
docs/tech/              技术文档与 ADR
```

## 常见任务决策树

**改 HTTP API（登录/钱包/开房/俱乐部）**
→ `docs/tech/openapi/` 改契约 → `server/internal/platform/api/` 实现 → 必要时 `server/migrations/`

**改微信 / OAuth**
→ 先读 [oauth.md](../../docs/tech/oauth.md) 与 `.cursor/rules/oauth.mdc` → 只改 `auth-service/`（表：`auth_accounts`、`auth_oauth_identities`）→ OpenAPI；**禁止**在游戏库建 OAuth 表或覆盖已有大厅 nickname

**改游戏规则或状态机**
→ `server/internal/game/{id}/` Engine → `server/internal/pitaya/handlers/{id}/` Handler → `server/proto/pitaya/{id}.proto` → `make gen-proto` + `make gen-client-proto`

**改客户端联调**
→ 先读 [client-structure.md](../../docs/tech/client-structure.md) → `client/assets/platform/sdk/`（ApiClient/PitayaClient/GameSession）→ `lobby/` → `games/{id}/`（Module/PushHandler/Bundle）；`cd client && npm test`

**改游戏大厅或用户偏好**
→ `server/internal/platform/lobby/` + `server/migrations/` + `LobbyScene.ts` + OpenAPI `lobby.yaml`

**改运营后台**
→ `admin/web/src/` + `admin/internal/api/`（admin-api）；登录仍在 `auth-service`；**禁止**让 game-api/game 依赖 admin-api

**改数据库**
→ `server/migrations/` 新增 up/down；**禁止**修改已发布 proto message 字段编号

**新增游戏**
→ 见 [new-game-checklist.md](new-game-checklist.md)；参考 `dawugui`（完整）与 `liuzichong`（第二范例）

## 代码原则

### Pitaya Route

格式：`{serverType}.{handler}.{method}`，MVP `serverType=game`。

| 域 | Handler | 示例 |
| :--- | :--- | :--- |
| 连接 | connector | `game.connector.entry` |
| 房间 | room | `game.room.join` |
| 游戏 | {gameId} | `game.dawugui.playcards` |

Push Route：`onRoomState`、`onDeal`、`onTurnNotify`、`onAlert`、`onSettlement`、`onError`。

### 审计与回放（ADR-005）

- 所有 Push 必含 `PushHeader.meta`（`audit_sn` + `action_seq`）
- Handler 统一 `commitEvents()`：每条 GameEvent → `action_seq++` → INSERT `game_action_log` → GroupBroadcast
- 详见 [audit-action-log.md](../../docs/tech/audit-action-log.md)、[replay.md](../../docs/tech/replay.md)

### GameEngine 契约

| 方法 | 职责 |
| :--- | :--- |
| `Meta()` | game_id、人数 |
| `NewState()` | 开局状态 |
| `ApplyAction()` | 玩家操作 |
| `OnTick()` | 超时托管（可选） |
| `VisibleState()` | 断线重连掩码 |
| `CheckRoundEnd()` | 胜负/无效局 |
| `CalcSettlement()` | 规则积分 |

接口变更需 ADR 修订；优先版本化（`GameEngineV2`）而非破坏现有实现。

### 增游戏注册点

1. `server/internal/game/registry.go` — `Get(gameID)` switch
2. `server/cmd/game/main.go` — `pitaya.Register(..., component.WithName("{gameId}"))`

### 编码惯例

- 状态机：**手写 phase enum**，不用 FSM 库
- Engine **禁止** import 框架；测试放 `server/internal/game/{id}/engine_test.go`
- HTTP 请求需 HMAC 签名，见 [http-signature.md](../../docs/tech/openapi/http-signature.md)
- 最小 diff：只改任务相关文件，匹配现有命名与 import 风格

## P0 验收流程

1. `POST /v1/auth/login` — HMAC 签名头
2. `POST /v1/rooms` — 开房，获 `room_id` + `ws_url`
3. Pitaya WS：`game.connector.entry` → `bind` → `game.room.join` → `ready`
4. 游戏路由：如 `game.dawugui.playcards` / `game.dawugui.pass`

## 延伸阅读

| 文档 | 内容 |
| :--- | :--- |
| [new-game-checklist.md](new-game-checklist.md) | 新游戏接入清单 |
| [platform-architecture.md](../../docs/tech/platform-architecture.md) | 分层与模块 |
| [game-framework.md](../../docs/tech/game-framework.md) | Handler + Engine |
| [client-architecture.md](../../docs/tech/client-architecture.md) | Cocos SDK |
| [protocol.md](../../docs/tech/protocol.md) | HTTP + Pitaya 全栈协议 |
| [adr/](../../docs/tech/adr/README.md) | 架构决策 |
