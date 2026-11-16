# 数据库初始化与 Auth / 游戏分离

> 本地启动总览见根 [README.md](../../README.md)。表归属细节另见 [oauth.md](oauth.md)、[auth-service.md](auth-service.md)。

## 结论（是否已分离）

| 维度 | 状态 |
| :--- | :--- |
| 进程 / 代码 | **已分离**：`auth-service` vs `server`（game-api / game） |
| 迁移目录 | **已分离**：`auth-service/migrations` vs `server/migrations` |
| 版本表 | **已分离**：Auth 用 `auth_schema_migrations`；游戏用 `schema_migrations`（互不踩版本号） |
| 表归属 | **已分离**：`auth_*` 仅 Auth；游戏侧 `players` / `admin_users` 等用 `auth_account_id` 软链 |
| 物理库 | **默认未拆**：本地/compose 默认同一 Postgres 库 `game`；生产可配不同 `DATABASE_URL` |

游戏库**禁止**出现 `auth_oauth_*` 等 Auth 表，也禁止跨库 JOIN。鉴权用 JWT `AccountID` → `EnsureFromAuth` 建/链本地行。

## 首次部署：创建角色 / 库 / 密码

主入口：[deploy/postgresql/](../../deploy/postgresql/)（自包含；详见该目录 [README.md](../../deploy/postgresql/README.md)）。

在**已有 Postgres 实例**上幂等创建应用用户与库，并可自动写 `.env`：

| 平台 | 命令 |
| :--- | :--- |
| Windows（`postgres` **默认可无密码**） | `cd deploy\postgresql` → `init-db.bat`；或根目录 `.\scripts\dev.ps1 init-db` |
| Linux/macOS | `cd deploy/postgresql && ./init-db.sh`；或 `make init-db` |
| 物理拆 Auth 库 | `init-db.bat all --split-auth` / `./init-db.sh all --split-auth` |
| 超级用户有密码 | 在 `config.ini` `[Superuser] password=` 填写，或设 `PGPASSWORD` |

> 官方/包安装的 Postgres 超级用户常为 **peer/trust、无密码**。`init-db` 默认走本机 socket。应用用户 `game` 密码空则随机生成。

脚本行为：

1. 确保 LOGIN 角色（默认 `game`）存在并设密码
2. 创建数据库（默认 `game`；`--split-auth` 再加 `auth`）
3. 写入 `deploy/postgresql/postgres.env`（**gitignore**）
4. `write_env=true`（默认）：回写 `server/.env`、`auth-service/.env` 的 `DATABASE_URL`

> 本地 `docker compose up` 已用 `POSTGRES_USER/PASSWORD/DB=game` 时，可不跑 init-db，直接用 `.env.example` 的 `game:game`。外置/生产 Postgres 请跑 `deploy/postgresql` 的 init-db。

## 本地初始化（迁移）

```powershell
# 1. 环境变量（可用 init-db -WriteEnv 自动生成 DATABASE_URL）
Copy-Item server\.env.example server\.env
Copy-Item auth-service\.env.example auth-service\.env
# 或：cd deploy\postgresql; .\init-db.bat   # 建用户/库并写 .env

# 2. 起 Postgres + Redis（Docker 开发）
.\scripts\dev.ps1 up

# 3. 游戏库迁移（含 seed；版本表 schema_migrations）
.\scripts\dev.ps1 seed-dev
# 等价：cd server; go run ./cmd/migrate up

# 4. Auth 库迁移（版本表 auth_schema_migrations）
# 方式 A：启动 Auth 时自动迁移
.\scripts\dev.ps1 run-auth
# 方式 B：只跑迁移
cd auth-service; go run ./cmd/migrate -path migrations
```

Linux / macOS：

```bash
# 可选首次建库：cd deploy/postgresql && ./init-db.sh
cp server/.env.example server/.env
cp auth-service/.env.example auth-service/.env
make up
make seed-dev          # 仅游戏库
make run-auth          # 内含 auth 迁移 + 启动 :8081
# 或：cd auth-service && go run ./cmd/migrate -path migrations
```

**注意**：`seed-dev` / `make seed-dev` **不会**跑 Auth 迁移；Auth 表需 `run-auth` 或单独执行 auth `cmd/migrate`。

### 默认连接

两边 `.env.example` 默认相同：

```
DATABASE_URL=postgres://game:game@localhost:5432/game?sslmode=disable
```

可选：`MIGRATE_DATABASE_URL`（仅迁移用；`dev.ps1 migrate-docker` 可能改写为 `host.docker.internal`）。

## 迁移命令一览

| 目的 | Windows | Linux/macOS / 直接 |
| :--- | :--- | :--- |
| 建用户/库/密码 | `deploy\postgresql\init-db.bat` 或 `.\scripts\dev.ps1 init-db` | `deploy/postgresql/init-db.sh` 或 `make init-db` |
| 游戏：状态 | `.\scripts\dev.ps1 migrate-status` | `make migrate-status` / `cd server && go run ./cmd/migrate status` |
| 游戏：升级 | `.\scripts\dev.ps1 migrate` 或 `seed-dev` | `make migrate` / `seed-dev` |
| 游戏：回滚 | `.\scripts\dev.ps1 migrate-down`（`-Steps N`） | `make migrate-down STEPS=N` |
| Auth：升级 | `cd auth-service; go run ./cmd/migrate -path migrations` | 同左；或随 `make run-auth` |
| Compose 生产 | `server/deploy/docker-compose.prod.yml` 中 `migrate` + `auth-migrate` 两服务 | 仍默认同一 `game` 库 |

Auth dirty：确认是否误连了游戏库的 `schema_migrations`；可用 `go run ./cmd/migrate -path migrations -force <V>` 清 dirty（见 [auth-service/README.md](../../auth-service/README.md)）。

## 迁移目录与职责

### Auth（`auth-service/migrations`）

典型表：`auth_accounts`、`auth_refresh_tokens`、`auth_oauth_identities`、实名相关字段等。  
版本记录：`auth_schema_migrations`。

### 游戏（`server/migrations`）

典型表：`players`、`admin_users`、钱包、房间、俱乐部、companion、审计等；`000007_seed_dev` 写入本地种子。  
`000013_auth_account_link` 等为软链列（`auth_account_id`），**不**复制 OAuth 明细。  
版本记录：`schema_migrations`。

## 生产拆库（可选）

逻辑上已支持「两套 DSN」：

1. Auth：`auth-service` 的 `DATABASE_URL` 指向独立库（如 `auth`）
2. 游戏：`server` / admin-api 的 `DATABASE_URL` 指向游戏库（如 `game`）
3. 分别跑两边 migrate；不要把 Auth SQL 打进游戏 migrate 工具

默认 compose **仍共用**一个 Postgres 与库名 `game`，靠两套版本表共存。真正物理拆分时改环境变量即可，无需改表归属约定。

## 相关文档

| 文档 | 内容 |
| :--- | :--- |
| [auth-service.md](auth-service.md) | Auth 职责、JWT ES256、Refresh |
| [oauth.md](oauth.md) | OAuth 表落点与修改约束 |
| [../../auth-service/README.md](../../auth-service/README.md) | Auth 本地 migrate 说明 |
| [../../README.md](../../README.md) | 一键启动 |
| [../../deploy/postgresql/README.md](../../deploy/postgresql/README.md) | 本机 PG 启停与 init-db |
| [gateway.md](gateway.md) | 单端口分流（与库无关） |
