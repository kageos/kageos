# Security Policy

## Reporting a Vulnerability

Please do not report suspected vulnerabilities in public issues.

Email the maintainers at `security@qiayanai.com` with:

- A short description of the issue and likely impact.
- Affected version, branch, commit, or deployment mode.
- Reproduction steps, proof of concept, logs, or screenshots when safe to share.
- Whether the issue may expose secrets, tenant data, workspace files, or remote
  code execution paths.

The maintainers will acknowledge reports as soon as practical, triage severity,
and coordinate a fix or mitigation before public disclosure.

## Supported Versions

The project is preparing public source releases. Until a formal support matrix is
published, security fixes target the current `main` branch and actively
maintained release branches.

## Sensitive Data Rules

Do not commit:

- Real `.env` files or generated deployment configs.
- Production `kage.yaml` files.
- Customer licenses, private keys, tokens, passwords, or API credentials.
- `namespace/`, `local/`, runtime databases, object storage data, logs, or
  generated deployment output.

When adding logs or diagnostics, log presence, length, stable IDs, or redacted
prefixes only when necessary. Never log bearer tokens, generated passwords,
license secrets, or raw customer payloads.

## Deployment Hardening

For production deployments, review the deployment documentation and optional
container confinement material in `deploy/security/`. Keep external ingress,
database, object storage, NATS, and generated application runtime boundaries
explicit in your environment.
