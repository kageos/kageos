#!/usr/bin/env python3
"""把已经生成的 kageos 套件制品上传到目录、S3 兼容存储或 GitHub Release。"""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
from pathlib import Path


def release_files(release_dir: Path) -> list[Path]:
    required = [release_dir / "release.json", release_dir / "SHA256SUMS"]
    metadata = json.loads(required[0].read_text(encoding="utf-8"))
    required.append(release_dir / metadata["filename"])
    for path in required:
        if not path.is_file():
            raise ValueError(f"缺少发布制品：{path}")
    return required


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("release_dir", type=Path)
    destination = parser.add_mutually_exclusive_group(required=True)
    destination.add_argument("--directory", type=Path, help="复制到本地或已挂载的存储目录")
    destination.add_argument("--s3-uri", help="使用 aws CLI 上传，例如 s3://bucket/kageos")
    destination.add_argument("--github-repo", help="上传到已经存在的 GitHub Release，例如 kageos/kageos")
    parser.add_argument("--tag", help="GitHub Release tag；使用 --github-repo 时必填")
    args = parser.parse_args()

    release_dir = args.release_dir.expanduser().resolve()
    files = release_files(release_dir)
    if args.directory:
        target = args.directory.expanduser().resolve()
        target.mkdir(parents=True, exist_ok=True)
        for path in files:
            shutil.copy2(path, target / path.name)
        print(target)
        return 0
    if args.s3_uri:
        if not shutil.which("aws"):
            raise ValueError("当前环境没有 aws CLI")
        for path in files:
            subprocess.run(["aws", "s3", "cp", str(path), f"{args.s3_uri.rstrip('/')}/{path.name}"], check=True)
        return 0
    if not args.tag:
        raise ValueError("使用 --github-repo 时必须同时提供 --tag")
    if not shutil.which("gh"):
        raise ValueError("当前环境没有 gh CLI")
    subprocess.run(["gh", "release", "view", args.tag, "--repo", args.github_repo], check=True)
    subprocess.run(
        ["gh", "release", "upload", args.tag, *map(str, files), "--repo", args.github_repo, "--clobber"],
        check=True,
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, ValueError, subprocess.CalledProcessError, json.JSONDecodeError) as error:
        raise SystemExit(f"错误：{error}") from error
