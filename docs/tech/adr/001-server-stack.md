# ADR-001：服务端技术栈

| 项 | 值 |
| :--- | :--- |
| **Status** | Accepted |
| **Date** | 2026-07 |
| **Context** | 棋牌多游戏平台；P0 房卡闭环；HTTP OpenAPI + 实时 WS；audit_sn 审计 |

---

## 决策

### 双通道架构

| 通道 | 框架 | 用途 |
| :--- | :--- | :--- |
| **HTTP REST** | **Gin** + OpenAPI 3.0 + HMAC | 登录、钱包、开房、俱乐部、配置 |
| **游戏实时** | **[Pitaya v2](https://github.com/topfreegames/pitaya) Standalone** | WS 连接、Session、Handler、Groups、Push |

Pitaya **不替代** Gin；平台业务与游戏实时分层部署（MVP 可同机两进程）。

### 锁定选型

| 类别 | 选型 |
| :--- | :--- |
| 语言 | Go 1.22+ |
| HTTP 框架 | **Gin** |
| HTTP 契约 | OpenAPI 3.0.3 + **oapi-codegen** |
| HTTP 签名 | HMAC-SHA256（见 [http-signature.md](../openapi/http-signature.md)） |
| JWT | golang-jwt/jwt/v5 |
| **游戏框架** | **Pitaya v2 Standalone**（MVP 无 etcd/NATS） |
| Pitaya 序列化 | **Protobuf**（`pitaya.SetSerializer(protobuf.NewSerializer())`） |
| Pitaya 传输 | WebSocket Acceptor（内置） |
| PG 驱动 | jackc/pgx/v5 |
| 数据访问 | **sqlx** |
| 迁移 | golang-migrate/migrate |
| 缓存 | redis/go-redis/v9 |
| 配置 | spf13/viper + ops-hooks JSONB |
| 日志 | log/slog |
| audit_sn | Snowflake 自研（41+10+12） |
| 持久化 | PostgreSQL 16+ |

### 进程划分（MVP）

| 进程 | 入口 | 职责 |
| :--- | :--- | :--- |
| `cmd/platform-api` | Gin | OpenAPI REST |
| `cmd/game` | Pitaya Standalone | WS + Handler + GameEngine |

---

## 弃选方案

| 弃选 | 原因 |
| :--- | :--- |
| Fiber | 非 net/http；OpenAPI 生态弱于 Gin |
| GORM | 钱包账本需显式 SQL；多游戏 JSONB 配置不适合 ORM 耦合 |
| 自研 WS Gateway + nhooyr | Pitaya 已提供 Session/Groups/Route/Push |
| Pitaya Cluster（MVP） | 运维复杂；Standalone 满足 P0，成长期再切 |
| 统一 Envelope 自定义帧 | 改为 Pitaya 报文 + 按 route 拆分 proto |

---

## 多游戏扩展性评估

| 维度 | Gin + Pitaya | 自研 Gateway |
| :--- | :--- | :--- |
| 增游戏改平台 HTTP | **否** | 否 |
| 增游戏 WS 接入 | **Register Handler 组件** | 改网关路由 |
| 房间广播 | **Groups API** | 自研广播 |
| Session 绑定 | **内置 Bind** | 自研 |
| 客户端 SDK | 自研 PitayaClient TS | 自研 Envelope |

---

## 后果

**正面：**

- Pitaya Groups/Session/Pipeline 减少自研网关代码
- Gin 与 Pitaya 职责清晰，增游戏主要改 Handler + Engine
- Standalone MVP 部署简单

**负面：**

- Cocos 无官方 Pitaya TS SDK，需自研 [pitaya-client.md](../pitaya-client.md)
- 成长期切 Cluster 需引入 NATS + etcd
- Handler 方法需遵循 Pitaya 签名约定

---

## 依赖清单（实施期参考）

```
github.com/gin-gonic/gin
github.com/topfreegames/pitaya/v2
github.com/jackc/pgx/v5
github.com/jmoiron/sqlx
github.com/redis/go-redis/v9
github.com/spf13/viper
github.com/golang-jwt/jwt/v5
google.golang.org/protobuf
```

---

## 相关文档

| 文档 | 内容 |
| :--- | :--- |
| [004-pitaya-game-framework.md](004-pitaya-game-framework.md) | Pitaya 游戏层 |
| [002-pluggable-game-constraints.md](002-pluggable-game-constraints.md) | 接入约束 |
| [platform-architecture.md](../platform-architecture.md) | 总体架构 |
