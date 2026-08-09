# Verification contract

Use `Authorization: Bearer $KAGEOS_OPENAPI_TOKEN` against the KageOS OpenAPI base URL. Start with API discovery/manifest endpoints available in the target deployment. Invoke only functions exposed by the resolved directory.

The report must be machine-readable JSON:

```json
{
  "status": "verified",
  "directory": "system/demos/meeting",
  "started_at": "RFC3339",
  "finished_at": "RFC3339",
  "checks": [{"operation": "form.submit", "ok": true, "evidence": "redacted summary"}],
  "browser_url": "http://localhost:5173/workspace/system/demos/meeting",
  "capture_targets": [{"title": "会议列表", "reason": "证明创建结果出现在真实列表"}],
  "sensitive_fields": ["attendee_email"],
  "cleanup": "removed synthetic meeting record",
  "issues": []
}
```

Redact secrets, personal data, internal IDs that identify real customers, and any content not intended for Hub publication.
