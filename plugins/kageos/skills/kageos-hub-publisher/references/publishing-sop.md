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

## Screenshot manifest

After inspecting every local capture, create a temporary media manifest. Keep the most representative image first because Hub uses the first gallery item as the catalog cover. Relative file paths resolve from the manifest directory.

```json
{
  "gallery_mode": "replace",
  "append_to_description": true,
  "description_heading": "真实使用效果",
  "items": [
    {
      "file": "screenshots/01-meeting-list.png",
      "alt": "会议列表与处理状态",
      "caption": "创建会议后，议题、参与人和待办会在同一工作区同步展示。",
      "include_in_description": true
    },
    {
      "file": "screenshots/02-dashboard.png",
      "alt": "会议执行统计看板",
      "caption": "看板汇总会议数量、完成率和待处理事项。",
      "include_in_description": true
    }
  ]
}
```

Use `gallery_mode=replace` for a complete new screenshot set. Use `append` only when intentionally preserving gallery items already present in the submission JSON. Hub accepts at most eight gallery items. Images default to inline rich-text evidence; videos remain gallery-only and must set `include_in_description` to `false`.

Upload all media and prepare the final payload without creating a submission:

```bash
python3 scripts/hub_publish.py prepare \
  /tmp/hub-submission.json \
  /tmp/hub-media.json \
  --output /tmp/hub-submission-ready.json
```

`prepare` performs `POST /uploads/intents`, direct PUT, and completion for each item. It uses each returned `asset.url`; never construct an `assets.hub.kageos.ai` path manually. It writes ordered `gallery` entries and safe `<img>` blocks into `description_html`, then prints only a preparation summary. Review `/tmp/hub-submission-ready.json` and obtain explicit user confirmation before running `submit`.

The token should expire and be revoked when no longer needed. A publisher token must not carry administrator review privileges.
