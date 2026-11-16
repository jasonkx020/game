# HTTP 请求签名（auth-service）

所有 `/v1/*` 接口须携带 HMAC 签名头。业务服验 JWT；本服务额外验签名。

## 请求头

| Header | 必填 | 说明 |
| :--- | :--- | :--- |
| `X-App-Id` | 是 | 应用 ID |
| `X-Timestamp` | 是 | Unix 秒（UTC），偏差 ≤ 300s |
| `X-Nonce` | 是 | 单次 UUID，Redis 防重放 |
| `X-Content-SHA256` | 是 | Body SHA256 hex；空 Body 用 empty hash |
| `X-Signature` | 是 | HMAC-SHA256 hex |
| `Authorization` | 条件 | 需登录的接口：`Bearer {access_token}` |

空 Body SHA256：

```
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Canonical String

各字段以 `\n` 连接（无尾部分隔符）：

```
{HTTP_METHOD}
{REQUEST_PATH}
{CANONICAL_QUERY_STRING}
{X-Timestamp}
{X-Nonce}
{X-Content-SHA256}
{AUTHORIZATION_VALUE}
```

| 字段 | 规则 |
| :--- | :--- |
| `HTTP_METHOD` | 大写 |
| `REQUEST_PATH` | 如 `/v1/auth/login`（不含 host） |
| `CANONICAL_QUERY_STRING` | key 字典序 `key=urlencode(value)` 用 `&`；无 query 为空串 |
| `AUTHORIZATION_VALUE` | 完整 `Bearer ...`；无 Token 时为空串 |

```
X-Signature = hex_lower(HMAC-SHA256(app_secret, canonical_string))
```

## 业务服接入注意

- 登录/刷新/注册：**直连 auth-service**（默认 `:8081`），不要打到 game-api。
- 业务 API：带未过期 `access_token`，用 Auth **ES256 公钥**验签；`sub` = `account_id`。
- Refresh：**必须**同时上传过期也可的 `access_token` + 有效 `refresh_token`。
