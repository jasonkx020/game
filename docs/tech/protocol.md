# 通信协议总览

> **HTTP OpenAPI 3.0** + **Pitaya WebSocket Protobuf** 双通道。  
> 旧 Envelope 方案已废弃，见 [proto/DEPRECATED.md](proto/DEPRECATED.md)。

---

## 1. 通道划分

| 通道 | 框架 | 序列化 | 用途 |
| :--- | :--- | :--- | :--- |
| HTTP REST | Gin + **OpenAPI 3.0.3** | JSON | 登录、钱包、开房、俱乐部 |
| WebSocket | **Pitaya** | **Protobuf** | 进房、准备、出牌、Push 广播 |

---

## 2. HTTP REST（OpenAPI 3.0）

契约：[openapi/openapi.yaml](openapi/openapi.yaml)

| 项 | 约定 |
| :--- | :--- |
| 鉴权 | JWT Bearer |
| 签名 | HMAC-SHA256（[http-signature.md](openapi/http-signature.md)） |
| 幂等 | `Idempotency-Key` |
| 错误 | `ApiError` |

代码生成：`oapi-codegen` → Gin。

---

## 3. Pitaya WebSocket

### 3.1 报文

- 传输：WebSocket binary
- 格式：**Pitaya 官方 packet**（type + id + route + protobuf data）
- 序列化：`pitaya.SetSerializer(protobuf.NewSerializer())`

详见 [pitaya-client.md](pitaya-client.md)。

### 3.2 Route 与 Proto

按 route 拆分 proto：[proto/pitaya/](proto/pitaya/README.md)

| 示例 Route | 类型 |
| :--- | :--- |
| `game.connector.entry` | Request |
| `game.room.join` | Request |
| `game.room.sync` | Request |
| `game.dawugui.playcards` | Request |
| `onDeal` | Push |
| `onSettlement` | Push |

### 3.3 EventMeta / PushHeader

所有 Push 含 `EventMeta`：`audit_sn`, `action_seq`, `round_id`, `round_no`, `room_id`, `server_ts`；外加 `game_id`。

与 `game_action_log` 一 event 一 log。详见 [audit-action-log.md](audit-action-log.md)。

---

## 4. 回放与补发

| 通道 | 用途 |
| :--- | :--- |
| HTTP `/v1/rounds/{id}/replay` | 局后战绩回放（全员手牌） |
| HTTP `/v1/rooms/{id}/replay` | 整房多局串联 |
| Pitaya `game.room.sync` | 断线 since_action_seq 补发 |

详见 [replay.md](replay.md)、[openapi/paths/replay.yaml](openapi/paths/replay.yaml)。

---

## 5. 进房闭环

1. HTTP login → JWT
2. HTTP create room → `room_id`, `ws_url`
3. PitayaClient connect → `game.connector.entry`
4. `game.connector.bind` → session.Bind
5. `game.room.join` → Group
6. `game.dawugui.*` / Push 开始对局

---

## 6. 原则

| 原则 | 说明 |
| :--- | :--- |
| 职责分离 | 非实时 HTTP；实时 Pitaya |
| 服务端权威 | Engine 校验，Push 为准 |
| 有序日志 | 每条 event 有 action_seq + audit_sn |
| 先写后推 | INSERT 先于 GroupBroadcast |
| 增游戏 | 新 route + proto + GameEvent，不改 OpenAPI 公共部分 |

---

## 7. HTTP 错误码（节选）

| code | 含义 |
| :--- | :--- |
| 1001 | Token 失效 |
| 1002 | 签名失败 |
| 1003 | 时间戳过期 |
| 1004 | Nonce 重放 |
| 2001 | 房卡不足 |

Pitaya Push 错误：`onError`（ErrorPush）。

---

## 8. 相关文档

| 文档 | 内容 |
| :--- | :--- |
| [replay.md](replay.md) | 回放 API |
| [audit-action-log.md](audit-action-log.md) | 日志 DDL |
| [openapi/README.md](openapi/README.md) | HTTP |
| [proto/pitaya/README.md](proto/pitaya/README.md) | WS proto |
| [pitaya-client.md](pitaya-client.md) | Cocos 客户端 |
| [adr/004-pitaya-game-framework.md](adr/004-pitaya-game-framework.md) | Route 表 |
