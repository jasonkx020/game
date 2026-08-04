# deploy/run — Windows 人工部署与运维手册

> **HTML 版（浏览器直接打开）：** [README.html](./README.html)

本文档面向现场运维人员，说明如何在 **Windows 10/11** 上通过 `deploy/run/` 启动包人工部署与日常维护 **PostgreSQL、Redis、MinIO、NanoMQ、Nginx** 五个基础设施组件。

本目录提供便携脚本（`config.ini` + `.bat`），与 Linux 侧 `scripts/service-*.sh` 行为对齐。本文**不包含** tenant-api / tenant-web 的构建与打包，仅说明上述组件与后端的对接参数。

---

## 目录结构

```
deploy/run/
  README.md                 # 本文（运维主入口）
  postgresql/
    config.ini              # 数据库配置
    start-postgres.bat      # 启动
    stop-postgres.bat       # 停止
    status-postgres.bat     # 状态
    init-db.bat             # 建库与 schema
    psql-client.bat         # psql 客户端
    README.md               # 补充说明（英文）
  Redis/
    config.ini
    redis-server.exe / redis-cli.exe
    start-redis.bat / stop-redis.bat / status-redis.bat / reload-redis.bat
    redis-cli.bat           # redis-cli 客户端
    render-redis-conf.ps1   # 渲染 conf/redis.conf
    README.md
  minio/
    config.ini
    start-minio.bat / stop-minio.bat / status-minio.bat
    minio-mc.bat            # mc 客户端与 bucket 初始化
    minio.exe / mc.exe      # 可放置于本目录或 bin/
    README.md
  nanomq/
    config.ini
    start-nanomq.bat / stop-nanomq.bat
    render-nanomq-conf.ps1  # 渲染 etc/nanomq.conf
    nanomq.conf.template
    bin/nanomq.exe
    README.md
  nginx/
    nginx.exe
    start-nginx.bat / stop-nginx.bat / reload-nginx.bat
    start-nginx.sh          # Linux 便携启动（本文不展开）
    conf/nginx.conf         # 主配置
    conf/proxy/*.conf       # 反代规则
    conf/stream.conf        # MQTT 四层代理
    conf/ssl/               # TLS 证书（dev-server.crt/key）
```

---

## 端口一览

| 组件 | 默认端口 | 说明 |
|------|----------|------|
| PostgreSQL | 5432 | 应用库 `freetv_edge`（EDGE）、`freetv_cloud`（CLOUD） |
| Redis | 6379 | 缓存/会话（可选，`redis.enabled=true` 时启用） |
| MinIO API | 9000 | S3 API，`listen_bind` 默认 `127.0.0.1` |
| MinIO Console | 9001 | Web 管理控制台 |
| NanoMQ MQTT | 1883 | 设备/Hub TCP 连接 |
| NanoMQ HTTP | 8082 | 本机运维 REST（仅 127.0.0.1） |
| NanoMQ WebSocket | 8083 | WS 监听（nginx 可代理为 WSS :8084） |
| Nginx HTTP | 80 | 跳转至 HTTPS :8081 |
| Nginx 租户入口 | 8081 | HTTPS：Web + API + `/files/` + MinIO 路径前缀 |
| Nginx MinIO API | 9002 | 独立 HTTPS 反代至 MinIO :9000 |
| Nginx MinIO Console | 9003 | 独立 HTTPS 反代至 MinIO :9001 |
| Nginx MQTTS | 8883 | 四层 TLS 代理至 NanoMQ :1883 |
| Nginx WSS | 8084 | 四层 TLS 代理至 NanoMQ WS :8083 |
| Nginx 平台后台 | 8080 | HTTPS 反代 platform-api（可选，见 `platform.conf`） |
| tenant-api（上游） | 8443 | 需另行启动，非本目录脚本 |
| tenant-web（上游） | 2001 | 需另行启动，nginx `tenant.conf` upstream |

端口可在各组件 `config.ini` 或 nginx `conf/proxy/*.conf` 中修改；修改后须同步 tenant-api 配置与防火墙规则。

---

## 依赖与启动顺序

```mermaid
flowchart LR
  PG[PostgreSQL] --> API[tenant-api]
  API --> NanoMQ
  MinIO --> Nginx
  Web[tenant-web] --> Nginx
  API --> Nginx
  NanoMQ --> Nginx
```

**要点：**

1. PostgreSQL 须最先就绪；tenant-api 依赖数据库 DSN。
2. NanoMQ 通过 HTTP 回调 tenant-api 完成 MQTT 认证；`start-nanomq.bat` 默认会探测 `http://127.0.0.1:8443/api/v1/health`（可在 `nanomq/config.ini` 关闭）。
3. Nginx 反代 MinIO（:9000/:9001）及 tenant-api（:8443）、tenant-web（:2001）；启动前确认上游已监听。
4. **勿同时**将 EMQX 与 NanoMQ 作为 Hub 目标；切换 Broker 须停旧服务、改 tenant-api、重启后再启新 Broker。

### 推荐启动顺序

| 顺序 | 组件 | 命令 |
|------|------|------|
| 1 | PostgreSQL | `postgresql\start-postgres.bat` |
| 2 | 初始化库 | `postgresql\init-db.bat tenant --schema` |
| 3 | Redis（可选） | `Redis\start-redis.bat` |
| 4 | MinIO | `minio\start-minio.bat` |
| 5 | MinIO bucket | `minio\minio-mc.bat`（菜单选 3 创建 bucket） |
| 6 | tenant-api | 见 `ycwl-freetv-server-go/bin` 或 `deploy/README.md` |
| 7 | tenant-web | 生产静态或 Vite 开发服务（端口与 nginx upstream 一致） |
| 8 | NanoMQ | `nanomq\start-nanomq.bat` |
| 9 | Nginx | `nginx\start-nginx.bat` |

### 推荐停止顺序（与启动相反）

Nginx → NanoMQ → tenant-web / tenant-api → MinIO → PostgreSQL

---

## 首次部署 Checklist

### 1. 准备二进制与证书

| 组件 | 操作 |
|------|------|
| PostgreSQL | 安装 PostgreSQL 16+，或在 `postgresql/config.ini` 设 `run_mode=pg_ctl` 并将 `bin\psql.exe` 等放入 `postgresql/bin/` |
| Redis | 确认 `Redis/redis-server.exe` 存在（已随仓库提供 Windows 便携版） |
| MinIO | 将 `minio.exe`、`mc.exe` 放入 `minio/`（缺失时 `start-minio.bat` 可自动下载 minio） |
| NanoMQ | 从 [nanomq.io/downloads](https://nanomq.io/downloads) 解压 Windows 包到 `nanomq/bin/` |
| Nginx | 确认 `nginx/nginx.exe` 存在；TLS 证书放入 `nginx/conf/ssl/`（开发可复制 `deploy/certs/edge/dev-server.crt` 与 `dev-server.key`） |

### 2. 编辑 config.ini（使用生产密码，勿使用示例占位符上线）

**PostgreSQL**（`postgresql/config.ini`）示例：

```ini
[PostgreSQL]
install_path = C:\Program Files\PostgreSQL\16
port = 5432
listen_bind = 127.0.0.1
run_mode = service

[Application]
user = freetv
password = changeme
db_edge = freetv_edge
db_cloud = freetv_cloud
```

**MinIO**（`minio/config.ini`）示例：

```ini
[MinIO]
api_port = 9000
console_port = 9001
listen_bind = 127.0.0.1
bucket = freetv-assets

[Credentials]
root_user = ycwlminio
root_password = changeme
```

**Redis**（`Redis/config.ini`）示例：

```ini
[Redis]
port = 6379
listen_bind = 127.0.0.1
data_path = .\data
appendonly = true
run_mode = background

[Credentials]
password =
```

**NanoMQ**（`nanomq/config.ini`）示例：

```ini
[NanoMQ]
mqtt_port = 1883
http_port = 8082
ws_port = 8083

[TenantAPI]
base_url = http://127.0.0.1:8443
check_health_before_start = true
```

查看解析结果（PowerShell）：

```powershell
cd deploy\run\postgresql
.\read-postgres-config.ps1 -PrintSummary

cd ..\minio
.\read-minio-config.ps1 -PrintSummary

cd ..\Redis
.\read-redis-config.ps1 -PrintSummary

cd ..\nanomq
.\read-nanomq-config.ps1 -PrintSummary
```

### 3. 以管理员运行启动脚本

首次启动会添加 Windows 防火墙入站规则（MinIO、NanoMQ、Nginx 等）。右键「以管理员身份运行」或在脚本内自动提权。

### 4. 按顺序启动并初始化

在仓库根目录 `bus` 下执行：

```bat
deploy\run\postgresql\start-postgres.bat
deploy\run\postgresql\init-db.bat tenant --schema
deploy\run\Redis\start-redis.bat
deploy\run\minio\start-minio.bat
deploy\run\minio\minio-mc.bat
REM 启动 tenant-api、tenant-web 后：
deploy\run\nanomq\start-nanomq.bat
deploy\run\nginx\start-nginx.bat
```

`start-minio.bat`、`start-postgres.bat` 会保持窗口打开；关闭窗口即停止进程。

### 5. 验证

| 组件 | 验证方式 |
|------|----------|
| PostgreSQL | `status-postgres.bat`；`psql-client.bat` |
| Redis | `status-redis.bat`；`redis-cli.bat` ping |
| MinIO | `status-minio.bat`；浏览器 `http://127.0.0.1:9001` |
| NanoMQ | `nanomq\bin\nanomq_cli.exe` 或 MQTT 客户端连 `tcp://127.0.0.1:1883` |
| Nginx | `https://127.0.0.1:8081/`；`reload-nginx.bat` 前先 `-t` 测试配置 |
| tenant-api | `curl http://127.0.0.1:8443/api/v1/health` |

---

## PostgreSQL

### 配置说明（config.ini）

| 段 | 键 | 说明 |
|----|-----|------|
| PostgreSQL | `install_path` | PG 根目录（含 `bin\psql.exe`）；`.` 表示本目录 |
| PostgreSQL | `data_path` | 数据目录（仅 `pg_ctl` 模式） |
| PostgreSQL | `port` / `listen_bind` | 监听端口与地址 |
| PostgreSQL | `run_mode` | `service` = Windows 服务；`pg_ctl` = 便携集群 |
| PostgreSQL | `windows_service_name` | 空则自动检测 `postgresql-x64-16` |
| Superuser | `user` / `password` | 超级用户（`init-db` 用）；本地 `pg_ctl` 可留空密码（trust） |
| Application | `user` / `password` | 应用用户，须与 tenant-api DSN 一致 |
| Application | `db_edge` / `db_cloud` | 数据库名 |

### 日常命令

| 命令 | 说明 |
|------|------|
| `start-postgres.bat` | 启动（service 或 pg_ctl） |
| `stop-postgres.bat` | 停止 |
| `status-postgres.bat` | 端口、pg_isready、集群状态 |
| `init-db.bat tenant --schema` | 建 EDGE 库并导入 schema |
| `init-db.bat all --schema` | EDGE + CLOUD |
| `psql-client.bat` | 交互式 psql |

`init-db.bat` 参数：`tenant` | `platform` | `all`，加 `--schema` 导入 SQL。

### 与 tenant-api 对接

`postgresql/config.ini` 中 `[Application]` 须与 tenant-api 一致：

```yaml
ycwl:
  db:
    driver: postgres
    dsn: postgres://freetv:changeme@127.0.0.1:5432/freetv_edge?sslmode=disable
```

或环境变量：

```bat
set YCWL_DB_DSN=postgres://freetv:changeme@127.0.0.1:5432/freetv_edge?sslmode=disable
```

### 常见问题

| 现象 | 处理 |
|------|------|
| `service` 模式启动失败 | 确认已安装 PostgreSQL 16 且服务名正确 |
| `pg_ctl` 首次启动慢 | 自动执行 `initdb` 并渲染 `postgresql.conf` / `pg_hba.conf` |
| `pg_isready` 超时 | 查看 `log/postgresql.log` |
| `init-db` 连接失败 | 确认 PostgreSQL 已启动；超级用户密码与 `config.ini` 一致 |

便携模式示例：

```ini
[PostgreSQL]
run_mode = pg_ctl
install_path = .
data_path = .\data
```

更多细节见 [postgresql/README.md](./postgresql/README.md)。

---

## Redis

> **可选组件**：tenant-api 默认 `redis.enabled=false`。启用缓存/会话存储时再启动本服务。

### 配置说明（config.ini）

| 段 | 键 | 说明 |
|----|-----|------|
| Redis | `install_path` | 安装根；`.` = 本目录 |
| Redis | `data_path` | RDB/AOF 数据目录 |
| Redis | `log_path` | 日志目录 |
| Redis | `port` / `listen_bind` | 监听端口与地址（默认 `127.0.0.1:6379`） |
| Redis | `appendonly` | AOF 持久化，与 `docker-compose.edge.yml` 一致 |
| Redis | `run_mode` | `background`（后台，同 nginx）或 `foreground`（保持窗口） |
| Redis | `add_firewall_rules` | 启动时添加入站规则 |
| Credentials | `password` | 空 = 无密码（本机 `127.0.0.1` 可接受）；生产建议设置 |

`start-redis.bat` 会调用 `render-redis-conf.ps1` 将 `redis.windows.conf` 渲染为 `conf/redis.conf`。

### 日常命令

| 命令 | 说明 |
|------|------|
| `start-redis.bat` | 渲染配置、防火墙、启动 `redis-server` |
| `stop-redis.bat` | `SHUTDOWN SAVE` 优雅停止，必要时强制结束 |
| `status-redis.bat` | 端口、PID、`PING` 健康检查 |
| `reload-redis.bat` | 重新渲染配置并重启（bind/port/密码变更须重启） |
| `redis-cli.bat` | 交互菜单或透传 redis-cli 参数 |

### 与 tenant-api 对接

```yaml
redis:
  enabled: true
  addr: 127.0.0.1:6379
  password: ""
  db: 0
```

### 常见问题

| 现象 | 处理 |
|------|------|
| 端口占用 | `netstat -ano \| findstr :6379` 排查 |
| 启动失败 | 查看 `Redis/logs/redis.log` |
| 修改密码后连不上 | `reload-redis.bat` 并同步 tenant-api `redis.password` |

更多细节见 [Redis/README.md](./Redis/README.md)。

---

## MinIO

### 配置说明（config.ini）

| 段 | 键 | 说明 |
|----|-----|------|
| MinIO | `install_path` | 安装根；`.` = 本目录 |
| MinIO | `data_path` | 数据目录 |
| MinIO | `api_port` / `console_port` | S3 API / Console 端口 |
| MinIO | `listen_bind` | `127.0.0.1` 仅本机；`0.0.0.0` 需配合防火墙 |
| MinIO | `bucket` | 默认 bucket（对应 `YCWL_STORAGE_BUCKET`） |
| MinIO | `add_firewall_rules` | 启动时添加入站规则 |
| Credentials | `root_user` / `root_password` | 根账号，须与 tenant-api 存储配置一致 |
| Client | `alias` / `endpoint` | mc 别名；`endpoint` 空则自动拼接 |

### 日常命令

| 命令 | 说明 |
|------|------|
| `start-minio.bat` | 启动（保持窗口） |
| `stop-minio.bat` | 按 API/Console 端口查找并结束 minio.exe |
| `status-minio.bat` | 端口与可执行文件状态 |
| `minio-mc.bat` | 配置 mc 别名；交互菜单创建 bucket、上传文件 |
| `minio-mc.bat ls ycwl-minio/` | 直接执行 mc 子命令 |

首次部署在 `minio-mc.bat` 菜单中选择 **3** 创建默认 bucket（如 `freetv-assets`）。

### 与 tenant-api 对接

```yaml
ycwl:
  storage:
    use_minio: true
    endpoint: http://127.0.0.1:9000
    bucket: freetv-assets
    access_key: ycwlminio
    secret_key: changeme
    public_url_prefix: https://127.0.0.1:8081/files
```

**重要：**

- `endpoint` 填 MinIO **直连**地址（`http://127.0.0.1:9000`），**不要**填 nginx URL。
- 经 nginx 对外提供素材时，`public_url_prefix` 使用统一入口，例如 `https://<主机>:8081/files`。
- 若 `data/` 已用旧密码初始化，改 `config.ini` 凭证后须删除 `data/` 重开，或保持原凭证。

环境变量对照：`YCWL_STORAGE_USE_MINIO`、`YCWL_STORAGE_ENDPOINT`、`YCWL_STORAGE_ACCESS_KEY`、`YCWL_STORAGE_SECRET_KEY`、`YCWL_STORAGE_PUBLIC_URL_PREFIX`。

### 常见问题

| 现象 | 处理 |
|------|------|
| 警告 `minioadmin:minioadmin` | 修改 `[Credentials]` 并重启 |
| 凭证错误 | 确认与 tenant-api 及已有 `data/` 初始化凭证一致 |
| 端口占用 | `netstat -ano | findstr :9000` 排查 |
| mc 连接失败 | 先 `start-minio.bat`，再 `minio-mc.bat` |

更多细节见 [minio/README.md](./minio/README.md)、[对象存储指南](../../docs/对象存储-MinIO与本地配置指南.md)。

---

## NanoMQ

### 配置说明（config.ini）

| 段 | 键 | 说明 |
|----|-----|------|
| NanoMQ | `mqtt_port` | MQTT TCP（默认 1883） |
| NanoMQ | `ssl_port` / `ssl_enabled` | MQTTS（NanoMQ 直连 TLS，生产常用 nginx :8883 代理） |
| NanoMQ | `http_port` | 本机 REST 管理（默认 8082，仅 127.0.0.1） |
| NanoMQ | `ws_port` | WebSocket（默认 8083） |
| NanoMQ | `allow_anonymous` | 须为 `false`（HTTP 认证） |
| TenantAPI | `base_url` | tenant-api 基址，用于 auth 回调 |
| TenantAPI | `check_health_before_start` | 启动前健康检查 |
| TLS | `cert_file` 等 | `ssl_enabled=true` 时使用 |

`start-nanomq.bat` 会调用 `render-nanomq-conf.ps1` 生成 `etc/nanomq.conf`，其中 `auth.http_auth` 回调：

- `POST {base_url}/api/v1/mqtt/auth`

模板中 `super_req`、`acl_req` 当前为注释状态；设备认证以 `auth` 端点为主。

### 日常命令

| 命令 | 说明 |
|------|------|
| `start-nanomq.bat` | 渲染配置并以 daemon 启动 |
| `stop-nanomq.bat` | `nanomq stop` |

手动渲染与启动：

```powershell
cd deploy\run\nanomq
.\render-nanomq-conf.ps1
.\bin\nanomq.exe start --conf .\etc\nanomq.conf --daemon
```

### 与 tenant-api 对接

```yaml
ycwl:
  mqtt:
    broker_type: nanomq
    broker_url: tcp://127.0.0.1:1883
```

环境变量：

```bat
set YCWL_MQTT_BROKER_TYPE=nanomq
set YCWL_MQTT_BROKER_URL=tcp://127.0.0.1:1883
set YCWL_MQTT_SERVER_USERNAME=gouser
set YCWL_MQTT_SERVER_PASSWORD=gouser@1234
```

对外 MQTTS（经 nginx 四层代理）：

```bat
set YCWL_MQTT_PUBLIC_BROKER_URL=ssl://<公网主机>:8883
set YCWL_MQTT_TLS_CA_FILE=certs\root_ca.crt
```

Hub 账号 `gouser` / `gouser@1234` 须与 tenant-api YAML 中 `server_username` / `server_password` 一致。

### 常见问题

| 现象 | 处理 |
|------|------|
| 设备无法连接 | 确认 tenant-api 已启动；检查 `etc/nanomq.conf` 中 auth URL |
| 与 EMQX 冲突 | 只保留一个 Broker；切换后重启 tenant-api |
| 修改端口 | 改 `config.ini` 后重新 `start-nanomq.bat`（会重新渲染） |

业务下行由 tenant-api **Hub 长连接**发布 `ycwl/.../down/*`，不经过 NanoMQ `http_server`。

更多细节见 [nanomq/README.md](./nanomq/README.md)、[NanoMQ 接入指南](../../docs/NanoMQ接入与HTTP认证配置指南.md)。

---

## Nginx

### 配置结构

| 文件 | 作用 |
|------|------|
| `conf/nginx.conf` | 主配置，include 反代与 stream |
| `conf/proxy/tenant.conf` | 租户入口 :8081（upstream tenant-api :8443、tenant-web :2001） |
| `conf/proxy/minio.conf` | MinIO 独立 HTTPS :9002 / :9003 |
| `conf/proxy/minio-locations.inc` | :8081 下 `/minio-console/`、`/minio-api/` |
| `conf/proxy/tenant-locations.inc` | :8081 下 `/api/`、`/files/`、静态页 |
| `conf/proxy/platform.conf` | 平台后台 :8080（可选） |
| `conf/stream.conf` | MQTTS :8883 → :1883，WSS :8084 → WS :8083 |
| `conf/ssl-params.inc` | 共用 TLS 参数 |
| `conf/ssl/dev-server.crt/key` | 默认开发证书 |

### :8081 路径对照

| 路径 | 反代目标 |
|------|----------|
| `/` | tenant-web :2001 |
| `/api/` | tenant-api :8443 |
| `/files/` | tenant-api `/files/`（素材代理） |
| `/minio-api/` | MinIO API :9000 |
| `/minio-console/` | MinIO Console :9001 |

独立端口：

| URL | 反代目标 |
|-----|----------|
| `https://<host>:9002/` | MinIO API :9000 |
| `https://<host>:9003/` | MinIO Console :9001 |
| `ssl://<host>:8883` | NanoMQ MQTT :1883（四层） |
| `wss://<host>:8084` | NanoMQ WS :8083（四层） |

`:80` 自动 301 跳转至 `https://<host>:8081`。

### TLS 证书

生产环境将正式证书替换 `conf/ssl/dev-server.crt` 与 `conf/ssl/dev-server.key`（或修改 `ssl-params.inc` 中的路径）。`stream.conf` 中 MQTTS/WSS 使用 `ssl/dev-server.crt/key`（相对 `conf/`）。

开发证书可从 `deploy/certs/edge/` 复制。

### 日常命令

| 命令 | 说明 |
|------|------|
| `start-nginx.bat` | 创建目录、防火墙、启动 nginx |
| `stop-nginx.bat` | 优雅停止，必要时强制结束 |
| `reload-nginx.bat` | `-t` 测试后 `reload` |

启动成功后的本机地址（与 `start-nginx.bat` 输出一致）：

- 租户入口：`https://127.0.0.1:8081/`
- MinIO API：`https://127.0.0.1:9002/`
- MinIO Console：`https://127.0.0.1:9003/`
- 平台后台（若启用）：`https://127.0.0.1:8080/`

### 修改 upstream 端口

若 tenant-api 非 :8443 或 tenant-web 非 :2001，编辑 `conf/proxy/tenant.conf`：

```nginx
upstream tenant_api {
    server 127.0.0.1:8443;
}
upstream tenant_web {
    server 127.0.0.1:2001;
}
```

修改后执行 `reload-nginx.bat`。

### 与对外地址对齐

启用 nginx 后，tenant-api 对外地址建议与 nginx 一致：

```bat
set SERVER_PUBLIC_URL=https://<主机>:8081
set YCWL_NGINX_PUBLIC_URL=https://<主机>:8081
set YCWL_STORAGE_PUBLIC_URL_PREFIX=https://<主机>:8081/files
```

参考 [deploy/.env.edge.example](../.env.edge.example)。

### 常见问题

| 现象 | 处理 |
|------|------|
| TLS 监听失败 | 检查 `conf/ssl/` 证书是否存在 |
| 502 / 空白页 | 确认 tenant-api :8443、tenant-web :2001 已启动 |
| MinIO 路径 502 | 确认 MinIO :9000/:9001 已启动 |
| 配置未生效 | `reload-nginx.bat`（内含 `-t`） |
| 外网无法访问 | 防火墙需开放 80、8081、8084、8883、9002、9003（`start-nginx.bat` 会尝试添加） |

日志：`nginx/logs/error.log`、`nginx/logs/access.log`。

---

## 日常运维速查

### 启动顺序

PostgreSQL → init-db（仅首装）→ Redis（可选）→ MinIO → tenant-api → tenant-web → NanoMQ → Nginx

### 停止顺序

Nginx → NanoMQ → tenant-web / tenant-api → MinIO → PostgreSQL

### 日志位置

| 组件 | 路径 |
|------|------|
| PostgreSQL | `postgresql/log/postgresql.log`（pg_ctl 模式） |
| Redis | `Redis/logs/redis.log` |
| MinIO | 启动窗口输出 |
| NanoMQ | `nanomq/log/` |
| Nginx | `nginx/logs/error.log`、`access.log` |

### 配置变更后动作

| 组件 | 动作 |
|------|------|
| PostgreSQL | `pg_ctl reload`（`start-postgres.bat` 集群已运行时自动 reload）或重启 |
| Redis | 修改 `config.ini` 后 `reload-redis.bat`（或 `stop-redis.bat` + `start-redis.bat`） |
| MinIO | 修改 `config.ini` 后 `stop-minio.bat` + `start-minio.bat` |
| NanoMQ | 修改 `config.ini` 后 `stop-nanomq.bat` + `start-nanomq.bat`（会重新渲染） |
| Nginx | 改 `conf/` 后 `reload-nginx.bat` |

---

## 安全与生产建议

1. **修改所有默认密码**：PostgreSQL 应用用户、MinIO root、Hub `gouser`、tenant-api `YCWL_JWT_SECRET`。
2. **监听地址**：默认 `127.0.0.1` 仅本机访问；需局域网访问时改 `listen_bind=0.0.0.0` 并限制防火墙源 IP。
3. **TLS**：替换开发证书；设备 MQTTS 使用正式 CA 或 step-ca 签发证书。
4. **对外 URL 一致**：`SERVER_PUBLIC_URL`、`YCWL_NGINX_PUBLIC_URL`、`YCWL_STORAGE_PUBLIC_URL_PREFIX` 与 nginx 实际对外 HTTPS 地址一致。
5. **数据备份**：定期备份 `postgresql/data`（pg_ctl）、`minio/data`、License 与配置文件。
6. **管理员权限**：仅首次添加防火墙规则需要管理员；日常 MinIO 窗口模式注意勿误关窗口导致服务停止。

---

## 相关文档

| 文档 | 说明 |
|------|------|
| [deploy/README.md](../README.md) | Compose、tenant-api 打包与整体部署 |
| [docs/部署指南-v1.0.0.md](../../docs/部署指南-v1.0.0.md) | 数据库迁移与 v1.0.0 发布流程 |
| [docs/对象存储-MinIO与本地配置指南.md](../../docs/对象存储-MinIO与本地配置指南.md) | MinIO、/files/ 代理 |
| [docs/NanoMQ接入与HTTP认证配置指南.md](../../docs/NanoMQ接入与HTTP认证配置指南.md) | NanoMQ HTTP 认证详解 |
| [docs/MQTT协议规范-设备通信.md](../../docs/MQTT协议规范-设备通信.md) | MQTT Topic 与载荷 |
| [postgresql/README.md](./postgresql/README.md) | PostgreSQL 补充（英文） |
| [Redis/README.md](./Redis/README.md) | Redis 补充（英文） |
| [minio/README.md](./minio/README.md) | MinIO 补充（英文） |
| [nanomq/README.md](./nanomq/README.md) | NanoMQ 补充 |

---

## Linux 说明

本目录 **Nginx** 提供 `nginx/start-nginx.sh`。**PostgreSQL** 在 Ubuntu 24.04 上可使用与 Windows 对齐的 shell 脚本：

```bash
cd deploy/run/postgresql
chmod +x *.sh
sudo ./install-postgres.sh          # 首次可选：apt 安装 PostgreSQL 16
./start-postgres.sh
./init-db.sh tenant --schema
./status-postgres.sh
./stop-postgres.sh
```

配置见 `config.linux.ini.example`（`run_mode=systemd` 或 `pg_ctl`）。详见 [postgresql/README.md](./postgresql/README.md)。

其他 Linux 裸机组件：`scripts/service-minio.sh`、`service-nanomq.sh`、`service-nginx.sh`；完整安装亦可使用 `scripts/service-postgres.sh`。
