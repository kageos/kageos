# kageos delivery run contract

`kageos.delivery-run.v1` 是交付证据索引，不是写操作执行计划。真实操作仍须遵守 Operator 的响应驱动 HTTP 契约。

## Required shape

```json
{
  "schema_version": "kageos.delivery-run.v1",
  "run_id": "kageos-delivery-...",
  "directory": "/user/app/package",
  "created_at": "RFC3339",
  "updated_at": "RFC3339",
  "stages": {
    "design": {"status": "pending", "recorded_at": "", "note": "", "artifacts": []},
    "local_build": {"status": "pending", "recorded_at": "", "note": "", "artifacts": []},
    "platform_build": {"status": "pending", "recorded_at": "", "note": "", "artifacts": []},
    "operator_verify": {"status": "pending", "recorded_at": "", "note": "", "artifacts": []},
    "bundle": {"status": "pending", "recorded_at": "", "note": "", "artifacts": []},
    "publish_prepare": {"status": "pending", "recorded_at": "", "note": "", "artifacts": []},
    "publish_submit": {"status": "pending", "recorded_at": "", "note": "", "artifacts": []},
    "publish_status": {"status": "pending", "recorded_at": "", "note": "", "artifacts": []}
  }
}
```

Each artifact contains an absolute path, byte size, SHA-256, and recording time. Reports may be kept in a user-approved workspace artifact directory; ephemeral delivery runs may use a protected temporary directory.

## Invariants

- A stage can pass only after every preceding stage passed.
- A blocked stage may be recorded without altering earlier evidence.
- `operator_verify=passed` requires a `kageos.operator-report.v1` JSON whose status is `verified` and directory matches the delivery run.
- The Operator report must not predate the latest recorded platform build.
- `bundle=passed` requires a `capability.bundle.v1` JSON.
- `publish_submit=passed` requires an explicit-confirmation flag at the time it is recorded.
- Sensitive key names or values are rejected. Credentials belong only in environment variables or an external secret store.
- If code or platform state changes after verification, start a new run or reset from `platform_build`; never reuse stale downstream evidence.
