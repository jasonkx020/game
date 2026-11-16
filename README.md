# 打乌龟游戏平台 — 本地开发

仓库按 **auth / server / admin / client** 分目录管理：

| 目录 | 职责 |
| :--- | :--- |
| [auth-service/](auth-service/) | 独立登录与 Token（auth-api :8081；双 Token refresh） |
| [server/](server/) | 游戏平台双进程（Gin game-api + Pitaya game）、迁移、proto、部署 |
| [admin/](admin/) | 运营后台（admin-api :8082 嵌入 Vue SPA）；**停服不影响游戏** |
| [client/](client/) | Cocos Creator 3.8.8 游戏客户端 + Platform SDK |

- **本地开发**：Windows / macOS / Linux 均可（Go 源码跨平台）
- **生产部署**：Linux（`server/bin/` 交叉编译或 `server/deploy/Dockerfile` 镜像）

## 前置

- Go 1.22+（含数据库迁移：`go run ./cmd/migrate`，在 `server/` 下执行）
- Node.js 18+（运营后台、客户端 Vitest）
- Cocos Creator 3.8.8（运行游戏客户端场景）
- Docker Desktop（可选：本地 Postgres + Redis；`dev.ps1 up`）

## Windows 快速启动

```powershell
# 1. 环境变量（放在 server/；生产/外置 PG 可改用 init-db 生成账号）
Copy-Item server\.env.example server\.env
Copy-Item auth-service\.env.example auth-service\.env
# 可选首次建库（外置/本机 PG；Compose 默认同名库可跳过）：
# cd deploy\postgresql; .\init-db.bat       # 或 .\scripts\dev.ps1 init-db
# 说明见 deploy/postgresql/README.md 与 docs/tech/database.md

# 2. 基础设施
.\scripts\dev.ps1 up

# 3. 数据库迁移 + 种子（使用 server/.env 中的 DATABASE_URL）
.\scripts\dev.ps1 seed-dev

# 4. 生成 proto（需 buf）
.\scripts\dev.ps1 gen-proto

# 5. 启动服务 — 任选其一：
#    A) 三个终端
.\scripts\dev.ps1 run-auth   # :8081 登录/刷新
.\scripts\dev.ps1 run-game-api    # :8080 业务 API
.\scripts\dev.ps1 run-game   # :3250 WS
#    B) VS Code/Cursor：Run and Debug -> Server (api + game)

# 6. 运营后台（可选；不启也不影响游戏三件套）
.\scripts\dev.ps1 run-admin      # 构建 Vue 并启动 admin-api :8082（API+SPA）

# 6b. 可选：单端口 HTTP 网关（auth + game-api + admin 同源）
# docker compose -f deploy/docker-compose.gateway.yml up -d
# → http://localhost:8090/   见 docs/tech/gateway.md

# 7. 游戏 Bundle 静态服务（可选，按需下载）
.\scripts\dev.ps1 serve-bundles  # :8787
```

## Linux / macOS 快速启动

```bash
cp server/.env.example server/.env
make up
make seed-dev          # 在 server/ 执行 go run ./cmd/migrate up
make gen-proto
make run-auth          # 终端 1 — auth-api :8081
make run-game-api           # 终端 2 — game-api :8080
make run-game          # 终端 3 — Pitaya :3250
# 可选运营：make run-admin   # :8082 API+SPA
```

根目录 `make` 会转发到 `server/Makefile`；也可直接 `cd server && make ...`。

### 数据库迁移

详细说明（Auth / 游戏两套迁移、是否拆库、首次建库脚本）：[docs/tech/database.md](docs/tech/database.md)。

| 命令 | Windows | Linux/macOS |
|------|---------|-------------|
| 首次建用户/库/密码 | `deploy\postgresql\init-db.bat` / `.\scripts\dev.ps1 init-db` | `deploy/postgresql/init-db.sh` / `make init-db` |
| 查看状态（游戏库） | `.\scripts\dev.ps1 migrate-status` | `make migrate-status` |
| 升级到最新（游戏库） | `.\scripts\dev.ps1 migrate` / `seed-dev` | `make migrate` / `seed-dev` |
| 回滚 1 步 | `.\scripts\dev.ps1 migrate-down` | `make migrate-down` |
| 回滚 N 步 | `.\scripts\dev.ps1 migrate-down -Steps 2` | `make migrate-down STEPS=2` |
| Auth 迁移 | `cd auth-service; go run ./cmd/migrate -path migrations`（或随 `run-auth`） | 同左 |

游戏：`cd server && go run ./cmd/migrate status|up|down [N]`（`schema_migrations`）。  
Auth：独立版本表 `auth_schema_migrations`；`seed-dev` **不含** Auth 迁移。

## 生产构建（Linux）

```bash
# 方式 A：交叉编译静态二进制（在开发机执行，产物拷贝到 Linux 服务器）
make build-linux       # 输出 server/bin/game-api, server/bin/game

# 方式 B：Docker 镜像（推荐）
make docker-build      # game-api:latest, game-server:latest
make docker-up-prod    # 参考 server/deploy/docker-compose.prod.yml
```

生产环境请修改 `HMAC_SECRET`、`POSTGRES_PASSWORD`，并替换 Auth ES256 密钥对（勿使用仓库内 dev PEM），业务服只配置公钥。

## 游戏客户端

| 项 | 说明 |
|----|------|
| 工程 | [client/](client/)（Cocos Creator 3.8.8） |
| SDK | ApiClient + PitayaClient + GameSession |
| 场景 | Launch → **Lobby** → Room（脚本已提供，需在编辑器挂载 UI） |
| 大厅 | API 驱动游戏列表，`GET /v1/lobby/games`；支持隐藏/置顶 |
| Bundle | `make serve-bundles` 本地托管；未构建时自动回退内置模块 |
| 单测 | `make test-client` 或 `cd client && npm test` |

## 运营后台（P1）

| 项 | 说明 |
|----|------|
| 地址 | http://localhost:8082/（SPA 嵌入 admin-api） |
| API | 同端口 `/v1/admin/*`（**非游戏关键路径**） |
| 管理员账号 | 手机 `13800000000`，验证码 `123456`（见 `server/.env` 中 `DEV_SMS_CODE`） |
| 工程 | [admin/](admin/)（`web/` 构建进 `cmd/admin-api`） |
| 功能 | 仪表盘 KPI、俱乐部管理、Mock 房卡充值、游戏 ops-hooks、伴侣人设 |
| 协议与接入文档 | http://localhost:8082/docs（**免登录**；OpenAPI / 协议 / 接入指南） |
| 可选 HMR | `run-admin-web` → :5173（需另开 `run-admin-api`） |

业务 API 与 SPA 同域；登录打 `auth-service`，请求需 HMAC 签名（见 `docs/tech/openapi/http-signature.md`）。

## P0 验收流程

1. `POST /v1/auth/sign-in` — body `{"phone":"13800000001","sms_code":"123456"}` + HMAC 签名头（打到 auth-api `:8081`；响应含 `is_new_user`）
2. `POST /v1/rooms` — 开房，获 `room_id` + `ws_url`（房卡消耗读 `game_ops_config`）
3. Pitaya WS：`game.connector.entry` → `game.connector.bind` → `game.room.join` → `game.room.ready`
4. `game.dawugui.playcards` / `game.dawugui.pass`

## P1 API 摘要

- 俱乐部：`/v1/clubs/*`（成员、房卡池划拨）
- 游戏目录：`GET /v1/games`、`GET /v1/games/{id}/config`
- 游戏大厅：`GET /v1/lobby/games`、`PUT /v1/lobby/games`（用户偏好）
- Mock 充值：`POST /v1/wallet/room-card/recharge`（`rc_10` / `rc_50` / `rc_200`）
- 运营报表（admin-api）：`GET /v1/admin/metrics/overview`、`/v1/admin/metrics/room-cards`

## 目录

```
auth-service/           # 登录 / Token（:8081）
server/                 # 游戏平台（独立 go.mod）
  cmd/game-api/     玩家/大厅 HTTP :8080
  cmd/game/             Pitaya WS :3250
  pkg/                  可导出共享包（供 admin-api replace 引用）
  internal/platform/    房间/大厅/companion/replay 等玩家侧
  internal/game/        GameEngine
  internal/pitaya/      Handler + commitEvents + bot
  migrations/           PostgreSQL DDL
  deploy/
admin/                  # 运营独立工程
  cmd/admin-api/        运营 HTTP + 嵌入 SPA :8082
  web/                  Vue 源码（build → internal/webui/dist）
client/                 # Cocos Creator 3 客户端
scripts/dev.ps1
docs/
```

## 文档

- **在线（推荐）**：[http://localhost:8082/docs](http://localhost:8082/docs) — 后台文档中心（协议、OpenAPI、接入指南）
- 技术架构：[docs/tech/README.md](docs/tech/README.md)
- Auth 契约：[auth-service/openapi/](auth-service/openapi/)
- 运营手册：[运营手册.md](运营手册.md)
