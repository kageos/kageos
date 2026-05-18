# Backend Open Source Readiness

> 状态：执行口径
> 更新时间：2026-05-17
> 负责人窗口：事项 7 / codex/backend-open-source-readiness

## Service List

| Service | Path | Role | Default dev config |
| --- | --- | --- | --- |
| API Gateway | `core/api-gateway` | Static frontend entry, route gateway, auth forwarding | `deploy/dev/config/api-gateway.yaml` |
| HR Server | `core/hr-server` | Login, users, departments, system user seed | `deploy/dev/config/hr-server.yaml` |
| App Server | `core/app-server` | Workspaces, metadata, service tree, app orchestration | `deploy/dev/config/app-server.yaml` |
| Agent Server | `core/agent-server` | Workstation, PRD/code generation, LLM config APIs | `deploy/dev/config/agent-server.yaml` |
| App Runtime | `core/app-runtime` | Build/run generated user apps through containers | `deploy/dev/config/app-runtime.yaml` |
| App Storage | `core/app-storage` | Upload tokens, object metadata, MinIO access | `deploy/dev/config/app-storage.yaml` |

`core/cmd/main` is the local all-in-one entry used by the development guide.

## Dependencies

- MySQL stores platform metadata and generated app table data.
- NATS carries service and generated application messages.
- MinIO stores uploaded objects.
- Podman or Docker is used by the runtime layer for generated apps.

Start the bundled development infrastructure with:

```bash
bash deploy/dev/scripts/infra.sh up
```

## Test Command

Use the official backend regression entry:

```bash
bash scripts/test-core-go.sh
```

The script runs:

```bash
go test ./cmd/... ./core/... ./dto/... ./pkg/... ./sdk/... ./scripts/sync-case-catalog/...
```

Do not use plain `go test ./...` as the public regression command. It walks generated or local workspaces such as `namespace/`, which may contain user apps that are mid-generation, experimental, or intentionally outside the core package contract.

## Migration And Seed Notes

- Development config is under `deploy/dev/config/`.
- Production config is generated from `deploy/prod/aos.example.yaml` by `cmd/aosctl`.
- Service model initialization currently performs table creation/verification in service startup code.
- `core/app-server/system-seed/` contains built-in seed material used by the app-server domain.
- `hr-server` can initialize the system user from config or `SYSTEM_USER_PASSWORD`.

## Public Package Boundaries

- `namespace/` is generated user workspace output and must not be part of a source release.
- `local/` is for checked-out third-party or local experiments and must not be part of a source release.
- `deploy/prod/aos.yaml` and `deploy/prod/.generated/` are private local deployment artifacts.
