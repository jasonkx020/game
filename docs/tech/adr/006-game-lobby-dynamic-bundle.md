# ADR-006：游戏大厅与动态 Bundle

| 项 | 值 |
| :--- | :--- |
| **Status** | Accepted |
| **Date** | 2026-07 |
| **Depends on** | ADR-002, ADR-004 |

---

## 背景

客户端原先在 `HallScene` 硬编码游戏入口，所有游戏资源随主包编译。随着游戏数量增加，需要：

1. **平台游戏大厅**：API 驱动列表，用户可隐藏/置顶/排序
2. **按需下载**：每游戏独立 Cocos Remote Bundle，主包仅保留平台壳
3. **用户资料扩展**：`users` 表补充头像、状态、登录时间等字段

---

## 决策

### 客户端分层

| 层 | 路径 | 职责 |
| :--- | :--- | :--- |
| 平台壳 | `platform/sdk/`、`platform/lobby/`、`scripts/scenes/Launch|Lobby|Room` | HTTP/WS、大厅、进房 |
| 游戏包 | `assets/games/{id}/`（Remote Bundle） | Scene、PushHandler、GameEntry |
| 注册表 | `GameModuleRegistry` | `gameId` → `registerPush` / `entryScene` |

`RoomScene` **禁止**静态 import 具体游戏；通过 `GameModuleRegistry` 调用。

### Bundle 加载

- 元数据存 `game_client_bundle` 表，经 `GET /v1/lobby/games` 下发
- 客户端 `GameBundleManager.ensureLoaded()`：`assetManager.loadBundle(url)` → 失败则 `import()` 内置模块（开发兜底）
- MVP 本地静态服务：`make serve-bundles`（:8787）

### 用户偏好

- 表 `user_game_prefs`：`visible` / `pinned` / `sort_order`
- 无记录时默认展示全部 `enabled` 游戏
- `PUT /v1/lobby/games` 批量更新

### 用户表扩展

`users` 增：`avatar_url`、`status`、`updated_at`、`last_login_at`、`settings`（JSONB）

---

## 包依赖（不变）

仍遵守 ADR-002：`internal/platform` 不 import 具体游戏引擎。

---

## API

| 方法 | 路径 |
| :--- | :--- |
| GET | `/v1/lobby/games` |
| PUT | `/v1/lobby/games` |

---

## 不在本期

- 服务端记录客户端安装状态
- 生产 CDN/OSS 发布流水线（字段已预留）
- Bundle 差分热更新

---

## 相关文档

| 文档 | 内容 |
| :--- | :--- |
| [client-architecture.md](../client-architecture.md) | 大厅与 Bundle 流程 |
| [client/README.md](../../../client/README.md) | Cocos 构建与场景挂载 |
