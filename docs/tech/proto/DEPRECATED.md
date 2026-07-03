# 已废弃：Envelope 主帧协议

> **Status:** Deprecated（2026-07）  
> **替代方案：** [Pitaya route proto](pitaya/README.md) + [pitaya-client.md](../pitaya-client.md)

---

## 废弃内容

| 文件/方案 | 说明 |
| :--- | :--- |
| [common.proto](common.proto) | 统一 `Envelope` + oneof body |
| [game_ws.md](game_ws.md) | Envelope 消息时序 |
| [dawugui.md](dawugui.md) | Envelope 扩展说明 |
| 自定义帧 `[4字节长度][Envelope protobuf]` | 改为 Pitaya 官方报文格式 |

---

## 迁移对照

| 旧（Envelope） | 新（Pitaya） |
| :--- | :--- |
| C2S JoinRoom | Request `game.room.join` |
| C2S PlayCards | Request `game.dawugui.playcards` |
| S2C Deal | Push `onDeal` |
| S2C Settlement | Push `onSettlement` |
| Envelope.audit_sn | PushHeader.audit_sn |

---

## 相关 ADR

- [ADR-001](../adr/001-server-stack.md) — Pitaya 选型
- [ADR-004](../adr/004-pitaya-game-framework.md) — Handler/Route 表
