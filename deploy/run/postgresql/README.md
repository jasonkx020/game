# PostgreSQL launcher (Windows + Ubuntu 24.04)

Aligned with [`scripts/service-postgres.sh`](../../../scripts/service-postgres.sh) and [`scripts/db-init.sh`](../../../scripts/db-init.sh) / [`scripts/db-init.ps1`](../../../scripts/db-init.ps1).

| Platform | Scripts | Config |
|----------|---------|--------|
| Windows | `*.bat`, `read-postgres-config.ps1` | `config.ini` (`run_mode=service` or `pg_ctl`) |
| Ubuntu 24.04 | `*.sh`, `read-postgres-config.sh` | `config.ini` (`run_mode=systemd` or `pg_ctl`) |

Windows PostgreSQL [download](https://get.enterprisedb.com/postgresql/postgresql-18.4-2-windows-x64-binaries.zip).

## Layout

```
deploy/run/postgresql/
  config.ini                    # shared INI (Windows + Linux)
  config.linux.ini.example      # Ubuntu defaults template
  read-postgres-config.ps1      # Windows config parser
  read-postgres-config.sh       # Linux config parser
  render-postgresql-conf.ps1 / .sh
  postgresql.conf.template
  pg_hba.conf.template
  start-postgres.bat / .sh
  stop-postgres.bat / .sh
  status-postgres.bat / .sh
  init-db.bat / .sh
  psql-client.bat / .sh
  install-postgres.sh           # Ubuntu apt install (optional)
  data/                         # pg_ctl mode only (gitignored)
```

## config.ini

| Section | Key | Description |
|---------|-----|-------------|
| PostgreSQL | `install_path` | PG root with `bin/psql`; `.` = auto-detect |
| PostgreSQL | `data_path` | Cluster data (`pg_ctl` mode only) |
| PostgreSQL | `port` / `listen_bind` | Listen port and bind address |
| PostgreSQL | `run_mode` | Windows: `service` \| `pg_ctl`; Linux: `systemd` \| `pg_ctl` |
| PostgreSQL | `windows_service_name` | Windows service (empty = auto-detect) |
| PostgreSQL | `systemd_service_name` | Linux systemd unit (empty = auto-detect) |
| Superuser | `user` / `password` | Superuser for init-db |
| Application | `user` / `password` | App user — match tenant-api DSN |
| Application | `db_edge` / `db_cloud` | Database names |

Print resolved config:

```powershell
.\read-postgres-config.ps1 -PrintSummary
```

```bash
./read-postgres-config.sh --print-summary
```

## Ubuntu 24.04 usage

**First-time install (apt):**

```bash
cd deploy/run/postgresql
chmod +x *.sh
# optional: cp config.linux.ini.example config.ini && edit passwords
sudo ./install-postgres.sh
```

**systemd mode (recommended after apt install):**

```ini
[PostgreSQL]
run_mode = systemd
```

```bash
./start-postgres.sh
./status-postgres.sh
./init-db.sh tenant --schema
./psql-client.sh
./stop-postgres.sh
```

`start-postgres.sh` / `stop-postgres.sh` use `sudo systemctl` when not root.

**Portable pg_ctl mode:**

```ini
[PostgreSQL]
run_mode = pg_ctl
install_path = .
data_path = ./data
```

First start runs `initdb` and renders `postgresql.conf` / `pg_hba.conf`.

## Windows usage

**Installed PostgreSQL:** `run_mode = service`

```bat
start-postgres.bat
status-postgres.bat
init-db.bat tenant --schema
psql-client.bat
stop-postgres.bat
```

**Portable pg_ctl:**

```ini
[PostgreSQL]
run_mode = pg_ctl
install_path = .
data_path = .\data
```

## tenant-api

Match `config.ini` `[Application]` with tenant-api DSN:

```yaml
ycwl:
  db:
    driver: postgres
    dsn: postgres://freetv:changeme@127.0.0.1:5432/freetv_edge?sslmode=disable
```

## init-db

Wraps `scripts/db-init.ps1` (Windows) or `scripts/db-init.sh` (Linux); writes local `postgres.env`.

```bash
./init-db.sh tenant --schema
./init-db.sh all --schema
```

## Ports

| Service | Default | Purpose |
|---------|---------|---------|
| PostgreSQL | 5432 | tenant-api / platform-api |
| tenant-api | 8443 | HTTP API |

## See also

- [deploy/run/README.md](../README.md) — full Windows ops manual
- [docs/部署指南-v1.0.0.md](../../../docs/部署指南-v1.0.0.md)
- [scripts/postgres.env.example](../../../scripts/postgres.env.example)
