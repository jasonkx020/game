# Protocol Buffers — WebSocket 游戏协议

> **当前方案：** [Pitaya route proto](pitaya/README.md)  
> **已废弃：** [DEPRECATED.md](DEPRECATED.md)（Envelope 主帧）

---

## 使用 Pitaya Route Proto

游戏 WS 走 Pitaya 报文 + 按 route 拆分的 proto：

- [pitaya/common.proto](pitaya/common.proto) — PushHeader
- [pitaya/connector.proto](pitaya/connector.proto)
- [pitaya/room.proto](pitaya/room.proto)
- [pitaya/dawugui.proto](pitaya/dawugui.proto)

文档：[pitaya/README.md](pitaya/README.md)

---

## 废弃文件（勿新引用）

| 文件 | 说明 |
| :--- | :--- |
| [common.proto](common.proto) | 旧 Envelope 方案 |
| [game_ws.md](game_ws.md) | 旧消息时序 |
| [dawugui.md](dawugui.md) | 旧 payload 说明 |

迁移对照见 [DEPRECATED.md](DEPRECATED.md)。

---

## 相关文档

| 文档 | 内容 |
| :--- | :--- |
| [protocol.md](../protocol.md) | 协议总览 |
| [pitaya-client.md](../pitaya-client.md) | Cocos 客户端 |
| [adr/004-pitaya-game-framework.md](../adr/004-pitaya-game-framework.md) | Route 表 |
