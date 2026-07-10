# 运营文档模块导航

从 [运营手册.md](../../运营手册.md) 提炼，按层快速定位应编辑的文件。

---

## 平台层（房卡与平台 KPI）

| 文档 | 路径 | 内容 |
| :--- | :--- | :--- |
| 架构 | [docs/platform/architecture.md](../../docs/platform/architecture.md) | 四层架构、货币绑定、模块边界 |
| 房卡经济 | [docs/platform/economy-base.md](../../docs/platform/economy-base.md) | 平台房卡定价、默认消耗、代理分成 |
| KPI | [docs/platform/metrics.md](../../docs/platform/metrics.md) | KPI、盈利测算、看板字段 |

---

## 运营层 — 房卡场（地推）

| 文档 | 路径 | 内容 |
| :--- | :--- | :--- |
| 总览 | [docs/ops/room-card/overview.md](../../docs/ops/room-card/overview.md) | 定位、Phase 1 目标、MVP 闭环 |
| 代理俱乐部 | [docs/ops/room-card/agent-club.md](../../docs/ops/room-card/agent-club.md) | 代理/俱乐部分成、KPI |
| SOP | [docs/ops/room-card/sop.md](../../docs/ops/room-card/sop.md) | 俱乐部孵化 SOP |
| 活动 | [docs/ops/room-card/activities.md](../../docs/ops/room-card/activities.md) | 俱乐部榜、代理返利 |

---

## 运营层 — 记分场（官方）

| 文档 | 路径 | 内容 |
| :--- | :--- | :--- |
| 总览 | [docs/ops/score-field/overview.md](../../docs/ops/score-field/overview.md) | 官方定位、游戏金币隔离 |
| 抽水 | [docs/ops/score-field/tiers-rake.md](../../docs/ops/score-field/tiers-rake.md) | 抽水框架（通用 5%） |
| 匹配破产 | [docs/ops/score-field/matching-bankruptcy.md](../../docs/ops/score-field/matching-bankruptcy.md) | 匹配、破产保护框架 |
| 活动 | [docs/ops/score-field/activities.md](../../docs/ops/score-field/activities.md) | 签到、锦标赛（按游戏金币） |

---

## 运营层 — 共用

| 文档 | 路径 | 内容 |
| :--- | :--- | :--- |
| 漏斗 | [docs/ops/shared/funnel.md](../../docs/ops/shared/funnel.md) | 新手转化漏斗 |
| 增值 | [docs/ops/shared/value-added.md](../../docs/ops/shared/value-added.md) | VIP、赛季通行证（Phase 3） |
| 回放申诉 | [docs/ops/shared/replay-ops.md](../../docs/ops/shared/replay-ops.md) | 战绩回放、申诉 SOP |
| 日志保留 | [docs/ops/shared/replay-retention.md](../../docs/ops/shared/replay-retention.md) | 对局日志保留策略 |

---

## 风控层（横切）

| 文档 | 路径 | 内容 |
| :--- | :--- | :--- |
| 合规 | [docs/risk/compliance.md](../../docs/risk/compliance.md) | 话术、房卡/游戏金币规则 |
| 反欺诈 | [docs/risk/anti-fraud.md](../../docs/risk/anti-fraud.md) | IP/GPS、异常输赢 |

---

## 游戏层

| 文档 | 路径 | 内容 |
| :--- | :--- | :--- |
| 接入指南 | [docs/game/README.md](../../docs/game/README.md) | 新游戏文档四步 |
| 打乌龟挂点 | [docs/game/dawugui/ops-hooks.md](../../docs/game/dawugui/ops-hooks.md) | 龟币经济、房卡消耗、规则挂点 |
| 打乌龟 PRD | [dawugui.md](../../dawugui.md) | 游戏规则 |
| 六子冲挂点 | [docs/game/liuzichong/ops-hooks.md](../../docs/game/liuzichong/ops-hooks.md) | 冲币经济 |
| 六子冲 PRD | [liuzichong.md](../../liuzichong.md) | 游戏规则 |
| 模板 | [docs/game/_template/ops-hooks.template.md](../../docs/game/_template/ops-hooks.template.md) | 新游戏 ops-hooks 模板 |

---

## 技术层（实现参考，非运营编辑主区）

| 文档 | 路径 | 内容 |
| :--- | :--- | :--- |
| 技术索引 | [docs/tech/README.md](../../docs/tech/README.md) | 阅读顺序 |
| ADR | [docs/tech/adr/README.md](../../docs/tech/adr/README.md) | 架构决策 |
| 平台架构 | [docs/tech/platform-architecture.md](../../docs/tech/platform-architecture.md) | Gin + Pitaya |
| 协议 | [docs/tech/protocol.md](../../docs/tech/protocol.md) | HTTP + Pitaya WS |
| OpenAPI | [docs/tech/openapi/openapi.yaml](../../docs/tech/openapi/openapi.yaml) | HTTP API 契约 |
| 回放 API | [docs/tech/replay.md](../../docs/tech/replay.md) | 战绩回放 |
| 审计日志 | [docs/tech/audit-action-log.md](../../docs/tech/audit-action-log.md) | action_seq DDL |

---

## 运营后台（P1）

| 项 | 路径/说明 |
| :--- | :--- |
| 代码 | [web/admin/](../../web/admin/) |
| 启动 | `make run-admin`（需先 `make run-api`） |
| API 契约 | [docs/tech/openapi/](../../docs/tech/openapi/openapi.yaml) |

---

## 原方案待办对照

| 原待办 | 模块化文档 |
| :--- | :--- |
| economy-design | economy-base.md + game/{id}/ops-hooks.md |
| agent-system | room-card/agent-club.md |
| room-card-mvp | room-card/overview.md |
| game-features | game/{id}/ops-hooks.md |
| 记分场 | score-field/ + game 层金币 |
| risk-control | risk/compliance.md |
| ops-activities | room-card/activities.md + score-field/activities.md |
| metrics-dashboard | platform/metrics.md |
