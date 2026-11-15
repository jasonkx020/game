# 公共 room / engine 变更影响评估

`handlers/room`、`pitaya/runtime`、`pitaya/commit`、`game/engine` 是**全游戏共用**路径。默认禁止随某一游戏需求顺手改；若必须改，PR 中勾选本清单，并取得 CODEOWNERS `@platform-room` 批准（见仓库根 [CODEOWNERS](../../CODEOWNERS)）。

## 提交前自检

- [ ] 能否改成「只动 `internal/game/{gameId}/` 或对应 handler」而不改 room/engine？
- [ ] 是否影响**所有** `game_id` 的 join / ready / leave / tick / 结算路径？
- [ ] 是否迫使各游戏 Handler 或 Engine 同步改接口（破坏面列表）？
- [ ] 若扩展 `engine.GameEngine` / `BotPicker` / `TurnAware`：是否已更新 [ADR-002](adr/002-pluggable-game-constraints.md)？
- [ ] 旧客户端 route / Push 是否兼容？不兼容时的回滚方案？
- [ ] 是否补充或更新跨游戏回归（至少 dawugui + liuzichong 开局/一局流程）？

## PR 描述建议段落

```markdown
### Room / engine impact
- Why room/engine (not game package): …
- Games affected: dawugui, liuzichong, …
- Interface changes: …
- Rollback: …
```

相关约束见 ADR-002「禁止改动」表。

## 已落地的平台能力（勿在游戏包重复造）

旁观 / 托管 / 重连 Join 幂等 / Sync / Session 离线标记 — 见 ADR-002「平台 room 钩子」与 [new-game-guide](new-game-guide.md)。游戏侧用可选接口接入即可。
