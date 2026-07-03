# 打乌龟游戏平台 — 本地开发

Go 双进程：**Gin platform-api** + **Pitaya game**；P1 运营后台 + **Cocos 客户端 SDK**。

## 前置

- Go 1.22+
- Node.js 18+（运营后台、客户端 Vitest）
- Cocos Creator 3.8+（运行游戏客户端场景）
- Docker / Docker Compose
- [migrate CLI](https://github.com/golang-migrate/migrate)（可选，亦可用 docker 运行）

## 快速启动

```bash
# 1. 环境变量
cp .env.example .env

# 2. 基础设施
make up

# 3. 数据库迁移 + 种子数据（含平台管理员、打乌龟 ops-hooks）
set DATABASE_URL=postgres://game:game@localhost:5432/game?sslmode=disable
make seed-dev

# 4. 生成 proto（需 buf）
make gen-proto

# 5. 启动服务（两个终端）
make run-api    # :8080
make run-game   # :3250 WS

# 6. 运营后台（可选，第三个终端）
cd web/admin && npm install
make run-admin  # :5173，代理至 platform-api

# 7. 游戏客户端（Cocos Creator 打开 client/ 目录）
make gen-client-proto
cd client && npm install && npm test   # SDK 单测
# 详见 client/README.md
```

## 游戏客户端

| 项 | 说明 |
|----|------|
| 工程 | [client/](client/)（Cocos Creator 3.8） |
| SDK | ApiClient + PitayaClient + GameSession |
| 场景 | Launch → Hall → Room（脚本已提供，需在编辑器挂载 UI） |
| 单测 | `make test-client` |

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
- Mock 充值：`POST /v1/wallet/room-card/recharge`（`rc_10` / `rc_50` / `rc_200`）
- 运营报表：`GET /v1/admin/metrics/overview`、`/v1/admin/metrics/room-cards`（需 `platform_admin` 或 `club_admin` 角色）

## 目录

```
cmd/platform-api/   Gin HTTP
cmd/game/           Pitaya WS
internal/platform/  用户/钱包/房间/俱乐部/catalog/metrics
internal/game/      GameEngine（dawugui）
internal/pitaya/    Handler + commitEvents
web/admin/          Vue 3 运营后台
client/             Cocos Creator 3 客户端 + Platform SDK
config/ops-hooks/   游戏运营配置 JSON
proto/pitaya/       Protobuf
migrations/         PostgreSQL DDL
deploy/             docker-compose.yml
```

## 文档

技术架构见 [docs/tech/README.md](docs/tech/README.md)。运营手册见 [运营手册.md](运营手册.md)。
