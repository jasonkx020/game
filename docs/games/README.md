# 新游戏接入指南

> 游戏层 — 可插拔模块。接入新游戏无需修改 platform 房卡定价与 ops/risk 主体文档。

---

## 1. 接入四步

| 步骤 | 动作 | 产出 |
| :--- | :--- | :--- |
| 1 | 编写游戏规则 PRD | `{gameId}.md`（如 [dawugui.md](../../dawugui.md)） |
| 2 | 填写运营挂点（**必填游戏金币经济 + 房卡消耗**） | [games/{gameId}/ops-hooks.md](dawugui/ops-hooks.md) |
| 3 | 在总索引注册 | [运营手册.md](../../运营手册.md) 游戏索引表 |
| 4 | 开放记分场（可选） | 按游戏逐步上线官方记分场 |
| 5 | 技术接入（开发） | Pitaya Handler + GameEngine + PitayaClient，见 [tech/adr/](docs/tech/adr/README.md) |

---

## 2. 货币职责

| 货币 | 绑定 | 新游戏需配置的位置 |
| :--- | :--- | :--- |
| **房卡** | 平台（全游戏通用） | ops-hooks §房卡消耗（可选覆盖数量） |
| **游戏金币** | 游戏（独立币种） | ops-hooks §游戏金币经济（**必填**） |

- 房卡定价、代理分成：沿用 [platform/economy-base.md](../platform/economy-base.md)，无需修改
- 游戏金币名称、充值比、场次、礼包、签到：必须在 ops-hooks 中独立定义

---

## 3. 文档职责边界

| 层 | 新游戏是否需要修改 |
| :--- | :--- |
| games/{id}/ops-hooks.md | **是** — 游戏金币 + 房卡消耗 + 规则挂点 |
| platform/ | **否** — 房卡定价不变 |
| ops/room-card/ | **否** |
| ops/score-field/ | **否** — 引用 games 层参数即可 |
| risk/ | **否** |

---

## 4. 模板

复制 [_template/ops-hooks.template.md](_template/ops-hooks.template.md) 至 `games/{gameId}/ops-hooks.md` 并填写，**§3 游戏金币经济为必填**。

---

## 5. 已接入游戏

| 游戏 ID | 规则 PRD | 运营挂点 | 金币名称 |
| :--- | :--- | :--- | :--- |
| dawugui | [dawugui.md](../../dawugui.md) | [dawugui/ops-hooks.md](dawugui/ops-hooks.md) | 龟币 |

---

## 6. 架构说明

| 视角 | 文档 |
| :--- | :--- |
| 运营/模块边界 | [platform/architecture.md](../platform/architecture.md) |
| 技术实现 | [tech/README.md](../tech/README.md) |
