# Soft-delete lifecycle

KageOS Table records use soft deletion when the SDK model contains `gorm.DeletedAt`.
The platform provides a recycle bin, complete delete snapshots in operation logs,
and an optional runtime-owned retention worker.

## User flow

- A Table with `OnTableDeleteRows` shows **Recycle bin** in its toolbar. The view
  is linkable with `_tab=recycle-bin`; opening and operating it requires the
  current function's Admin or Owner role.
- The recycle-bin dialog lists deleted rows and supports single or batch restore,
  permanent deletion with typed confirmation, and a persistent table-level
  automatic-cleanup policy (enabled state, dry-run/purge mode, and retention days).
- Delete, restore, and permanent-delete operations write one audit entry per row. Delete entries keep
  the pre-delete row snapshot in `old_values_json`.
- Operation logs hide scheduled-task entries by default. **Show scheduled tasks**
  includes them without changing deep links to a specific log or trace.

## Retention worker

The worker runs inside `app-runtime` with platform-owned admin credentials. App
runtime and migration users do not receive `DELETE` permission. It scans only
runtime-managed active databases and only tables that contain a `deleted_at`
column.

The default configuration is disabled and read-only:

```yaml
app_database:
  soft_delete_cleanup:
    enabled: false
    mode: dry_run
    retention_days: 30
    interval_minutes: 1440
    batch_size: 500
```

Recommended rollout:

1. Set `enabled: true` and keep `mode: dry_run`; inspect
   `[AppDatabaseCleanup]` candidate counts.
2. Confirm backups and retention policy.
3. Set `mode: purge` to hard-delete expired rows in bounded batches.

The deployment values are the default for tables without an override. An Admin
or Owner can open a Table's recycle bin and save an explicit table policy. Table
overrides are stored by app-runtime and survive restarts. The deployment
`interval_minutes` and `batch_size` remain platform-wide safety controls.

Equivalent environment variables use the
`KAGEOS_APP_DB_SOFT_DELETE_*` prefix; cleanup enablement and mode are
`KAGEOS_APP_DB_SOFT_DELETE_CLEANUP_ENABLED` and
`KAGEOS_APP_DB_SOFT_DELETE_CLEANUP_MODE`.

## API and SDK compatibility

The platform endpoints are:

- `GET /workspace/api/v1/table/deleted/*full-code-path`
- `POST /workspace/api/v1/table/restore/*full-code-path`
- `DELETE /workspace/api/v1/table/purge/*full-code-path` (admin only)
- `GET|PUT /workspace/api/v1/table/recycle-policy/*full-code-path` (admin only)

They rely on the private SDK callbacks `__table_get_deleted_rows` and
`__table_restore_rows`. Permanent deletion is sent from app-server to the
runtime-owned admin database connection; application credentials retain their
no-`DELETE` privilege boundary. The matching SDK release must be available before the
platform release enables the recycle-bin UI in deployed applications.
