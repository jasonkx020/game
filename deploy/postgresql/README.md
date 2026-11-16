# GameS PostgreSQL launcher（首次建库）

本目录提供本机/生产外置 Postgres 的启停与**首次创建应用用户 + 数据库**。  
Docker Compose 开发（默认 `game:game`）可跳过；外置 PG 推荐走这里。

迁移与种子仍用仓库根命令：`seed-dev` / `run-auth`。详见 [docs/tech/database.md](../../docs/tech/database.md)。

## 布局

```
deploy/postgresql/
  config.ini                 # 共享配置（Windows + Linux）
  config.linux.ini.example   # Ubuntu 示例
  read-postgres-config.ps1 / .sh
  load-postgres-config.cmd
  db-init.ps1 / db-init.sh   # 自包含建角色/库
  init-db.bat / init-db.sh   # 入口
  start|stop|status-postgres.*
  psql-client.*
  postgres.env               # init-db 生成（gitignore）
```

## config.ini

| Section | Key | 说明 |
|---------|-----|------|
| PostgreSQL | `install_path` | PG 根目录（含 `bin/psql`）；`.` = 自动探测 |
| PostgreSQL | `run_mode` | Windows: `service` \| `pg_ctl`；Linux: `systemd` \| `pg_ctl` |
| Superuser | `user` / `password` | 超级用户；**password 默认可空**（peer/trust） |
| Application | `user` / `password` | 应用角色（默认 `game`）；password 空则 init-db 随机生成 |
| Application | `db` / `db_auth` | 游戏库；可选独立 Auth 库名 |
| Init | `default_scope` | `game` \| `auth` \| `all` |
| Init | `write_env` | `true` 时回写 `server/.env` 与 `auth-service/.env` |

查看解析结果：

```powershell
.\read-postgres-config.ps1 -PrintSummary
```

```bash
./read-postgres-config.sh --print-summary
```

## 首次部署

**超级用户无密码（默认）：**

```bat
cd deploy\postgresql
start-postgres.bat
init-db.bat
```

```bash
cd deploy/postgresql
chmod +x *.sh
./start-postgres.sh
./init-db.sh
```

可选：

```bat
init-db.bat all --split-auth
init-db.bat --no-write-env
```

```bash
./init-db.sh all --split-auth
./init-db.sh --no-write-env
```

然后在仓库根：

```powershell
.\scripts\dev.ps1 seed-dev
.\scripts\dev.ps1 run-auth
```

```bash
make seed-dev
make run-auth
```

## 与根目录入口

`.\scripts\dev.ps1 init-db` / `make init-db` 会转发到本目录的 `db-init`，避免两套实现。

## Windows / Linux 启停

- Windows 安装版：`run_mode=service`
- 便携 `pg_ctl`：`run_mode=pg_ctl`，`data_path=.\data`
- Ubuntu apt：`run_mode=systemd`（可参考 `config.linux.ini.example`、`install-postgres.sh`）
