---
name: kageos-operator
description: Operate and verify an already-developed KageOS workspace directory through its real OpenAPI actions and UI. Use after kageos-developer and before publishing when a user asks to test, operate, validate, walk through, or prove a KageOS app works with real Form, Table, Chart, scheduled-function, and agent-task behavior.
---

# KageOS Operator

Verify the product as an operator would. Do not redesign or publish it.

## Inputs

Collect or discover:

- KageOS base URL and bearer token.
- Workspace directory path or `full_code_path`.
- Expected user scenario and safe test data.
- Optional browser URL for visual verification.

Never print tokens. Never use real customer data when synthetic fixtures can prove the flow.

## Workflow

1. Read `references/verification-contract.md`.
2. Discover the directory manifest before invoking functions.
3. Build a short scenario covering the main write, read, update, aggregate/chart, and automation paths that actually exist.
4. Ask for confirmation immediately before any write unless the user already explicitly authorized running the test scenario.
5. Call real OpenAPI functions. Do not substitute source inspection for runtime evidence.
6. If a browser URL is supplied, use the available Browser or Chrome control skill to exercise the corresponding UI. Do not inspect cookies, local storage, or hidden application state.
7. Clean up only synthetic records created by this run, when cleanup is safe and the user authorized writes.
8. Save a JSON verification report containing timestamps, called functions, redacted inputs, outputs/assertions, cleanup, and unresolved issues.

## Gate

Return `verified` only when the primary scenario succeeds with real data and the UI, when supplied, agrees with the API results. Otherwise return `blocked` with exact failing operations. A publisher must not publish a blocked report.

## Handoff

Give the publisher:

- directory `full_code_path`;
- verification report path;
- browser URL and key screens worth capturing;
- known sensitive fields to avoid;
- concise release facts, not marketing copy.
