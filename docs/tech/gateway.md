# HTTP 网关（单端口分流）

对外只暴露 **一个 HTTP 端口**（本地默认 `:8090`，生产可用 `:80`/`:443`），按路径前缀转发到 auth / game-api / admin-api。**不改**业务 URI，**禁止**剥 `/v1` 前缀（HMAC 按完整 path 验签）。

## 分流

| 前缀 | 上游 | 说明 |
| :--- | :--- | :--- |
| `/v1/admin/auth` | auth-api `:8081` | 运营登录 / refresh / 账号管控 |
| `/v1/auth` | auth-api `:8081` | 玩家 sign-in、OAuth 微信、实名等 |
| `/v1/admin` | admin-api `:8082` | 运营业务 API |
| `/v1/` | game-api `:8080` | 玩家大厅 / 钱包 / 开房等 |
| `/`（含 `/docs`） | admin-api `:8082` | 运营 SPA |

WebSocket 对局仍走 `PITAYA_WS_URL`（默认 `:3250`），不经本 HTTP 网关。

## 本地

```powershell
# 先起三服务
.\scripts\dev.ps1 run-auth
.\scripts\dev.ps1 run-game-api
.\scripts\dev.ps1 run-admin

# 再起网关
docker compose -f deploy/docker-compose.gateway.yml up -d
# 浏览器: http://localhost:8090/
```

配置文件：[deploy/nginx/gateway.conf](../../deploy/nginx/gateway.conf)（本机 nginx）、[gateway.docker.conf](../../deploy/nginx/gateway.docker.conf)（Compose）。

## 前端同源

运营后台将 `VITE_API_BASE` / `VITE_AUTH_BASE` 留空，请求打到**当前页面源**（网关或 Vite）。  
Vite HMR（`:5173`）已 proxy：`/v1/admin/auth`、`/v1/auth` → `:8081`，其余 `/v1` → `:8082`。

直连多端口调试时可设：

```
VITE_AUTH_BASE=http://localhost:8081
```

此时 auth CORS 需包含前端源（如 `http://localhost:8082`）。走网关同源后 CORS 可收敛为网关域名。

## 验收速查

```text
POST /v1/auth/sign-in              → auth
POST /v1/admin/auth/login|refresh  → auth
GET  /v1/admin/users               → admin
GET  /v1/lobby/games               → game-api
GET  /                             → admin SPA
```
