---
name: game-dev
description: Guides Go/Pitaya/Cocos development for the game platform: dual-process architecture, GameEngine constraints, proto generation, local dev, and new-game onboarding. Use when modifying backend, client, admin, proto, migrations, or adding a new game in this repository.
---

# game 技术开发

打乌龟游戏平台（game）全栈开发约定。详细架构见 [docs/tech/README.md](../../docs/tech/README.md)。

## 项目速览

| 组件 | 路径 | 端口 |
| :--- | :--- | :--- |
| HTTP 平台 | `cmd/platform-api` | :8080 |
| Pitaya 游戏 | `cmd/game` | :3250 WS |
| 运营后台 | `web/admin/` | :5173 |
| Cocos 客户端 | `client/` | Cocos 3.8.8 |

技术栈：Go 1.22+ / Gin + OpenAPI / Pitaya v2 Standalone / PostgreSQL + Redis / Cocos Creator 3 + Vue 3。

本地启动见 [README.md](../../README.md)。

## 架构硬约束

摘自 [ADR-002](../../docs/tech/adr/002-pluggable-game-constraints.md)，**违反即拒绝合并**：

| 规则 | 说明 |
| :--- | :--- |
| `internal/platform/*` | **禁止** import `internal/game/{具体游戏}` |
| `internal/game/{id}/` | **禁止** import Pitaya / Gin / Redis |
| `cmd/platform-api` | **禁止** import pitaya handlers |
| `internal/game/engine` | 仅 interface 与类型，无 IO |
| `internal/pitaya/handlers/*` | 薄层：Session/Group/Push/audit，调用 Engine |

**职责分工**：GameEngine = 纯规则（服务端权威）；Pitaya Handler = IO + `commitEvents()` 落库与 Push。

## 本地开发命令

### Windows

```powershell
.\scripts\dev.ps1 up          # Postgres + Redis
.\scripts\dev.ps1 seed-dev    # 迁移 + 种子
.\scripts\dev.ps1 gen-proto   # Go proto
.\scripts\dev.ps1 run-api     # :8080
.\scripts\dev.ps1 run-game    # :3250
.\scripts\dev.ps1 run-admin   # :5173（需先 run-api）
.\scripts\dev.ps1 test        # go test ./...
```

### Linux / macOS

```bash
make up && make seed-dev
make gen-proto
make run-api    # 终端 1
make run-game   # 终端 2
make test
```

### Proto 与客户端

```bash
make gen-proto              # Go: internal/gen/pitaya/
make gen-client-proto       # TS: client/assets/platform/generated/pitaya/
cd client && npm test       # Vitest（不依赖 Cocos 编辑器）
```

### 生产构建

```bash
make build-linux    # bin/platform-api, bin/game
make docker-build   # Linux 镜像
```

环境变量：复制 `.env.example` → `.env`。生产勿用 dev 默认 `JWT_SECRET` / `HMAC_SECRET`。

## 目录地图

```
cmd/platform-api/     Gin HTTP
cmd/game/             Pitaya WS
internal/platform/    用户/钱包/房间/俱乐部/metrics
internal/game/        GameEngine（每游戏一套）
internal/pitaya/      Handler + commitEvents
web/admin/            Vue 3 运营后台
client/               Cocos + Platform SDK
config/ops-hooks/     游戏运营配置 JSON
proto/pitaya/         Protobuf
migrations/           PostgreSQL DDL
deploy/               docker-compose, Dockerfile
docs/tech/            技术文档与 ADR
```

## 常见任务决策树

**改 HTTP API（登录/钱包/开房/俱乐部）**
→ `docs/tech/openapi/` 改契约 → `internal/platform/api/` 实现 → 必要时 `migrations/`

**改游戏规则或状态机**
→ `internal/game/{id}/` Engine → `internal/pitaya/handlers/{id}/` Handler → `proto/pitaya/{id}.proto` → `make gen-proto` + `make gen-client-proto`

**改客户端联调**
→ `client/assets/platform/sdk/`（ApiClient/PitayaClient/gameession）→ `client/assets/game/{id}/`（PushHandler/Scene）

**改运营后台**
→ `web/admin/src/`；API 契约见 `docs/tech/openapi/openapi.yaml`

**改数据库**
→ `migrations/` 新增 up/down；**禁止**修改已发布 proto message 字段编号

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

1. `internal/game/registry.go` — `Get(gameID)` switch
2. `cmd/game/main.go` — `pitaya.Register(..., component.WithName("{gameId}"))`

### 编码惯例

- 状态机：**手写 phase enum**，不用 FSM 库
- Engine **禁止** import 框架；测试放 `internal/game/{id}/engine_test.go`
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
