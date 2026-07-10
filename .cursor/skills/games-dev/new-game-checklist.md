# 新游戏技术接入清单

基于 [docs/game/README.md](../../docs/game/README.md) 与 [ADR-002](../../docs/tech/adr/002-pluggable-game-constraints.md)。

参考实现：
- **dawugui**（打乌龟）— 完整范例：3~5 人扑克、playcards/pass
- **liuzichong**（六子冲）— 第二游戏：独立 Scene + Board

---

## 文档层（与 game-ops 协作）

```
- [ ] 编写规则 PRD：{gameId}.md（参考 dawugui.md）
- [ ] 填写运营挂点：docs/game/{gameId}/ops-hooks.md（§3 游戏金币经济必填）
- [ ] 在 运营手册.md 游戏层表格注册
- [ ] 同步 JSON 配置：config/ops-hooks/{gameId}.json
```

## 后端 — GameEngine

```
- [ ] 创建 internal/game/{gameId}/（engine.go, state.go, rules.go 等）
- [ ] 实现 engine.GameEngine 全部方法（Meta/NewState/ApplyAction/...）
- [ ] 定义 GameID 常量
- [ ] 编写 engine_test.go 覆盖核心规则
- [ ] 确认无 Pitaya/Gin/Redis import
- [ ] 在 internal/game/registry.go 注册 Get() case
```

## 后端 — Pitaya Handler

```
- [ ] 创建 internal/pitaya/handlers/{gameId}/handler.go
- [ ] 实现游戏 C2S 路由方法
- [ ] 调用 Engine + commitEvents() 落库与 Push
- [ ] 在 cmd/game/main.go pitaya.Register(..., WithName("{gameId}"))
- [ ] Push 含 audit_sn + action_seq
```

## Proto

```
- [ ] 新增 proto/pitaya/{gameId}.proto（不修改已有 message 字段编号）
- [ ] Route 命名：game.{gameId}.{method}
- [ ] 扩展 GameEvent oneof（见 ADR-005）
- [ ] make gen-proto（Go）
- [ ] make gen-client-proto（TS）
```

## 客户端（Cocos）

```
- [ ] client/assets/game/{gameId}/ — Scene 或 Board 组件
- [ ] {GameId}PushHandler.ts — 映射 Push → UI 事件
- [ ] HallScene 添加入口按钮（onCreate{GameId}Room）
- [ ] 如需独立场景：Cocos 编辑器创建场景并挂载脚本
- [ ] cd client && npm test 通过
```

## 平台层（通常不改）

以下路径 **禁止** 为新游戏单独修改（除非平台级变更）：

| 禁止改动 | 说明 |
| :--- | :--- |
| `internal/platform/wallet` | 房卡/游戏币逻辑 |
| `openapi/components/` 公共 schema | 除游戏 config 字段外 |
| `internal/pitaya/handlers/connector` | 除非平台级变更 |
| `internal/pitaya/handlers/room` | 除非房间流程变更 |

游戏目录与配置通过 `GET /v1/game/{id}/config` 读取 ops-hooks，无需改 wallet。

## 验证

```
- [ ] go test ./...
- [ ] make gen-proto && make gen-client-proto 无报错
- [ ] P0 流程：login → 开房 → WS entry/bind/join/ready → 游戏操作
- [ ] Push 日志含连续 action_seq
- [ ] 多客户端联调（Cocos 或手动 WS）
```

## 文件对照表

| 层 | 路径 | dawugui 范例 | liuzichong 范例 |
| :--- | :--- | :--- | :--- |
| 规则 PRD | `{gameId}.md` | dawugui.md | liuzichong.md |
| 运营挂点 | docs/game/{id}/ops-hooks.md | dawugui/ | liuzichong/ |
| Engine | internal/game/{id}/ | dawugui/ | liuzichong/ |
| Handler | internal/pitaya/handlers/{id}/ | dawugui/ | liuzichong/ |
| Proto | proto/pitaya/{id}.proto | dawugui.proto | liuzichong.proto |
| 客户端 | client/assets/game/{id}/ | DawuguiPushHandler.ts | LiuzichongScene.ts 等 |
| 配置 | config/ops-hooks/{id}.json | dawugui.json | — |
