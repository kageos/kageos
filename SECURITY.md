# Security Policy

> 状态：执行口径
> 更新时间：2026-05-17
> 负责人窗口：事项 1 / codex/open-source-governance

## Reporting A Vulnerability

Please report suspected vulnerabilities privately to the maintainers before public disclosure. Include:

- Affected component and version or commit.
- Reproduction steps or proof of concept.
- Expected and observed impact.
- Any temporary mitigation you found.

If no private security contact is available in the public hosting provider, open a minimal issue that asks for a private disclosure channel and avoid exploit details.

## Secret Handling

Do not commit real credentials, customer licenses, private keys, tokens, production `aos.yaml`, generated deployment output, `namespace/`, or `local/` workspaces.

Before publishing or opening a pull request, run:

```bash
bash scripts/check-sensitive-files.sh
```

The script is intentionally conservative. If it flags an example value, rename the file or value so it is clearly non-secret.
