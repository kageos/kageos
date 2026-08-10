#!/usr/bin/env python3
"""Upload artifacts directly to R2 through Sona intents and submit to Hub."""

import argparse
import json
import mimetypes
import os
import pathlib
import sys
import urllib.error
import urllib.request


def request_json(url, token, method="GET", body=None):
    data = None if body is None else json.dumps(body).encode("utf-8")
    request = urllib.request.Request(url, data=data, method=method)
    request.add_header("Authorization", f"Bearer {token}")
    request.add_header("Accept", "application/json")
    if data is not None:
        request.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(request, timeout=65) as response:
            raw = response.read()
            return {} if not raw else json.loads(raw)
    except urllib.error.HTTPError as error:
        detail = error.read().decode("utf-8", "replace")
        raise RuntimeError(f"Hub returned HTTP {error.code}: {detail[:800]}") from error


def upload(base, token, filename, purpose, public):
    path = pathlib.Path(filename).resolve()
    content_type = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
    intent = request_json(f"{base}/uploads/intents", token, "POST", {
        "file_name": path.name, "content_type": content_type, "size": path.stat().st_size,
        "purpose": purpose, "visibility": "public" if public else "private",
    })
    upload_request = urllib.request.Request(intent["upload_url"], data=path.read_bytes(), method="PUT")
    upload_request.add_header("Content-Type", content_type)
    for key, value in (intent.get("headers") or {}).items():
        upload_request.add_header(key, value)
    with urllib.request.urlopen(upload_request, timeout=120):
        pass
    completed = request_json(f"{base}/uploads/intents/{intent['intent']['id']}/complete", token, "POST", {})
    print(json.dumps(completed, ensure_ascii=False))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", default=os.getenv("HUB_API_BASE", "https://hub.kageos.ai/api/v1"))
    sub = parser.add_subparsers(dest="command", required=True)
    upload_parser = sub.add_parser("upload")
    upload_parser.add_argument("file")
    upload_parser.add_argument("--purpose", choices=["attachment", "product-media"], required=True)
    upload_parser.add_argument("--public", action="store_true")
    for name in ("assist", "submit"):
        command = sub.add_parser(name)
        command.add_argument("json_file")
    sub.add_parser("status")
    args = parser.parse_args()
    token = os.getenv("HUB_PUBLISH_TOKEN", "").strip()
    if not token:
        raise SystemExit("HUB_PUBLISH_TOKEN is required")
    base = args.base.rstrip("/")
    if args.command == "upload":
        upload(base, token, args.file, args.purpose, args.public)
    elif args.command == "status":
        print(json.dumps(request_json(f"{base}/hub/submissions", token), ensure_ascii=False))
    else:
        payload = json.loads(pathlib.Path(args.json_file).read_text(encoding="utf-8"))
        suffix = "assist" if args.command == "assist" else ""
        endpoint = f"{base}/hub/submissions" + (f"/{suffix}" if suffix else "")
        print(json.dumps(request_json(endpoint, token, "POST", payload), ensure_ascii=False))


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, urllib.error.URLError, KeyError, json.JSONDecodeError) as error:
        print(str(error), file=sys.stderr)
        raise SystemExit(1)
