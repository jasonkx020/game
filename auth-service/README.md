# auth-service

独立登录与 Token 服务。客户端直连本进程完成注册 / 登录 / 微信 / 实名 / refresh；GameS 等业务服仅用共享 `JWT_SECRET` 校验未过期的 access JWT。

## 端口与接口

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| POST | `/v1/auth/send-code` | 手机/邮箱验证码（register\|login） |
| POST | `/v1/auth/register` | 验证码注册 → TokenPair |
| POST | `/v1/auth/login` | **已有账号**登录（phone\|email + code） |
| POST | `/v1/auth/oauth/wechat` | `{platform:web\|miniprogram, code}` 自动建号 |
| POST | `/v1/auth/realname/verify` | Bearer；阿里云二要素实名 |
| GET | `/v1/auth/me` | Bearer；基础资料（含实名状态） |
| POST | `/v1/auth/refresh` | **双 Token** 刷新 |
| POST | `/v1/auth/logout` | 注销 refresh |
| GET | `/v1/auth/introspect` | 校验未过期 access |
| POST | `/v1/admin/auth/login` | 管理员登录 |
| GET | `/healthz` | 健康检查 |

HMAC 签名头与 GameS `/v1` 一致。

## 本地

```bash
cp .env.example .env
go run ./cmd/migrate -path migrations
go run ./cmd/auth-api
```

默认 `WECHAT_MODE=mock`、`KYC_MODE=mock`、`DEV_SMS_CODE=123456`。

## 测试

```bash
go test ./...
```
