# Kageos

Kageos is a source-available AI light-application workspace for individuals and small teams. Users describe an internal tool in natural language, confirm a lightweight PRD, and let the system generate runnable `Form`, `Table`, and `Chart` applications.

The project is currently licensed under the Business Source License 1.1. It is not OSI open source today; see [LICENSE](LICENSE) for the exact grant, Hosted Service restriction, Change Date, and future Apache-2.0 change license.

## What It Does

- AI workstation for requirement clarification, PRD generation, code generation, build repair, and runtime verification.
- Dynamic UI rendering for generated forms, tables, details, and charts.
- Service Tree for organizing application capabilities inside a workspace.
- Go SDK for generated light applications.
- Single-machine development and production deployment paths based on Go, Vue, MySQL, NATS, MinIO, and containers.

## Repository Map

| Path | Purpose |
| --- | --- |
| `core/agent-server` | Agent workstation, tool calling, PRD, and code generation flow |
| `core/app-server` | Workspace APIs, metadata, and runtime orchestration |
| `core/app-runtime` | Runtime manager for generated user applications |
| `core/app-storage` | Upload and object-storage service |
| `core/api-gateway` | API gateway, static frontend entry, and auth forwarding |
| `core/hr-server` | Login, user, and basic organization data |
| `sdk/agent-app` | Go SDK used by generated applications |
| `web` | Vue 3 frontend |
| `deploy` | Development, production, image, and security deployment material |
| `docs` | Project documentation index and governance documents |

## Quick Start

Prerequisites:

- Go 1.25 or newer.
- Node.js 20.19 or newer, or 22.12 or newer.
- Docker Compose or Podman Compose.
- MySQL, NATS, and MinIO from the bundled development compose stack.

Bootstrap local development backend:

```bash
go run ./cmd/kagectl bootstrap --dev
```

Start the frontend in another terminal:

```bash
cd web
npm install
npm run dev
```

The frontend uses relative API paths by default. To point only the frontend at a remote backend, create `web/.env.development.local` from `web/.env.development.local.example` and set `VITE_PROXY_TARGET`.

Detailed onboarding, dependency notes, smoke tests, and troubleshooting live in [docs/local-development.md](docs/local-development.md).

## Verification

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

Repository governance checks:

```bash
bash scripts/check-sensitive-files.sh
bash scripts/check-doc-links.sh
```

The same checks are wired into GitHub Actions in `.github/workflows/ci.yml`.

## Documentation

- [Documentation index](docs/README.md)
- [Local development](docs/local-development.md)
- [Backend readiness](docs/backend-open-source-readiness.md)
- [Examples and SDK guide](docs/examples/README.md)
- [Production deployment](deploy/prod/README.md)
- [Release process](docs/governance/RELEASE_PROCESS.md)

## Contributing And Security

Please read [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before opening issues or pull requests.

Do not commit real `.env` files, production `kage.yaml`, customer licenses, generated deployment output, `namespace/`, `local/`, or other private workspaces. Use the example files and local overrides instead.
