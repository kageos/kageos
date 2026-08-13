# Contributing to kageos

Thanks for taking the time to improve kageos. This repository is
source-available under the Business Source License 1.1, and each release
converts to Apache-2.0 after four years. Before contributing, read `LICENSE`
and `LICENSE_FAQ.md` so you understand the current use grant and change license.

## Before You Start

- Discuss larger feature work with the maintainers before opening a pull
  request.
- Keep changes focused. Avoid mixing refactors, formatting sweeps, dependency
  upgrades, and product behavior changes in one pull request.
- Do not commit real secrets, customer data, generated production files,
  licenses, `namespace/`, `local/`, or runtime data directories.
- Follow `docs/source-available-messaging.md` and describe the core platform as
  "source-available and self-hostable" while it is BSL-licensed.

## Contribution Workflow

1. Fork the repository and create a branch from the latest `main`.
2. Set up local development with the commands below.
3. Make a small, focused change.
4. Add tests for bug fixes or new behavior, or explain verification in the
   pull request.
5. Update README or docs when behavior, deployment, or public positioning
   changes.
6. Open a pull request that explains what changed, why it changed, and how it
   was verified.

All code changes should go through pull requests. Pull requests need passing CI
and maintainer review before merge.

## Branch Model

Create branches from `main`. Recommended names:

```text
feat/short-feature-name
fix/short-bug-name
docs/short-doc-change
chore/short-maintenance-change
```

Before opening a pull request, sync with the latest `main`, resolve conflicts,
and make sure no local runtime files or private data are included.

## Commit Messages

Use Conventional Commits when possible:

```text
feat: add scheduled agent notification tool
fix: prevent incomplete tool messages after interruption
docs: update production deployment guide
test: cover service tree bundle import
chore: refresh generated API docs
```

Common types include `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, and
`ci`. Keep subjects clear and describe the user-visible behavior or maintenance
purpose.

## Local Development

Use `kagectl` as the contributor entrypoint for local development. The
repository still contains Compose files for the underlying development stack,
but those files are implementation details and troubleshooting tools. A new
contributor should not need to remember historical `customer`, `embedding`, or
root-level Compose paths just to run kageos locally.

If you plan to open a pull request, fork the repository first and clone your
fork. For a read-only local run, clone the upstream repository:

```bash
git clone https://github.com/kageos/kageos.git
cd kageos
```

For the normal local backend and frontend workflow, do not create or export
environment variables by hand. `kagectl bootstrap --dev` creates the local
backend env files:

| File | Purpose |
| --- | --- |
| `.kageos/kageos.env` | Records workspace mode, for example `KAGEOS_MODE=dev` and the selected dev engine. |
| `.kageos/dev/env/kageos.env` | Stores local-only secrets and service settings such as MySQL, NATS, MinIO, JWT, SMTP log mode, and `SYSTEM_USER_PASSWORD`. |

Those files are private runtime state and must not be committed.

From the source checkout, run `kagectl` through Go so contributors do not need
to install a separate binary first:

```bash
go run ./cmd/kagectl bootstrap --dev
```

This single command includes the dev initialization step. It initializes the
local dev workspace, starts MySQL / NATS / MinIO, ensures the local user-app
base image exists, and runs the backend in the foreground. By default, local dev
uses Podman when available. If you want Docker instead, choose it on first run:

```bash
go run ./cmd/kagectl bootstrap --dev --engine docker
```

Stop the backend with `Ctrl-C`. Stop the bundled infrastructure with:

```bash
go run ./cmd/kagectl down
```

This is the recommended first-run path when you want the whole product running
locally.

Start the frontend in another terminal:

```bash
cd web
npm install
npm run dev
```

After startup, open `http://localhost:5173`. `kagectl bootstrap --dev` prints a
`kageos dev initialization summary`; use username `system` and the printed
`Admin password` to sign in. If the summary has scrolled away, re-run
`go run ./cmd/kagectl init --dev` or read `SYSTEM_USER_PASSWORD` from
`.kageos/dev/env/kageos.env`.

Local email verification uses log mode. If you register another account, the
verification code is printed in backend logs and returned as `debug_code` for
local development.

Choose the workflow that matches your change:

| Goal | Recommended path |
| --- | --- |
| Run the full local product | `go run ./cmd/kagectl bootstrap --dev`, then `cd web && npm run dev` |
| Debug backend code in an IDE | Run `go run ./cmd/kagectl init --dev` once, then start `core/cmd/main/main.go` from the repository root |
| Work only on frontend code | Run the frontend dev server and set `VITE_PROXY_TARGET` when you want to use an existing backend |
| Troubleshoot infrastructure | Use `kagectl status`, `kagectl doctor`, `kagectl verify`, and `kagectl logs` before touching raw Compose commands |

For IDE debugging, initialize dev mode once:

```bash
go run ./cmd/kagectl init --dev
```

Then run `core/cmd/main/main.go` from the repository root. The dev marker in
`.kageos/kageos.env` tells the platform to load `.kageos/dev/config/*.yaml`.

Useful dev commands:

```bash
go run ./cmd/kagectl status
go run ./cmd/kagectl doctor
go run ./cmd/kagectl verify
go run ./cmd/kagectl logs main
go run ./cmd/kagectl logs infra
go run ./cmd/kagectl down
```

Common local ports:

| Port | Service |
| --- | --- |
| `5173` | Vite frontend |
| `9090` | API gateway used by the frontend proxy |
| `9091` | `app-server` |
| `9092` | `app-storage` |
| `9093` | `app-runtime` |
| `9095` | `agent-server` |
| `9096` | `connector-server` |
| `9097` | `hr-server` |
| `9098` | `timer-scheduler` |
| `9099` | `message-server` |
| `3318` | Local dev MySQL on the host |
| `4222` | NATS |
| `9000` | MinIO API |
| `9001` | MinIO console |

When something fails, check in this order:

1. `go run ./cmd/kagectl doctor` for workspace and compose setup.
2. `go run ./cmd/kagectl status` for local infrastructure container state.
3. `go run ./cmd/kagectl verify` for infrastructure and platform health checks.
4. `go run ./cmd/kagectl logs main` for backend logs, or `go run ./cmd/kagectl logs infra` for MySQL / NATS / MinIO logs.

`go run ./cmd/kagectl down` stops local infrastructure but keeps local data. If
you need a full reset, back up anything important under `.kageos/dev/namespace/`
first, then remove local dev env/state and the matching container volumes for
your Docker or Podman engine.

If you only work on the frontend, create `web/.env.development.local` from
`web/.env.development.local.example` and set `VITE_PROXY_TARGET` to the backend
you want to use.

Optional local settings:

| Need | Use |
| --- | --- |
| Force Docker instead of Podman | `go run ./cmd/kagectl bootstrap --dev --engine docker` |
| Point frontend to a remote backend | `web/.env.development.local` with `VITE_PROXY_TARGET` |
| Set a remote WebSocket endpoint | `VITE_WS_URL` in `web/.env.development.local` |
| Configure real email delivery | Change local config to `SMTP_MODE=smtp` and set `SMTP_*` values |
| Use AI workstation features | Add an LLM configuration after logging in; an LLM API key is not required just to boot and sign in |

First startup can take a while because the local user-app base image may need to
be built. Local email verification defaults to log mode, so development
verification codes are written to backend logs instead of being sent through a
real SMTP provider.

Local dev files under `.kageos/dev/config/`, `.kageos/dev/env/`, and
`.kageos/dev/namespace/` are private runtime state. Do not commit them. The
full local development reference lives in `deploy/dev/README.md`.

## Local Verification

Run the checks that match your change before opening a pull request.

Backend:

```bash
bash scripts/test-core-go.sh
```

Frontend:

```bash
cd web
npm ci
npm run check:architecture
npm run lint
npm run type-check
npm run test:unit -- --run
npm run build
```

Repository governance:

```bash
bash scripts/check-sensitive-files.sh
bash scripts/check-sdk-boundaries.sh
bash scripts/check-doc-links.sh
git diff --check
```

Security dependency checks:

```bash
cd web
npm audit --omit=dev
```

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./core/... ./dto/... ./pkg/...
```

## Pull Request Checklist

- The change has a clear reason and a narrow scope.
- User-facing behavior is covered by tests or explained in the pull request.
- Documentation is updated when behavior, deployment, or public positioning
  changes.
- New logs avoid secrets, tokens, license keys, customer data, and private
  workspace paths.
- New dependencies are necessary, maintained, and compatible with the project
  license model.

## Conduct And Security

Follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) when participating in project
spaces. Report security issues privately through [SECURITY.md](SECURITY.md);
do not open public issues for suspected vulnerabilities.

By submitting a contribution, you agree that your contribution is provided under
the repository's current license terms and will follow the Change License
mechanism described in [LICENSE](LICENSE), unless separately agreed in writing
with the maintainers.
