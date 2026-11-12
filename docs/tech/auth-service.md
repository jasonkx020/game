# 独立 Auth 服务

## 职责

| 进程 | 职责 |
| :--- | :--- |
| `auth-service` (`:8081`) | 注册 / 登录 / 微信 / 实名 / refresh / logout / me |
| `platform-api` / Pitaya | 仅校验**未过期** access JWT；懒链接本地 `players` / `admin_users` |

共享 `JWT_SECRET`、`HMAC_SECRET`、`APP_ID`。

## 注册与登录

- `POST /v1/auth/send-code`：`{channel: phone|email, target, purpose: register|login}`
- `POST /v1/auth/register`：验证码建号（已存在 → 409）
- `POST /v1/auth/login`：**不再隐式建号**；需已有账号 + 验证码（兼容 `phone`/`sms_code` 字段）

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

Body 必须同时携带可过期的 `access_token` 与未过期的 `refresh_token`，且主体一致后轮换。

## 本地

```bash
cp auth-service/.env.example auth-service/.env
make run-auth
```
