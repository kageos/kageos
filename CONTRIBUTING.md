# Contributing

> 状态：执行口径
> 更新时间：2026-05-17
> 负责人窗口：事项 1 / codex/open-source-governance

Thanks for helping improve AI-Agent-OS. This repository is source-available under BSL 1.1, so contributions should assume the license terms in [LICENSE](LICENSE).

## Development Setup

1. Install Go 1.25+, Node.js 20.19+ or 22.12+, and Docker Compose or Podman Compose.
2. Start infrastructure with `bash deploy/dev/scripts/infra.sh up`.
3. Start the backend with `APP_ENV=dev go run ./core/cmd/main`.
4. Start the frontend from `web/` with `npm install` and `npm run dev`.

See [docs/local-development.md](docs/local-development.md) for the full local guide.

## Pull Request Checklist

- Keep changes scoped to one concern.
- Add or update tests for user-visible behavior and shared contracts.
- Update documentation when commands, configuration, APIs, or behavior change.
- Do not commit secrets, real customer data, generated workspaces, or local deployment output.
- Run the relevant checks before asking for review.

Backend checks:

```bash
bash scripts/test-core-go.sh
```

Frontend checks:

```bash
cd web
npm run check:architecture
npm run type-check
npm run test:unit -- --run
npm run build
```

Governance checks:

```bash
bash scripts/check-sensitive-files.sh
bash scripts/check-doc-links.sh
```

## Coding Notes

- Follow existing package boundaries and naming before adding new abstractions.
- Keep generated `namespace/` applications out of core backend tests.
- Prefer example configuration files over committing runnable local configuration.
- If a change touches licensing, public packaging boundaries, or enterprise/commercial positioning, call that out explicitly in the pull request.
