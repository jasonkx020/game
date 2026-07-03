# 记分场专属活动（官方）

> 由平台官方运营。活动奖励中的**游戏金币**按当前游戏发放；**房卡**为平台通用。  
> 房卡场活动见 [room-card/activities.md](../room-card/activities.md)。

---

## 1. 周期性活动

| 周期 | 活动 | 内容 | 目的 |
| :--- | :--- | :--- | :--- |
| **每日** | 签到 | 发放**当前游戏金币**（数量见该游戏 ops-hooks） | 日活 |
| **节点** | 包牌王排行榜 | 游戏专属规则排名（如打乌龟包牌） | 传播 + 充值 |

跨场域房卡任务见 [room-card/activities.md](../room-card/activities.md)。

---

## 2. 每日签到

| 项目 | 规则 |
| :--- | :--- |
| 奖励币种 | **当前游戏的独立金币**（非平台统一币） |
| 产出控制 | 遵循该游戏 ops-hooks 中的平衡目标 |

**打乌龟示例：** 普通 200 龟币/日，VIP 400 龟币/日 — 见 [dawugui/ops-hooks.md](../../games/dawugui/ops-hooks.md) §2.3。

---

## 3. 包牌王排行榜

| 项目 | 规则 |
| :--- | :--- |
| 时间 | 春节 7 天 / 国庆 7 天 |
| 排名依据 | 活动期间包牌次数（规则因游戏而异） |
| 奖励 | 房卡（平台通用）+ 可选游戏金币 |
| 传播 | 每日推送 Top 10 |

**打乌龟：** 规则与奖励见 [dawugui/ops-hooks.md](../../games/dawugui/ops-hooks.md) §8。

---

## 4. 周末锦标赛

详见 [shared/value-added.md](../shared/value-added.md)。

| 项目 | 规则 |
| :--- | :--- |
| 报名费 | 以**该游戏金币**支付，金额 = 底分 × 20（底分见 ops-hooks） |
| 平台抽成 | 报名费的 10% |

【运营动作】锦标赛前 3 天推送报名提醒。

---

## 5. 游戏专属传播活动

依赖具体游戏规则，在 [games/{id}/ops-hooks.md](../../games/README.md) 中定义挂点。

**打乌龟示例：** 最快出完榜、零失误榜 — 见 [dawugui/ops-hooks.md](../../games/dawugui/ops-hooks.md) §8。
