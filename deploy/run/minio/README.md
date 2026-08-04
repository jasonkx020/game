# MinIO Windows launcher

Aligned with Linux [`scripts/service-minio.sh`](../../../scripts/service-minio.sh) and EDGE Compose for local **Windows** development.

## Layout

```
deploy/run/minio/
  config.ini
  read-minio-config.ps1
  start-minio.bat
  stop-minio.bat
  status-minio.bat
  minio-mc.bat
  data/
```

Place `minio.exe` / `mc.exe` in this folder or under `bin/`.

## config.ini

| Section | Key | Description |
|---------|-----|-------------|
| MinIO | `install_path` | Install root, `.` = this directory |
| MinIO | `data_path` | Data dir, supports `.\data` |
| MinIO | `api_port` / `console_port` | S3 API / web console |
| MinIO | `listen_bind` | `127.0.0.1` local; `0.0.0.0` LAN |
| MinIO | `bucket` | Default bucket (`YCWL_STORAGE_BUCKET`) |
| MinIO | `add_firewall_rules` | Add firewall rules on start |
| Credentials | `root_user` / `root_password` | Match tenant-api storage |
| Client | `alias` / `endpoint` | mc alias and API URL |

Print resolved config:

```powershell
.\read-minio-config.ps1 -PrintSummary
```

## Usage

Run as administrator (first start adds firewall rules):

```bat
start-minio.bat
stop-minio.bat
status-minio.bat
minio-mc.bat
minio-mc.bat ls ycwl-minio/
```

## tenant-api

```yaml
ycwl:
  storage:
    use_minio: true
    endpoint: http://127.0.0.1:9000
    bucket: freetv-assets
    access_key: ycwlminio
    secret_key: YcwlMinioDev123
    public_url_prefix: http://127.0.0.1:8443/files
```

Use the same values in `config.ini` `[Credentials]` when running MinIO via `deploy/run/minio/start-minio.bat`.

**Note:** If you already initialized `data/` with `minioadmin`, either keep those credentials in `config.ini` or delete `data/` and start fresh after changing passwords.

For production EDGE behind nginx, use `/minio-api/` and `/minio-console/`; do not put nginx URLs in MinIO `endpoint`.

## Ports

| Service | Default | Purpose |
|---------|---------|---------|
| MinIO API | 9000 | S3 / tenant-api |
| MinIO Console | 9001 | Web UI (or nginx :9003) |
| tenant-api | 8443 | `/files` proxy |
