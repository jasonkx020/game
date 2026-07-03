# Pitaya Route Protobuf

> WS 游戏协议 — 按 **Pitaya route** 拆分 proto3。  
> 已废弃 Envelope 主帧方案见 [../DEPRECATED.md](../DEPRECATED.md)。

---

## 1. 文件列表

| 文件 | 包 | 说明 |
| :--- | :--- | :--- |
| [common.proto](common.proto) | pitaya.common | EventMeta、PushHeader、公共枚举 |
| [event.proto](event.proto) | pitaya.event | GameEvent oneof（落库与回放） |
| [connector.proto](connector.proto) | pitaya.connector | entry、bind |
| [room.proto](room.proto) | pitaya.room | join、ready、leave、**sync** |
| [dawugui.proto](dawugui.proto) | pitaya.dawugui | 打乌龟 req/rsp/push |

---

## 2. Route 对照表

### Request / Response

| Route | Request | Response |
| :--- | :--- | :--- |
| `game.connector.entry` | EntryReq | EntryRsp |
| `game.connector.bind` | BindReq | BindRsp |
| `game.room.join` | JoinReq | JoinRsp |
| `game.room.ready` | ReadyReq | ReadyRsp |
| `game.room.leave` | LeaveReq | LeaveRsp |
| `game.room.sync` | SyncReq | SyncRsp |
| `game.dawugui.playcards` | PlayCardsReq | PlayCardsRsp（含 EventMeta） |
| `game.dawugui.pass` | PassReq | PassRsp（含 EventMeta） |

### Push（服务端→客户端）

| Route | Message | event_type |
| :--- | :--- | :--- |
| `onRoomState` | RoomStatePush | ROOM_STATE |
| `onDeal` | DealPush | DEAL |
| `onTurnNotify` | TurnNotifyPush | TURN |
| `onPlayResult` | PlayResultPush | PLAY / PASS |
| `onAlert` | AlertPush | ALERT |
| `onRoundInvalid` | RoundInvalidPush | ROUND_INVALID |
| `onSettlement` | SettlementPush | SETTLEMENT |
| `onError` | ErrorPush | — |

---

## 3. EventMeta / PushHeader（所有 Push 必含）

```protobuf
message EventMeta {
  uint64 audit_sn = 1;
  uint32 action_seq = 2;
  string round_id = 3;
  uint32 round_no = 4;
  string room_id = 5;
  int64 server_ts = 6;
}

message PushHeader {
  EventMeta meta = 1;
  string game_id = 2;
}
```

与 `game_action_log` 行 **一一对应**。详见 [audit-action-log.md](../../audit-action-log.md)。

---

## 4. GameEvent 与 Push 映射

| GameEvent oneof | Push Route | 备注 |
| :--- | :--- | :--- |
| `room_state` | `onRoomState` | |
| `deal` | `onDeal` | 按 seat 掩码手牌 |
| `turn` | `onTurnNotify` | |
| `play` / `pass` | `onPlayResult` | play_type 区分 |
| `alert` | `onAlert` | |
| `round_invalid` | `onRoundInvalid` | |
| `settlement` | `onSettlement` | 含 final_hands（局后） |

Handler 实现 `eventToPush()` 统一映射；Live 与 Replay 共用。

---

## 5. 代码生成

```bash
buf generate proto/pitaya
# Go → server/internal/gen/pitaya
# TS → client/assets/platform/generated/pitaya
```

---

## 6. 新游戏

1. 新增 `proto/pitaya/{gameId}.proto`（Push/Req）
2. 在 `event.proto` 或 `{gameId}` 扩展 `GameEvent` game_payload
3. 注册 Handler routes（[ADR-004](../../adr/004-pitaya-game-framework.md)）
4. 实现 `eventToPush` 映射表

---

## 7. 相关文档

| 文档 | 内容 |
| :--- | :--- |
| [audit-action-log.md](../../audit-action-log.md) | DDL、seq 规则 |
| [replay.md](../../replay.md) | HTTP 回放 API |
| [pitaya-client.md](../../pitaya-client.md) | TS 客户端 |
| [ADR-005](../../adr/005-ordered-action-log-replay.md) | 有序日志 ADR |
