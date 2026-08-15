#!/usr/bin/env python3
"""Upload artifacts directly to R2 through Sona intents and submit to Hub."""

import argparse
import copy
import html
import json
import mimetypes
import os
import pathlib
import sys
import urllib.error
import urllib.parse
import urllib.request


USER_AGENT = "kageos-Hub-Publisher/1.0"


def request_json(url, token, method="GET", body=None):
    data = None if body is None else json.dumps(body).encode("utf-8")
    request = urllib.request.Request(url, data=data, method=method)
    request.add_header("Authorization", f"Bearer {token}")
    request.add_header("Accept", "application/json")
    request.add_header("User-Agent", USER_AGENT)
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
    if not path.is_file():
        raise ValueError(f"Upload file does not exist: {path}")
    content_type = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
    intent = request_json(f"{base}/uploads/intents", token, "POST", {
        "file_name": path.name, "content_type": content_type, "size": path.stat().st_size,
        "purpose": purpose, "visibility": "public" if public else "private",
    })
    upload_request = urllib.request.Request(intent["upload_url"], data=path.read_bytes(), method="PUT")
    upload_request.add_header("Content-Type", content_type)
    upload_request.add_header("User-Agent", USER_AGENT)
    for key, value in (intent.get("headers") or {}).items():
        upload_request.add_header(key, value)
    with urllib.request.urlopen(upload_request, timeout=120):
        pass
    return request_json(f"{base}/uploads/intents/{intent['intent']['id']}/complete", token, "POST", {})


def read_json_object(filename, label):
    path = pathlib.Path(filename).resolve()
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{label} must be a JSON object")
    return value, path


def normalize_media_manifest(filename):
    manifest, path = read_json_object(filename, "Media manifest")
    raw_items = manifest.get("items")
    if not isinstance(raw_items, list) or not raw_items:
        raise ValueError("Media manifest items must contain at least one screenshot or video")
    if len(raw_items) > 8:
        raise ValueError("Hub accepts at most 8 gallery items")

    items = []
    for index, raw_item in enumerate(raw_items, start=1):
        if not isinstance(raw_item, dict):
            raise ValueError(f"Media item {index} must be a JSON object")
        raw_file = str(raw_item.get("file") or "").strip()
        alt = str(raw_item.get("alt") or "").strip()
        caption = str(raw_item.get("caption") or "").strip()
        if not raw_file:
            raise ValueError(f"Media item {index} is missing file")
        media_path = pathlib.Path(raw_file).expanduser()
        if not media_path.is_absolute():
            media_path = path.parent / media_path
        media_path = media_path.resolve()
        if not media_path.is_file():
            raise ValueError(f"Media item {index} does not exist: {media_path}")
        content_type = mimetypes.guess_type(media_path.name)[0] or ""
        if not (content_type.startswith("image/") or content_type.startswith("video/")):
            raise ValueError(f"Media item {index} must be an image or video")
        if not alt or len(alt) > 160:
            raise ValueError(f"Media item {index} alt must contain 1-160 characters")
        if not caption or len(caption) > 300:
            raise ValueError(f"Media item {index} caption must contain 1-300 characters")
        include_in_description = raw_item.get("include_in_description", content_type.startswith("image/"))
        if not isinstance(include_in_description, bool):
            raise ValueError(f"Media item {index} include_in_description must be a boolean")
        if include_in_description and not content_type.startswith("image/"):
            raise ValueError(f"Media item {index} cannot embed video in description_html")
        items.append({
            "file": str(media_path),
            "content_type": content_type,
            "alt": alt,
            "caption": caption,
            "include_in_description": include_in_description,
        })

    gallery_mode = str(manifest.get("gallery_mode") or "replace").strip().lower()
    if gallery_mode not in ("replace", "append"):
        raise ValueError("gallery_mode must be replace or append")
    heading = str(manifest.get("description_heading") or "真实使用效果").strip()
    if not heading or len(heading) > 120:
        raise ValueError("description_heading must contain 1-120 characters")
    append_to_description = manifest.get("append_to_description", True)
    if not isinstance(append_to_description, bool):
        raise ValueError("append_to_description must be a boolean")
    return {
        "items": items,
        "gallery_mode": gallery_mode,
        "append_to_description": append_to_description,
        "description_heading": heading,
    }


def validate_public_asset(asset, index):
    if not isinstance(asset, dict):
        raise ValueError(f"Completed media upload {index} has no asset")
    url = str(asset.get("url") or "").strip()
    parsed = urllib.parse.urlparse(url)
    if parsed.scheme not in ("http", "https") or not parsed.netloc:
        raise ValueError(f"Completed media upload {index} has no public URL")
    content_type = str(asset.get("content_type") or "").strip().lower()
    if not (content_type.startswith("image/") or content_type.startswith("video/")):
        raise ValueError(f"Completed media upload {index} is not an image or video")
    return url, "video" if content_type.startswith("video/") else "image"


def compose_submission(submission, manifest, assets):
    if not isinstance(submission, dict):
        raise ValueError("Submission must be a JSON object")
    if len(assets) != len(manifest["items"]):
        raise ValueError("Uploaded asset count does not match media manifest")

    uploaded_gallery = []
    inline_images = []
    for index, (item, asset) in enumerate(zip(manifest["items"], assets), start=1):
        url, kind = validate_public_asset(asset, index)
        uploaded_gallery.append({
            "url": url,
            "kind": kind,
            "alt": item["alt"],
            "caption": item["caption"],
        })
        if item["include_in_description"]:
            inline_images.append((url, item["alt"], item["caption"]))

    existing_gallery = submission.get("gallery") or []
    if not isinstance(existing_gallery, list):
        raise ValueError("Submission gallery must be a JSON array")
    combined = uploaded_gallery if manifest["gallery_mode"] == "replace" else existing_gallery + uploaded_gallery
    deduplicated = []
    seen_urls = set()
    for item in combined:
        if not isinstance(item, dict) or not str(item.get("url") or "").strip():
            raise ValueError("Submission gallery contains an invalid item")
        url = str(item["url"]).strip()
        if url in seen_urls:
            continue
        seen_urls.add(url)
        deduplicated.append(item)
    if len(deduplicated) > 8:
        raise ValueError("Combined submission gallery exceeds 8 items")
    submission["gallery"] = deduplicated

    if manifest["append_to_description"] and inline_images:
        blocks = [f"<h2>{html.escape(manifest['description_heading'])}</h2>"]
        for url, alt, caption in inline_images:
            safe_url = html.escape(url, quote=True)
            safe_alt = html.escape(alt, quote=True)
            safe_caption = html.escape(caption)
            blocks.extend([
                f"<h3>{html.escape(alt)}</h3>",
                f'<p><img src="{safe_url}" alt="{safe_alt}" title="{html.escape(caption, quote=True)}"></p>',
                f"<p>{safe_caption}</p>",
            ])
        existing_description = str(submission.get("description_html") or "").strip()
        submission["description_html"] = "\n".join(filter(None, [existing_description, *blocks]))
    return submission


def prepare_submission(base, token, submission_file, manifest_file, output_file):
    submission, submission_path = read_json_object(submission_file, "Submission")
    manifest = normalize_media_manifest(manifest_file)
    output_path = pathlib.Path(output_file).resolve()
    manifest_path = pathlib.Path(manifest_file).resolve()
    if output_path in (submission_path, manifest_path):
        raise ValueError("Prepared output must not overwrite the submission or media manifest")
    if not output_path.parent.is_dir():
        raise ValueError(f"Prepared output directory does not exist: {output_path.parent}")
    placeholder_assets = [
        {"url": f"https://upload-preflight.invalid/{index}", "content_type": item["content_type"]}
        for index, item in enumerate(manifest["items"], start=1)
    ]
    compose_submission(copy.deepcopy(submission), manifest, placeholder_assets)

    assets = []
    for item in manifest["items"]:
        completed = upload(base, token, item["file"], "product-media", True)
        assets.append(completed.get("asset"))
    prepared = compose_submission(submission, manifest, assets)
    output_path.write_text(json.dumps(prepared, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return {
        "output": str(output_path),
        "uploaded": len(assets),
        "gallery_count": len(prepared.get("gallery") or []),
        "inline_image_count": sum(1 for item in manifest["items"] if item["include_in_description"]),
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", default=os.getenv("HUB_API_BASE", "https://hub.kageos.ai/api/v1"))
    sub = parser.add_subparsers(dest="command", required=True)
    upload_parser = sub.add_parser("upload")
    upload_parser.add_argument("file")
    upload_parser.add_argument("--purpose", choices=["attachment", "product-media"], required=True)
    upload_parser.add_argument("--public", action="store_true")
    prepare_parser = sub.add_parser("prepare")
    prepare_parser.add_argument("submission_json")
    prepare_parser.add_argument("media_manifest")
    prepare_parser.add_argument("--output", required=True)
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
        print(json.dumps(upload(base, token, args.file, args.purpose, args.public), ensure_ascii=False))
    elif args.command == "prepare":
        result = prepare_submission(base, token, args.submission_json, args.media_manifest, args.output)
        print(json.dumps(result, ensure_ascii=False))
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
    except (RuntimeError, urllib.error.URLError, KeyError, OSError, ValueError, json.JSONDecodeError) as error:
        print(str(error), file=sys.stderr)
        raise SystemExit(1)
