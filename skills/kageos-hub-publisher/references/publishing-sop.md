# Publishing SOP

## Closed loop

`kageos-developer` builds and documents the app → `kageos-operator` proves the real workflow → `kageos-hub-publisher` captures evidence, packages, uploads, and submits.

## Hub protocol

Use `Authorization: Bearer $HUB_PUBLISH_TOKEN` for every authenticated request.

1. `POST /uploads/intents` with `purpose=attachment` for the bundle or `purpose=product-media` for screenshots, plus `visibility=public` for gallery media.
2. PUT raw bytes to the returned `upload_url` using only the returned upload headers.
3. `POST /uploads/intents/{id}/complete`.
4. Optional: `POST /hub/submissions/assist`.
5. After explicit confirmation: `POST /hub/submissions`.
6. `GET /hub/submissions` and confirm the returned revision/status.

Recommended screenshot captions state what is visible and why it matters, for example: “创建会议后，议题、参与人和待办会在同一工作区同步展示。” Do not burn annotations into screenshots by default; captions remain editable Hub content.

The token should expire and be revoked when no longer needed. A publisher token must not carry administrator review privileges.
