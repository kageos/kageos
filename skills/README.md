# kageos skills

Install the `kageos` plugin and use `kageos` as the normal entrypoint. It routes work across three deliberately separate roles:

1. `kageos-developer` designs and builds the workspace directory and its docs.
2. `kageos-operator` runs the real user workflow and emits a verification report.
3. `kageos-hub-publisher` captures safe browser evidence, uploads artifacts directly to R2/S3, and submits to Hub after explicit confirmation.

The operator and publisher are separate so publishing cannot silently replace runtime verification. The Hub artifact is the deterministic `capability.bundle.v1` directory package; the Codex skill itself remains an automation/runbook and is not installed into a customer's kageos workspace.

The repository `skills/` directory is the source of truth. `scripts/package-kageos-plugin.py` copies the four skills into `plugins/kageos/` for Codex and Claude Code distribution.
