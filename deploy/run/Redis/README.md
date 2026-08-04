# Redis (Windows portable)

YCWL Windows launcher for [Redis for Windows](https://github.com/tporadowski/redis) (tporadowski port).

## Quick start

1. Edit `config.ini` (port, password, paths).
2. Run `start-redis.bat` as administrator (first run adds firewall rule).
3. Verify with `status-redis.bat` or `redis-cli.bat`.

## Scripts

| Script | Description |
|--------|-------------|
| `start-redis.bat` | Render `conf/redis.conf`, start `redis-server` |
| `stop-redis.bat` | Graceful `SHUTDOWN SAVE`, then force kill if needed |
| `status-redis.bat` | Port, PID, PING health |
| `reload-redis.bat` | Re-render config and restart (bind/port/password need restart) |
| `redis-cli.bat` | Interactive client menu or pass-through args |

## Configuration (`config.ini`)

| Section | Key | Description |
|---------|-----|-------------|
| Redis | `port` | Default `6379` |
| Redis | `listen_bind` | `127.0.0.1` (local only) or `0.0.0.0` |
| Redis | `data_path` | RDB/AOF directory |
| Redis | `appendonly` | `true` aligns with `docker-compose.edge.yml` |
| Redis | `run_mode` | `background` (default) or `foreground` |
| Credentials | `password` | Empty = no auth (OK on localhost) |

Render resolved settings:

```powershell
.\read-redis-config.ps1 -PrintSummary
```

## tenant-api alignment

```yaml
redis:
  enabled: true
  addr: 127.0.0.1:6379
  password: ""
  db: 0
```

Environment variables (if used): `YCWL_REDIS_ADDR`, `YCWL_REDIS_PASSWORD`.

## Logs and data

- Logs: `logs/redis.log`
- Data: `data/dump.rdb`, `data/appendonly.aof` (when AOF enabled)

See [deploy/run/README.md](../README.md) for full deployment order.
