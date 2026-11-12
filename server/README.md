# GameS Server

Go 后端：`platform-api`（Gin HTTP）+ `game`（Pitaya WS）。

```
cmd/              入口
internal/         业务与引擎
migrations/       PostgreSQL
proto/            Protobuf（生成到 internal/gen 与 ../client）
web/admin/        运营后台
config/ops-hooks/ 游戏运营配置
deploy/           Docker / compose
```

## 常用命令

在仓库根目录：

```powershell
.\scripts\dev.ps1 run-api
.\scripts\dev.ps1 run-game
```

或在本目录：

```bash
cp .env.example .env   # 首次
make up && make seed-dev
make run-api
make run-game
```

环境变量文件为 **本目录 `.env`**（与 `go.mod` 同级）。
