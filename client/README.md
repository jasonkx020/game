# Cocos Creator 3 游戏客户端

> Platform SDK + 最小场景脚手架。架构见 [docs/tech/client-architecture.md](../docs/tech/client-architecture.md)。

## 前置

- Cocos Creator **3.8.x**
- Node.js 18+（运行 Vitest / 生成 proto）
- 后端 `make run-api` + `make run-game` 已启动

## 代码生成

```bash
# 仓库根目录
make gen-client-proto
```

生成 TS proto 到 `assets/platform/generated/pitaya/`。

## 单元测试（不依赖 Cocos 编辑器）

```bash
cd client
npm install
npm test
```

## Cocos 工程设置

1. 用 Cocos Creator 打开 **`client/`** 目录
2. 创建三个场景并命名：`Launch`、`Hall`、`Room`
3. 挂载脚本（`assets/scripts/scenes/`）：
   - **Launch**：`LaunchScene` + 两个 EditBox（手机号/验证码）+ 登录按钮 → `onLoginClick`
   - **Hall**：`HallScene` + Label（用户/房卡）+ 按钮 → `onQuickCreateRoom`
   - **Room**：`RoomScene` + Label（Push 日志）+ 占位按钮 → `onPassClick` / `onPlayClick`
4. Build Settings 将 **Launch** 设为起始场景

## 联调流程

1. `make up && make seed-dev`
2. `make run-api`（:8080）与 `make run-game`（:3250 WS）
3. Cocos 运行 Launch 场景
4. 默认 dev 登录：`13800000001` / `123456`
5. Hall → 快速开房 → Room 应看到 Pitaya Push 日志（`onRoomState`、`onDeal` 等）

## SDK 目录

```
assets/platform/sdk/
  ApiClient.ts       HTTP + HMAC
  PitayaClient.ts    WS + Pitaya 报文
  PitayaPacket.ts    Pomelo packet / message 编解码
  EventTracker.ts    action_seq 连续性 + sync
  GameSession.ts     进房闭环编排
  ReplayPlayer.ts    回放骨架（P2）
assets/games/dawugui/DawuguiPushHandler.ts
```

## 配置

开发环境默认对齐 `.env.example`：

| 项 | 默认 |
|----|------|
| API | `http://localhost:8080` |
| WS | `ws://localhost:3250` |
| APP_ID | `cocos-dev` |
| HMAC | `dev-hmac-secret-change-me` |

可在 Cocos 构建脚本中注入 `GAME_API_BASE` 等全局常量（见 `sdk/config.ts`）。
