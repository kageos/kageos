# Release Process

> 状态：执行口径
> 更新时间：2026-05-17
> 负责人窗口：事项 11 / codex/release-process

AI-Agent-OS does not yet promise a stable public API. Until `v1.0.0`, minor versions may contain breaking changes, and release notes must call them out explicitly.

## Versioning

- Use semantic versions: `vMAJOR.MINOR.PATCH`.
- Before `v1.0.0`, breaking changes are allowed in minor releases, but must be documented.
- Patch releases should be limited to fixes, documentation corrections, and low-risk operational improvements.

## Release Checklist

1. Confirm license and source-available wording still match [LICENSE](../../LICENSE).
2. Run backend tests: `bash scripts/test-core-go.sh`.
3. Run frontend checks: `npm run check:architecture`, `npm run type-check`, `npm run test:unit -- --run`, and `npm run build` in `web/`.
4. Run governance checks: `bash scripts/check-sensitive-files.sh` and `bash scripts/check-doc-links.sh`.
5. Verify `deploy/prod/aos.example.yaml` is current and `deploy/prod/aos.yaml` is not tracked.
6. Update [CHANGELOG.md](../../CHANGELOG.md).
7. Add a release note with breaking changes, migration steps, and known limitations.

## Breaking Change Template

```markdown
### Breaking Changes

- Component:
- What changed:
- Who is affected:
- Migration:
- Rollback:
```
