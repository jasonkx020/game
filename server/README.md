# server — 游戏平台

双进程：`game-api`（:8080）+ `game`（:3250）。**不含运营后台**（见仓库根目录 [`admin/`](../admin/)）。

```
cmd/game-api/              玩家/大厅 HTTP（不依赖 admin-api）
cmd/game/                  Pitaya WS（单进程注册全部游戏）
cmd/check-boundaries/      游戏 import 边界检查
pkg/                       可导出共享领域包（admin-api 通过 replace 引用）
internal/platform/         房间/大厅/companion/replay 等
internal/game/engine/      GameEngine / BotPicker / TurnAware
internal/game/{id}/        游戏公共面（GameID, New, NewBotPicker）
internal/game/{id}/internal/  游戏私有实现（其他包不可引用）
internal/game/registry.go  组合根：Get / Bot
internal/pitaya/handlers/  connector / room / {gameId}
migrations/
deploy/
```

## 增游戏

白名单见 [ADR-002](../docs/tech/adr/002-pluggable-game-constraints.md)：规则包 + handler + `registry` + `cmd/game` Register + proto。  
游戏之间**禁止**互相 import；bot 走 `engine.BotPicker`；改公共 room 须 [影响评估](../docs/tech/room-change-checklist.md)。

```bash
make lint   # go run ./cmd/check-boundaries（+ 可选 golangci-lint）
make test
```

本地启动见根 [README.md](../README.md)。运营：`.\scripts\dev.ps1 run-admin-api` + `run-admin`。
