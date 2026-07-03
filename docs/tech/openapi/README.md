# OpenAPI 3.0 HTTP API 规范

> HTTP REST 契约，遵循 **OpenAPI 3.0.3**。总览见 [protocol.md](../protocol.md)。

---

## 1. 目录结构

```
openapi/
├── README.md              # 本说明
├── http-signature.md      # HMAC-SHA256 请求签名规范
├── openapi.yaml           # 根入口（$ref 聚合）
├── paths/
│   ├── auth.yaml
│   ├── user.yaml
│   ├── wallet.yaml
│   ├── room.yaml
│   ├── club.yaml
│   ├── game.yaml
│   └── replay.yaml        # 战绩回放、审计查询
└── components/
    ├── schemas.yaml
    ├── parameters.yaml    # 签名 Header 参数
    ├── responses.yaml
    └── securitySchemes.yaml
```

---

## 2. 约定

| 项 | 说明 |
| :--- | :--- |
| 版本 | `openapi: 3.0.3` |
| 路径前缀 | `/v1/` |
| Content-Type | `application/json; charset=utf-8` |
| 鉴权 | JWT Bearer + **HMAC 签名校验**（见 [http-signature.md](http-signature.md)） |
| 签名 | Header `X-App-Id`、`X-Timestamp`、`X-Nonce`、`X-Content-SHA256`、`X-Signature` |
| 幂等 | 写操作 Header `Idempotency-Key: {uuid}` |
| 错误 | 统一 `ApiError` schema |

---

## 3. 校验与生成

```bash
# 校验
npx @apidevtools/swagger-cli validate openapi.yaml
# 或
npx @redocly/cli lint openapi.yaml

# Go 生成
oapi-codegen -package api -generate types,server openapi.yaml

# TypeScript 生成
openapi-generator-cli generate -i openapi.yaml -g typescript-fetch -o ../../../client/assets/platform/generated/api
```

**Design First：** 修改 YAML 后再对齐 Handler 实现。

---

## 4. 安全

见 [components/securitySchemes.yaml](components/securitySchemes.yaml)：

- `bearerAuth`：HTTP Bearer JWT（login 之后）
- `signatureAuth`：HMAC-SHA256 请求签名（**所有接口**，含 login）

签名算法、规范字符串、防重放规则详见 **[http-signature.md](http-signature.md)**。

---

## 5. 公共 Schema

见 [components/schemas.yaml](components/schemas.yaml)：

- `ApiError`
- `PageResult`
- `WalletBalance`
- `RoomSummary`
- `GameConfig`
- `RoundReplayResponse`、`MatchSummary`、`ActionLogEntry`（回放）

---

## 6. MVP 路径清单

| 文件 | 路径 |
| :--- | :--- |
| auth.yaml | POST `/v1/auth/login`, POST `/v1/auth/refresh` |
| user.yaml | GET `/v1/user/profile` |
| wallet.yaml | GET `/v1/wallet/room-card`, GET `/v1/wallet/game-coin/{game_id}` |
| room.yaml | POST `/v1/rooms`, GET `/v1/rooms/{room_id}`, POST `/v1/rooms/{room_id}/join` |
| club.yaml | GET `/v1/clubs/{club_id}` |
| game.yaml | GET `/v1/games`, GET `/v1/games/{game_id}/config` |
| replay.yaml | GET `/v1/users/me/matches`, GET `/v1/rounds/{round_id}/replay`, GET `/v1/rooms/{room_id}/replay`, GET `/v1/rounds/{round_id}/events`, GET `/v1/admin/audit/actions` |

---

## 7. audit_sn、action_seq 与 HTTP

资金/敏感写操作响应可携带 `audit_sn`，与 `wallet_ledger.audit_sn` 一致。回放 API 返回 `events[].action_seq` 有序列表，详见 [replay.md](../replay.md)。

---

## 8. 相关文档

| 文档 | 内容 |
| :--- | :--- |
| [replay.md](../replay.md) | 回放 API 说明 |
| [protocol.md](../protocol.md) | 双通道总览 |
| [platform-architecture.md](../platform-architecture.md) | 服务模块 |
| [economy-base.md](../../platform/economy-base.md) | 房卡经济 |
