# 新游戏技术接入清单

完整说明见 **[docs/tech/new-game-guide.md](../../docs/tech/new-game-guide.md)**。  
运营侧：[docs/games/README.md](../../docs/games/README.md)。约束：[ADR-002](../../docs/tech/adr/002-pluggable-game-constraints.md)。

参考：`dawugui`、`liuzichong`。单进程 `cmd/game` 注册，不另起 WS。

---

## 文档层

```
- [ ] 规则 PRD：{gameId}.md
- [ ] docs/games/{gameId}/ops-hooks.md（§游戏金币经济必填）
- [ ] 运营手册.md 游戏索引
- [ ] （可选）CODEOWNERS @game-{id}
```

## 后端 — GameEngine 公共面 + internal

```
- [ ] server/internal/game/{id}/factory.go（GameID / New / NewBotPicker）
- [ ] server/internal/game/{id}/internal/rules/ 实现 Engine + 单测
- [ ] （推荐）TurnAware + BotPicker（托管/bot 代打）
- [ ] （按需）Phased / Snapshotter / SpectatorViewer；Action.Opcode+Payload；Seed/Rank/TeamID/Draw
- [ ] 无 Pitaya/Gin；不 import 其他游戏
- [ ] registry.go：Get + Bot case
```

## 后端 — Handler / Proto / 种子

```
- [ ] handlers/{id}/handler.go + commitEvents；可调 bot.RunPending
- [ ] cmd/game Register WithName("{id}")
- [ ] proto/pitaya/{id}.proto；按需扩展 event.proto（不改旧字段号）
- [ ] make gen-proto && make gen-client-proto
- [ ] migration：game_catalog + game_ops_config + game_client_bundle
```

## 客户端

```
- [ ] client/assets/games/{id}/ Module + PushHandler + Bundle 入口
- [ ] GameModuleRegistry；不改 Lobby 硬编码
- [ ] cd client && npm test
```

## 验证

```
- [ ] make lint（check-boundaries）
- [ ] go test ./...
- [ ] P0：login → 大厅 → 开房 → WS join/ready → game.{id}.* → 结算
```

## 禁止（默认可不改）

| 路径 | 说明 |
| :--- | :--- |
| `platform/wallet`、openapi 公共 schema | 平台货币/契约 |
| `handlers/connector` | 平台连接 |
| `handlers/room` / `game/engine` | 须 [room-change-checklist](../../docs/tech/room-change-checklist.md) |
| 其他 `internal/game/{other}/` | 跨游戏互引 |
