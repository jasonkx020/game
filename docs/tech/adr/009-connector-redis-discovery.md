# ADR-009：Connector Redis 发现与按用户单点分配

| 项 | 值 |
| :--- | :--- |
| **Status** | Accepted |
| **Date** | 2026-11 |
| **Depends on** | ADR-001, ADR-004 |

---

## 背景

多 Connector 时需要发现与逃生，但：

- **auth-service** 必须保持纯鉴权，不维护节点表。
- 不可向客户端下发**全量** Connector 列表（避免盲探、放大探测负担）。
- 独立 registry 进程增加运维面，本阶段不引入。

---

## 决策

### 职责

| 组件 | 职责 |
| :--- | :--- |
| **auth-service** | 仅账号 / JWT / refresh；**零**发现职责 |
| **Connector**（MVP：`cmd/game` Standalone） | 启动即写 Redis、1s 刷新、TTL 3s、退出 DEL，并 Pub/Sub 通知 |
| **game-api** | 订阅维护**服务端内存全量表**；`GET /v1/discovery/assign` 按用户分配 **1 个** 节点 |
| **客户端** | 只连网关或当前分配节点；失败带 `exclude_id` 再 assign；禁止遍历全量 |

### 及时性

- 启动：就绪后立刻 `SET` + `PUBLISH`
- 心跳：间隔 **1s**，key **TTL 3s**
- 优雅退出：`DEL` + `PUBLISH`（近乎即时摘除）
- 崩溃：依赖 TTL（≤3s）从存活集消失
- assign **不做** TCP/WS 探活

### 选点

对 `user_id` 在当前存活集上做 **consistent hash**（粘性）。  
请求可带 `exclude_id`，在剩余存活集中再选。

### 双层逃生（客户端）

1. `ws_gateway_url`（若有）  
2. 分配到的 `connector.ws_url`  
3. 带 `exclude_id` 重新 assign（有限次）  
4. 本地 `defaultWsUrl` / 开房返回的 `ws_url` 降级  

### Cluster 预留

成长期拆 `cmd/connector` Frontend 后，由 Frontend 进程写 Redis；选点与 assign 契约不变。

---

## 明确不做

- 独立 registry 进程 / auth 发现 API  
- 向玩家下发 `connectors[]` 全量  
- 本 ADR 引入 etcd / NATS（见 ADR-001 成长期）

---

## 相关

- [platform-architecture.md](../platform-architecture.md)
- OpenAPI：`GET /v1/discovery/assign`
