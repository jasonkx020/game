# 打乌龟游戏平台 — 本地开发

Go 双进程：**Gin platform-api** + **Pitaya game**；P1 运营后台 + **Cocos 客户端 SDK**。

- **本地开发**：Windows / macOS / Linux 均可（Go 源码跨平台）
- **生产部署**：Linux（`bin/` 交叉编译或 `deploy/Dockerfile` 镜像）

## 前置

- Go 1.22+
- Node.js 18+（运营后台、客户端 Vitest）
- Cocos Creator 3.8.8（运行游戏客户端场景）
- Docker Desktop（Postgres + Redis；Windows 上 migrate 也走 Docker）
- [migrate CLI](https://github.com/golang-migrate/migrate)（可选；Windows 推荐用 `scripts/dev.ps1 migrate`）

## Windows 快速启动

```powershell
# 1. 环境变量
Copy-Item .env.example .env

# 2. 基础设施
.\scripts\dev.ps1 up

# 3. 数据库迁移 + 种子（无需安装 migrate CLI）
.\scripts\dev.ps1 seed-dev

# 4. 生成 proto（需 buf）
.\scripts\dev.ps1 gen-proto

# 5. 启动服务 — 任选其一：
#    A) 两个终端
.\scripts\dev.ps1 run-api    # :8080
.\scripts\dev.ps1 run-game   # :3250 WS
#    B) VS Code/Cursor：Run and Debug -> Server (api + game)

# 6. 运营后台（可选）
cd web/admin; npm install; cd ../..
.\scripts\dev.ps1 run-admin  # :5173

# 7. 游戏 Bundle 静态服务（可选，按需下载）
.\scripts\dev.ps1 serve-bundles  # :8787
```

## Linux / macOS 快速启动

```bash
cp .env.example .env
make up
make seed-dev          # 使用 migrate-docker，无需本地 migrate
make gen-proto
make run-api           # 终端 1
make run-game          # 终端 2
```

若已安装 `migrate` CLI，可将 Makefile 中 `seed-dev` 改为依赖 `migrate` 目标。

## 生产构建（Linux）

```bash
# 方式 A：交叉编译静态二进制（在开发机执行，产物拷贝到 Linux 服务器）
make build-linux       # 输出 bin/platform-api, bin/game

# 方式 B：Docker 镜像（推荐）
make docker-build      # game-platform-api:latest, game-server:latest
make docker-up-prod    # 参考 deploy/docker-compose.prod.yml
```

生产环境请修改 `.env` 中的 `JWT_SECRET`、`HMAC_SECRET`、`POSTGRES_PASSWORD`，勿使用 dev 默认值。

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
| 地址 | http://localhost:5173 |
| 管理员账号 | 手机 `13800000000`，验证码 `123456`（见 `.env` 中 `DEV_SMS_CODE`） |
| 功能 | 仪表盘 KPI、俱乐部管理、Mock 房卡充值、游戏 ops-hooks 只读 |

Admin Web 与 Cocos 客户端共用同一 `platform-api`，请求需 HMAC 签名（见 `docs/tech/openapi/http-signature.md`）。

## P0 验收流程

1. `POST /v1/auth/login` — body `{"phone":"13800000001","sms_code":"123456"}` + HMAC 签名头
2. `POST /v1/rooms` — 开房，获 `room_id` + `ws_url`（房卡消耗读 `game_ops_config`）
3. Pitaya WS：`game.connector.entry` → `game.connector.bind` → `game.room.join` → `game.room.ready`
4. `game.dawugui.playcards` / `game.dawugui.pass`

## P1 API 摘要

- 俱乐部：`/v1/clubs/*`（成员、房卡池划拨）
- 游戏目录：`GET /v1/games`、`GET /v1/games/{id}/config`
- 游戏大厅：`GET /v1/lobby/games`、`PUT /v1/lobby/games`（用户偏好）
- Mock 充值：`POST /v1/wallet/room-card/recharge`（`rc_10` / `rc_50` / `rc_200`）
- 运营报表：`GET /v1/admin/metrics/overview`、`/v1/admin/metrics/room-cards`（需 `platform_admin` 或 `club_admin` 角色）

## 目录

```
cmd/platform-api/   Gin HTTP
cmd/game/           Pitaya WS
scripts/dev.ps1     Windows 开发脚本
internal/platform/  用户/钱包/房间/俱乐部/catalog/metrics
internal/game/      GameEngine（dawugui）
internal/pitaya/    Handler + commitEvents
web/admin/          Vue 3 运营后台
client/             Cocos Creator 3 客户端 + Platform SDK
config/ops-hooks/   游戏运营配置 JSON
proto/pitaya/       Protobuf
migrations/         PostgreSQL DDL
deploy/             docker-compose.yml, Dockerfile, prod compose
.vscode/launch.json Go 调试配置
```

## 文档

技术架构见 [docs/tech/README.md](docs/tech/README.md)。运营手册见 [运营手册.md](运营手册.md)。
