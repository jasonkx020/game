# 独立 Auth 服务

## 职责

| 进程 | 职责 |
| :--- | :--- |
| `auth-service` (`:8081`) | sign-in / 微信 / 实名 / refresh / logout / me |
| `platform-api` / Pitaya | 仅校验**未过期** access JWT；懒链接本地 `players` / `admin_users` |

共享 `JWT_SECRET`、`HMAC_SECRET`、`APP_ID`。

## 玩家登录（sign-in）

统一入口：无号建号、有号登录，响应带 `is_new_user`。

- `POST /v1/auth/send-code`：`{channel: phone|email, target, purpose: signin}`（`register`/`login` 为兼容别名；不预检账号存在性）
- `POST /v1/auth/sign-in`：`{channel, target, code, nickname?}`（兼容 `phone`/`sms_code`/`email`）
- 旧路径 `POST /v1/auth/register`、`POST /v1/auth/login`（玩家）内部转调 sign-in，已 deprecated
- Admin：`POST /v1/admin/auth/login` **不变**（账号必须已存在，不自动建号）

Dev：`DEV_SMS_CODE` / `DEV_EMAIL_CODE` 可直接通过校验。

## 微信登录

客户端判断渠道，服务端适配：

```json
POST /v1/auth/oauth/wechat
{ "platform": "web" | "miniprogram", "code": "..." }
```

- `miniprogram` → `jscode2session`
- `web` → OAuth access_token + userinfo
- 按 `union_id` → `open_id+platform` 绑定；无账户则自动建号
- `WECHAT_MODE=mock` 时用稳定假 openid（本地可测）

## 实名（阿里云二要素）

```json
POST /v1/auth/realname/verify
Authorization: Bearer <access>
{ "real_name": "张三", "id_card_no": "..." }
```

- 生产：`KYC_MODE=live` + `ALIYUN_ACCESS_KEY_*` → `Id2MetaVerify`
- Dev：`KYC_MODE=mock`（身份证校验位合法即通过；姓名填 `拒绝` 强制不一致）
- **不落库明文证件号**；仅 hash / 脱敏字段；`GET /v1/auth/me` 返回 `realname_status`

## Refresh（双 Token）

Body 必须同时携带可过期的 `access_token` 与未过期的 `refresh_token`，且主体一致后轮换（新旧 refresh 都会换发）。

**轮换宽限**（`REFRESH_GRACE_SECONDS`，默认 120s）：新 refresh 已写入后，上一枚 refresh 在短窗内仍可再换一对。用于客户端网络中断、未落盘新 token 时重试，避免被踢去重新登录。客户端应在 refresh 失败/超时时用**同一对旧 token**重试；成功后再落盘新的 access+refresh。

Refresh 只存 SHA-256 hash（明文 32 字节随机）；`token_hash` 唯一约束冲突时服务端会重生 opaque 再写。

## OpenAPI 契约

其他项目接入请直接使用独立契约包：

- 入口：[auth-service/openapi/openapi.yaml](../../auth-service/openapi/openapi.yaml)
- 说明：[auth-service/openapi/README.md](../../auth-service/openapi/README.md)
- 签名：[auth-service/openapi/http-signature.md](../../auth-service/openapi/http-signature.md)
- **后台在线文档**：运营后台 `http://localhost:5173/docs`（免登录；含协议 / OpenAPI / 接入指南）

平台总览 `docs/tech/openapi/openapi.yaml` 仅引用 Auth 路径作索引；**以 auth-service/openapi 为准**。

## 本地

```bash
cp auth-service/.env.example auth-service/.env
make run-auth
```
