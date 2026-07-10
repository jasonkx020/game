---
name: game-ops
description: Guides operations and product documentation for the game platform: room-card vs score-field domains, currency binding model, modular docs under docs/, and ops-hooks templates. Use when editing 运营手册, ops-hooks, economy, risk, metrics, or game PRD in this repository.
---

# game 运营文档

打乌龟平台运营/产品文档编写约定。总索引：[运营手册.md](../../运营手册.md)。

## 文档体系

| 入口 | 用途 |
| :--- | :--- |
| [运营手册.md](../../运营手册.md) | 模块化总索引 |
| [docs/game/README.md](../../docs/game/README.md) | 新游戏文档接入 |
| [docs/platform/architecture.md](../../docs/platform/architecture.md) | 四层架构与模块边界 |

**版本**：V2.1（货币绑定模型：房卡=平台，游戏金币=游戏）

**适用对象**：运营团队、代理渠道、俱乐部管理员、产品经理

## 货币绑定模型

**不可混用**，编辑任何经济相关文档前先确认绑定关系：

| 货币 | 绑定 | 用途 | 文档位置 |
| :--- | :--- | :--- | :--- |
| **房卡** | 平台 | 全游戏通用；仅开房消耗，不可局内输赢 | [economy-base.md](../../docs/platform/economy-base.md) |
| **游戏金币** | 游戏 | 各游戏独立（龟币/冲币）；记分场专用；不可跨游戏、不可提现 | `docs/game/{id}/ops-hooks.md` §3 |

- 房卡定价、代理分成：沿用 platform 层，**新游戏无需修改**
- 游戏金币名称、充值比、场次、礼包：必须在 ops-hooks 中独立定义

## 场域划分

| 场域 | 运营主体 | 消耗货币 | 文档入口 |
| :--- | :--- | :--- | :--- |
| **房卡场** | 地推/代理/俱乐部 | 平台房卡（通用） | [docs/ops/room-card/](../../docs/ops/room-card/overview.md) |
| **记分场** | 平台官方 | 该游戏独立金币 + 抽水 | [docs/ops/score-field/](../../docs/ops/score-field/overview.md) |

商业模式：房卡场（地推）+ 记分场（官方）双轨并行。

## 标注规范

沿用运营手册约定：

- **【运营动作】** — 运营团队可直接执行的事项
- **【产品需求】** — 需产品/技术配合实现，文档仅描述业务要求

## 新游戏文档流程

1. 编写规则 PRD → `{gameId}.md`（参考 [dawugui.md](../../dawugui.md)）
2. 复制 [ops-hooks 模板](../../docs/game/_template/ops-hooks.template.md) → `docs/game/{gameId}/ops-hooks.md`
   - **§3 游戏金币经济为必填**
   - 房卡消耗可在 ops-hooks 中可选覆盖数量
3. 在 [运营手册.md](../../运营手册.md) 游戏层表格注册
4. 可选：按游戏逐步开放记分场

### 新游戏不需改的文档

| 层 | 是否修改 |
| :--- | :--- |
| `docs/game/{id}/ops-hooks.md` | **是** |
| `docs/platform/` | **否** — 房卡定价不变 |
| `docs/ops/room-card/` | **否** |
| `docs/ops/score-field/` | **否** — 引用 game 层参数 |
| `docs/risk/` | **否** |

### 已接入游戏

| gameId | 规则 PRD | ops-hooks | 金币名称 |
| :--- | :--- | :--- | :--- |
| dawugui | dawugui.md | docs/game/dawugui/ | 龟币 |
| liuzichong | liuzichong.md | docs/game/liuzichong/ | 冲币 |

## 术语表（精简）

完整版见运营手册 §术语表。

| 术语 | 定义 |
| :--- | :--- |
| **底分** | 记分场场次倍数；规则积分 × 底分 = 游戏金币变动 |
| **抽水** | 平台从赢家净赢中抽取 5%（全平台统一） |
| **audit_sn** | 全局唯一操作 ID；申诉与风控定位 |
| **action_seq** | 局内事件序号；回放与断线补发 |
| **战绩回放** | 局结束后查看完整对局，见 [replay-ops.md](../../docs/ops/shared/replay-ops.md) |
| **报单/包牌/无效局** | 打乌龟特殊规则，见 [dawugui/ops-hooks.md](../../docs/game/dawugui/ops-hooks.md) |

## 与技术文档边界

| 视角 | 文档 | 职责 |
| :--- | :--- | :--- |
| 运营/产品 | `docs/platform/`、`docs/ops/`、`docs/game/`、`docs/risk/` | 业务要求、SOP、KPI |
| 技术实现 | `docs/tech/` | 架构、协议、API、DDL |

- ops-hooks Markdown 描述业务参数；技术同步至 `config/ops-hooks/{gameId}.json` 与 PG `game_ops_config`
- 不要在运营文档中写实现细节（Handler/Proto 等），技术接入见 game-dev skill

## 常见编辑场景

**调整房卡定价或代理分成**
→ 仅改 [economy-base.md](../../docs/platform/economy-base.md)，不动 game 层

**新增记分场活动（签到/锦标赛）**
→ [score-field/activities.md](../../docs/ops/score-field/activities.md) + 对应游戏 ops-hooks

**俱乐部运营 SOP**
→ [room-card/sop.md](../../docs/ops/room-card/sop.md)、[agent-club.md](../../docs/ops/room-card/agent-club.md)

**风控话术或合规**
→ [risk/compliance.md](../../docs/risk/compliance.md)

**KPI 与盈利测算**
→ [metrics.md](../../docs/platform/metrics.md)

**战绩申诉流程**
→ [replay-ops.md](../../docs/ops/shared/replay-ops.md)

## 模块导航

完整速查表见 [doc-map.md](doc-map.md)。

## 收入结构参考

收入来源：房卡销售（平台）、记分场抽水（按游戏）、桌费/服务费、道具装扮、VIP 订阅。

成本：代理分成、服务器带宽、支付通道费、推广补贴。

详见运营手册 §收入结构概览。
