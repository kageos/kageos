# Complex input contract

Resolve complex widget values through their real platform APIs before submitting a Form or Table request. Apply the same authentication and audit headers used for workspace operations.

## Files

A `files` field stores stable object refs as one comma-separated string. It does not store JSON, local paths, or expiring download URLs.

Before upload, enforce the discovered widget configuration: `accept`, `max_size`, `max_count`, and any required-field rule. Compute the actual byte size, MIME type, and SHA-256 hash locally.

Use this direct HTTP sequence for each local file:

1. Request credentials:

```http
POST /storage/api/v1/upload_token
```

```json
{
  "router": "system/democase/example/example.form",
  "file_name": "evidence.png",
  "content_type": "image/png",
  "file_size": 12345,
  "hash": "<sha256>",
  "upload_source": "server"
}
```

Use the target function `full_code_path` without the leading slash as `router`. The response must contain `method: presigned_url`, `key`, `bucket`, `ref`, exact upload `headers`, and a usable `server_upload_url` or `upload_url`.

2. Upload raw bytes with `PUT` to `server_upload_url` when available; otherwise use `upload_url`. Send the exact returned storage headers. Do not send KageOS authentication headers to the presigned storage URL. Never print or persist the signed URL.

3. Notify completion:

```http
POST /storage/api/v1/upload_complete
```

```json
{
  "key": "<returned key>",
  "bucket": "<returned bucket>",
  "success": true,
  "router": "system/democase/example/example.form",
  "file_name": "evidence.png",
  "file_size": 12345,
  "content_type": "image/png",
  "hash": "<sha256>"
}
```

Require API `code: 0` and capture the stable returned `ref`. For multiple files, preserve user order and submit `ref1,ref2`. Batch credential and completion endpoints exist at `/storage/api/v1/batch_upload_token` and `/storage/api/v1/batch_upload_complete`, but use them only when partial-failure handling is explicit.

4. Verify refs:

```http
POST /storage/api/v1/files/resolve
```

```json
{"refs":["<ref>"],"audience":"all"}
```

Assert the expected file name, byte size, content type, and hash. Treat returned download URLs as temporary access URLs, not stored field values.

5. Cleanup only run-owned uploads after removing every business record that references them:

```http
DELETE /storage/api/v1/files/<URL-encoded key>
```

Do not delete an object merely because a Form/Table call failed; first determine whether another successful record references the same run-owned ref.

## User and users

Never invent a username or submit a display name, nickname, or email as a user field.

Use the authenticated HR APIs:

| Purpose | Method | Endpoint |
| --- | --- | --- |
| Resolve the current authenticated user | `GET` | `/hr/api/v1/user/info` |
| Search candidates | `GET` | `/hr/api/v1/user/search_fuzzy?keyword=<text>&limit=20` |
| Hydrate one username | `GET` | `/hr/api/v1/user/query?username=<username>` |
| Hydrate several usernames | `POST` | `/hr/api/v1/users` with `{"usernames":[...]}` |

For a `user` widget, submit exactly one verified `username` string. For a `users` widget, submit verified usernames as one comma-separated string in the selected order. Resolve `Me()` to the concrete username returned by `/user/info`; do not send UI default expressions through raw HTTP. Hydrate every selected username immediately before a write and reject missing, disabled, or ambiguous candidates.

Do not expose emails, signatures, avatars, department data, or other profile fields in verification reports. Record only that the intended username count and identity checks passed.

## Rich text

A `richtext` field submits an HTML string. Use simple valid HTML and never submit base64 media, scripts, event handlers, iframes, objects, embeds, `srcdoc`, or unsafe URL schemes.

For images, video, or downloadable files inside rich text:

1. Upload and complete each asset through the Files sequence above.
2. Use the completion response `download_url` as the immediate `src` or `href` and retain the stable ref in a `data-file-ref` attribute for later resolution.
3. Insert images as `<img src="..." alt="..." data-file-ref="bucket/key">`.
4. Insert downloadable files as `<a href="..." data-file-ref="bucket/key" target="_blank" rel="noopener noreferrer">name</a>`.
5. Insert video only when the widget and content type support it; include `controls` and the stable `data-file-ref`.
6. Read the saved record back and assert that the sanitized HTML retains the intended safe element, alt/link text, and stable ref without embedding credentials.

The browser editor already uses the storage credential, upload, and completion APIs for pasted or dropped assets. Direct HTTP operation must mirror that flow instead of embedding a local path or data URL.

Cleanup the business record before deleting run-owned rich-text assets. A report may name the widget and media type but must not include signed URLs, raw HTML containing private content, or object keys.
