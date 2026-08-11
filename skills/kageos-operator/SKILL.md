---
name: kageos-operator
description: Operate and verify an already-developed KageOS workspace directory through direct authenticated HTTP requests, live schema discovery, Form/Table/Chart and OnSelectFuzzy calls, file uploads, user selectors, rich-text assets, business readback, cleanup, automation evidence, and optional UI checks. Use after kageos-developer and before publishing when a user asks to test, operate, validate, walk through, or prove a KageOS directory works end to end.
---

# KageOS Operator

Verify the installed product as an operator would. Do not redesign, edit, build, or publish it.

## Inputs

Collect or discover:

- KageOS gateway base URL.
- Workspace directory `full_code_path`, not a local filesystem path.
- `KAGEOS_OPENAPI_TOKEN`, or temporarily `KAGEOS_ACCESS_TOKEN` for local/test use.
- Expected user scenario and safe synthetic test data.
- Optional browser URL for visual verification.

Keep credentials only in the environment. Never print them or place them in commands saved to files, reports, or source control. Prefer a dedicated low-permission test identity; never use an administrator access JWT for unattended or production automation.

## Workflow

1. Read `references/http-operation-contract.md`, `references/complex-input-contract.md`, and `references/verification-contract.md`.
2. Generate one stable source reference and trace ID for the run. Send the required audit headers on every request.
3. Call the discovery endpoints directly over HTTP. Inspect the current function schemas, Table callbacks, field-level `OnSelectFuzzy` callbacks, scheduled functions, and AgentTasks. Never guess paths, fields, enums, or callback support.
4. Inspect every input widget. Resolve dynamic values before use: upload `files` and rich-text assets, search and hydrate `user/users`, and query `OnSelectFuzzy` fields. Never guess refs, usernames, or foreign IDs.
5. Invoke read operations directly. Inspect each real response before deciding the next request; do not assume response fields such as `data.list` or `data.total`.
6. Before the first write, show the user a concise human-readable sequence containing the exact functions, local files, synthetic marker, expected business effects, uploaded-object cleanup, and record cleanup. Obtain one explicit authorization for that sequence. Do not create a JSON execution plan.
7. Invoke approved writes directly over HTTP, one at a time. Never retry writes or uploads automatically. Capture returned identifiers and file refs only in the current execution context and use read operations to prove saved or updated business state.
8. For each discovered `OnSelectFuzzy`, call `by_keyword`. Before using a returned scalar or array value, call `by_value` or `by_values`; never guess foreign IDs.
9. Test only operations relevant to the real workflow. Do not mechanically invoke all CRUD endpoints merely because callbacks exist.
10. Always attempt deterministic cleanup of synthetic records and uploaded objects created by this run. Never delete pre-existing or customer data.
11. If a browser URL is supplied, use Browser or Chrome control to exercise the same scenario. Do not inspect cookies, local storage, or hidden state.
12. Write an after-the-fact `kageos.operator-report.v1` JSON report when an artifact is needed. Write scenario and evidence text in the user's language while keeping operation names, paths, status values, and schema fields stable. The JSON records machine-checkable evidence; it never drives execution.
13. Render every saved JSON report for humans with `python3 scripts/render_report.py <report.json>`. Return links to the generated Markdown and self-contained HTML alongside the JSON. Do not hand-edit a rendered report to change verification status or evidence.

## Authentication

Choose exactly one mode:

- OpenAPI: send `Authorization: Bearer $KAGEOS_OPENAPI_TOKEN`.
- Temporary local/test access JWT: send `X-Token: $KAGEOS_ACCESS_TOKEN`.

Never send an access JWT as Bearer and never send an OpenAPI token as `X-Token`. Record only `auth_mode` in evidence.

## Automation

Inventory every scheduled function and AgentTask returned by discovery. Definition discovery proves packaging only. Mark automation runtime-verified only when a real read or poll observes its output. If it is disabled or cannot be safely triggered, report it as blocked.

## Gate

Return `verified` only when every required business assertion passes, cleanup succeeds, all discovered automation has runtime evidence, and the UI agrees when supplied. Otherwise return `blocked` with the exact missing or failing evidence.

## Handoff

Give the publisher the directory `full_code_path`, JSON/Markdown/HTML verification report paths, browser URL and key screens, known sensitive fields, and concise release facts.
