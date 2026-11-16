# JWT ES256 keys (dev)

- `jwt-es256-dev.pem` — **private** (Auth only)
- `jwt-es256-dev.pub.pem` — public (platform / Pitaya / admin)

Regenerate:

```powershell
# Windows（推荐）
.\scripts\dev.ps1 gen-jwt-keys

# 或直接
cd auth-service; go run ./keys/gen_jwt_keys.go
```

```bash
# Linux / macOS / 已安装 make
make gen-jwt-keys
```

Production: generate a new keypair, never reuse these files; mount via `JWT_PRIVATE_KEY_FILE` / `JWT_PUBLIC_KEY_FILE` or PEM env vars. JWKS: `GET http://auth:8081/.well-known/jwks.json`.
