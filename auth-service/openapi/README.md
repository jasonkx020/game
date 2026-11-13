# Auth Service OpenAPI

其他项目接入登录体系时，以本目录为**权威契约**：

| 文件 | 说明 |
| :--- | :--- |
| [openapi.yaml](./openapi.yaml) | OpenAPI 3.0 入口（可导入 Swagger / Insomnia / codegen） |
| [http-signature.md](./http-signature.md) | HMAC 签名算法 |
| [paths/](./paths/) | 接口路径 |
| [components/](./components/) | Schema / 参数 / 响应 / 安全定义 |

## 快速接入

1. **直连 auth-api**（默认 `http://localhost:8081`），不要把 sign-in/refresh 打到业务 API。
2. 所有 `/v1/*` 请求按 [http-signature.md](./http-signature.md) 签名（共享 `APP_ID` / `HMAC_SECRET`）。
3. 登录成功后保存 `access_token` + `refresh_token`；关注 `is_new_user`；业务请求只带未过期 access。
4. Refresh 时 **同时**上传 access（可过期）与 refresh。
5. 业务服用共享 `JWT_SECRET` 验 HS256 JWT；`sub` = `account_id`。
6. Admin 用 `/v1/admin/auth/login`（账号须已存在，不自动建号）。

## 推荐调用顺序

```text
send-code(purpose=signin) → sign-in
  或 oauth/wechat
→（可选）realname/verify
→ 业务 API（Bearer access）
→ access 将过期：refresh（双 Token）
→ logout
```

旧路径 `/v1/auth/register`、`/v1/auth/login`（玩家）仍可用，内部转调 sign-in，OpenAPI 已标 deprecated。

## 在线查看（推荐）

启动运营后台后打开：

- 免登录：`http://localhost:5173/docs`（文档中心：协议 · OpenAPI · 接入指南）
- 登录后侧栏：**协议与接入文档** → `/api-docs`

可在侧栏打开 Auth/平台 OpenAPI（Swagger）、HTTP 签名、Pitaya/Proto 协议与接入指南全文。

## 导入示例

```bash
# Swagger UI / Redoc / openapi-generator 等指向：
#   auth-service/openapi/openapi.yaml
```

本地服务：`go run ./cmd/auth-api`（先 `go run ./cmd/migrate -path migrations`）。
