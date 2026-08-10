# Verification contract

Use direct authenticated HTTP calls and evaluate each real response before continuing. A transport-level success is not sufficient: assert the saved, read, updated, aggregated, selected, or rejected business outcome appropriate to the operation.

Use one source reference and trace ID across discovery, operation, readback, and cleanup. Redact credentials, personal data, customer identifiers, raw response bodies, and internal IDs from user-facing evidence.

## Evidence

Track these facts during execution without creating an input plan artifact:

- directory and scenario;
- authentication mode, never the credential;
- called HTTP method and function path;
- business assertions and concise results;
- synthetic marker and names of captured variables, never sensitive values;
- complex-input evidence: file metadata/hash checks, resolved ref count, user hydration count, and rich-text asset type without signed URLs or profile data;
- cleanup attempts and outcomes;
- automation runtime evidence or exact blocker;
- optional UI agreement and capture targets.

When a machine-readable artifact is needed, write it only after or during execution as evidence:

```json
{
  "schema_version": "kageos.operator-report.v1",
  "status": "verified",
  "directory": "/system/demos/meeting",
  "scenario": "Create, find, update, and remove a synthetic meeting",
  "started_at": "RFC3339",
  "finished_at": "RFC3339",
  "auth_mode": "access_token",
  "source_ref": "meeting-release-20260810",
  "trace_id": "kageos-operator-...",
  "checks": [
    {
      "operation": "table.search",
      "method": "GET",
      "full_code_path": "/system/demos/meeting/rooms.table",
      "status": "passed",
      "evidence": "The synthetic room appeared in the real table response."
    }
  ],
  "automations": [],
  "cleanup": [{"operation": "table.delete", "status": "passed"}],
  "sensitive_fields": [],
  "issues": []
}
```

Never edit a failed report into a passing one. Return `verified` only when all required checks pass, every run-owned business record and uploaded object is cleaned up, and every discovered automation has runtime evidence. Otherwise return `blocked`.
