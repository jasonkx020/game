# Admin（运营后台）

独立工程：Vue 前端 **嵌入** `admin-api`（默认 `:8082`，API + SPA 同端口）。

**硬约束**：`admin-api` 停服不影响玩家登录、大厅与对局。游戏关键路径仅为 `auth-service` + `game-api` + `game`。

## 本地启动

```powershell
# 游戏三件套（可不启 admin）
.\scripts\dev.ps1 run-auth
.\scripts\dev.ps1 run-game-api
.\scripts\dev.ps1 run-game

# 运营后台（构建前端并启动；浏览器打开 http://localhost:8082/）
.\scripts\dev.ps1 run-admin

# 可选：Vite HMR（需另开 run-admin-api）
.\scripts\dev.ps1 run-admin-api
.\scripts\dev.ps1 run-admin-web   # :5173
```

Admin 登录走 auth-service `:8081`（`/v1/admin/auth/*`）。auth CORS 需包含 `http://localhost:8082`。

### 用户管控（platform_admin）

| 动作 | API |
| :--- | :--- |
| 封禁/解封玩家 | `POST /v1/admin/players/:id/ban` \| `unban` |
| 踢下线 | `POST /v1/admin/players/:id/kick` |
| 新建管理员 | `POST /v1/admin/users`（Auth 建号 + `admin_users` 双写） |
| 管理员详情/审计 | `GET /v1/admin/users/:id` |
| 改昵称 | `PATCH /v1/admin/users/:id` |
| 停用/启用管理员 | `POST /v1/admin/users/:id/disable` \| `enable` |
| 改角色 | `PATCH /v1/admin/users/:id/role`（先同步 Auth `auth_accounts.role`，再本地） |
| 强制下线 | `POST /v1/admin/users/:id/logout`（Auth `revoke-sessions`） |

封禁会同步 Auth `disabled`、吊销 refresh、镜像游戏侧 status，并经 Redis 通知 game 踢线。管理员新建/改角色须与 Auth 一致，否则 JWT `role` 会与列表漂移。迁移：`server/migrations/000014_moderation_actions`。

后续若要独立部署前端：去掉 `webui.Mount`，改用 CDN/Nginx 即可，API 契约不变。
