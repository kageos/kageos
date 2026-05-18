# Local Development

> 状态：执行口径
> 更新时间：2026-05-17
> 负责人窗口：事项 5 / codex/local-dev-onboarding

This is the short path for running the minimum AI-Agent-OS system locally.

## Dependencies

| Dependency | Purpose |
| --- | --- |
| Go 1.25+ | Backend services and `aosctl` |
| Node.js 20.19+ or 22.12+ | Vue frontend build and tests |
| Docker Compose or Podman Compose | MySQL, NATS, MinIO, and runtime containers |
| MySQL 8 | Platform metadata |
| NATS | Generated app and service messaging |
| MinIO | Object storage and upload flow |

## Start Order

1. Start infrastructure:

   ```bash
   bash deploy/dev/scripts/infra.sh up
   ```

2. Start the backend:

   ```bash
   APP_ENV=dev go run ./core/cmd/main
   ```

3. Start the frontend:

   ```bash
   cd web
   npm install
   npm run dev
   ```

4. Open the Vite URL printed by `npm run dev`.

## Frontend Environment

The tracked frontend env files have been removed from git. Use examples instead:

```bash
cp web/.env.example web/.env.development.local
```

For remote-backend frontend development, use:

```bash
cp web/.env.development.local.example web/.env.development.local
```

Then set `VITE_PROXY_TARGET`.

## Smoke Tests

Backend:

```bash
bash scripts/test-core-go.sh
```

Frontend:

```bash
cd web
npm run check:architecture
npm run type-check
npm run test:unit -- --run
npm run build
```

Deployment config:

```bash
go run ./cmd/aosctl doctor --config deploy/prod/aos.example.yaml
```

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Backend cannot connect to MySQL, NATS, or MinIO | Run `bash deploy/dev/scripts/infra.sh up` again and inspect compose logs |
| Frontend API calls fail with CORS | Keep `VITE_API_BASE_URL` empty for local proxy mode, or set `VITE_PROXY_TARGET` in `web/.env.development.local` |
| Generated app tests fail under `go test ./...` | Use `bash scripts/test-core-go.sh`; it intentionally excludes generated `namespace/` apps |
| Production config appears in git status | `deploy/prod/aos.yaml` is local private config; keep it ignored and commit only `deploy/prod/aos.example.yaml` |
