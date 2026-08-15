#!/usr/bin/env python3
"""生成确定性的 kageos 套件发布包，并可同步到官网静态下载目录。"""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import subprocess
import sys
import zipfile
from datetime import datetime, timezone
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PLUGIN = ROOT / "plugins" / "kageos"
PACKAGE_SCRIPT = ROOT / "scripts" / "package-kageos-plugin.py"
DEFAULT_OUTPUT = ROOT.parent / "artifacts" / "kageos-plugin"
EXCLUDED_PARTS = {"__pycache__", ".DS_Store"}


def read_version() -> str:
    version = (PLUGIN / "VERSION").read_text(encoding="utf-8").strip()
    if not version or "+" in version:
        raise ValueError("VERSION 必须是无构建后缀的稳定语义版本")
    return version


def read_json(path: Path) -> dict:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"必须是 JSON 对象：{path}")
    return value


def validate_versions(version: str) -> None:
    codex = read_json(PLUGIN / ".codex-plugin" / "plugin.json")
    claude = read_json(PLUGIN / ".claude-plugin" / "plugin.json")
    compatibility = read_json(PLUGIN / "compatibility.json")
    facts = {
        "Codex manifest": codex.get("version"),
        "Claude manifest": claude.get("version"),
        "compatibility.json": compatibility.get("plugin_version"),
    }
    mismatches = [f"{name}={actual}" for name, actual in facts.items() if actual != version]
    if mismatches:
        raise ValueError(f"版本不一致，VERSION={version}；" + "；".join(mismatches))


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def plugin_files() -> list[Path]:
    return [
        path
        for path in sorted(PLUGIN.rglob("*"))
        if path.is_file() and not any(part in EXCLUDED_PARTS for part in path.parts)
    ]


def write_deterministic_zip(output: Path) -> None:
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for path in plugin_files():
            relative = path.relative_to(PLUGIN)
            info = zipfile.ZipInfo(str(Path("kageos") / relative), date_time=(2020, 1, 1, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = (path.stat().st_mode & 0xFFFF) << 16
            archive.writestr(info, path.read_bytes())


def sync_website(website_root: Path, release_dir: Path, metadata: dict) -> None:
    public = website_root.resolve() / "public" / "downloads" / "kageos"
    public.mkdir(parents=True, exist_ok=True)
    filename = metadata["filename"]
    shutil.copy2(release_dir / filename, public / filename)
    shutil.copy2(release_dir / "SHA256SUMS", public / f"{filename}.sha256")
    shutil.copy2(PLUGIN / "CHANGELOG.zh-CN.md", public / "CHANGELOG.zh-CN.md")
    (public / "latest.json").write_text(
        json.dumps(metadata, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    shutil.copy2(ROOT / "scripts" / "install-kageos-plugin.sh", website_root.resolve() / "public" / "install-kageos-plugin.sh")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-root", type=Path, default=DEFAULT_OUTPUT)
    parser.add_argument("--website-root", type=Path)
    args = parser.parse_args()

    subprocess.run([sys.executable, str(PACKAGE_SCRIPT)], check=True)
    version = read_version()
    validate_versions(version)
    release_dir = args.output_root.expanduser().resolve() / version
    release_dir.mkdir(parents=True, exist_ok=True)
    filename = f"kageos-plugin-{version}.zip"
    archive = release_dir / filename
    write_deterministic_zip(archive)
    digest = sha256(archive)
    (release_dir / "SHA256SUMS").write_text(f"{digest}  {filename}\n", encoding="utf-8")
    metadata = {
        "schema_version": "kageos.plugin-release.v1",
        "name": "kageos",
        "version": version,
        "language": "zh-CN",
        "filename": filename,
        "size": archive.stat().st_size,
        "sha256": digest,
        "download_url": f"https://kageos.ai/downloads/kageos/{filename}",
        "changelog_url": "https://kageos.ai/downloads/kageos/CHANGELOG.zh-CN.md",
        "released_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
    }
    (release_dir / "release.json").write_text(
        json.dumps(metadata, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    if args.website_root:
        sync_website(args.website_root, release_dir, metadata)
    print(release_dir)
    print(f"{filename}  sha256={digest}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, subprocess.CalledProcessError, json.JSONDecodeError) as error:
        raise SystemExit(f"错误：{error}") from error
