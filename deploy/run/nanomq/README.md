# NanoMQ Windows 本地启动包

与 Linux `scripts/service-nanomq.sh` 对齐，用于开发/EDGE 节点在 **Windows** 上运行 NanoMQ，并通过 tenant-api `auth_http` 完成设备与 Hub 认证。

## 前置条件

1. 将 [NanoMQ](https://nanomq.io/downloads) Windows 发行包解压到本目录（或 `config.ini` 中 `install_path` 指向的路径），目录结构示例：

   ```
   deploy/run/nanomq/
     bin/nanomq.exe
     config.ini
     read-nanomq-config.ps1
     nanomq.conf.template
     render-nanomq-conf.ps1
     start-nanomq.bat
   ```

2. tenant-api 已启动，且配置：

   ```yaml
   ycwl:
     mqtt:
       broker_type: nanomq   # 或环境变量 YCWL_MQTT_BROKER_TYPE=nanomq
       broker_url: tcp://127.0.0.1:1883
   ```

3. Hub 账号与 YAML 中 `server_username` / `server_password` 一致（默认 `gouser` / `gouser@1234`）。

## 配置说明（config.ini）

| 段 | 键 | 说明 |
|----|-----|------|
| NanoMQ | `mqtt_port` | MQTT TCP 端口（默认 1883） |
| NanoMQ | `http_port` | 本机管理 REST（默认 8081，仅 127.0.0.1） |
| NanoMQ | `log_dir` | 日志目录（相对 `install_path`） |
| NanoMQ | `add_firewall_rules` | 启动时添加入站防火墙规则（默认 true） |
| TenantAPI | `base_url` | tenant-api 基址，用于 auth/super/acl 回调 |
| TenantAPI | `check_health_before_start` | 启动前探测 `/api/v1/health` |
| TLS | `cert_file` 等 | 启用 `ssl_enabled` 时填写 MQTTS 证书 |

`read-nanomq-config.ps1` 一次性解析 `config.ini`；`render-nanomq-conf.ps1` 根据模板生成 `etc/nanomq.conf`，内含 `auth.http_auth` 三端点。

查看解析结果：

```powershell
.\read-nanomq-config.ps1 -PrintSummary
```

## 启动

以管理员运行：

```bat
start-nanomq.bat
```

或手动渲染后启动：

```powershell
.\render-nanomq-conf.ps1
.\bin\nanomq.exe start --conf .\etc\nanomq.conf --daemon
```

## 端口对照

| 服务 | 默认端口 | 用途 |
|------|----------|------|
| NanoMQ MQTT | 1883 | 设备/Hub 连接 |
| NanoMQ MQTTS | 8883 | 生产 TLS（需配置证书） |
| NanoMQ http_server | 8081 | 本机运维 REST，**非**业务下行 |
| tenant-api | 8443 | HTTP 认证回调 |

业务下行仍由 tenant-api **Hub 长连接**发布 `ycwl/.../down/*`，不经过 NanoMQ `http_server`。

## 与 EMQX 切换

同一 EDGE 节点请勿同时连接两个 Broker。切换步骤：

1. 停止当前 Broker（EMQX 或 NanoMQ）
2. 修改 `YCWL_MQTT_BROKER_TYPE` 与 `YCWL_MQTT_BROKER_URL`
3. 重启 tenant-api
4. 启动目标 Broker

详见 [`docs/NanoMQ接入与HTTP认证配置指南.md`](../../../docs/NanoMQ接入与HTTP认证配置指南.md)。
