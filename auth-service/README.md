# auth-service

独立登录与 Token 服务。客户端直连本进程完成 sign-in / 微信 / 实名 / refresh；GameS 等业务服仅用共享 `JWT_SECRET` 校验未过期的 access JWT。

## 端口与接口

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| POST | `/v1/auth/send-code` | 手机/邮箱验证码（推荐 purpose=`signin`） |
| POST | `/v1/auth/sign-in` | 玩家统一入口：无号建号、有号登录 → TokenPair + `is_new_user` |
| POST | `/v1/auth/register` | **deprecated** → 转调 sign-in |
| POST | `/v1/auth/login` | **deprecated**（玩家）→ 转调 sign-in |
| POST | `/v1/auth/oauth/wechat` | `{platform:web\|miniprogram, code}`；无号自动建号 |
| POST | `/v1/auth/realname/verify` | Bearer；阿里云二要素实名 |
| GET | `/v1/auth/me` | Bearer；基础资料（含实名状态） |
| POST | `/v1/auth/refresh` | **双 Token** 刷新 |
| POST | `/v1/auth/logout` | 注销 refresh |
| GET | `/v1/auth/introspect` | 校验未过期 access |
| POST | `/v1/admin/auth/login` | 管理员登录（须已有账号，不建号） |
| GET | `/healthz` | 健康检查 |

HMAC 签名头与 GameS `/v1` 一致。

单端口对外时由 nginx 分流 `/v1/auth`、`/v1/admin/auth` → 本服务（**勿剥路径**），见 [docs/tech/gateway.md](../docs/tech/gateway.md)。

## OpenAPI（其他项目接入）

权威契约目录：[openapi/](./openapi/)（入口 `openapi/openapi.yaml`）。

微信 OAuth 表归属与修改约束：[docs/tech/oauth.md](../docs/tech/oauth.md)、仓库规则 `.cursor/rules/oauth.mdc`。

## 本地

```bash
cp .env.example .env
go run ./cmd/migrate -path migrations
go run ./cmd/auth-api
```

迁移使用独立表 `auth_schema_migrations`（可与 GameS 共用同一 Postgres，互不踩版本号）。  
若出现 `Dirty database version`：先确认是否误连了游戏库的 `schema_migrations`；用 `go run ./cmd/migrate -path migrations -force <V>` 清 dirty。

默认 `WECHAT_MODE=mock`、`KYC_MODE=mock`、`DEV_SMS_CODE=123456`。

## 测试

```bash
go test ./...
```
