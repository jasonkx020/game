# Cocos Creator 3 游戏客户端

> Platform SDK + 最小场景脚手架。架构见 [docs/tech/client-architecture.md](../docs/tech/client-architecture.md)。

## 前置

- Cocos Creator **3.8.8**（`package.json` 中 `creator.version` 已锁定；请用同版本编辑器打开 `client/`）
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

1. 安装 npm 依赖（proto 生成代码依赖 `@bufbuild/protobuf/wire`）：
   ```bash
   cd client && npm install
   ```
2. 用 Cocos Creator 打开 **`client/`** 目录（Import Map 已写入 `settings/v2/packages/project.json`，指向 `import-map.json`；若仍报模块找不到，请重启编辑器）
3. 创建场景并命名：`Launch`、`Lobby`、`Room`、**`Profile`（我的）**、**`Liuzichong`**（六子冲棋盘）
4. 挂载脚本（`assets/scripts/scenes/` 与 `assets/games/liuzichong/`）：
   - 设计分辨率：**720×1280 竖屏**（`settings/v2/packages/project.json`，`fitWidth`）
   - **Launch**：挂 `LaunchScene` 即可（默认 `autoBuildUI=true`，代码搭登录页：品牌头/手机号/验证码/登录按钮/社交占位/Toast）
     - 对接现有短信登录 API（验证码，非密码）；dev 默认 `13800000001` / `123456`
   - **Lobby**：挂 `LobbyScene` 即可（默认 `autoBuildUI=true`）
     - 运行时代码搭建：顶栏 / 游戏网格(3列) / 推荐 / 最近 / 伴侣底栏 / Toast / **我的覆盖层**
     - 若场景无 Canvas，会自动创建 UI Canvas（`UI_2D`）；代码节点必须挂 Canvas 下且 layer=`UI_2D` 才可见
     - 无需拖引用；旧手工 UI 自动隐藏；点头像→个人资料在覆盖层打开（不必单独建 Profile 场景）
   - **Profile**（可选）：仅当单独开 Profile 场景时挂 `ProfileScene`（同样 `autoBuildUI`）
   - **Room**：`RoomScene` + Label（Push 日志）；打乌龟可保留 `onPassClick` / `onPlayClick`
   - **Liuzichong**：`LiuzichongScene` 或 `LiuzichongBoard` + 可选 Label（`statusLabel` / `overlayLabel`）+ 返回按钮 → `onBackToHall`
5. Build Settings 将 **Launch** 设为起始场景（Lobby 内已含「我的」覆盖层，可不单独加 Profile 场景）

### 大厅 vs 个人页数据

| 位置 | 展示 |
|------|------|
| Lobby | 昵称、头像占位、房卡、游戏架、推荐、最近玩过（最多 3） |
| Profile | 脱敏手机、改昵称、房卡/游戏金币、充值流水、游戏偏好、设置、战绩列表、退出 |

## 游戏 Bundle（按需下载）

1. 在 Cocos 编辑器将 `assets/games/{gameId}/` 配置为 **Remote Bundle**
2. **构建发布 → Web Desktop**，产物在 `client/build/web-desktop/remote/{gameId}/`（含 `index.js`）
3. 同步到托管目录并起服务：

```powershell
.\scripts\sync-bundles.ps1          # remote → build/bundles，并生成 index.1.0.0.js
.\scripts\dev.ps1 serve-bundles     # :8787，自动 sync + CORS + JS MIME
```

4. 验证：浏览器打开 `http://localhost:8787/bundles/liuzichong/index.1.0.0.js`
5. 数据库 `game_client_bundle.bundle_url` 指向上述地址（migration 000009 已种子）

### Creator 预览（:7456）重要说明

从编辑器预览进入游戏时，页面源是 `http://localhost:7456`，大厅下发的 Bundle 在 `:8787`，属于**跨域**。  
Cocos `loadBundle` 会用带 `crossorigin` 的 `<script>` 拉远程包；含脚本的 Bundle 还会解析 `chunks://`，即使用 CORS 头也常失败并弹出系统 Error。

因此 `GameBundleManager` 在 **PREVIEW / EDITOR / DEV**（以及跨域 HTTP URL）下会：

1. **跳过** `http://localhost:8787/...` 跨域加载
2. 改为 `assetManager.loadBundle('liuzichong')` 加载工程内**本地 Asset Bundle**（含场景）
3. 再注册 `LiuzichongModule`

本地进六子冲不依赖 `:8787`，但必须按包名加载本地 Bundle，否则 `loadScene('Liuzichong')` 会失败。

远程 HTTP Bundle 用于**同域正式发布包**联调。

说明：`client/build/bundles/*/config.json` 若只有 `"Dev stub"` 字样，说明还没 sync，不是真正的 Bundle。

## 资源目录

```
assets/
├── platform/ui/              # 平台主包 UI（登录/大厅壳）
│   ├── common/               # 通用按钮、背景、Logo
│   └── lobby/icons/          # 大厅游戏列表本地占位图标（可选）
├── scripts/scenes/             # 平台场景 + 脚本
│   ├── Launch.scene / Lobby.scene / Room.scene
│   └── images/               # 平台场景专用图片
└── games/{gameId}/           # Remote Bundle（按需下载）
    ├── *.ts / *.scene        # 游戏逻辑与场景
    ├── images/               # 图标、小图
    ├── textures/             # 牌面、棋盘、背景
    ├── ui/                   # 游戏内 UI
    └── audio/                # 音效（可选）
```

新游戏：复制 `games/_template/` 目录结构，实现 `{GameId}Module.ts` 与 `GameEntry.ts`，并在编辑器中将 `games/{gameId}/` 设为 Bundle。

初始化或补全资源目录：

```bash
cd client && node scripts/scaffold-asset-dirs.mjs
```

## 联调流程

1. `make up && make seed-dev`
2. `make run-api`（:8080）与 `make run-game`（:3250 WS）
3. 可选：`make serve-bundles`（:8787）
4. Cocos 运行 Launch 场景
5. 默认 dev 登录：`13800000001` / `123456`
6. Lobby →「我的」进 Profile 查看脱敏手机/资产/战绩；或选游戏开房 → Room：
   - **打乌龟**：留在 Room，看 Push 日志
   - **六子冲**：自动进入 Liuzichong 场景，2 人 ready 后可走子

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
assets/games/liuzichong/LiuzichongPushHandler.ts
assets/games/liuzichong/LiuzichongBoard.ts
```

## 配置

开发环境默认对齐 `.env.example`：

| 项 | 默认 |
|----|------|
| API | `http://localhost:8080` |
| WS | `ws://localhost:3250` |
| APP_ID | `cocos-dev` |
| HMAC | `dev-hmac-secret-change-me` |
| CORS | 需含 Cocos 预览源，默认含 `http://localhost:7456`（见根目录 `.env` 的 `CORS_ORIGINS`） |

可在 Cocos 构建脚本中注入 `GAME_API_BASE` 等全局常量（见 `sdk/config.ts`）。
