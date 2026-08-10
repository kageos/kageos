---
name: kageos-hub-publisher
description: Package and publish a verified KageOS workspace directory to KageOS Hub using a scoped Sona personal access token, direct R2/S3 uploads, AI-assisted metadata, and browser screenshots with editable captions. Use after kageos-operator when a user asks to release, submit, update, or publish a KageOS directory or skill to Hub.
---

# KageOS Hub Publisher

Publish evidence, not an untested demo. Read `references/publishing-sop.md` before acting.

## Required inputs

- A `verified` operator report.
- Directory `full_code_path` and exportable `capability.bundle.v1` package.
- Hub base URL, default `https://hub.kageos.ai/api/v1`.
- `HUB_PUBLISH_TOKEN` with `uploads:write` and `hub:publish` scopes.
- Optional workspace browser URL supplied by the user.

Never log or embed the token. Never request R2 credentials: all files must use Sona upload intents followed by direct PUT to R2/S3.

## Workflow

1. Reject a missing, stale, or blocked operator report.
2. Export and validate the deterministic directory bundle. Do not package `SKILL.md` as the directory artifact.
3. If a browser URL is available, use the available Browser or Chrome control skill:
   - open the exact URL and operate the verified scenario;
   - capture 2–6 key viewport screenshots at a consistent desktop size;
   - capture the result state, not empty setup screens;
   - inspect every image for tokens, email, phone, customer names, private URLs, or unrelated tabs;
   - discard unsafe captures and re-run with synthetic data;
   - write a concise editable caption for each capture explaining the user value and visible evidence.
4. Upload the bundle with `scripts/hub_publish.py upload`. For screenshots, write the reviewed media manifest described in the SOP and run `scripts/hub_publish.py prepare`; it uploads every local file through a Sona intent, builds the ordered gallery, and appends selected images to `description_html`.
5. Optionally call Hub AI assist with bundle-derived facts and operator evidence. Treat suggestions as a draft; never invent capabilities.
6. Inspect the prepared submission JSON. The first gallery item is the catalog cover; every gallery item must include `url`, `kind`, useful `alt`, and `caption`. Inline screenshots must explain the visible outcome instead of duplicating empty UI.
7. Show the final name, summary, version facts, screenshot captions, and target Hub to the user. Obtain explicit confirmation immediately before `submit`.
8. Submit with `scripts/hub_publish.py submit`, then query `status` until the pending submission is visible.

## Update releases

Use the same directory code/namespace. Hub assigns the next public version; preserve the bundle's release version as source evidence and include real release notes. Never create a second identity for an update.

## Stop conditions

Stop without submitting when verification is not `verified`, package validation fails, screenshots expose sensitive data, token scopes are insufficient, upload completion fails, or the user has not confirmed the final submission.
