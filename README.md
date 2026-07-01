# Kageos

Kageos is a source-available, self-hostable business capability directory for individuals and teams, created by QiaYan AI (qiayanai.com). It turns ready-to-run scenario directories into governed workspace capabilities: users install mature directories from Hub, run them with private data, customize or fork them inside their own namespace, and publish stable capabilities back to the ecosystem.

The core project is currently licensed under the Business Source License 1.1. It is source-available and self-hostable, but it is not OSI open source today. The license restricts third-party commercial Hosted Service offerings, and each version converts to Apache License 2.0 after its Change Date. See [LICENSE](LICENSE) for the exact grant, Hosted Service restriction, Change Date, and future Apache-2.0 change license.

## Naming And Current Focus

- Write the product name as `Kageos` in prose, UI copy, docs, SDK docs, and website content. Use lowercase `kageos` for package, module, path, and domain identifiers, and keep all-caps `KAGEOS_*` only for environment variables or config keys.
- In public copy, prefer "source-available and self-hostable" for the core platform. Do not call the BSL-licensed core "open source" until the relevant version has converted to Apache-2.0. SDKs, examples, templates, and docs may use permissive open-source licenses separately when their own license files say so.
- The current product and engineering focus is namespace scenario-package ecology: build, harden, publish, install, and reuse ready-to-run scenario directories. Do not position Kageos as a low-cost AI app generator; Hub-first installation, private customization, and directory reuse are the main narrative. Do not add new platform cross-cutting capabilities unless maintainers explicitly reopen that track; existing governance, message, timer, audit, and runtime capabilities should be reused by scenario packages.

## What It Does

- Hub-oriented directory lifecycle: install ready-to-run scenario directories, adapt them in a private workspace, fork useful variants, and publish stable capabilities back.
- Service Tree for organizing application capabilities inside a workspace, with shared access, audit, operation-log, message, and schedule entry points.
- Workstation for requirement clarification, PRD drafting, code changes, build repair, runtime verification, and persistent workspace sessions.
- Dynamic UI rendering for generated forms, tables, details, charts, function panels, inbox threads, and operation logs.
- Platform inbox and notification flow backed by `message-server`, `message.v1.cmd.send`, SDK `ctx.SendNotification`, and the Agent `send_notification` tool.
- Scheduled automation backed by `timer-scheduler`, covering fixed Form/Table/Chart execution and unattended Agent workspace sessions.
- Go SDK for workspace applications.
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
| `core/timer-scheduler` | Shared schedule state, execution records, leases, and timer workers |
| `core/message-server` | Station inbox, message threads, unread state, and notification command consumer |
| `github.com/kageos/kageos-sdk/agent-app` | Go SDK module used by generated applications |
| `pkg/scheduledsdk` | Timer HTTP/NATS client and executor contract shared by scheduler and workers |
| `pkg/subjects` | NATS subject constants shared across services |
| `web` | Vue 3 frontend |
| `deploy` | Development, production, image, and security deployment material |
| `docs` | Project documentation index and operating notes |

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

Detailed onboarding, dependency notes, smoke tests, and troubleshooting live in [deploy/dev/README.md](deploy/dev/README.md).

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
- [Kageos Blueprint](docs/kageos-blueprint.md)
- [Current architecture](docs/current-architecture.md)
- [Platform capabilities](docs/platform-capabilities.md)
- [Scheduled task architecture](docs/scheduled-tasks-architecture-design.md)
- [Local development](deploy/dev/README.md)
- [Production deployment](deploy/prod/README.md)
- [Kageos lifecycle SOP](docs/kagectl-lifecycle-sop.md)
- [Kageos release SOP](docs/release-sop.md)
- [Product governance note](docs/product-thinking-ai-era-application-governance.md)

## Contributing And Security

Coordinate contribution, security, and conduct expectations with the repository maintainers before opening public issues or pull requests.

Do not commit real `.env` files, production `kage.yaml`, customer licenses, generated deployment output, `namespace/`, `local/`, or other private workspaces. Use the example files and local overrides instead.
